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

/** First segment of an action type, title-cased (e.g. slack.send_message → Slack). */
export function humanizeConnectorPrefix(actionType: string): string {
  const dot = actionType.indexOf(".");
  const prefix = dot > 0 ? actionType.slice(0, dot) : actionType;
  if (!prefix) return actionType;
  return prefix.charAt(0).toUpperCase() + prefix.slice(1);
}

/** Reads frozen multi-instance label from stored action JSON, if present. */
export function connectorInstanceLabelFromAction(action: {
  _connector_instance_label?: unknown;
}): string | undefined {
  const v = action._connector_instance_label;
  if (typeof v === "string" && v.trim() !== "") return v;
  return undefined;
}

/** Builds a plain-text summary for an action, similar to the web frontend. */
export function buildActionSummary(
  actionType: string,
  parameters: Record<string, unknown>,
  displayTemplate?: string | null,
  resourceDetails?: Record<string, unknown> | null,
): string {
  // 1. Try display template from manifest.
  // Merge resourceDetails so templates can resolve human-readable names (#862).
  if (displayTemplate) {
    const lookup = resourceDetails
      ? { ...parameters, ...resourceDetails }
      : parameters;
    const result = renderDisplayTemplate(displayTemplate, lookup);
    if (result) return result;
  }

  // 2. Try action-specific formatter.
  const formatter = ACTION_FORMATTERS[actionType];
  if (formatter) {
    const result = formatter(parameters, resourceDetails ?? undefined);
    if (result) return result;
  }

  // 3. Fall back to generic.
  return buildGenericSummary(actionType, parameters, resourceDetails);
}

type ActionFormatter = (params: Record<string, unknown>, resourceDetails?: Record<string, unknown>) => string | null;

/** Pattern matching {{param}} and {{param:directive}} placeholders. */
const TEMPLATE_RE = /\{\{(\w+)(?::(\w+))?\}\}/g;

/**
 * Plain-text display template renderer for mobile.
 * Mirrors the web displayTemplate.ts logic but outputs a plain string
 * instead of SummaryPart[].
 *
 * Supports directives: :datetime (human-readable date), :count (array length).
 */
function renderDisplayTemplate(
  template: string,
  parameters: Record<string, unknown>,
): string | null {
  let result = "";
  let lastIndex = 0;
  let hasValue = false;

  TEMPLATE_RE.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = TEMPLATE_RE.exec(template)) !== null) {
    result += template.slice(lastIndex, match.index);
    const paramName = match[1]!;
    const directive = match[2];
    const rawValue = parameters[paramName];

    let display: string | null = null;
    if (directive === "datetime" && typeof rawValue === "string") {
      display = tryFormatDateTime(rawValue);
    } else if (directive === "count" && Array.isArray(rawValue)) {
      display = String(rawValue.length);
    }

    if (display === null && rawValue != null) {
      display = formatParamValue(rawValue);
    }

    if (display !== null) {
      result += `\u201C${display}\u201D`;
      hasValue = true;
    }

    lastIndex = match.index + match[0].length;
  }

  result += template.slice(lastIndex);
  return hasValue ? result : null;
}

/** Attempts to parse a string as an ISO datetime and return a human-readable form. */
function tryFormatDateTime(value: string): string | null {
  if (!/^\d{4}-\d{2}/.test(value)) return null;
  const d = new Date(value);
  if (isNaN(d.getTime())) return null;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

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

  "slack.send_message": (params, rd) => {
    const channel = strVal(rd?.channel_name) ?? strVal(params.channel);
    if (!channel) return null;
    const message = strVal(params.message);
    let result = `Send message to ${channel}`;
    if (message) result += ` \u2014 ${truncate(message, 60)}`;
    return result;
  },

  "slack.send_dm": (params, rd) => {
    const user = strVal(rd?.user_name) ?? strVal(params.user_id);
    if (!user) return null;
    const message = strVal(params.message);
    let result = `Send DM to ${user}`;
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
 * Converts a snake_case parameter key into a human-readable label.
 * e.g. "channel_name" → "Channel name", "user_id" → "User id"
 *
 * NOTE: This is intentionally duplicated from the web frontend's
 * frontend/src/lib/formatValues.ts `humanizeKey`. The web and mobile
 * apps share no code at the module level, so keep both in sync when
 * changing the logic.
 */
function humanizeKey(key: string): string {
  const words = key.replace(/_/g, " ").toLowerCase();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/**
 * Fallback summary builder when no action-specific formatter matches.
 * Constructs a label from the action type plus up to 3 parameter highlights.
 * Uses humanized key names instead of raw keys for readability (#862).
 */
function buildGenericSummary(
  actionType: string,
  parameters: Record<string, unknown>,
  resourceDetails?: Record<string, unknown> | null,
): string {
  const label = humanizeActionType(actionType);
  // Merge resourceDetails so resolved names appear instead of raw IDs (#862).
  // Skip raw params whose resolved counterpart exists in resourceDetails
  // (e.g. skip "channel" when "channel_name" is present) to avoid showing both.
  const filtered = resourceDetails
    ? Object.fromEntries(
        Object.entries(parameters).filter(
          ([key]) => !(resourceDetails[`${key}_name`] != null),
        ),
      )
    : parameters;
  const merged = resourceDetails
    ? { ...filtered, ...resourceDetails }
    : parameters;
  const entries = Object.entries(merged);
  if (entries.length === 0) return label;

  const highlights: string[] = [];
  for (const [key, value] of entries) {
    if (highlights.length >= 3) break;
    if (value == null) continue;
    const display = formatValue(value);
    if (display) highlights.push(`${humanizeKey(key)}: ${display}`);
  }

  if (highlights.length === 0) return label;
  return `${label}: ${highlights.join(", ")}`;
}

/**
 * Returns a consistent avatar background + text color pair based on the name.
 * Uses a simple hash to pick from 8 soft pastel pairs.
 */
import { colors } from "../../theme/colors";

const AVATAR_COLORS = [
  { bg: "#DBEAFE", text: "#1E40AF" }, // blue
  { bg: colors.approvedBg, text: colors.approvedText }, // green — reuses theme
  { bg: "#EDE9FE", text: "#5B21B6" }, // purple
  { bg: "#FCE7F3", text: "#9D174D" }, // pink
  { bg: colors.pendingBg, text: colors.pendingText }, // amber — reuses theme
  { bg: "#CFFAFE", text: "#155E75" }, // cyan
  { bg: "#FFEDD5", text: "#9A3412" }, // orange
  { bg: "#FFE4E6", text: "#9F1239" }, // rose
] as const;

export function getAvatarColors(name: string): { bg: string; text: string } {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  }
  const index = Math.abs(hash) % AVATAR_COLORS.length;
  const entry = AVATAR_COLORS[index]!;
  return { bg: entry.bg, text: entry.text };
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
 * Formats an epoch-ms timestamp as a human-friendly "last updated" string.
 * Returns null when the timestamp is 0 (data never fetched).
 */
export function formatLastUpdated(epochMs: number): string | null {
  if (epochMs === 0) return null;
  // Clamp to 0 so clock skew (epochMs in the future) still shows "just now".
  const diffSec = Math.max(0, Math.floor((Date.now() - epochMs) / 1_000));
  if (diffSec < 10) return "Updated just now";
  if (diffSec < 60) return `Updated ${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `Updated ${diffMin}m ago`;
  return `Updated ${Math.floor(diffMin / 60)}h ago`;
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
