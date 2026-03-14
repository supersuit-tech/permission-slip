import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import validation from "@/lib/validation";

/** Reserved action_type value meaning "all actions on this connector". */
export const WILDCARD_ACTION_TYPE = "*";

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

interface NameFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function NameField({ id, value, onChange, disabled }: NameFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Name</Label>
      <Input
        id={id}
        placeholder="e.g. Create bug issues in webapp"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        maxLength={validation.actionConfigName.maxLength}
        disabled={disabled}
        required
      />
    </div>
  );
}

interface DescriptionFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function DescriptionField({
  id,
  value,
  onChange,
  disabled,
}: DescriptionFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Description (optional)</Label>
      <Input
        id={id}
        placeholder="Describe what this configuration permits"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        maxLength={validation.actionConfigDescription.maxLength}
        disabled={disabled}
      />
    </div>
  );
}

interface StatusSelectProps {
  id: string;
  value: "active" | "disabled";
  onChange: (value: "active" | "disabled") => void;
  disabled?: boolean;
}

export function StatusSelect({
  id,
  value,
  onChange,
  disabled,
}: StatusSelectProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Status</Label>
      <select
        id={id}
        className={selectClassName}
        value={value}
        // Cast is safe: the only <option> values are "active" and "disabled"
        onChange={(e) => onChange(e.target.value as "active" | "disabled")}
        disabled={disabled}
      >
        <option value="active">Active</option>
        <option value="disabled">Disabled</option>
      </select>
    </div>
  );
}

interface ActionSelectProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  actions: Array<{ action_type: string; name: string }>;
  disabled?: boolean;
}

export function ActionSelect({
  id,
  value,
  onChange,
  actions,
  disabled,
}: ActionSelectProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Action</Label>
      <select
        id={id}
        className={selectClassName}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      >
        <option value="">Select an action...</option>
        {actions.map((action) => (
          <option key={action.action_type} value={action.action_type}>
            {action.name} ({action.action_type})
          </option>
        ))}
      </select>
    </div>
  );
}

/**
 * Returns the names of required parameters that have empty (non-wildcard) values.
 * Used to prevent submitting a form with required params accidentally omitted
 * (e.g. after toggling from wildcard to fixed without entering a value).
 */
export function getEmptyRequiredParams(
  paramValues: Record<string, string>,
  requiredFields?: string[],
): string[] {
  if (!requiredFields?.length) return [];
  return requiredFields.filter((key) => {
    const value = paramValues[key];
    return value === undefined || value === "";
  });
}

export type ParamMode = "fixed" | "pattern" | "wildcard";

/**
 * Convert form parameter values (string map) into the API request format.
 * Filters out empty strings so only user-provided values are sent.
 * When schema properties are provided, coerces values back to their declared
 * types (integer, number, boolean) so the backend receives correct JSON types.
 *
 * When paramModes is provided, parameters in "pattern" mode are wrapped as
 * {"$pattern": "<glob>"} objects for explicit pattern matching on the backend.
 */
export function buildParametersFromForm(
  paramValues: Record<string, string>,
  schemaProperties?: Record<string, { type?: string }>,
  paramModes?: Record<string, ParamMode>,
): Record<string, unknown> {
  const parameters: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(paramValues)) {
    if (value === "") continue;

    // Pattern mode: wrap in $pattern object for explicit backend handling.
    if (paramModes?.[key] === "pattern") {
      parameters[key] = { $pattern: value };
      continue;
    }

    const type = schemaProperties?.[key]?.type;
    if (type === "integer") {
      const parsed = Number.parseInt(value, 10);
      if (!Number.isNaN(parsed)) { parameters[key] = parsed; continue; }
    } else if (type === "number") {
      const parsed = Number(value);
      if (!Number.isNaN(parsed)) { parameters[key] = parsed; continue; }
    } else if (type === "boolean") {
      if (value === "true") { parameters[key] = true; continue; }
      if (value === "false") { parameters[key] = false; continue; }
    }

    parameters[key] = value;
  }
  return parameters;
}

// Re-exported from shared lib for backward compatibility.
export { isPatternWrapper } from "@/lib/constraints";
