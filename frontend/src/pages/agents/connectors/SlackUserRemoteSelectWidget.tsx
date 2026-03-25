import { useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { SchemaPropertyUI } from "@/lib/parameterSchema";
import { useAgentConnectorUsers } from "@/hooks/useAgentConnectorUsers";
import { readLabel } from "./remoteSelectUtils";

export interface SlackUserRemoteSelectWidgetProps {
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
  "Connect a Slack credential to select a user.";

export function SlackUserRemoteSelectWidget({
  inputId,
  value,
  onChange,
  disabled,
  placeholder,
  className,
  agentId,
  connectorId,
  ui,
}: SlackUserRemoteSelectWidgetProps) {
  const [manual, setManual] = useState(false);
  const {
    users,
    isLoading,
    isFetching,
    error,
    hasCredential,
    isCredentialBindingPending,
    refetch,
  } = useAgentConnectorUsers(agentId, connectorId);

  const labelKey = ui.remote_select_label_key ?? "display_label";

  const options = useMemo(() => {
    const rows = users.map((row) => {
      const idRaw = row[ui.remote_select_id_key ?? "id"];
      const id = typeof idRaw === "string" ? idRaw : String(idRaw ?? "");
      const label = readLabel(row, labelKey);
      return { id, label: label || id };
    });
    if (value && !rows.some((r) => r.id === value)) {
      return [{ id: value, label: value }, ...rows];
    }
    return rows;
  }, [users, ui.remote_select_id_key, labelKey, value]);

  const noCredentialText = ui.help_text?.trim() || DEFAULT_NO_CREDENTIAL;
  const selectDisabled = disabled || isLoading || isFetching;
  const usersReady = hasCredential && !isLoading && !isFetching && !error;

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
            "User ID (e.g. U01234567)"
          }
          className={className}
          data-testid={`slack-user-remote-select-manual-${inputId}`}
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
          data-testid={`slack-user-remote-select-binding-pending-${inputId}`}
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
          data-testid={`slack-user-remote-select-disabled-${inputId}`}
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
          Enter user ID manually
        </Button>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-2">
        <p className="text-muted-foreground text-xs">
          Could not load users.{" "}
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
        disabled={selectDisabled}
        aria-busy={isLoading || isFetching ? "true" : undefined}
        className={`border-input bg-background ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50${className ? ` ${className}` : ""}`}
        data-testid={`slack-user-remote-select-${inputId}`}
      >
        <option value="">
          {isLoading || isFetching ? "Loading…" : placeholder ?? "Select user…"}
        </option>
        {options.map((opt) => (
          <option key={opt.id} value={opt.id}>
            {opt.label}
          </option>
        ))}
      </select>

      {usersReady && options.length === 0 && (
        <p className="text-muted-foreground text-xs">
          No users returned.{" "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => setManual(true)}
          >
            Enter a user ID manually
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
