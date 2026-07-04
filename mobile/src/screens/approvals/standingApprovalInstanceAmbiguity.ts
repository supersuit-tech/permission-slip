export const STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING =
  "Applies to an unspecified account";

export function hasFrozenStandingApprovalInstanceDisplay(
  connectorInstanceDisplay: string | null | undefined,
): boolean {
  return (
    typeof connectorInstanceDisplay === "string" &&
    connectorInstanceDisplay.trim() !== ""
  );
}

export function shouldShowStandingApprovalInstanceAmbiguityWarning(
  connectorInstanceDisplay: string | null | undefined,
  instanceCount: number | undefined,
): boolean {
  if (hasFrozenStandingApprovalInstanceDisplay(connectorInstanceDisplay)) {
    return false;
  }
  return instanceCount !== undefined && instanceCount > 1;
}
