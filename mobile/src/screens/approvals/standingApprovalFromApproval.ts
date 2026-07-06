import type { components } from "../../api/schema";
import type { ApprovalSummary } from "../../hooks/useApprovals";
import { findMatchingActionConfigForApproval } from "./matchActionConfig";

type CreateStandingApprovalRequest =
  components["schemas"]["CreateStandingApprovalRequest"];

const META_NAMESPACE_KEY = "$meta";

const UID_EMAIL_ACTION_TYPES = new Set([
  "protonmail.read_email",
  "protonmail.archive_email",
  "protonmail.reply_email",
  "protonmail.mark_read",
  "protonmail.mark_unread",
  "protonmail.flag",
  "protonmail.unflag",
  "protonmail.move_to_folder",
  "protonmail.delete",
  "protonmail.apply_label",
  "protonmail.remove_label",
]);

function deriveConstraintsFromParams(
  parameters: Record<string, unknown>,
): Record<string, unknown> {
  const constraints: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(parameters)) {
    constraints[key] = value ?? "";
  }
  return constraints;
}

function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
): Record<string, unknown> {
  if (Object.keys(params).length === 0) {
    return {};
  }
  return deriveConstraintsFromParams(params);
}

function normalizeEmailAddress(raw: string): string {
  const trimmed = raw.trim();
  const match = /<([^>]+)>/.exec(trimmed);
  return (match?.[1] ?? trimmed).trim();
}

function primarySenderAddress(from: unknown): string | null {
  if (typeof from === "string" && from.length > 0) return normalizeEmailAddress(from);
  if (Array.isArray(from)) {
    const first = from.find((value): value is string => typeof value === "string" && value.length > 0);
    return first ? normalizeEmailAddress(first) : null;
  }
  return null;
}

function deriveEmailSenderConstraint(
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> | null {
  if (!resourceDetails) return null;
  const source =
    resourceDetails.in_reply_to && typeof resourceDetails.in_reply_to === "object"
      ? (resourceDetails.in_reply_to as Record<string, unknown>)
      : resourceDetails;
  const sender = primarySenderAddress(source.from);
  if (!sender) return null;
  return {
    message_id: "*",
    folder: "*",
    [META_NAMESPACE_KEY]: {
      from: sender,
    },
  };
}

function deriveStandingApprovalConstraints(
  approval: ApprovalSummary,
  sourceActionConfigurationId?: string,
): Record<string, unknown> {
  if (sourceActionConfigurationId !== undefined) {
    return {};
  }

  const params = approval.action.parameters as Record<string, unknown>;
  const resourceDetails = approval.resource_details as Record<string, unknown> | undefined;

  if (UID_EMAIL_ACTION_TYPES.has(approval.action.type)) {
    const senderConstraint = deriveEmailSenderConstraint(resourceDetails);
    if (senderConstraint) {
      return senderConstraint;
    }
    const constraints = deriveConstraintsFromParams(params);
    constraints.message_id = "*";
    return constraints;
  }

  return standingApprovalConstraintsForCreate(params);
}

export { findMatchingActionConfigForApproval };

export function buildCreateStandingApprovalFromApproval(
  approval: ApprovalSummary,
  sourceActionConfigurationId?: string,
): CreateStandingApprovalRequest {
  const version =
    typeof approval.action.version === "string" && approval.action.version !== ""
      ? approval.action.version
      : "1";

  const request: CreateStandingApprovalRequest = {
    agent_id: approval.agent_id,
    action_type: approval.action.type,
    action_version: version,
    constraints: deriveStandingApprovalConstraints(approval, sourceActionConfigurationId),
    expires_at: null,
  };

  if (sourceActionConfigurationId !== undefined) {
    request.source_action_configuration_id = sourceActionConfigurationId;
  }

  return request;
}
