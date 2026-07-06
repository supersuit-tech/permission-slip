import {
  DATA_WINDOW_NAMESPACE_KEY,
  META_NAMESPACE_KEY,
  isPatternWrapper,
} from "@/lib/constraints";
import type { ParamMode } from "@/pages/agents/connectors/StandingApprovalFormFields";

export function metaValuesFromConstraints(
  constraints: Record<string, unknown> | null | undefined,
): Record<string, string> {
  const meta = constraints?.[META_NAMESPACE_KEY];
  if (!meta || typeof meta !== "object") return {};

  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(meta as Record<string, unknown>)) {
    if (value === "*") {
      result[key] = "*";
    } else if (isPatternWrapper(value)) {
      result[key] = value.$pattern;
    } else if (value != null && typeof value !== "object") {
      result[key] = String(value);
    }
  }
  return result;
}

export function metaModesFromConstraints(
  constraints: Record<string, unknown> | null | undefined,
): Record<string, ParamMode> {
  const meta = constraints?.[META_NAMESPACE_KEY];
  if (!meta || typeof meta !== "object") return {};

  const modes: Record<string, ParamMode> = {};
  for (const [key, value] of Object.entries(meta as Record<string, unknown>)) {
    if (value === "*") {
      modes[key] = "wildcard";
    } else if (isPatternWrapper(value)) {
      modes[key] = "pattern";
    } else {
      modes[key] = "fixed";
    }
  }
  return modes;
}

export function buildMetaConstraintsFromForm(
  metaValues: Record<string, string>,
  metaModes?: Record<string, ParamMode>,
): Record<string, unknown> | undefined {
  const meta: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(metaValues)) {
    if (value === "") continue;
    if (metaModes?.[key] === "wildcard") {
      meta[key] = "*";
      continue;
    }
    if (value.includes("*") || metaModes?.[key] === "pattern") {
      meta[key] = { $pattern: value };
      continue;
    }
    meta[key] = value;
  }

  if (Object.keys(meta).length === 0) {
    return undefined;
  }
  return meta;
}

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

export function mergeStandingApprovalConstraints(
  params: Record<string, unknown>,
  meta?: Record<string, unknown>,
  preserved?: { data_window?: unknown },
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...params };
  if (meta && Object.keys(meta).length > 0) {
    out[META_NAMESPACE_KEY] = meta;
  }
  if (preserved?.data_window !== undefined) {
    out[DATA_WINDOW_NAMESPACE_KEY] = preserved.data_window;
  }
  return out;
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
