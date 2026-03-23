import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Asterisk, ChevronDown, ChevronRight } from "lucide-react";
import type { ParamMode } from "./ActionConfigFormFields";
import { ParameterFieldWidget } from "./ParameterFieldWidget";
import type { ParametersSchema, SchemaProperty, FieldGroup } from "@/lib/parameterSchema";
import {
  getOrderedFieldKeys,
  getFieldLabel,
  isFieldVisible,
  inferWidgetFromProperty,
} from "@/lib/parameterSchema";

interface ActionConfigParameterFieldsProps {
  parametersSchema: ParametersSchema | null;
  values: Record<string, string>;
  onValueChange: (key: string, value: string) => void;
  modes: Record<string, ParamMode>;
  onModeChange: (key: string, mode: ParamMode) => void;
  disabled?: boolean;
  agentId?: number;
  connectorId?: string;
}

/**
 * Renders parameter fields for action configuration, with an "Any value"
 * checkbox per parameter. Wildcards in values (e.g. `*@company.com`) are
 * auto-detected and serialized as patterns. Used in both the Add and Edit
 * action configuration dialogs.
 *
 * Supports x-ui rendering hints: field ordering, labels, placeholders,
 * grouped collapsible sections, conditional visibility, and widget types.
 */
export function ActionConfigParameterFields({
  parametersSchema,
  values,
  onValueChange,
  modes,
  onModeChange,
  disabled,
  agentId,
  connectorId,
}: ActionConfigParameterFieldsProps) {
  if (!parametersSchema?.properties) {
    return (
      <p className="text-muted-foreground text-sm">
        This action has no configurable parameters.
      </p>
    );
  }

  const properties = parametersSchema.properties;
  const requiredFields = parametersSchema.required ?? [];
  const orderedKeys = getOrderedFieldKeys(parametersSchema);
  const groups = parametersSchema["x-ui"]?.groups;

  function getMode(key: string): ParamMode {
    return modes[key] ?? "fixed";
  }

  function renderField(key: string) {
    const prop = properties[key];
    if (!prop) return null;
    if (!isFieldVisible(prop, values)) return null;

    const isRequired = requiredFields.includes(key);
    const currentValue = values[key] ?? "";
    const mode = getMode(key);
    const label = getFieldLabel(key, prop);

    return (
      <ParameterField
        key={key}
        paramKey={key}
        property={prop}
        label={label}
        isRequired={isRequired}
        value={currentValue}
        allValues={values}
        mode={mode}
        disabled={disabled}
        onValueChange={onValueChange}
        onModeChange={onModeChange}
        agentId={agentId}
        connectorId={connectorId}
      />
    );
  }

  const introText = (
    <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
      <p className="text-muted-foreground text-xs leading-relaxed">
        Use{" "}
        <code className="rounded bg-muted px-1 font-mono text-foreground/70">*</code>{" "}
        as a wildcard in any value (e.g.{" "}
        <code className="rounded bg-muted px-1 font-mono text-foreground/70">*@mycompany.com</code>
        ). Check <strong className="text-foreground/70">Any value</strong> to let the agent choose freely.
      </p>
    </div>
  );

  // If groups are defined, render ungrouped fields then each group section
  if (groups && groups.length > 0) {
    const groupIds = new Set(groups.map((g) => g.id));
    const ungroupedKeys = orderedKeys.filter(
      (k) => !properties[k]?.["x-ui"]?.group || !groupIds.has(properties[k]?.["x-ui"]?.group ?? ""),
    );
    const keysByGroup = new Map<string, string[]>();
    for (const g of groups) {
      keysByGroup.set(g.id, []);
    }
    for (const key of orderedKeys) {
      const groupId = properties[key]?.["x-ui"]?.group;
      if (groupId && keysByGroup.has(groupId)) {
        keysByGroup.get(groupId)!.push(key);
      }
    }

    return (
      <div className="space-y-3">
        {introText}
        {ungroupedKeys.map(renderField)}
        {groups.map((group) => {
          const groupKeys = keysByGroup.get(group.id) ?? [];
          // Filter by visibility so empty groups don't render when all fields are hidden
          const visibleGroupKeys = groupKeys.filter(
            (k) => properties[k] !== undefined && isFieldVisible(properties[k]!, values),
          );
          if (visibleGroupKeys.length === 0) return null;
          return (
            <CollapsibleFieldGroup key={group.id} group={group}>
              {groupKeys.map(renderField)}
            </CollapsibleFieldGroup>
          );
        })}
      </div>
    );
  }

  // No groups: flat ordered list
  return (
    <div className="space-y-3">
      {introText}
      {orderedKeys.map(renderField)}
    </div>
  );
}

/** Renders a single parameter field with label, widget, and "Any value" checkbox. */
function ParameterField({
  paramKey,
  property,
  label,
  isRequired,
  value,
  allValues,
  mode,
  disabled,
  onValueChange,
  onModeChange,
  agentId,
  connectorId,
}: {
  paramKey: string;
  property: SchemaProperty;
  label: string;
  isRequired: boolean;
  value: string;
  allValues: Record<string, string>;
  mode: ParamMode;
  disabled?: boolean;
  onValueChange: (key: string, value: string) => void;
  onModeChange: (key: string, mode: ParamMode) => void;
  agentId?: number;
  connectorId?: string;
}) {
  const isWildcard = mode === "wildcard";
  const widgetValue = isWildcard ? "" : value;
  const widgetDisabled = disabled || isWildcard;
  const widgetClassName = isWildcard ? "bg-muted" : "";
  const widgetPlaceholder = isWildcard ? "Agent can use any value" : undefined;
  const effectiveWidget = property["x-ui"]?.widget ?? inferWidgetFromProperty(property);
  const isMultiRow = effectiveWidget === "list";

  // In constraint context, use plain text only when the value needs freeform entry
  // (wildcards / patterns). Otherwise keep datetime-local so bounds like time_min /
  // time_max are easy to set; pair hints still apply via x-ui.datetime_range_*.
  const constraintProperty: typeof property =
    isDatetimeLikeProperty(property) && shouldUseTextDatetimeConstraint(value)
      ? { ...property, "x-ui": { ...(property["x-ui"] ?? {}), widget: "text" } }
      : property;

  const pairKey = property["x-ui"]?.datetime_range_pair;
  const siblingDatetimeValue =
    pairKey && allValues[pairKey] !== undefined ? allValues[pairKey] : undefined;

  const anyValueCheckbox = (
    <label
      className="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs whitespace-nowrap"
    >
      <Checkbox
        checked={isWildcard}
        disabled={disabled}
        onCheckedChange={(checked) => {
          if (checked === true) {
            onModeChange(paramKey, "wildcard");
            onValueChange(paramKey, "*");
          } else if (checked === false) {
            onModeChange(paramKey, "fixed");
            onValueChange(paramKey, "");
          }
        }}
      />
      <Asterisk className="size-3" />
      Any value
    </label>
  );

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Label htmlFor={`param-${paramKey}`} className="text-sm font-medium">
            {label}
          </Label>
          {isRequired && (
            <Badge variant="secondary" className="text-xs">
              required
            </Badge>
          )}
        </div>
        {anyValueCheckbox}
      </div>
      {isMultiRow ? (
        !isWildcard && (
          <ParameterFieldWidget
            paramKey={paramKey}
            property={constraintProperty}
            value={widgetValue}
            onChange={(v) => onValueChange(paramKey, v)}
            disabled={widgetDisabled}
            className={widgetClassName}
            placeholder={widgetPlaceholder}
            siblingDatetimeValue={siblingDatetimeValue}
            agentId={agentId}
            connectorId={connectorId}
          />
        )
      ) : (
        <ParameterFieldWidget
          paramKey={paramKey}
          property={constraintProperty}
          value={widgetValue}
          onChange={(v) => onValueChange(paramKey, v)}
          disabled={widgetDisabled}
          className={widgetClassName}
          placeholder={widgetPlaceholder}
          siblingDatetimeValue={siblingDatetimeValue}
          agentId={agentId}
          connectorId={connectorId}
        />
      )}
      {!isWildcard && value.includes("*") && (
        <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
          <p className="text-muted-foreground text-xs leading-relaxed">
            <code className="rounded bg-muted px-1 font-mono text-foreground/70">*</code> matches any text
          </p>
        </div>
      )}
    </div>
  );
}

/** Collapsible section for a group of related fields. */
function CollapsibleFieldGroup({
  group,
  children,
}: {
  group: FieldGroup;
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(!group.collapsed);

  return (
    <div className="rounded-md border">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm font-medium hover:bg-muted/50"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-expanded={isOpen}
      >
        {isOpen ? (
          <ChevronDown className="size-4 shrink-0" />
        ) : (
          <ChevronRight className="size-4 shrink-0" />
        )}
        {group.label}
        {group.description && (
          <span className="text-muted-foreground text-xs font-normal">
            — {group.description}
          </span>
        )}
      </button>
      {isOpen && (
        <div className="space-y-3 border-t px-3 py-3">{children}</div>
      )}
    </div>
  );
}


function isDatetimeLikeProperty(prop: SchemaProperty): boolean {
  if (prop.format === "date-time") return true;
  if (prop["x-ui"]?.widget === "datetime") return true;
  if (prop.type === "string" && prop.description) {
    const d = prop.description.toLowerCase();
    return (
      (d.includes("rfc 3339") || d.includes("rfc3339") || d.includes("iso 8601")) &&
      !d.includes("epoch")
    );
  }
  return false;
}

/** True when the value is a single concrete instant (not a wildcard pattern). */
function isConcreteDatetimeString(value: string): boolean {
  if (!value || value.includes("*")) return false;
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) return true;
  const t = new Date(value).getTime();
  return !Number.isNaN(t);
}

function shouldUseTextDatetimeConstraint(value: string): boolean {
  if (value.includes("*")) return true;
  if (value !== "" && !isConcreteDatetimeString(value)) return true;
  return false;
}

// Re-export the shared parseParametersSchema so callers don't break.
export { parseParametersSchema } from "@/lib/parameterSchema";
