/** Check if a stored parameter value is a $pattern wrapper object. */
export function isPatternWrapper(value: unknown): value is { $pattern: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "$pattern" in value &&
    typeof (value as Record<string, unknown>).$pattern === "string"
  );
}

export const META_NAMESPACE_KEY = "$meta";

/** Human-readable labels for verified $meta constraint fields. */
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

export type ConstraintMode = "fixed" | "pattern" | "wildcard";

export interface ParsedConstraint {
  name: string;
  mode: ConstraintMode;
  value: string;
}

function parseConstraintValue(
  name: string,
  raw: unknown,
): ParsedConstraint {
  if (raw === "*") {
    return { name, mode: "wildcard", value: "*" };
  }
  if (isPatternWrapper(raw)) {
    return { name, mode: "pattern", value: raw.$pattern };
  }
  return { name, mode: "fixed", value: String(raw) };
}

/** Flatten standing-approval constraints into display rows (params + $meta). */
export function parseStandingApprovalConstraints(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraint[] {
  if (!constraints || typeof constraints !== "object") return [];

  const parsed: ParsedConstraint[] = [];
  for (const [key, raw] of Object.entries(constraints)) {
    if (key === META_NAMESPACE_KEY) {
      if (!raw || typeof raw !== "object") continue;
      for (const [metaKey, metaVal] of Object.entries(
        raw as Record<string, unknown>,
      )) {
        const label = metaConstraintLabel(metaKey);
        parsed.push(parseConstraintValue(label, metaVal));
      }
      continue;
    }
    parsed.push(parseConstraintValue(key, raw));
  }
  return parsed;
}
