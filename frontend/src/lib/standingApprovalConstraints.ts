import { DATA_WINDOW_NAMESPACE_KEY } from "@/lib/constraints";

export function preservedNamespacesFromConstraints(
  constraints: Record<string, unknown> | null | undefined,
): { data_window?: unknown } {
  if (!constraints || typeof constraints !== "object") {
    return {};
  }
  const dataWindow = constraints[DATA_WINDOW_NAMESPACE_KEY];
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
