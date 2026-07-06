import type { components } from "@/api/schema";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { META_NAMESPACE_KEY } from "@/lib/constraints";
import { resourceDetailsToConstraintMeta } from "@/lib/approvalConstraintMeta";

type CreateStandingApprovalRequest =
  components["schemas"]["CreateStandingApprovalRequest"];

/** UID-targeted Proton Mail actions where message_id is ephemeral. */
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

/**
 * Derive initial constraints from an approval's action parameters.
 * Each parameter becomes a fixed constraint pinned to its exact value.
 */
export function deriveConstraintsFromParams(
  parameters: Record<string, unknown>,
): Record<string, unknown> {
  const constraints: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(parameters)) {
    constraints[key] = value ?? "";
  }
  return constraints;
}

export function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return {};
  }
  return deriveConstraintsFromParams(params);
}

function deriveEmailSenderConstraint(
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> | null {
  const meta = resourceDetailsToConstraintMeta(resourceDetails);
  const sender = meta?.sender;
  if (typeof sender !== "string" || sender.length === 0) {
    return null;
  }
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
): Record<string, unknown> {
  const params = approval.action.parameters as Record<string, unknown>;
  const resourceDetails = approval.resource_details as
    | Record<string, unknown>
    | undefined;

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

export function buildCreateStandingApprovalFromApproval(
  approval: ApprovalSummary,
): CreateStandingApprovalRequest {
  const version =
    typeof approval.action.version === "string" && approval.action.version !== ""
      ? approval.action.version
      : "1";

  const description =
    typeof approval.context.description === "string" &&
    approval.context.description.trim() !== ""
      ? approval.context.description.trim()
      : null;

  return {
    agent_id: approval.agent_id,
    action_type: approval.action.type,
    action_version: version,
    name: description ?? approval.action.type,
    description,
    constraints: deriveStandingApprovalConstraints(approval),
    expires_at: null,
  };
}
