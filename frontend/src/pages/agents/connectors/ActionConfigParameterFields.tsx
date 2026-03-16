import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Asterisk, ChevronDown, ChevronRight, Lock, Regex } from "lucide-react";
import type { ParamMode } from "./ActionConfigFormFields";
import { ParameterFieldWidget } from "./ParameterFieldWidget";
import type { ParametersSchema, SchemaProperty, FieldGroup } from "@/lib/parameterSchema";
import {
  getOrderedFieldKeys,
  getFieldLabel,
  isFieldVisible,
} from "@/lib/parameterSchema";

interface ActionConfigParameterFieldsProps {
  parametersSchema: ParametersSchema | null;
  values: Record<string, string>;
  onValueChange: (key: string, value: string) => void;
  modes: Record<string, ParamMode>;
  onModeChange: (key: string, mode: ParamMode) => void;
  disabled?: boolean;
}

/**
 * Renders parameter fields for action configuration, with a constraint mode
 * dropdown (Fixed/Pattern/Wildcard) per parameter. Used in both the Add and
 * Edit action configuration dialogs.
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
        mode={mode}
        disabled={disabled}
        onValueChange={onValueChange}
        onModeChange={onModeChange}
      />
    );
  }

  const introText = (
    <p className="text-muted-foreground text-sm leading-relaxed">
      For each parameter, choose a constraint mode: <strong>Fixed</strong>{" "}
      locks in an exact value,{" "}
      <strong>Pattern</strong> uses{" "}
      <code className="rounded bg-muted px-1 font-mono">*</code> as a glob
      wildcard (e.g.{" "}
      <code className="rounded bg-muted px-1 font-mono">*@mycompany.com</code>
      ), or <strong>Wildcard</strong> lets the agent choose freely.
    </p>
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

/** Renders a single parameter field with label, widget, and constraint mode dropdown. */
function ParameterField({
  paramKey,
  property,
  label,
  isRequired,
  value,
  mode,
  disabled,
  onValueChange,
  onModeChange,
}: {
  paramKey: string;
  property: SchemaProperty;
  label: string;
  isRequired: boolean;
  value: string;
  mode: ParamMode;
  disabled?: boolean;
  onValueChange: (key: string, value: string) => void;
  onModeChange: (key: string, mode: ParamMode) => void;
}) {
  // Mode-aware placeholder: wildcard/pattern override x-ui.placeholder
  const modePlaceholder =
    mode === "wildcard"
      ? "Agent can use any value"
      : mode === "pattern"
        ? "e.g. *@mycompany.com, supersuit-tech/*"
        : undefined; // let ParameterFieldWidget use x-ui.placeholder or its default

  const widgetValue = mode === "wildcard" ? "" : value;
  const widgetDisabled = disabled || mode === "wildcard";
  const widgetClassName = mode === "wildcard" ? "bg-muted" : "";

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2">
        <Label htmlFor={`param-${paramKey}`} className="text-sm font-medium">
          {label}
        </Label>
        {isRequired && (
          <Badge variant="secondary" className="text-xs">
            required
          </Badge>
        )}
        {property.type && (
          <span className="text-muted-foreground text-xs">
            ({property.type})
          </span>
        )}
      </div>
      {property.description && (
        <p className="text-muted-foreground text-sm">
          {property.description}
        </p>
      )}
      <div className="flex items-center gap-2">
        <ParameterFieldWidget
          paramKey={paramKey}
          property={property}
          value={widgetValue}
          onChange={(v) => onValueChange(paramKey, v)}
          disabled={widgetDisabled}
          className={widgetClassName}
          placeholder={modePlaceholder}
        />
        <ConstraintModeDropdown
          mode={mode}
          disabled={disabled}
          onChange={(nextMode) => {
            const prevMode = mode;
            onModeChange(paramKey, nextMode);
            if (nextMode === "wildcard") {
              onValueChange(paramKey, "*");
            } else if (prevMode === "wildcard") {
              onValueChange(paramKey, "");
            }
          }}
        />
      </div>
      {mode === "pattern" && value !== "" && !value.includes("*") && (
        <p className="text-muted-foreground text-sm">
          Tip: Include <code className="rounded bg-muted px-1 font-mono">*</code> in the value
          for glob matching (e.g. <code className="rounded bg-muted px-1 font-mono">*@mycompany.com</code>).
        </p>
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

const modeConfig: Record<
  ParamMode,
  { icon: React.ReactNode; label: string; description: string }
> = {
  fixed: {
    icon: <Lock className="size-3" />,
    label: "Fixed",
    description: "Exact value",
  },
  pattern: {
    icon: <Regex className="size-3" />,
    label: "Pattern",
    description: "Glob matching with *",
  },
  wildcard: {
    icon: <Asterisk className="size-3" />,
    label: "Wildcard",
    description: "Agent chooses freely",
  },
};

const allModes: ParamMode[] = ["fixed", "pattern", "wildcard"];

function isParamMode(value: string): value is ParamMode {
  return allModes.includes(value as ParamMode);
}

/** Dropdown selector for constraint mode (Fixed/Pattern/Wildcard) with radio semantics. */
function ConstraintModeDropdown({
  mode,
  disabled,
  onChange,
}: {
  mode: ParamMode;
  disabled?: boolean;
  onChange: (next: ParamMode) => void;
}) {
  const current = modeConfig[mode];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          title={current.description}
          className="shrink-0 gap-1.5"
        >
          {current.icon}
          {current.label}
          <ChevronDown className="size-3 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={mode}
          onValueChange={(v) => {
            if (v !== mode && isParamMode(v)) onChange(v);
          }}
        >
          {allModes.map((m) => {
            const cfg = modeConfig[m];
            return (
              <DropdownMenuRadioItem key={m} value={m}>
                {cfg.icon}
                <span className="font-medium">{cfg.label}</span>
                <span className="text-muted-foreground text-xs">
                  {cfg.description}
                </span>
              </DropdownMenuRadioItem>
            );
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// Re-export the shared parseParametersSchema so callers don't break.
export { parseParametersSchema } from "@/lib/parameterSchema";
