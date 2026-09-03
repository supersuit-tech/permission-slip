import {
  DATA_WINDOW_NAMESPACE_KEY,
  isPatternWrapper,
  META_NAMESPACE_KEY,
  formatStandingApprovalConstraints,
  comparisonOpLabel,
  type ConstraintMode,
  type ParsedConstraintLine,
} from "@permission-slip/constraints-format";
import type { ParamMode } from "@/pages/agents/connectors/StandingApprovalFormFields";
import type { DataWindowFormState } from "@/lib/dataWindow";
import { buildDataWindowConstraint } from "@/lib/dataWindow";

export const CONSTRAINT_VERSION = 2;

export type RowOperator =
  | "matches"
  | "does_not_match"
  | "lte"
  | "gte"
  | "lt"
  | "gt";

export interface ConstraintValueRow {
  id: string;
  operator: RowOperator;
  mode: ParamMode;
  value: string;
}

export interface ConstraintScenario {
  id: string;
  paramRows: Record<string, ConstraintValueRow[]>;
  metaRows: Record<string, ConstraintValueRow[]>;
}

export interface StructuredConstraintFormState {
  scenarios: ConstraintScenario[];
  dataWindow?: DataWindowFormState;
}

export function isStructuredConstraints(
  constraints: Record<string, unknown> | null | undefined,
): boolean {
  return constraints?.$version === CONSTRAINT_VERSION;
}

function newRowId(): string {
  return `row-${Math.random().toString(36).slice(2, 9)}`;
}

export function newScenarioId(): string {
  return `scenario-${Math.random().toString(36).slice(2, 9)}`;
}

export function emptyConstraintRow(
  operator: RowOperator = "matches",
): ConstraintValueRow {
  return { id: newRowId(), operator, mode: "fixed", value: "" };
}

export function emptyScenario(): ConstraintScenario {
  return { id: newScenarioId(), paramRows: {}, metaRows: {} };
}

function isComparisonOperator(op: string): op is Exclude<RowOperator, "matches" | "does_not_match"> {
  return op === "lte" || op === "gte" || op === "lt" || op === "gt";
}

/** True when a glob is only `*` characters (`*`, `**`, `***`, …). */
export function isAllStarGlob(value: string): boolean {
  return value.length > 0 && [...value].every((ch) => ch === "*");
}

/** True when a stored constraint value matches every possible string. */
export function isSemanticWildcardValue(raw: unknown): boolean {
  if (raw === "*") return true;
  if (isPatternWrapper(raw) && isAllStarGlob(raw.$pattern)) return true;
  return false;
}

function encodeRowValue(row: ConstraintValueRow): unknown {
  if (isComparisonOperator(row.operator)) {
    const trimmed = row.value.trim();
    if (trimmed === "") return "";
    const asNumber = Number(trimmed);
    if (trimmed !== "" && !Number.isNaN(asNumber)) {
      return asNumber;
    }
    return trimmed;
  }
  if (row.mode === "wildcard" || row.value === "*") {
    return "*";
  }
  if (row.mode === "pattern" || row.value.includes("*")) {
    return { $pattern: row.value };
  }
  return row.value;
}

function decodeStoredValue(raw: unknown): { mode: ParamMode; value: string } {
  if (raw === "*") {
    return { mode: "wildcard", value: "*" };
  }
  if (isPatternWrapper(raw)) {
    return { mode: "pattern", value: raw.$pattern };
  }
  if (raw != null && typeof raw !== "object") {
    return { mode: "fixed", value: String(raw) };
  }
  if (raw != null && typeof raw === "object") {
    return { mode: "fixed", value: JSON.stringify(raw) };
  }
  return { mode: "fixed", value: "" };
}

function rowsFromCondition(
  operator: RowOperator,
  values: unknown[],
): ConstraintValueRow[] {
  return values.map((raw) => {
    const decoded = decodeStoredValue(raw);
    return {
      id: newRowId(),
      operator,
      mode: decoded.mode,
      value: decoded.value,
    };
  });
}

function appendConditionRows(
  target: Record<string, ConstraintValueRow[]>,
  field: string,
  op: string,
  value?: unknown,
  values?: unknown[],
) {
  const key = field;
  if (!target[key]) {
    target[key] = [];
  }
  if (op === "matches" || op === "does_not_match") {
    const operator: RowOperator =
      op === "does_not_match" ? "does_not_match" : "matches";
    if (value !== undefined) {
      target[key].push(...rowsFromCondition(operator, [value]));
    }
    return;
  }
  if (isComparisonOperator(op)) {
    if (value !== undefined) {
      target[key].push(...rowsFromCondition(op, [value]));
    }
    return;
  }
  if (op === "any_of") {
    if (values?.length) {
      target[key].push(...rowsFromCondition("matches", values));
    }
    return;
  }
  if (op === "none_of" && values?.length) {
    target[key].push(...rowsFromCondition("does_not_match", values));
  }
}

function parseStructuredDocument(
  constraints: Record<string, unknown>,
): StructuredConstraintFormState {
  const groups = constraints.groups;
  if (!Array.isArray(groups) || groups.length === 0) {
    return { scenarios: [emptyScenario()] };
  }

  const scenarios: ConstraintScenario[] = groups.map((group) => {
    const scenario = emptyScenario();
    const conditions = (group as { conditions?: unknown[] }).conditions;
    if (!Array.isArray(conditions)) {
      return scenario;
    }
    for (const cond of conditions) {
      if (!cond || typeof cond !== "object") continue;
      const c = cond as Record<string, unknown>;
      const field = String(c.field ?? "");
      const op = String(c.op ?? "matches");
      if (field === DATA_WINDOW_NAMESPACE_KEY) {
        continue;
      }
      if (field.startsWith(`${META_NAMESPACE_KEY}.`)) {
        const metaKey = field.slice(`${META_NAMESPACE_KEY}.`.length);
        appendConditionRows(
          scenario.metaRows,
          metaKey,
          op,
          c.value,
          c.values as unknown[] | undefined,
        );
        continue;
      }
      appendConditionRows(
        scenario.paramRows,
        field,
        op,
        c.value,
        c.values as unknown[] | undefined,
      );
    }
    return scenario;
  });

  return {
    scenarios: scenarios.length > 0 ? scenarios : [emptyScenario()],
  };
}

/** Parse flat or v2 constraints into editable scenario form state. */
export function constraintsToFormState(
  constraints: Record<string, unknown> | null | undefined,
): StructuredConstraintFormState {
  if (!constraints || typeof constraints !== "object") {
    return { scenarios: [emptyScenario()] };
  }
  if (isStructuredConstraints(constraints)) {
    return parseStructuredDocument(constraints);
  }

  const scenario = emptyScenario();
  for (const [key, raw] of Object.entries(constraints)) {
    if (key === META_NAMESPACE_KEY) {
      if (raw && typeof raw === "object") {
        for (const [metaKey, metaVal] of Object.entries(
          raw as Record<string, unknown>,
        )) {
          appendConditionRows(scenario.metaRows, metaKey, "matches", metaVal);
        }
      }
      continue;
    }
    if (key === DATA_WINDOW_NAMESPACE_KEY) {
      continue;
    }
    appendConditionRows(scenario.paramRows, key, "matches", raw);
  }
  return { scenarios: [scenario] };
}

function buildFieldConditions(
  rowsByField: Record<string, ConstraintValueRow[]>,
  metaPrefix: boolean,
): Array<Record<string, unknown>> {
  const conditions: Array<Record<string, unknown>> = [];
  for (const [field, rows] of Object.entries(rowsByField)) {
    const effectiveField = metaPrefix ? `${META_NAMESPACE_KEY}.${field}` : field;
    const allow: unknown[] = [];
    const deny: unknown[] = [];
    const comparisons: Array<{ op: RowOperator; value: unknown }> = [];
    for (const row of rows) {
      if (row.value === "" && row.mode !== "wildcard") continue;
      const encoded = encodeRowValue(row);
      if (isComparisonOperator(row.operator)) {
        comparisons.push({ op: row.operator, value: encoded });
        continue;
      }
      if (row.operator === "does_not_match") {
        deny.push(encoded);
      } else {
        allow.push(encoded);
      }
    }
    for (const comp of comparisons) {
      conditions.push({
        field: effectiveField,
        op: comp.op,
        value: comp.value,
      });
    }
    if (allow.length === 1 && deny.length === 0) {
      conditions.push({
        field: effectiveField,
        op: "matches",
        value: allow[0],
      });
    } else if (allow.length > 0) {
      conditions.push({
        field: effectiveField,
        op: "any_of",
        values: allow,
      });
    }
    if (deny.length === 1) {
      conditions.push({
        field: effectiveField,
        op: "does_not_match",
        value: deny[0],
      });
    } else if (deny.length > 1) {
      conditions.push({
        field: effectiveField,
        op: "none_of",
        values: deny,
      });
    }
  }
  return conditions;
}

/** Build v2 structured constraints from scenario form state. */
export function buildStructuredConstraintsFromForm(
  form: StructuredConstraintFormState,
  dataWindow?: unknown,
): Record<string, unknown> {
  const groups = form.scenarios.map((scenario) => {
    const conditions = [
      ...buildFieldConditions(scenario.paramRows, false),
      ...buildFieldConditions(scenario.metaRows, true),
    ];
    return { match: "all", conditions };
  });

  if (dataWindow !== undefined) {
    const first = groups[0];
    if (first) {
      first.conditions.push({
        field: DATA_WINDOW_NAMESPACE_KEY,
        op: "matches",
        value: dataWindow,
      });
    }
  }

  return {
    $version: CONSTRAINT_VERSION,
    match: "any",
    groups,
  };
}

/** Collapse a v2 document with no conditions to `{}` (parameterless unrestricted). */
export function collapseEmptyStructuredConstraints(
  constraints: Record<string, unknown>,
): Record<string, unknown> {
  if (!isStructuredConstraints(constraints)) {
    return constraints;
  }
  const groups = constraints.groups;
  if (!Array.isArray(groups) || groups.length === 0) {
    return {};
  }
  const hasCondition = groups.some((group) => {
    if (!group || typeof group !== "object") return false;
    const conditions = (group as { conditions?: unknown }).conditions;
    return Array.isArray(conditions) && conditions.length > 0;
  });
  return hasCondition ? constraints : {};
}

/**
 * Replace blank (unset) rows with explicit wildcards so an unrestricted
 * document still has well-formed group conditions.
 */
export function fillEmptyRowsAsWildcards(
  form: StructuredConstraintFormState,
  paramKeys: string[],
): StructuredConstraintFormState {
  return {
    ...form,
    scenarios: form.scenarios.map((scenario) => {
      const paramRows = { ...scenario.paramRows };
      for (const key of paramKeys) {
        const rows = paramRows[key] ?? [];
        const hasValue = rows.some(
          (row) => row.mode === "wildcard" || row.value !== "",
        );
        if (!hasValue) {
          paramRows[key] = [
            { ...emptyConstraintRow(), mode: "wildcard", value: "*" },
          ];
        }
      }
      return { ...scenario, paramRows };
    }),
  };
}

export function constraintsObjectHasNonWildcard(
  constraints: Record<string, unknown>,
  dataWindowForm?: DataWindowFormState,
): boolean {
  if (isStructuredConstraints(constraints)) {
    return formStateHasNonWildcardConstraint(
      constraintsToFormState(constraints),
      dataWindowForm,
    );
  }
  if (dataWindowForm && buildDataWindowConstraint(dataWindowForm)) {
    return true;
  }
  for (const [key, value] of Object.entries(constraints)) {
    if (key === META_NAMESPACE_KEY) {
      if (value && typeof value === "object") {
        for (const metaVal of Object.values(value as Record<string, unknown>)) {
          if (!isSemanticWildcardValue(metaVal)) return true;
        }
      }
      continue;
    }
    if (key === DATA_WINDOW_NAMESPACE_KEY) return true;
    if (!isSemanticWildcardValue(value)) return true;
  }
  return false;
}

export function formStateHasNonWildcardConstraint(
  form: StructuredConstraintFormState,
  dataWindowForm?: DataWindowFormState,
): boolean {
  if (dataWindowForm && buildDataWindowConstraint(dataWindowForm)) {
    return true;
  }
  for (const scenario of form.scenarios) {
    for (const rows of [
      ...Object.values(scenario.paramRows),
      ...Object.values(scenario.metaRows),
    ]) {
      for (const row of rows) {
        if (isComparisonOperator(row.operator) && row.value !== "") {
          return true;
        }
        if (row.operator === "does_not_match" && row.value !== "") {
          return true;
        }
        if (row.mode === "wildcard" || row.value === "" || isAllStarGlob(row.value)) {
          continue;
        }
        if (row.value !== "") {
          return true;
        }
      }
    }
  }
  return false;
}

export interface ParsedConstraintDisplay {
  name: string;
  mode: ConstraintMode;
  value: string;
  negated?: boolean;
  scenarioIndex?: number;
}

/** Human-readable constraint lines including negation and scenario grouping. */
export function parseStructuredConstraintsForDisplay(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraintDisplay[] {
  if (!constraints || typeof constraints !== "object") return [];
  if (!isStructuredConstraints(constraints)) return [];

  return formatStandingApprovalConstraints(constraints).map((line: ParsedConstraintLine) => ({
    name: line.label,
    mode: line.mode,
    value: line.mode === "wildcard" ? "any" : line.value,
    negated: line.negated,
    scenarioIndex: /^Scenario (\d+): /.test(line.label)
      ? Number.parseInt(/^Scenario (\d+): /.exec(line.label)?.[1] ?? "0", 10) - 1
      : undefined,
  }));
}

/** Plain-language summary for a field's rows within one scenario. */
export function summarizeFieldRows(
  fieldLabel: string,
  rows: ConstraintValueRow[],
): string | null {
  const comparisons = rows.filter((r) => isComparisonOperator(r.operator));
  const allow = rows.filter((r) => r.operator === "matches");
  const deny = rows.filter((r) => r.operator === "does_not_match");
  const parts: string[] = [];

  for (const row of comparisons) {
    const decoded = decodeStoredValue(encodeRowValue(row));
    if (decoded.value === "") continue;
    parts.push(
      `${fieldLabel} is ${comparisonOpLabel(row.operator)} ${decoded.value}`,
    );
  }

  if (allow.length > 0) {
    const allowText = allow
      .map((r) => {
        const d = decodeStoredValue(encodeRowValue(r));
        return d.mode === "wildcard" ? "any value" : d.value;
      })
      .join(" or ");
    parts.push(`${fieldLabel} is ${allowText}`);
  }
  if (deny.length > 0) {
    const denyText = deny
      .map((r) => decodeStoredValue(encodeRowValue(r)).value)
      .join(" and ");
    parts.push(`${fieldLabel} is not ${denyText}`);
  }
  return parts.length > 0 ? parts.join(", and ") : null;
}
