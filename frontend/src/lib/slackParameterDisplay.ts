/**
 * Maps action parameters to human-readable display when backend resource_details
 * resolved Slack IDs (e.g. channel_name for slack.send_message).
 */
export function slackResolvedDisplayValue(
  actionType: string,
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  if (resourceDetails == null) return null;
  if (typeof rawValue !== "string" || rawValue.length === 0) return null;

  if (paramKey === "channel") {
    const name = resourceDetails.channel_name;
    if (typeof name === "string" && name.length > 0 && name !== rawValue) {
      return `${name} (${rawValue})`;
    }
    return null;
  }

  if (paramKey === "user_id" && actionType === "slack.send_dm") {
    const name = resourceDetails.user_name;
    if (typeof name === "string" && name.length > 0 && name !== rawValue) {
      return `${name} (${rawValue})`;
    }
  }

  return null;
}
