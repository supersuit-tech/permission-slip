/**
 * Shared value formatting utilities for action parameter display.
 *
 * These functions are used by both ActionPreviewSummary (highlights) and
 * SchemaParameterDetails (full parameter rendering) to ensure consistent
 * formatting across the approval UI.
 */

const DATE_ONLY_RE = /^\d{4}-\d{2}-\d{2}$/;

/** Formats a Date's clock portion as "8PM" or "8:30PM" (12-hour, no space before AM/PM). */
function formatClockTime(date: Date): string {
  const minutes = date.getMinutes();
  const formatted = date.toLocaleTimeString(undefined, {
    hour: "numeric",
    ...(minutes !== 0 ? { minute: "2-digit" } : {}),
    hour12: true,
  });
  return formatted.replace(/\s+(AM|PM)\b/i, "$1");
}

/**
 * Formats an all-day calendar date (YYYY-MM-DD) without timezone conversion.
 * Preserves the existing abbreviated-month style used before this change.
 */
function formatAllDayDate(iso: string): string {
  const [year, month, day] = iso.split("-").map(Number);
  const date = new Date(year!, month! - 1, day!);
  const now = new Date();
  const sameYear = date.getFullYear() === now.getFullYear();
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: sameYear ? undefined : "numeric",
  });
}

/** Formats the calendar date portion as "June 25 2026" (no comma). */
function formatDatePart(date: Date): string {
  const parts = new Intl.DateTimeFormat(undefined, {
    month: "long",
    day: "numeric",
    year: "numeric",
  }).formatToParts(date);
  const month = parts.find((p) => p.type === "month")?.value ?? "";
  const day = parts.find((p) => p.type === "day")?.value ?? "";
  const year = parts.find((p) => p.type === "year")?.value ?? "";
  return `${month} ${day} ${year}`;
}

function formatTimedDate(date: Date): string {
  return `${formatDatePart(date)} @ ${formatClockTime(date)}`;
}

/**
 * Formats a calendar event start/end timestamp for display.
 * Example: "June 25 2026 @ 8PM" (minutes shown only when non-zero).
 * All-day date-only strings (YYYY-MM-DD) are formatted without a time component
 * and without timezone conversion.
 */
export function formatDateTime(iso: string): string {
  if (DATE_ONLY_RE.test(iso)) {
    return formatAllDayDate(iso);
  }
  const date = new Date(iso);
  if (isNaN(date.getTime())) return iso;
  return formatTimedDate(date);
}

/**
 * Formats a start/end time range, collapsing the shared date when both fall on
 * the same local calendar day. Example: "June 25 2026 @ 8PM – 9PM".
 */
export function formatDateTimeRange(startIso: string, endIso: string): string {
  if (DATE_ONLY_RE.test(startIso)) {
    return formatAllDayDate(startIso);
  }
  const start = new Date(startIso);
  const end = new Date(endIso);
  if (isNaN(start.getTime())) return startIso;
  if (isNaN(end.getTime())) return formatDateTime(startIso);

  const sameDay =
    start.getFullYear() === end.getFullYear() &&
    start.getMonth() === end.getMonth() &&
    start.getDate() === end.getDate();

  if (sameDay) {
    const datePart = formatDatePart(start);
    return `${datePart} @ ${formatClockTime(start)} \u2013 ${formatClockTime(end)}`;
  }
  return `${formatDateTime(startIso)} \u2013 ${formatDateTime(endIso)}`;
}

/**
 * Attempts to parse a string as an ISO 8601 / RFC 3339 datetime and
 * return a human-readable representation. Returns null if the value
 * is not a recognisable datetime string.
 */
export function tryFormatDateTime(value: unknown): string | null {
  if (typeof value !== "string") return null;
  // Quick sanity check: must look like a date (YYYY-MM or ISO-ish).
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

/**
 * Converts a snake_case or camelCase parameter key into a human-readable
 * label. Splits on underscores and camelCase boundaries, lowercases, then
 * capitalises the first letter.
 *
 * Examples:
 * - "start_time" → "Start time"
 * - "spreadsheet_id" → "Spreadsheet id"
 * - "calendarId" → "Calendar id"
 */
export function humanizeKey(key: string): string {
  const words = key
    .replace(/_/g, " ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .toLowerCase();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

/**
 * Formats an unknown value for display in action parameter summaries.
 * Handles strings (with truncation), numbers, booleans, and arrays.
 * Returns null for objects and other non-displayable types.
 *
 * Consolidated from ActionPreviewSummary's formatHighlightValue to
 * avoid duplicate formatting logic.
 */
/** Masks Slack opaque resource IDs embedded in free-text params (e.g. search query). */
function redactSlackOpaqueIdsInString(s: string): string {
  return s.replace(/\b[CGD][A-Z0-9]{8,}\b/g, "\u2014");
}

export function formatHighlightValue(
  value: unknown,
  maxLen = 60,
): string | null {
  if (typeof value === "string")
    return truncate(redactSlackOpaqueIdsInString(value), maxLen);
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) {
    const strs = value
      .filter((x) => typeof x === "string" || typeof x === "number")
      .map(String);
    if (strs.length === 0) return null;
    if (strs.length <= 2) return strs.join(", ");
    return `${strs[0] ?? ""}, ${strs[1] ?? ""}, +${strs.length - 2} more`;
  }
  return null;
}

/**
 * Formats a value for full parameter display (SchemaParameterDetails).
 * Applies datetime detection for string values, and handles all types.
 */
export function formatParameterValue(value: unknown): string {
  if (value === null || value === undefined) return "\u2014";
  if (typeof value === "string") {
    return tryFormatDateTime(value) ?? value;
  }
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) return value.map(String).join(", ");
  return JSON.stringify(value);
}

/** Truncates a string to `maxLen` characters with an ellipsis. */
export function truncate(s: string, maxLen: number): string {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen - 1) + "\u2026";
}
