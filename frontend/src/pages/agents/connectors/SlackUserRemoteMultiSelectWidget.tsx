import { useCallback, useMemo, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import type { SchemaPropertyUI } from "@/lib/parameterSchema";
import { useAgentConnectorUsers } from "@/hooks/useAgentConnectorUsers";
import { readLabel } from "./remoteSelectUtils";

export interface SlackUserRemoteMultiSelectWidgetProps {
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
  "Connect a Slack credential to select users.";

/** Parse comma-separated IDs into a deduplicated array. */
function parseSelected(value: string): string[] {
  if (!value) return [];
  const seen = new Set<string>();
  return value
    .split(",")
    .map((s) => s.trim())
    .filter((s) => {
      if (!s || seen.has(s)) return false;
      seen.add(s);
      return true;
    });
}

export function SlackUserRemoteMultiSelectWidget({
  inputId,
  value,
  onChange,
  disabled,
  placeholder,
  className,
  agentId,
  connectorId,
  ui,
}: SlackUserRemoteMultiSelectWidgetProps) {
  const [manual, setManual] = useState(false);
  const [search, setSearch] = useState("");
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);

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
  const idKey = ui.remote_select_id_key ?? "id";

  const selectedIds = useMemo(() => parseSelected(value), [value]);
  const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);

  const allOptions = useMemo(() => {
    return users.map((row) => {
      const idRaw = row[idKey];
      const id = typeof idRaw === "string" ? idRaw : String(idRaw ?? "");
      const label = readLabel(row, labelKey);
      return { id, label: label || id };
    });
  }, [users, idKey, labelKey]);

  /** Map from ID to label for chip display. */
  const labelMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const opt of allOptions) {
      m.set(opt.id, opt.label);
    }
    return m;
  }, [allOptions]);

  /** Options filtered by search, excluding already-selected. */
  const filteredOptions = useMemo(() => {
    const available = allOptions.filter((opt) => !selectedSet.has(opt.id));
    if (!search) return available;
    const q = search.toLowerCase();
    return available.filter(
      (opt) =>
        opt.label.toLowerCase().includes(q) ||
        opt.id.toLowerCase().includes(q),
    );
  }, [allOptions, selectedSet, search]);

  const serialize = useCallback(
    (ids: string[]) => onChange(ids.join(",")),
    [onChange],
  );

  const addUser = useCallback(
    (id: string) => {
      if (selectedSet.has(id)) return;
      serialize([...selectedIds, id]);
      setSearch("");
      setActiveIndex(-1);
    },
    [selectedIds, selectedSet, serialize],
  );

  const removeUser = useCallback(
    (id: string) => {
      serialize(selectedIds.filter((sid) => sid !== id));
    },
    [selectedIds, serialize],
  );

  const handleBlur = useCallback((e: React.FocusEvent) => {
    if (
      containerRef.current &&
      !containerRef.current.contains(e.relatedTarget as Node)
    ) {
      setDropdownOpen(false);
      setActiveIndex(-1);
    }
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!dropdownOpen || filteredOptions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setActiveIndex((prev) =>
            prev < filteredOptions.length - 1 ? prev + 1 : 0,
          );
          break;
        case "ArrowUp":
          e.preventDefault();
          setActiveIndex((prev) =>
            prev > 0 ? prev - 1 : filteredOptions.length - 1,
          );
          break;
        case "Enter": {
          e.preventDefault();
          const opt = filteredOptions[activeIndex];
          if (opt) addUser(opt.id);
          break;
        }
        case "Escape":
          e.preventDefault();
          setDropdownOpen(false);
          setActiveIndex(-1);
          break;
      }
    },
    [dropdownOpen, filteredOptions, activeIndex, addUser],
  );

  const noCredentialText = ui.help_text?.trim() || DEFAULT_NO_CREDENTIAL;
  const selectDisabled = disabled || isLoading || isFetching;
  const usersReady = hasCredential && !isLoading && !isFetching && !error;

  const listboxId = `${inputId}-listbox`;

  const selectClassName = `border-input bg-muted ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none disabled:opacity-50${className ? ` ${className}` : ""}`;

  // Manual entry mode
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
            "Comma-separated user IDs (e.g. U01234567,U09876543)"
          }
          className={className}
          data-testid={`slack-user-remote-multi-select-manual-${inputId}`}
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

  // Credential binding pending
  if (isCredentialBindingPending) {
    return (
      <div className="space-y-2">
        <select
          id={inputId}
          value=""
          disabled
          aria-busy="true"
          className={selectClassName}
          data-testid={`slack-user-remote-multi-select-binding-pending-${inputId}`}
        >
          <option value="">Checking credentials\u2026</option>
        </select>
      </div>
    );
  }

  // No credential
  if (!hasCredential) {
    return (
      <div className="space-y-2">
        <select
          id={inputId}
          value=""
          disabled
          className={selectClassName}
          data-testid={`slack-user-remote-multi-select-disabled-${inputId}`}
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
          Enter user IDs manually
        </Button>
      </div>
    );
  }

  // Error state
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
          {" \u00b7 "}
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

  // Main multi-select UI
  return (
    <div className="space-y-2" ref={containerRef} onBlur={handleBlur}>
      {/* Selected user chips */}
      {selectedIds.length > 0 && (
        <div
          className="flex flex-wrap gap-1.5"
          data-testid={`slack-user-remote-multi-select-chips-${inputId}`}
        >
          {selectedIds.map((id) => (
            <span
              key={id}
              className="bg-secondary text-secondary-foreground inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium"
            >
              {labelMap.get(id) ?? id}
              <button
                type="button"
                className="hover:text-destructive ml-0.5 rounded-sm outline-none focus-visible:ring-1"
                onClick={() => removeUser(id)}
                disabled={disabled}
                aria-label={`Remove ${labelMap.get(id) ?? id}`}
              >
                <X className="size-3" />
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Search input + dropdown */}
      <div className="relative">
        <Input
          id={inputId}
          type="text"
          role="combobox"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setDropdownOpen(true);
            setActiveIndex(-1);
          }}
          onFocus={() => setDropdownOpen(true)}
          onKeyDown={handleKeyDown}
          disabled={selectDisabled}
          placeholder={
            isLoading || isFetching
              ? "Loading\u2026"
              : placeholder ?? "Search users\u2026"
          }
          aria-busy={isLoading || isFetching ? "true" : undefined}
          aria-expanded={dropdownOpen}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            activeIndex >= 0
              ? `${listboxId}-option-${activeIndex}`
              : undefined
          }
          className={className}
          autoComplete="off"
          data-testid={`slack-user-remote-multi-select-search-${inputId}`}
        />
        {dropdownOpen && usersReady && (
          <ul
            id={listboxId}
            className="border-input bg-popover text-popover-foreground absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-md border shadow-md"
            role="listbox"
            aria-label="User suggestions"
            data-testid={`slack-user-remote-multi-select-dropdown-${inputId}`}
          >
            {filteredOptions.length === 0 ? (
              <li className="text-muted-foreground px-3 py-2 text-sm">
                {allOptions.length === 0
                  ? "No users available"
                  : "No matching users"}
              </li>
            ) : (
              filteredOptions.map((opt, idx) => (
                <li
                  key={opt.id}
                  id={`${listboxId}-option-${idx}`}
                  role="option"
                  aria-selected={idx === activeIndex}
                  className={`cursor-pointer px-3 py-1.5 text-sm${
                    idx === activeIndex
                      ? " bg-accent text-accent-foreground"
                      : " hover:bg-accent hover:text-accent-foreground"
                  }`}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    addUser(opt.id);
                  }}
                  onMouseEnter={() => setActiveIndex(idx)}
                >
                  {opt.label}
                </li>
              ))
            )}
          </ul>
        )}
      </div>

      {usersReady && allOptions.length === 0 && (
        <p className="text-muted-foreground text-xs">
          No users returned.{" "}
          <button
            type="button"
            className="text-foreground underline"
            disabled={disabled}
            onClick={() => setManual(true)}
          >
            Enter user IDs manually
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
