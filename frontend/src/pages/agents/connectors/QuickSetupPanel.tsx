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
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <Label
            htmlFor="quick-read"
            className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
          >
            Read actions
          </Label>
          <Select
            value={quickRead}
            onValueChange={(v) => onQuickReadChange(v as ApprovalMode)}
            disabled={disabled}
          >
            <SelectTrigger
              id="quick-read"
              data-testid="quick-setup-read"
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
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <Label
            htmlFor="quick-write"
            className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
          >
            Write actions
          </Label>
          <Select
            value={quickWrite}
            onValueChange={(v) => onQuickWriteChange(v as ApprovalMode)}
            disabled={disabled}
          >
            <SelectTrigger
              id="quick-write"
              data-testid="quick-setup-write"
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
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <Label
            htmlFor="quick-edit"
            className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
          >
            Edit actions
          </Label>
          <Select
            value={quickEdit}
            onValueChange={(v) => onQuickEditChange(v as ApprovalMode)}
            disabled={disabled}
          >
            <SelectTrigger
              id="quick-edit"
              data-testid="quick-setup-edit"
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
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <Label
            htmlFor="quick-delete"
            className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
          >
            Delete actions
          </Label>
          <Select
            value={quickDelete}
            onValueChange={(v) => onQuickDeleteChange(v as ApprovalMode)}
            disabled={disabled}
          >
            <SelectTrigger
              id="quick-delete"
              data-testid="quick-setup-delete"
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
