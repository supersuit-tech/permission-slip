/**
 * Pure utility functions for the approval screens — formatting, date math,
 * and action summary generation. No React dependencies; fully unit-testable.
 */

/** Safely coerce action parameters to a plain object. */
export function safeParams(raw: unknown): Record<string, unknown> {
  if (raw != null && typeof raw === "object" && !Array.isArray(raw)) {
    return raw as Record<string, unknown>;
  }
  return {};
}

/** Returns true when a pending approval has passed its expiry. */
export function isExpired(status: string, expiresAt: string): boolean {
  return status === "pending" && secondsUntil(expiresAt) <= 0;
}

/** Formats a param/context value for display. */
export function formatParamValue(value: unknown): string {
  if (value == null) return "null";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) {
    return value.map((v) => formatParamValue(v)).join(", ");
  }
  return JSON.stringify(value, null, 2);
}

/** Formats an ISO timestamp for display (e.g. "Jan 5, 3:04 PM"). */
export function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  if (isNaN(date.getTime())) return iso;
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** Returns seconds remaining until `expiresAt`, clamped to 0. */
export function secondsUntil(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now();
  return Math.max(0, Math.floor(diff / 1_000));
}

/** Formats seconds as "M:SS". */
export function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

/**
 * Converts an action type string into a readable label.
 * Extracts the operation portion (last segment after the final dot),
 * replaces underscores with spaces, and capitalizes the first letter.
 */
export function humanizeActionType(actionType: string): string {
  const parts = actionType.split(".");
  const operation =
    parts.length >= 2 ? (parts[parts.length - 1] ?? actionType) : actionType;
  const words = operation.replace(/_/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/** Builds a plain-text summary for an action, similar to the web frontend. */
export function buildActionSummary(
  actionType: string,
  parameters: Record<string, unknown>,
): string {
  const formatter = ACTION_FORMATTERS[actionType];
  if (formatter) {
    const result = formatter(parameters);
    if (result) return result;
  }
  return buildGenericSummary(actionType, parameters);
}

type ActionFormatter = (params: Record<string, unknown>) => string | null;

/** Extracts a non-empty string from an unknown value, or returns null. */
function strVal(v: unknown): string | null {
  if (typeof v === "string" && v.length > 0) return v;
  return null;
}

/** Truncates a string to `maxLen` characters, appending an ellipsis if needed. */
function truncate(s: string, maxLen: number): string {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen - 1) + "\u2026";
}

/** Formats a recipient list (string or string[]) for display in email summaries. */
function formatRecipients(v: unknown): string | null {
  if (typeof v === "string") return v;
  if (Array.isArray(v)) {
    const strs = v.filter((x): x is string => typeof x === "string");
    if (strs.length === 0) return null;
    if (strs.length <= 3) return strs.join(", ");
    return `${strs[0] ?? ""}, ${strs[1] ?? ""}, and ${strs.length - 2} more`;
  }
  return null;
}

/**
 * Action-specific formatters keyed by action type. Each returns a
 * human-readable summary string, or null to fall back to the generic formatter.
 */
const ACTION_FORMATTERS: Record<string, ActionFormatter> = {
  "github.create_issue": (params) => {
    const owner = strVal(params.owner);
    const repo = strVal(params.repo);
    const title = strVal(params.title);
    if (!title) return null;
    const repoRef = owner && repo ? `${owner}/${repo}` : repo;
    let result = `Create issue \u201C${title}\u201D`;
    if (repoRef) result += ` in ${repoRef}`;
    return result;
  },

  "github.merge_pr": (params) => {
    const owner = strVal(params.owner);
    const repo = strVal(params.repo);
    const prNumber = params.pull_number;
    if (prNumber == null) return null;
    const repoRef = owner && repo ? `${owner}/${repo}` : repo;
    let result = `Merge PR #${String(prNumber)}`;
    if (repoRef) result += ` in ${repoRef}`;
    return result;
  },

  "slack.send_message": (params) => {
    const channel = strVal(params.channel);
    if (!channel) return null;
    const message = strVal(params.message);
    let result = `Send message to ${channel}`;
    if (message) result += ` \u2014 ${truncate(message, 60)}`;
    return result;
  },

  "email.send": (params) => {
    const to = formatRecipients(params.to);
    const subject = strVal(params.subject);
    if (!to) return null;
    let result = `Send email to ${to}`;
    if (subject) result += ` \u2014 ${truncate(subject, 60)}`;
    return result;
  },
};

/**
 * Fallback summary builder when no action-specific formatter matches.
 * Constructs a label from the action type plus up to 3 parameter highlights.
 */
function buildGenericSummary(
  actionType: string,
  parameters: Record<string, unknown>,
): string {
  const label = humanizeActionType(actionType);
  const entries = Object.entries(parameters);
  if (entries.length === 0) return label;

  const highlights: string[] = [];
  for (const [key, value] of entries) {
    if (highlights.length >= 3) break;
    if (value == null) continue;
    const display = formatValue(value);
    if (display) highlights.push(`${key}: ${display}`);
  }

  if (highlights.length === 0) return label;
  return `${label}: ${highlights.join(", ")}`;
}

/** Formats an unknown value as a short display string for generic summaries. */
function formatValue(value: unknown): string | null {
  if (typeof value === "string") return truncate(value, 40);
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) {
    const strs = value
      .filter((x) => typeof x === "string" || typeof x === "number")
      .map(String);
    if (strs.length === 0) return null;
    if (strs.length <= 2) return strs.join(", ");
    return `${strs[0] ?? ""}, +${strs.length - 1} more`;
  }
  return null;
}

/**
 * Formats an ISO timestamp as a relative time string for list rows.
 * Shows "Just now", "2m ago", "1h ago", or falls back to date for older items.
 */
export function formatRelativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const diffSec = Math.floor(diffMs / 1_000);

  if (diffSec < 60) return "Just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay}d ago`;

  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}
