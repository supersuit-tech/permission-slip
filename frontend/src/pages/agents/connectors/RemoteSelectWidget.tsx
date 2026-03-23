import { useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { SchemaPropertyUI } from "@/lib/parameterSchema";
import { useAgentConnectorCalendars } from "@/hooks/useAgentConnectorCalendars";

export interface RemoteSelectWidgetProps {
  inputId: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  agentId: number;
  connectorId: string;
  ui: Pick<
    SchemaPropertyUI,
    | "help_text"
    | "remote_select_options_path"
    | "remote_select_id_key"
    | "remote_select_label_key"
    | "remote_select_fallback_placeholder"
  >;
}

const DEFAULT_NO_CREDENTIAL =
  "Connect a credential to select a calendar.";

function optionKeys(ui: RemoteSelectWidgetProps["ui"]): {
  idKey: string;
  labelKey: string;
} {
  return {
    idKey: ui.remote_select_id_key ?? "id",
    labelKey: ui.remote_select_label_key ?? "summary",
  };
}

function readLabel(row: Record<string, unknown>, labelKey: string): string {
  const direct = row[labelKey];
  if (typeof direct === "string" && direct.length > 0) return direct;
  const summary = row.summary;
  if (typeof summary === "string" && summary.length > 0) return summary;
  const name = row.name;
  if (typeof name === "string" && name.length > 0) return name;
  const id = row.id;
  return typeof id === "string" ? id : "";
}

function formatOptionLabel(
  baseLabel: string,
  row: Record<string, unknown>,
): string {
  if (row.primary === true) {
    return baseLabel.includes("(primary)") ? baseLabel : `${baseLabel} (primary)`;
  }
  if (row.isDefaultCalendar === true) {
    return baseLabel.includes("(default)")
      ? baseLabel
      : `${baseLabel} (default)`;
  }
  return baseLabel;
}

/**
 * Select populated from a session-authenticated API path, with manual entry fallback.
 */
export function RemoteSelectWidget({
  inputId,
  value,
  onChange,
  disabled,
  placeholder,
  className,
  agentId,
  connectorId,
  ui,
}: RemoteSelectWidgetProps) {
  const [manual, setManual] = useState(false);
  const {
    calendars,
    isLoading,
    isFetching,
    error,
    hasCredential,
    isCredentialBindingPending,
    refetch,
  } = useAgentConnectorCalendars(agentId, connectorId);

  const { idKey, labelKey } = optionKeys(ui);

  const options = useMemo(() => {
    const rows = calendars.map((row) => {
      const idRaw = row[idKey];
      const id = typeof idRaw === "string" ? idRaw : String(idRaw ?? "");
      const raw = readLabel(row, labelKey);
      const base = raw || id;
      return { id, label: formatOptionLabel(base, row) };
    });
    if (value && !rows.some((r) => r.id === value)) {
      return [{ id: value, label: value }, ...rows];
    }
    return rows;
  }, [calendars, idKey, labelKey, value]);

  const noCredentialText = ui.help_text?.trim() || DEFAULT_NO_CREDENTIAL;
  const selectDisabled = disabled || isLoading || isFetching;
  const calendarsReady =
    hasCredential && !isLoading && !isFetching && !error;

  if (manual) {
    return (
      <div className="space-y-2">
        <Input
          id={inputId}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          placeholder={
            ui.remote_select_fallback_placeholder ??
            placeholder ??
            "Enter value"
          }
          className={className}
          data-testid={`remote-select-manual-${inputId}`}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-auto px-0 text-xs"
          disabled={disabled}
          onClick={() => setManual(false)}
        >
          Use list
        </Button>
      </div>
    );
  }

  if (isCredentialBindingPending) {
    return (
      <div className="space-y-2">
        <select
          id={inputId}
          value=""
          disabled
          aria-busy="true"
          className={`border-input bg-muted ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none disabled:opacity-50${className ? ` ${className}` : ""}`}
          data-testid={`remote-select-binding-pending-${inputId}`}
        >
          <option value="">Checking credentials…</option>
        </select>
      </div>
    );
  }

  if (!hasCredential) {
    return (
      <div className="space-y-2">
        <select
          id={inputId}
          value=""
          disabled
          className={`border-input bg-muted ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none disabled:opacity-50${className ? ` ${className}` : ""}`}
          data-testid={`remote-select-disabled-${inputId}`}
        >
          <option value="">{noCredentialText}</option>
        </select>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-auto px-0 text-xs"
          disabled={disabled}
          onClick={() => setManual(true)}
        >
          Enter calendar ID manually
        </Button>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-2">
        <p className="text-muted-foreground text-xs">
          Could not load calendars.{" "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => void refetch()}
          >
            Retry
          </button>
          {" · "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => setManual(true)}
          >
            Enter manually
          </button>
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <select
        id={inputId}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={selectDisabled || disabled}
        aria-busy={isLoading || isFetching ? "true" : undefined}
        className={`border-input bg-background ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50${className ? ` ${className}` : ""}`}
        data-testid={`remote-select-${inputId}`}
      >
        <option value="">
          {isLoading || isFetching ? "Loading…" : placeholder ?? "Select…"}
        </option>
        {options.map((opt) => (
          <option key={opt.id} value={opt.id}>
            {opt.label}
          </option>
        ))}
      </select>

      {calendarsReady && options.length === 0 && (
        <p className="text-muted-foreground text-xs">
          No calendars returned.{" "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => setManual(true)}
          >
            Enter a calendar ID manually
          </button>
        </p>
      )}

      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-auto px-0 text-xs"
        disabled={disabled}
        onClick={() => setManual(true)}
      >
        Enter manually
      </Button>
    </div>
  );
}
