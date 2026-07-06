export type ConstraintMode = "fixed" | "pattern" | "wildcard";

export interface ParsedConstraintLine {
  label: string;
  value: string;
  mode: ConstraintMode;
  verified: boolean;
  negated?: boolean;
}

export const META_NAMESPACE_KEY = "$meta";
export const DATA_WINDOW_NAMESPACE_KEY = "$data_window";
export const CONSTRAINT_VERSION = 2;

/** Check if a stored parameter value is a `$pattern` wrapper object. */
export function isPatternWrapper(value: unknown): value is { $pattern: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "$pattern" in value &&
    typeof (value as Record<string, unknown>).$pattern === "string"
  );
}

/** Human-readable labels for verified `$meta` constraint fields. */
export function metaConstraintLabel(key: string): string {
  switch (key) {
    case "from":
    case "sender":
    case "senders":
      return "Verified sender";
    case "to":
      return "Verified To";
    case "cc":
      return "Verified Cc";
    case "bcc":
      return "Verified Bcc (sent mail only)";
    default:
      return `Verified ${key}`;
  }
}

export function formatDataWindowConstraint(raw: unknown): string | null {
  if (!raw || typeof raw !== "object") return null;
  const dw = raw as Record<string, unknown>;
  if (typeof dw.last_days === "number" && dw.last_days >= 1) {
    const n = dw.last_days;
    return `last ${n} day${n === 1 ? "" : "s"}`;
  }
  const parts: string[] = [];
  if (typeof dw.starts_at === "string" && dw.starts_at) {
    parts.push(`from ${dw.starts_at}`);
  }
  if (typeof dw.ends_at === "string" && dw.ends_at) {
    parts.push(`until ${dw.ends_at}`);
  }
  return parts.length > 0 ? parts.join(" ") : null;
}

function decodeDisplayValue(raw: unknown): { mode: ConstraintMode; value: string } {
  if (raw === "*") {
    return { mode: "wildcard", value: "any" };
  }
  if (isPatternWrapper(raw)) {
    return { mode: "pattern", value: raw.$pattern };
  }
  return { mode: "fixed", value: String(raw) };
}

function parseStructuredConstraints(
  constraints: Record<string, unknown>,
): ParsedConstraintLine[] {
  const groups = constraints.groups;
  if (!Array.isArray(groups)) return [];

  const lines: ParsedConstraintLine[] = [];
  groups.forEach((group, scenarioIndex) => {
    const prefix = groups.length > 1 ? `Scenario ${scenarioIndex + 1}: ` : "";
    const conditions = (group as { conditions?: unknown[] }).conditions;
    if (!Array.isArray(conditions)) return;

    for (const cond of conditions) {
      if (!cond || typeof cond !== "object") continue;
      const c = cond as Record<string, unknown>;
      const field = String(c.field ?? "");
      const op = String(c.op ?? "matches");
      const negated = op === "none_of" || op === "does_not_match";

      if (field === DATA_WINDOW_NAMESPACE_KEY) {
        const text = formatDataWindowConstraint(c.value);
        if (text) {
          lines.push({
            label: `${prefix}Data window`,
            mode: "fixed",
            value: text,
            verified: false,
          });
        }
        continue;
      }

      const isMeta = field.startsWith(`${META_NAMESPACE_KEY}.`);
      const label = isMeta
        ? metaConstraintLabel(field.slice(`${META_NAMESPACE_KEY}.`.length))
        : field;

      const values: unknown[] =
        op === "any_of" || op === "none_of"
          ? ((c.values as unknown[]) ?? [])
          : c.value !== undefined
            ? [c.value]
            : [];

      for (const raw of values) {
        const decoded = decodeDisplayValue(raw);
        lines.push({
          label: `${prefix}${label}`,
          mode: decoded.mode,
          value: decoded.value,
          verified: isMeta,
          negated,
        });
      }
    }
  });
  return lines;
}

function parseValue(
  label: string,
  raw: unknown,
  verified: boolean,
): ParsedConstraintLine {
  const decoded = decodeDisplayValue(raw);
  return { label, mode: decoded.mode, value: decoded.value, verified };
}

/** Flatten standing approval constraints into human-readable display lines. */
export function formatStandingApprovalConstraints(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraintLine[] {
  if (!constraints || typeof constraints !== "object") return [];

  if (constraints.$version === CONSTRAINT_VERSION) {
    return parseStructuredConstraints(constraints);
  }

  const lines: ParsedConstraintLine[] = [];
  for (const [key, raw] of Object.entries(constraints)) {
    if (key === META_NAMESPACE_KEY) {
      if (!raw || typeof raw !== "object") continue;
      for (const [metaKey, metaVal] of Object.entries(
        raw as Record<string, unknown>,
      )) {
        lines.push(parseValue(metaConstraintLabel(metaKey), metaVal, true));
      }
      continue;
    }
    if (key === DATA_WINDOW_NAMESPACE_KEY) {
      const text = formatDataWindowConstraint(raw);
      if (text) {
        lines.push({
          label: "Data window",
          mode: "fixed",
          value: text,
          verified: false,
        });
      }
      continue;
    }
    lines.push(parseValue(key, raw, false));
  }
  return lines;
}

export function formatStandingApprovalConstraintsText(
  constraints: Record<string, unknown> | null | undefined,
): string {
  const lines = formatStandingApprovalConstraints(constraints);
  if (lines.length === 0) return "No constraints";
  return lines
    .map((line) => {
      const prefix = line.negated ? "not " : "";
      const value =
        line.mode === "wildcard" ? "any value" : `${prefix}${line.value}`;
      return `${line.label}: ${value}`;
    })
    .join("\n");
}
