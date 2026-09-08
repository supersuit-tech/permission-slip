import type { components } from "@/api/schema";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { META_NAMESPACE_KEY } from "@/lib/constraints";
import { resourceDetailsToConstraintMeta } from "@/lib/approvalConstraintMeta";
import { constraintsObjectHasNonWildcard } from "@/lib/structuredConstraints";
import {
  GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES,
  buildSharedDriveWorkspaceConstraints,
  isGoogleSharedDriveWorkspaceAction,
  sharedDriveIdFromResourceDetails,
  sharedDriveWorkspaceStandingApprovalName,
} from "./googleSharedDriveWorkspace";

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

function deriveSharedDriveConstraint(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> | null {
  if (!isGoogleSharedDriveWorkspaceAction(actionType)) {
    return null;
  }
  const driveId = sharedDriveIdFromResourceDetails(resourceDetails);
  if (!driveId) {
    return null;
  }
  return buildSharedDriveWorkspaceConstraints(actionType, driveId);
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

  const driveConstraint = deriveSharedDriveConstraint(
    approval.action.type,
    resourceDetails,
  );
  if (driveConstraint) {
    return driveConstraint;
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

  const constraints = deriveStandingApprovalConstraints(approval);

  return {
    agent_id: approval.agent_id,
    action_type: approval.action.type,
    action_version: version,
    name: description ?? approval.action.type,
    description,
    constraints,
    expires_at: null,
    ...(!constraintsObjectHasNonWildcard(constraints)
      ? { confirm_unrestricted: true }
      : {}),
  };
}

function createRequestForAction(
  approval: ApprovalSummary,
  actionType: string,
  constraints: Record<string, unknown>,
  name: string,
  description: string | null,
): CreateStandingApprovalRequest {
  const version =
    typeof approval.action.version === "string" && approval.action.version !== ""
      ? approval.action.version
      : "1";
  return {
    agent_id: approval.agent_id,
    action_type: actionType,
    action_version: version,
    name,
    description,
    constraints,
    expires_at: null,
    ...(!constraintsObjectHasNonWildcard(constraints)
      ? { confirm_unrestricted: true }
      : {}),
  };
}

/**
 * One or more standing approvals to create from "always allow".
 * Shared Drive workspace actions expand into the Drive + Sheets set.
 */
export function buildCreateStandingApprovalsFromApproval(
  approval: ApprovalSummary,
): CreateStandingApprovalRequest[] {
  const resourceDetails = approval.resource_details as
    | Record<string, unknown>
    | undefined;
  const driveId = sharedDriveIdFromResourceDetails(resourceDetails);
  if (
    !driveId ||
    !isGoogleSharedDriveWorkspaceAction(approval.action.type)
  ) {
    return [buildCreateStandingApprovalFromApproval(approval)];
  }

  const description = "Routine Drive and Sheets work inside this Shared Drive";
  return GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES.map((actionType) =>
    createRequestForAction(
      approval,
      actionType,
      buildSharedDriveWorkspaceConstraints(actionType, driveId),
      sharedDriveWorkspaceStandingApprovalName(actionType, resourceDetails),
      description,
    ),
  );
}

export function standingApprovalsToCreateFromApproval(
  approval: ApprovalSummary,
  existing: Array<{ agent_id: number; action_type: string }>,
): CreateStandingApprovalRequest[] {
  const existingTypes = new Set(
    existing
      .filter((sa) => sa.agent_id === approval.agent_id)
      .map((sa) => sa.action_type),
  );
  return buildCreateStandingApprovalsFromApproval(approval).filter(
    (req) => !existingTypes.has(req.action_type),
  );
}
