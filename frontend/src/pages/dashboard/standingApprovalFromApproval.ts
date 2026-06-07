import type { components } from "@/api/schema";
import type { ApprovalSummary } from "@/hooks/useApprovals";

type CreateStandingApprovalRequest =
  components["schemas"]["CreateStandingApprovalRequest"];

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

/** With source_action_configuration_id, `{}` means match-all (backend stores NULL constraints). */
export function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return {};
  }
  return deriveConstraintsFromParams(params);
}

export function buildCreateStandingApprovalFromApproval(
  approval: ApprovalSummary,
  sourceActionConfigurationId?: string,
): CreateStandingApprovalRequest {
  const params = approval.action.parameters as Record<string, unknown>;
  const version =
    typeof approval.action.version === "string" && approval.action.version !== ""
      ? approval.action.version
      : "1";

  const request: CreateStandingApprovalRequest = {
    agent_id: approval.agent_id,
    action_type: approval.action.type,
    action_version: version,
    constraints: standingApprovalConstraintsForCreate(params),
    expires_at: null,
  };

  if (sourceActionConfigurationId !== undefined) {
    request.source_action_configuration_id = sourceActionConfigurationId;
  }

  return request;
}
