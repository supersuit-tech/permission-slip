import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Asterisk, ChevronDown, Lock, Regex } from "lucide-react";
import type { ParamMode } from "./ActionConfigFormFields";
import type { ParametersSchema } from "@/lib/parameterSchema";

interface ActionConfigParameterFieldsProps {
  parametersSchema: ParametersSchema | null;
  values: Record<string, string>;
  onValueChange: (key: string, value: string) => void;
  modes: Record<string, ParamMode>;
  onModeChange: (key: string, mode: ParamMode) => void;
  disabled?: boolean;
}

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

  function getMode(key: string): ParamMode {
    return modes[key] ?? inferModeFromValue(values[key] ?? "");
  }

  return (
    <div className="space-y-3">
      <p className="text-muted-foreground text-xs">
        For each parameter, choose a constraint mode: <strong>Fixed</strong>{" "}
        locks in an exact value,{" "}
        <strong>Pattern</strong> uses{" "}
        <Badge variant="outline" className="border-dashed font-mono text-xs">
          *
        </Badge>{" "}
        as a glob wildcard (e.g.{" "}
        <code className="text-xs">*@mycompany.com</code>), or{" "}
        <strong>Wildcard</strong> lets the agent choose freely.
      </p>
      {Object.entries(properties).map(([key, prop]) => {
        const isRequired = requiredFields.includes(key);
        const currentValue = values[key] ?? "";
        const mode = getMode(key);

        return (
          <div key={key} className="space-y-1.5">
            <div className="flex items-center gap-2">
              <Label htmlFor={`param-${key}`} className="text-sm font-medium">
                {key}
              </Label>
              {isRequired && (
                <Badge variant="secondary" className="text-xs">
                  required
                </Badge>
              )}
              {prop.type && (
                <span className="text-muted-foreground text-xs">
                  ({prop.type})
                </span>
              )}
            </div>
            {prop.description && (
              <p className="text-muted-foreground text-xs">
                {prop.description}
              </p>
            )}
            <div className="flex items-center gap-2">
              <Input
                id={`param-${key}`}
                placeholder={
                  mode === "wildcard"
                    ? "Agent can use any value"
                    : mode === "pattern"
                      ? "e.g. *@mycompany.com, supersuit-tech/*"
                      : `Enter value for ${key}`
                }
                value={mode === "wildcard" ? "" : currentValue}
                onChange={(e) => onValueChange(key, e.target.value)}
                disabled={disabled || mode === "wildcard"}
                className={mode === "wildcard" ? "bg-muted" : ""}
              />
              <ConstraintModeDropdown
                mode={mode}
                disabled={disabled}
                onChange={(nextMode) => {
                  const prevMode = mode;
                  onModeChange(key, nextMode);
                  if (nextMode === "wildcard") {
                    onValueChange(key, "*");
                  } else if (prevMode === "wildcard") {
                    // Coming from wildcard — clear the "*" sentinel
                    onValueChange(key, "");
                  }
                  // Switching between fixed ↔ pattern preserves the typed value
                }}
              />
            </div>
            {mode === "pattern" && currentValue !== "" && !currentValue.includes("*") && (
              <p className="text-muted-foreground text-xs">
                Tip: Include <code className="font-mono">*</code> in the value
                for glob matching (e.g. <code>*@mycompany.com</code>).
              </p>
            )}
          </div>
        );
      })}
    </div>
  );
}

/**
 * Infer the mode from a plain string value. Used as a fallback when no
 * explicit mode override exists (e.g., first render before any user clicks).
 * Note: this only handles plain string values. $pattern wrapper objects should
 * be unwrapped before the value reaches this component.
 */
function inferModeFromValue(value: string): ParamMode {
  if (value === "*") return "wildcard";
  return "fixed";
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
            if (v !== mode) onChange(v as ParamMode);
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
