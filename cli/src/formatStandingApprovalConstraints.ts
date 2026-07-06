export {
  formatStandingApprovalConstraints,
  formatStandingApprovalConstraintsText,
  type ConstraintMode,
  type ParsedConstraintLine,
} from "@permission-slip/constraints-format";

export function parseConstraintsJson(raw: string): Record<string, unknown> {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error(`--constraints must be valid JSON. Got: ${raw}`);
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("--constraints must be a JSON object");
  }
  return parsed as Record<string, unknown>;
}
