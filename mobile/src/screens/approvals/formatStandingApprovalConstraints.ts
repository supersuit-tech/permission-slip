export type ConstraintMode = "fixed" | "pattern" | "wildcard";

export interface ParsedConstraintLine {
  label: string;
  mode: ConstraintMode;
  value: string;
  verified: boolean;
}

function isPatternWrapper(value: unknown): value is { $pattern: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "$pattern" in value &&
    typeof (value as Record<string, unknown>).$pattern === "string"
  );
}

function metaConstraintLabel(key: string): string {
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

function formatDataWindowConstraint(raw: unknown): string | null {
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

function parseValue(
  label: string,
  raw: unknown,
  verified: boolean,
): ParsedConstraintLine {
  if (raw === "*") {
    return { label, mode: "wildcard", value: "any", verified };
  }
  if (isPatternWrapper(raw)) {
    return { label, mode: "pattern", value: raw.$pattern, verified };
  }
  return { label, mode: "fixed", value: String(raw), verified };
}

export function formatStandingApprovalConstraints(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraintLine[] {
  if (!constraints || typeof constraints !== "object") return [];

  const lines: ParsedConstraintLine[] = [];
  for (const [key, raw] of Object.entries(constraints)) {
    if (key === "$meta") {
      if (!raw || typeof raw !== "object") continue;
      for (const [metaKey, metaVal] of Object.entries(
        raw as Record<string, unknown>,
      )) {
        lines.push(parseValue(metaConstraintLabel(metaKey), metaVal, true));
      }
      continue;
    }
    if (key === "$data_window") {
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
      const value =
        line.mode === "wildcard" ? "any value" : line.value;
      return `${line.label}: ${value}`;
    })
    .join("\n");
}
