import { Switch } from "@/components/ui/switch";

type Props = {
  enabled: boolean;
  disabled: boolean;
  onCheckedChange: (checked: boolean) => void;
};

/**
 * "Notify me about" row for auto-approval execution notifications.
 * Extracted so Storybook can mirror production markup without mocking hooks.
 */
export function NotifyAboutAutoApprovalsRow({
  enabled,
  disabled,
  onCheckedChange,
}: Props) {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <p className="text-sm font-medium">Auto-approval executions</p>
          <p className="text-xs text-muted-foreground">
            When an action runs automatically because it matched a standing
            approval.
          </p>
        </div>
        <Switch
          checked={enabled}
          disabled={disabled}
          aria-label="Auto-approval execution notifications"
          onCheckedChange={onCheckedChange}
        />
      </div>
    </div>
  );
}
