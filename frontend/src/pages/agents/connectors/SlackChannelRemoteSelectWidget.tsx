import { useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { SchemaPropertyUI } from "@/lib/parameterSchema";
import { useAgentConnectorChannels } from "@/hooks/useAgentConnectorChannels";

export interface SlackChannelRemoteSelectWidgetProps {
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
  "Connect a Slack credential to select a channel.";

function optionKeys(ui: SlackChannelRemoteSelectWidgetProps["ui"]): {
  idKey: string;
  labelKey: string;
} {
  return {
    idKey: ui.remote_select_id_key ?? "id",
    labelKey: ui.remote_select_label_key ?? "display_label",
  };
}

function readLabel(row: Record<string, unknown>, labelKey: string): string {
  const direct = row[labelKey];
  if (typeof direct === "string" && direct.length > 0) return direct;
  const displayLabel = row.display_label;
  if (typeof displayLabel === "string" && displayLabel.length > 0)
    return displayLabel;
  const id = row.id;
  return typeof id === "string" ? id : "";
}

function slackChannelTypeSuffix(row: Record<string, unknown>): string {
  if (row.is_im === true) return "DM";
  if (row.is_mpim === true) return "group DM";
  if (row.is_private === true) return "private";
  return "public";
}

function formatSlackOptionLabel(
  baseLabel: string,
  row: Record<string, unknown>,
): string {
  const parts: string[] = [baseLabel];
  const kind = slackChannelTypeSuffix(row);
  parts.push(`(${kind})`);
  const n = row.num_members;
  if (typeof n === "number" && Number.isFinite(n) && n > 0) {
    parts.push(`· ${n} members`);
  }
  return parts.join(" ");
}

export function SlackChannelRemoteSelectWidget({
  inputId,
  value,
  onChange,
  disabled,
  placeholder,
  className,
  agentId,
  connectorId,
  ui,
}: SlackChannelRemoteSelectWidgetProps) {
  const [manual, setManual] = useState(false);
  const {
    channels,
    isLoading,
    isFetching,
    error,
    hasCredential,
    isCredentialBindingPending,
    refetch,
  } = useAgentConnectorChannels(agentId, connectorId);

  const { idKey, labelKey } = optionKeys(ui);

  const options = useMemo(() => {
    const rows = channels.map((row) => {
      const idRaw = row[idKey];
      const id = typeof idRaw === "string" ? idRaw : String(idRaw ?? "");
      const raw = readLabel(row, labelKey);
      const base = raw || id;
      return { id, label: formatSlackOptionLabel(base, row) };
    });
    if (value && !rows.some((r) => r.id === value)) {
      return [{ id: value, label: value }, ...rows];
    }
    return rows;
  }, [channels, idKey, labelKey, value]);

  const noCredentialText = ui.help_text?.trim() || DEFAULT_NO_CREDENTIAL;
  const selectDisabled = disabled || isLoading || isFetching;
  const channelsReady =
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
            "Channel ID (e.g. C01234567)"
          }
          className={className}
          data-testid={`slack-remote-select-manual-${inputId}`}
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
          data-testid={`slack-remote-select-binding-pending-${inputId}`}
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
          data-testid={`slack-remote-select-disabled-${inputId}`}
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
          Enter channel ID manually
        </Button>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-2">
        <p className="text-muted-foreground text-xs">
          Could not load channels.{" "}
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
        data-testid={`slack-remote-select-${inputId}`}
      >
        <option value="">
          {isLoading || isFetching ? "Loading…" : placeholder ?? "Select channel…"}
        </option>
        {options.map((opt) => (
          <option key={opt.id} value={opt.id}>
            {opt.label}
          </option>
        ))}
      </select>

      {channelsReady && options.length === 0 && (
        <p className="text-muted-foreground text-xs">
          No channels returned.{" "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => setManual(true)}
          >
            Enter a channel ID manually
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
