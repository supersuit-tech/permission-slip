/**
 * Strict ISO 8601 / RFC 3339 datetime regex.
 * Matches formats like:
 *   - 2026-03-16T17:00
 *   - 2026-03-16T17:00:00
 *   - 2026-03-16T17:00:00Z
 *   - 2026-03-16T17:00:00+05:00
 *   - 2026-03-16T17:00:00.123Z
 */
const ISO_DATETIME_RE =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:\d{2})?$/;

/**
 * Returns true when the value is a single concrete datetime instant
 * (not a wildcard pattern like "2026-*").
 *
 * Uses a strict ISO 8601 regex instead of `new Date()` parsing, which
 * is locale/engine-dependent and accepts non-datetime strings.
 */
export function isConcreteDatetimeString(value: string): boolean {
  if (!value || value.includes("*")) return false;
  return ISO_DATETIME_RE.test(value);
}
