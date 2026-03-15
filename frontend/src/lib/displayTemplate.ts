/**
 * Display Template Engine
 *
 * Parses manifest-defined display_template strings into structured
 * SummaryPart arrays for dual rendering (rich JSX / plain text).
 *
 * Template syntax:
 * - {{param}}          — insert raw value (auto-truncated)
 * - {{param:datetime}} — format as human-readable date
 * - {{param:count}}    — array length (e.g., "3 attendees" context)
 *
 * The formatter registry is extensible — add new directives by
 * inserting entries into FORMATTERS.
 */

import { tryFormatDateTime, formatHighlightValue } from "./formatValues";

/** A segment of the summary — either plain text or a highlighted value. */
export type SummaryPart =
  | { kind: "text"; text: string }
  | { kind: "value"; text: string };

/**
 * Extensible formatter registry. Each formatter converts an unknown
 * parameter value into a display string, or returns null to skip.
 *
 * To add a new directive (e.g., {{param:email}}), add an entry here:
 * ```
 * FORMATTERS.set("email", (v) => typeof v === "string" ? v.toLowerCase() : null);
 * ```
 */
export const FORMATTERS = new Map<
  string,
  (value: unknown) => string | null
>();

// Built-in formatters
FORMATTERS.set("datetime", (value) => tryFormatDateTime(value));

FORMATTERS.set("count", (value) => {
  if (Array.isArray(value)) return String(value.length);
  return null;
});

/** Pattern matching {{param}} and {{param:directive}} placeholders. */
const PLACEHOLDER_RE = /\{\{(\w+)(?::(\w+))?\}\}/g;

/**
 * Renders a display template into SummaryPart[].
 *
 * Returns null if the template is empty or if critical placeholders
 * resolve to empty values (making the summary meaningless).
 */
export function renderTemplate(
  template: string,
  parameters: Record<string, unknown>,
): SummaryPart[] | null {
  if (!template) return null;

  const parts: SummaryPart[] = [];
  let lastIndex = 0;
  let hasValue = false;

  // Reset lastIndex for global regex reuse.
  PLACEHOLDER_RE.lastIndex = 0;

  let match: RegExpExecArray | null;
  while ((match = PLACEHOLDER_RE.exec(template)) !== null) {
    // Add text before this placeholder.
    if (match.index > lastIndex) {
      parts.push({ kind: "text", text: template.slice(lastIndex, match.index) });
    }

    const paramName = match[1]!;
    const directive = match[2]; // may be undefined
    const rawValue = parameters[paramName];

    let display: string | null = null;

    if (directive) {
      const formatter = FORMATTERS.get(directive);
      if (formatter) {
        display = formatter(rawValue);
      }
    }

    // Fall back to generic formatting if directive didn't produce a result.
    if (display === null) {
      display = formatHighlightValue(rawValue);
    }

    if (display !== null) {
      parts.push({ kind: "value", text: display });
      hasValue = true;
    }

    lastIndex = match.index + match[0].length;
  }

  // Add trailing text after the last placeholder.
  if (lastIndex < template.length) {
    parts.push({ kind: "text", text: template.slice(lastIndex) });
  }

  // If no placeholders resolved to actual values, the template is useless.
  if (!hasValue) return null;

  return parts;
}
