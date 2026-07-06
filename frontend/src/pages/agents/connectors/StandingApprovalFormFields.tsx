import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import validation from "@/lib/validation";

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
        placeholder="Describe what this standing approval authorizes"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        maxLength={validation.actionConfigDescription.maxLength}
        disabled={disabled}
      />
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

export type ParamMode = "fixed" | "pattern" | "wildcard";
