import { extractDataWindowConstraint } from "@/lib/dataWindow";

export function preservedNamespacesFromConstraints(
  constraints: Record<string, unknown> | null | undefined,
): { data_window?: unknown } {
  const dataWindow = extractDataWindowConstraint(constraints);
  if (dataWindow === undefined) {
    return {};
  }
  return { data_window: dataWindow };
}

/** Hide boilerplate descriptions from auto-created rule proposals. */
export function isBoilerplateStandingApprovalDescription(
  description: string | null | undefined,
): boolean {
  if (!description) return false;
  const normalized = description.trim().toLowerCase();
  return (
    normalized === "standing auto-approve rule" ||
    normalized.startsWith("created automatically when approving")
  );
}
