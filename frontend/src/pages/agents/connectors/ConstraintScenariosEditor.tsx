import { useCallback } from "react";
import { Copy, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { ParametersSchema, SchemaProperty } from "@/lib/parameterSchema";
import {
  getFieldLabel,
  getOrderedFieldKeys,
  isBooleanField,
  isComparableField,
  isEnumField,
  isFieldHidden,
  isFieldVisible,
} from "@/lib/parameterSchema";
import type { ParamMode } from "./StandingApprovalFormFields";
import { ParameterFieldWidget } from "./ParameterFieldWidget";
import {
  type ConstraintScenario,
  type ConstraintValueRow,
  type RowOperator,
  type StructuredConstraintFormState,
  emptyConstraintRow,
  emptyScenario,
  newScenarioId,
  summarizeFieldRows,
} from "@/lib/structuredConstraints";
import { metaConstraintLabel } from "@/lib/constraints";

interface ConstraintScenariosEditorProps {
  form: StructuredConstraintFormState;
  onChange: (form: StructuredConstraintFormState) => void;
  parametersSchema: ParametersSchema | null;
  metaFields: string[];
  disabled?: boolean;
  agentId?: number;
  connectorId?: string;
}

export function ConstraintScenariosEditor({
  form,
  onChange,
  parametersSchema,
  metaFields,
  disabled,
  agentId,
  connectorId,
}: ConstraintScenariosEditorProps) {
  const properties = parametersSchema?.properties ?? {};
  const orderedKeys = parametersSchema
    ? getOrderedFieldKeys(parametersSchema)
    : [];
  const requiredFields = parametersSchema?.required ?? [];
  const multiScenario = form.scenarios.length > 1;

  const updateScenario = useCallback(
    (index: number, scenario: ConstraintScenario) => {
      const scenarios = [...form.scenarios];
      scenarios[index] = scenario;
      onChange({ ...form, scenarios });
    },
    [form, onChange],
  );

  const addScenario = () => {
    const clone = emptyScenario();
    const last = form.scenarios[form.scenarios.length - 1];
    if (last) {
      clone.paramRows = structuredClone(last.paramRows);
      clone.metaRows = structuredClone(last.metaRows);
    }
    onChange({ ...form, scenarios: [...form.scenarios, clone] });
  };

  const removeScenario = (index: number) => {
    if (form.scenarios.length <= 1) return;
    onChange({
      ...form,
      scenarios: form.scenarios.filter((_, i) => i !== index),
    });
  };

  const duplicateScenario = (index: number) => {
    const source = form.scenarios[index];
    if (!source) return;
    const copy: ConstraintScenario = {
      id: newScenarioId(),
      paramRows: structuredClone(source.paramRows),
      metaRows: structuredClone(source.metaRows),
    };
    const scenarios = [...form.scenarios];
    scenarios.splice(index + 1, 0, copy);
    onChange({ ...form, scenarios });
  };

  return (
    <div className="space-y-4">
      {multiScenario && (
        <p className="text-muted-foreground text-sm">
          Auto-approve if the action matches{" "}
          <strong className="text-foreground">any</strong> of these scenarios.
        </p>
      )}

      {form.scenarios.map((scenario, index) => (
        <ScenarioCard
          key={scenario.id}
          scenario={scenario}
          index={index}
          showScenarioHeader={multiScenario}
          canRemove={form.scenarios.length > 1}
          orderedKeys={orderedKeys}
          properties={properties}
          requiredFields={requiredFields}
          metaFields={metaFields}
          disabled={disabled}
          agentId={agentId}
          connectorId={connectorId}
          onChange={(next) => updateScenario(index, next)}
          onRemove={() => removeScenario(index)}
          onDuplicate={() => duplicateScenario(index)}
        />
      ))}

      <Button
        type="button"
        variant="outline"
        size="sm"
        className="gap-1.5"
        disabled={disabled}
        onClick={addScenario}
      >
        <Plus className="size-4" />
        Add another scenario
      </Button>
    </div>
  );
}

function ScenarioCard({
  scenario,
  index,
  showScenarioHeader,
  canRemove,
  orderedKeys,
  properties,
  requiredFields,
  metaFields,
  disabled,
  agentId,
  connectorId,
  onChange,
  onRemove,
  onDuplicate,
}: {
  scenario: ConstraintScenario;
  index: number;
  showScenarioHeader: boolean;
  canRemove: boolean;
  orderedKeys: string[];
  properties: Record<string, SchemaProperty>;
  requiredFields: string[];
  metaFields: string[];
  disabled?: boolean;
  agentId?: number;
  connectorId?: string;
  onChange: (scenario: ConstraintScenario) => void;
  onRemove: () => void;
  onDuplicate: () => void;
}) {
  const paramValues = flattenRowsToValues(scenario.paramRows);

  function updateParamRows(field: string, rows: ConstraintValueRow[]) {
    onChange({
      ...scenario,
      paramRows: { ...scenario.paramRows, [field]: rows },
    });
  }

  function updateMetaRows(field: string, rows: ConstraintValueRow[]) {
    onChange({
      ...scenario,
      metaRows: { ...scenario.metaRows, [field]: rows },
    });
  }

  return (
    <div className="rounded-lg border bg-card p-3 shadow-sm">
      {showScenarioHeader && (
        <div className="mb-3 flex items-center justify-between gap-2">
          <p className="text-sm font-medium">Scenario {index + 1}</p>
          <div className="flex gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-8"
              disabled={disabled}
              onClick={onDuplicate}
              title="Duplicate scenario"
            >
              <Copy className="size-4" />
            </Button>
            {canRemove && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-8 text-destructive"
                disabled={disabled}
                onClick={onRemove}
                title="Remove scenario"
              >
                <Trash2 className="size-4" />
              </Button>
            )}
          </div>
        </div>
      )}

      <div className="space-y-4">
        {orderedKeys.map((key) => {
          const prop = properties[key];
          if (!prop || isFieldHidden(prop)) return null;
          if (!isFieldVisible(prop, paramValues)) return null;
          const rows = scenario.paramRows[key] ?? [emptyConstraintRow()];
          const label = getFieldLabel(key, prop);
          return (
            <FieldConstraintRows
              key={key}
              fieldKey={key}
              label={label}
              isRequired={requiredFields.includes(key)}
              property={prop}
              rows={rows}
              disabled={disabled}
              agentId={agentId}
              connectorId={connectorId}
              onRowsChange={(next) => updateParamRows(key, next)}
            />
          );
        })}

        {metaFields.map((field) => {
          const rows = scenario.metaRows[field] ?? [];
          if (rows.length === 0) return null;
          return (
            <FieldConstraintRows
              key={`meta-${field}`}
              fieldKey={field}
              label={metaConstraintLabel(field)}
              isRequired={false}
              property={{ type: "string" }}
              rows={rows}
              disabled={disabled}
              onRowsChange={(next) => updateMetaRows(field, next)}
            />
          );
        })}

        {metaFields.map((field) => {
          const rows = scenario.metaRows[field] ?? [];
          if (rows.length > 0) return null;
          return (
            <div key={`meta-add-${field}`} className="flex justify-end">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={disabled}
                onClick={() =>
                  updateMetaRows(field, [emptyConstraintRow()])
                }
              >
                Add {metaConstraintLabel(field)} rule
              </Button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function FieldConstraintRows({
  fieldKey,
  label,
  isRequired,
  property,
  rows,
  disabled,
  agentId,
  connectorId,
  onRowsChange,
}: {
  fieldKey: string;
  label: string;
  isRequired: boolean;
  property: SchemaProperty;
  rows: ConstraintValueRow[];
  disabled?: boolean;
  agentId?: number;
  connectorId?: string;
  onRowsChange: (rows: ConstraintValueRow[]) => void;
}) {
  const summary = summarizeFieldRows(label, rows);

  function updateRow(rowId: string, patch: Partial<ConstraintValueRow>) {
    onRowsChange(
      rows.map((row) => (row.id === rowId ? { ...row, ...patch } : row)),
    );
  }

  function removeRow(rowId: string) {
    const next = rows.filter((r) => r.id !== rowId);
    onRowsChange(next.length > 0 ? next : [emptyConstraintRow()]);
  }

  function addRow() {
    onRowsChange([...rows, emptyConstraintRow()]);
  }

  return (
    <div className="space-y-2 rounded-md border border-dashed px-3 py-2">
      <div className="flex items-center justify-between gap-2">
        <Label className="text-sm font-medium">{label}</Label>
        {isRequired && (
          <span className="text-muted-foreground text-xs">required</span>
        )}
      </div>
      {summary && (
        <p className="text-muted-foreground text-xs italic">{summary}</p>
      )}
      <div className="space-y-2">
        {rows.map((row) => (
          <ConstraintValueRowEditor
            key={row.id}
            row={row}
            fieldKey={fieldKey}
            property={property}
            disabled={disabled}
            canRemove={rows.length > 1}
            agentId={agentId}
            connectorId={connectorId}
            onChange={(patch) => updateRow(row.id, patch)}
            onRemove={() => removeRow(row.id)}
          />
        ))}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-8 gap-1 px-2 text-xs"
        disabled={disabled}
        onClick={addRow}
      >
        <Plus className="size-3" />
        Add rule for {label}
      </Button>
    </div>
  );
}

function ConstraintValueRowEditor({
  row,
  fieldKey,
  property,
  disabled,
  canRemove,
  agentId,
  connectorId,
  onChange,
  onRemove,
}: {
  row: ConstraintValueRow;
  fieldKey: string;
  property: SchemaProperty;
  disabled?: boolean;
  canRemove: boolean;
  agentId?: number;
  connectorId?: string;
  onChange: (patch: Partial<ConstraintValueRow>) => void;
  onRemove: () => void;
}) {
  const comparable = isComparableField(property);
  const enumField = isEnumField(property);
  const booleanField = isBooleanField(property);
  const discreteField = enumField || booleanField;
  const isComparison =
    row.operator === "lte" ||
    row.operator === "gte" ||
    row.operator === "lt" ||
    row.operator === "gt";
  const isWildcard = row.mode === "wildcard" && !isComparison;

  const operatorOptions: Array<{ value: RowOperator; label: string }> = comparable
    ? [
        { value: "matches", label: "is exactly" },
        { value: "does_not_match", label: "is not" },
        { value: "lte", label: "is at most" },
        { value: "gte", label: "is at least" },
        { value: "lt", label: "is less than" },
        { value: "gt", label: "is greater than" },
      ]
    : [
        { value: "matches", label: "is" },
        { value: "does_not_match", label: "is not" },
      ];

  return (
    <div className="flex flex-wrap items-start gap-2">
      <Select
        value={row.operator}
        onValueChange={(v) => {
          const nextOp = v as RowOperator;
          const patch: Partial<ConstraintValueRow> = { operator: nextOp };
          if (
            nextOp === "lte" ||
            nextOp === "gte" ||
            nextOp === "lt" ||
            nextOp === "gt"
          ) {
            patch.mode = "fixed";
            if (row.mode === "wildcard") {
              patch.value = "";
            }
          }
          onChange(patch);
        }}
        disabled={disabled}
      >
        <SelectTrigger className="h-9 w-[160px] shrink-0">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {operatorOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="min-w-[160px] flex-1">
        {booleanField ? (
          <BooleanConstraintSelect
            row={row}
            disabled={disabled || row.operator === "does_not_match"}
            onChange={onChange}
          />
        ) : enumField ? (
          <EnumConstraintSelect
            property={property}
            row={row}
            disabled={disabled || row.operator === "does_not_match"}
            onChange={onChange}
          />
        ) : (
          <ParameterFieldWidget
            paramKey={fieldKey}
            property={property}
            value={isWildcard ? "" : row.value}
            onChange={(v) => {
              const mode: ParamMode =
                v === "*"
                  ? "wildcard"
                  : v.includes("*")
                    ? "pattern"
                    : "fixed";
              onChange({ value: v, mode });
            }}
            disabled={disabled || isWildcard}
            className={isWildcard ? "bg-muted" : ""}
            placeholder={isWildcard ? "Any value" : undefined}
            agentId={agentId}
            connectorId={connectorId}
          />
        )}
      </div>

      {!isComparison && !discreteField && (
        <label className="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs">
          <Checkbox
            checked={isWildcard}
            disabled={disabled || row.operator === "does_not_match"}
            onCheckedChange={(checked) => {
              if (checked === true) {
                onChange({ mode: "wildcard", value: "*" });
              } else {
                onChange({ mode: "fixed", value: "" });
              }
            }}
          />
          Any value
        </label>
      )}

      {canRemove && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0"
          disabled={disabled}
          onClick={onRemove}
        >
          <Trash2 className="size-4" />
        </Button>
      )}
    </div>
  );
}

function EnumConstraintSelect({
  property,
  row,
  disabled,
  onChange,
}: {
  property: SchemaProperty;
  row: ConstraintValueRow;
  disabled?: boolean;
  onChange: (patch: Partial<ConstraintValueRow>) => void;
}) {
  const isWildcard = row.mode === "wildcard" || row.value === "*";
  const selectValue = isWildcard ? "*" : row.value;

  return (
    <Select
      value={selectValue}
      onValueChange={(v) => {
        if (v === "*") {
          onChange({ mode: "wildcard", value: "*" });
        } else {
          onChange({ mode: "fixed", value: v });
        }
      }}
      disabled={disabled}
    >
      <SelectTrigger className="h-9 w-full">
        <SelectValue placeholder="Select…" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="*">Any value</SelectItem>
        {(property.enum ?? []).map((opt) => (
          <SelectItem key={opt} value={opt}>
            {opt}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function BooleanConstraintSelect({
  row,
  disabled,
  onChange,
}: {
  row: ConstraintValueRow;
  disabled?: boolean;
  onChange: (patch: Partial<ConstraintValueRow>) => void;
}) {
  const isWildcard = row.mode === "wildcard" || row.value === "*";
  let selectValue = "any";
  if (!isWildcard) {
    if (row.value === "true") {
      selectValue = "true";
    } else if (row.value === "false") {
      selectValue = "false";
    }
  }

  return (
    <Select
      value={selectValue}
      onValueChange={(v) => {
        if (v === "any") {
          onChange({ mode: "wildcard", value: "*" });
        } else {
          onChange({ mode: "fixed", value: v });
        }
      }}
      disabled={disabled}
    >
      <SelectTrigger className="h-9 w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="any">Any value</SelectItem>
        <SelectItem value="true">Yes</SelectItem>
        <SelectItem value="false">No</SelectItem>
      </SelectContent>
    </Select>
  );
}

function flattenRowsToValues(
  rowsByField: Record<string, ConstraintValueRow[]>,
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, rows] of Object.entries(rowsByField)) {
    const first = rows[0];
    if (first) {
      out[key] = first.value;
    }
  }
  return out;
}

/** Ensure every schema param has at least one empty row in the first scenario. */
export function ensureScenarioFieldRows(
  form: StructuredConstraintFormState,
  paramKeys: string[],
  metaFields: string[],
): StructuredConstraintFormState {
  const scenarios = form.scenarios.map((scenario, index) => {
    if (index > 0) return scenario;
    const paramRows = { ...scenario.paramRows };
    for (const key of paramKeys) {
      if (!paramRows[key]?.length) {
        paramRows[key] = [emptyConstraintRow()];
      }
    }
    const metaRows = { ...scenario.metaRows };
    for (const field of metaFields) {
      if (!metaRows[field]) {
        metaRows[field] = [];
      }
    }
    return { ...scenario, paramRows, metaRows };
  });
  return { ...form, scenarios };
}
