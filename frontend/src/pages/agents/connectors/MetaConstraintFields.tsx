import { Asterisk } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { metaConstraintLabel } from "@/lib/constraints";
import type { ParamMode } from "./StandingApprovalFormFields";

interface MetaConstraintFieldsProps {
  fields: string[];
  values: Record<string, string>;
  onValueChange: (key: string, value: string) => void;
  modes: Record<string, ParamMode>;
  onModeChange: (key: string, mode: ParamMode) => void;
  disabled?: boolean;
}

export function MetaConstraintFields({
  fields,
  values,
  onValueChange,
  modes,
  onModeChange,
  disabled,
}: MetaConstraintFieldsProps) {
  if (fields.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
        <p className="text-muted-foreground text-xs leading-relaxed">
          Verified metadata constraints match server-fetched envelope data (for
          example sender/recipient), not agent-supplied parameters.
        </p>
      </div>
      {fields.map((fieldKey) => {
        const label = metaConstraintLabel(fieldKey);
        const mode = modes[fieldKey] ?? "fixed";
        const isWildcard = mode === "wildcard";
        const value = isWildcard ? "" : (values[fieldKey] ?? "");

        return (
          <div key={fieldKey} className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor={`meta-${fieldKey}`} className="text-sm font-medium">
                {label}
              </Label>
              <label className="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs whitespace-nowrap">
                <Checkbox
                  checked={isWildcard}
                  disabled={disabled}
                  onCheckedChange={(checked) => {
                    if (checked === true) {
                      onModeChange(fieldKey, "wildcard");
                      onValueChange(fieldKey, "*");
                    } else if (checked === false) {
                      onModeChange(fieldKey, "fixed");
                      onValueChange(fieldKey, "");
                    }
                  }}
                />
                <Asterisk className="size-3" />
                Any value
              </label>
            </div>
            <Input
              id={`meta-${fieldKey}`}
              value={value}
              disabled={disabled || isWildcard}
              placeholder={
                isWildcard
                  ? "Agent can use any value"
                  : fieldKey === "from" || fieldKey === "sender"
                    ? "e.g. automated@airbnb.com or *@company.com"
                    : undefined
              }
              className={isWildcard ? "bg-muted" : undefined}
              onChange={(e) => onValueChange(fieldKey, e.target.value)}
            />
          </div>
        );
      })}
    </div>
  );
}
