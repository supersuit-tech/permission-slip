export const STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING =
  "Applies to an unspecified account";

/** True when the proposal has a frozen instance label from the server. */
export function hasFrozenStandingApprovalInstanceDisplay(
  connectorInstanceDisplay: string | null | undefined,
): boolean {
  return (
    typeof connectorInstanceDisplay === "string" &&
    connectorInstanceDisplay.trim() !== ""
  );
}

/**
 * Show a reviewer warning when the proposal has no instance label but the agent
 * has multiple enabled connector instances. Pass `undefined` for instanceCount
 * while loading to avoid a flash of the warning.
 */
export function shouldShowStandingApprovalInstanceAmbiguityWarning(
  connectorInstanceDisplay: string | null | undefined,
  instanceCount: number | undefined,
): boolean {
  if (hasFrozenStandingApprovalInstanceDisplay(connectorInstanceDisplay)) {
    return false;
  }
  return instanceCount !== undefined && instanceCount > 1;
}
