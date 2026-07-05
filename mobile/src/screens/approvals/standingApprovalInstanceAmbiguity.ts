export function hasFrozenStandingApprovalInstanceDisplay(
  connectorInstanceDisplay: string | null | undefined,
): boolean {
  return (
    typeof connectorInstanceDisplay === "string" &&
    connectorInstanceDisplay.trim() !== ""
  );
}

export type ConnectorInstanceForScope = {
  display?: string | null;
};

/**
 * Returns a neutral scope label for rule-proposal approval screens.
 * Resolution order:
 * 1. frozen connector_instance_display → "Applies to <name>"
 * 2. exactly one instance → "Applies to <that account's display name>"
 * 3. otherwise → "Applies to all accounts"
 *
 * Returns null while instance count is still loading to avoid a flash.
 */
export function getStandingApprovalInstanceScopeLabel(
  connectorInstanceDisplay: string | null | undefined,
  instances: ConnectorInstanceForScope[],
  instanceCountLoaded: boolean,
): string | null {
  if (!instanceCountLoaded) {
    return null;
  }

  if (hasFrozenStandingApprovalInstanceDisplay(connectorInstanceDisplay)) {
    return `Applies to ${connectorInstanceDisplay!.trim()}`;
  }

  if (instances.length === 1) {
    const display = instances[0]?.display?.trim();
    return display ? `Applies to ${display}` : "Applies to this account";
  }

  return "Applies to all accounts";
}
