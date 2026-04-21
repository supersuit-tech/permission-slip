/** First segment of an action type, title-cased (e.g. slack.send_message → Slack). */
function humanizeConnectorPrefix(actionType: string): string {
  const dot = actionType.indexOf(".");
  const prefix = dot > 0 ? actionType.slice(0, dot) : actionType;
  if (!prefix) return actionType;
  return prefix.charAt(0).toUpperCase() + prefix.slice(1);
}

/**
 * Connector line for approval UI: "Slack (Engineering)" when a multi-instance label is frozen on the action.
 */
export function formatConnectorDisplayName(args: {
  connectorName: string | null | undefined;
  actionType: string;
  instanceLabel?: string | null;
}): string {
  const base =
    args.connectorName?.trim() || humanizeConnectorPrefix(args.actionType);
  const inst = args.instanceLabel?.trim();
  if (inst) {
    return `${base} (${inst})`;
  }
  return base;
}
