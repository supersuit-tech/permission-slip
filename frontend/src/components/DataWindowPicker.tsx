import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import type { DataWindowFormState } from "@/lib/dataWindow";

const selectClassName =
  "border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50";

export function DataWindowPicker({
  value,
  onChange,
  disabled,
}: {
  value: DataWindowFormState;
  onChange: (next: DataWindowFormState) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border bg-muted/30 p-3">
      <div className="flex items-start gap-2">
        <Checkbox
          id="data-window-enabled"
          checked={value.enabled}
          disabled={disabled}
          onCheckedChange={(checked) =>
            onChange({ ...value, enabled: checked === true })
          }
        />
        <div className="space-y-1">
          <Label htmlFor="data-window-enabled" className="cursor-pointer">
            Limit how far back data can be read
          </Label>
          <p className="text-muted-foreground text-xs leading-relaxed">
            Restricts the time range the agent may read, even if it omits date
            parameters in the request.
          </p>
        </div>
      </div>

      {value.enabled ? (
        <div className="space-y-3 pl-6">
          <div className="space-y-1">
            <Label htmlFor="data-window-mode">Window type</Label>
            <select
              id="data-window-mode"
              className={selectClassName}
              disabled={disabled}
              value={value.mode}
              onChange={(e) =>
                onChange({
                  ...value,
                  mode: e.target.value as DataWindowFormState["mode"],
                })
              }
            >
              <option value="last_days">Rolling window (last N days)</option>
              <option value="absolute">Fixed date range</option>
            </select>
          </div>

          {value.mode === "last_days" ? (
            <div className="space-y-1">
              <Label htmlFor="data-window-last-days">Last N days</Label>
              <Input
                id="data-window-last-days"
                type="number"
                min={1}
                disabled={disabled}
                value={value.lastDays}
                onChange={(e) =>
                  onChange({ ...value, lastDays: e.target.value })
                }
              />
            </div>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label htmlFor="data-window-starts-at">From</Label>
                <Input
                  id="data-window-starts-at"
                  type="datetime-local"
                  disabled={disabled}
                  value={value.startsAt}
                  onChange={(e) =>
                    onChange({ ...value, startsAt: e.target.value })
                  }
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="data-window-ends-at">Until</Label>
                <Input
                  id="data-window-ends-at"
                  type="datetime-local"
                  disabled={disabled}
                  value={value.endsAt}
                  onChange={(e) =>
                    onChange({ ...value, endsAt: e.target.value })
                  }
                />
              </div>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}
