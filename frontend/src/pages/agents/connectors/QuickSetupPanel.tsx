import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";

export function QuickSetupPanel({
  quickRead,
  quickWrite,
  quickEdit,
  quickDelete,
  onQuickReadChange,
  onQuickWriteChange,
  onQuickEditChange,
  onQuickDeleteChange,
  onApply,
  disabled,
  applyDisabled,
}: {
  quickRead: boolean;
  quickWrite: boolean;
  quickEdit: boolean;
  quickDelete: boolean;
  onQuickReadChange: (v: boolean) => void;
  onQuickWriteChange: (v: boolean) => void;
  onQuickEditChange: (v: boolean) => void;
  onQuickDeleteChange: (v: boolean) => void;
  onApply: () => void;
  disabled: boolean;
  applyDisabled: boolean;
}) {
  return (
    <div className="bg-muted/40 space-y-3 rounded-lg border border-input p-3">
      <p className="text-sm font-semibold">Quick setup</p>
      <p className="text-muted-foreground text-xs">
        Select standing approval templates by action category, then apply to
        enable auto-approve for those actions.
      </p>
      <div className="space-y-2">
        <QuickSetupRow
          id="quick-read"
          testId="quick-setup-read"
          label="Read actions"
          checked={quickRead}
          onChange={onQuickReadChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-write"
          testId="quick-setup-write"
          label="Write actions"
          checked={quickWrite}
          onChange={onQuickWriteChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-edit"
          testId="quick-setup-edit"
          label="Edit actions"
          checked={quickEdit}
          onChange={onQuickEditChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-delete"
          testId="quick-setup-delete"
          label="Delete actions"
          checked={quickDelete}
          onChange={onQuickDeleteChange}
          disabled={disabled}
        />
      </div>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        className="w-full sm:w-auto"
        onClick={onApply}
        disabled={disabled || applyDisabled}
      >
        Select matching templates
      </Button>
    </div>
  );
}

function QuickSetupRow({
  id,
  testId,
  label,
  checked,
  onChange,
  disabled,
}: {
  id: string;
  testId: string;
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled: boolean;
}) {
  return (
    <label
      htmlFor={id}
      className="flex flex-wrap items-center gap-2 sm:gap-3"
      data-testid={testId}
    >
      <Checkbox
        id={id}
        checked={checked}
        onCheckedChange={(v) => onChange(v === true)}
        disabled={disabled}
      />
      <span className="text-muted-foreground text-xs sm:text-sm">{label}</span>
    </label>
  );
}
