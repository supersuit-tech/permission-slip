import {
  DATA_WINDOW_NAMESPACE_KEY,
  META_NAMESPACE_KEY,
  isPatternWrapper,
  metaConstraintLabel,
  type ConstraintMode,
} from "@/lib/constraints";
import type { ParamMode } from "@/pages/agents/connectors/StandingApprovalFormFields";
import type { DataWindowFormState } from "@/lib/dataWindow";
import { buildDataWindowConstraint } from "@/lib/dataWindow";

export const CONSTRAINT_VERSION = 2;

export type RowOperator = "matches" | "does_not_match";

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

function encodeRowValue(row: ConstraintValueRow): unknown {
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
    for (const row of rows) {
      if (row.value === "" && row.mode !== "wildcard") continue;
      const encoded = encodeRowValue(row);
      if (row.operator === "does_not_match") {
        deny.push(encoded);
      } else {
        allow.push(encoded);
      }
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
        if (row.mode !== "wildcard" && row.value !== "" && row.value !== "*") {
          return true;
        }
        if (row.operator === "does_not_match" && row.value !== "") {
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

function formatRowDisplay(
  label: string,
  row: ConstraintValueRow,
): ParsedConstraintDisplay {
  const decoded = decodeStoredValue(encodeRowValue(row));
  return {
    name: label,
    mode: decoded.mode,
    value: decoded.mode === "wildcard" ? "any" : decoded.value,
    negated: row.operator === "does_not_match",
  };
}

/** Human-readable constraint lines including negation and scenario grouping. */
export function parseStructuredConstraintsForDisplay(
  constraints: Record<string, unknown> | null | undefined,
): ParsedConstraintDisplay[] {
  if (!constraints || typeof constraints !== "object") return [];

  if (!isStructuredConstraints(constraints)) {
    return [];
  }

  const form = parseStructuredDocument(constraints);
  const lines: ParsedConstraintDisplay[] = [];

  form.scenarios.forEach((scenario, scenarioIndex) => {
    const prefix =
      form.scenarios.length > 1 ? `Scenario ${scenarioIndex + 1}: ` : "";

    for (const [field, rows] of Object.entries(scenario.paramRows)) {
      for (const row of rows) {
        if (row.value === "" && row.mode !== "wildcard") continue;
        lines.push({
          ...formatRowDisplay(`${prefix}${field}`, row),
          scenarioIndex,
        });
      }
    }
    for (const [field, rows] of Object.entries(scenario.metaRows)) {
      const label = metaConstraintLabel(field);
      for (const row of rows) {
        if (row.value === "" && row.mode !== "wildcard") continue;
        lines.push({
          ...formatRowDisplay(`${prefix}${label}`, row),
          scenarioIndex,
        });
      }
    }
  });

  return lines;
}

/** Plain-language summary for a field's rows within one scenario. */
export function summarizeFieldRows(
  fieldLabel: string,
  rows: ConstraintValueRow[],
): string | null {
  const allow = rows.filter((r) => r.operator === "matches");
  const deny = rows.filter((r) => r.operator === "does_not_match");
  const parts: string[] = [];

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
