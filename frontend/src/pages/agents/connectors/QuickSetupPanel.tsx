import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { approvalModeOptions, type ApprovalMode } from "./recommendedTemplatesTypes";

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
  quickRead: ApprovalMode;
  quickWrite: ApprovalMode;
  quickEdit: ApprovalMode;
  quickDelete: ApprovalMode;
  onQuickReadChange: (v: ApprovalMode) => void;
  onQuickWriteChange: (v: ApprovalMode) => void;
  onQuickEditChange: (v: ApprovalMode) => void;
  onQuickDeleteChange: (v: ApprovalMode) => void;
  onApply: () => void;
  disabled: boolean;
  applyDisabled: boolean;
}) {
  return (
    <div className="bg-muted/40 space-y-3 rounded-lg border border-input p-3">
      <p className="text-sm font-semibold">Quick setup</p>
      <div className="space-y-2">
        <QuickSetupRow
          id="quick-read"
          testId="quick-setup-read"
          label="Read actions"
          value={quickRead}
          onChange={onQuickReadChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-write"
          testId="quick-setup-write"
          label="Write actions"
          value={quickWrite}
          onChange={onQuickWriteChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-edit"
          testId="quick-setup-edit"
          label="Edit actions"
          value={quickEdit}
          onChange={onQuickEditChange}
          disabled={disabled}
        />
        <QuickSetupRow
          id="quick-delete"
          testId="quick-setup-delete"
          label="Delete actions"
          value={quickDelete}
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
        Apply
      </Button>
    </div>
  );
}

function QuickSetupRow({
  id,
  testId,
  label,
  value,
  onChange,
  disabled,
}: {
  id: string;
  testId: string;
  label: string;
  value: ApprovalMode;
  onChange: (v: ApprovalMode) => void;
  disabled: boolean;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 sm:gap-3">
      <Label
        htmlFor={id}
        className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
      >
        {label}
      </Label>
      <Select
        value={value}
        onValueChange={(v) => onChange(v as ApprovalMode)}
        disabled={disabled}
      >
        <SelectTrigger
          id={id}
          data-testid={testId}
          size="sm"
          className="max-w-[11rem] min-w-0 flex-1"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {approvalModeOptions.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
