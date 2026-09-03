/** True when a glob is only `*` characters (`*`, `**`, `***`, …). */
function isAllStarGlob(value: string): boolean {
  return value.length > 0 && [...value].every((ch) => ch === "*");
}

function isSemanticWildcardValue(raw: unknown): boolean {
  if (raw === "*") return true;
  if (raw && typeof raw === "object" && !Array.isArray(raw) && "$pattern" in raw) {
    const pattern = (raw as { $pattern?: unknown }).$pattern;
    return typeof pattern === "string" && isAllStarGlob(pattern);
  }
  return false;
}

function structuredGroupsAreUnrestricted(constraints: Record<string, unknown>): boolean {
  const groups = constraints.groups;
  if (!Array.isArray(groups) || groups.length === 0) return true;
  return groups.every((group) => {
    if (!group || typeof group !== "object") return true;
    const conditions = (group as { conditions?: unknown }).conditions;
    if (!Array.isArray(conditions) || conditions.length === 0) return true;
    return conditions.every((cond) => {
      if (!cond || typeof cond !== "object") return true;
      const c = cond as {
        field?: string;
        op?: string;
        value?: unknown;
        values?: unknown[];
      };
      if (c.field === "$data_window") return false;
      if (c.op === "lte" || c.op === "gte" || c.op === "lt" || c.op === "gt") {
        return false;
      }
      if (c.op === "none_of" || c.op === "does_not_match") {
        return false;
      }
      if (Array.isArray(c.values)) {
        return c.values.every(isSemanticWildcardValue);
      }
      return isSemanticWildcardValue(c.value);
    });
  });
}

/** True when every constraint matches any parameter value. */
export function constraintsAreUnrestricted(
  constraints: Record<string, unknown> | null | undefined,
): boolean {
  if (!constraints || typeof constraints !== "object") return true;
  if (constraints.$version === 2) {
    return structuredGroupsAreUnrestricted(constraints);
  }
  const entries = Object.entries(constraints);
  if (entries.length === 0) return true;
  return entries.every(([key, value]) => {
    if (key === "$data_window") return false;
    if (key === "$meta" && value && typeof value === "object") {
      return Object.values(value as Record<string, unknown>).every(
        isSemanticWildcardValue,
      );
    }
    return isSemanticWildcardValue(value);
  });
}
