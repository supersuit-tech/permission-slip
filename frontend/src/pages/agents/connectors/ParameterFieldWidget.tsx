import { useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { ExternalLink, Plus, X } from "lucide-react";
import type { SchemaProperty, SchemaPropertyUI, WidgetType } from "@/lib/parameterSchema";
import { inferWidgetFromProperty } from "@/lib/parameterSchema";

export interface DynamicSelectOption {
  value: string;
  label: string;
}

export interface ParameterFieldWidgetProps {
  /** The parameter key (used for id, fallback label). */
  paramKey: string;
  /** The schema property definition. */
  property: SchemaProperty;
  /** Current string value of the field. */
  value: string;
  /** Callback when the value changes. */
  onChange: (value: string) => void;
  /** Whether the field is disabled. */
  disabled?: boolean;
  /** Extra className for the input element. */
  className?: string;
  /** Override the placeholder from x-ui (e.g. for constraint mode hints). */
  placeholder?: string;
  /** Runtime options for `x-ui.options_from` select widgets. */
  dynamicSelectOptions?: DynamicSelectOption[];
  /** True while dynamic options are loading. */
  dynamicSelectLoading?: boolean;
}

/**
 * Renders the correct input widget for a parameter based on its `x-ui.widget`
 * hint. Falls back to a plain text `<Input>` when no widget is specified.
 */
export function ParameterFieldWidget({
  paramKey,
  property,
  value,
  onChange,
  disabled,
  className,
  placeholder: placeholderOverride,
  dynamicSelectOptions,
  dynamicSelectLoading,
}: ParameterFieldWidgetProps) {
  const ui = property["x-ui"];
  const widget: WidgetType = ui?.widget ?? inferWidgetFromProperty(property);
  const placeholder = placeholderOverride ?? ui?.placeholder;
  const inputId = `param-${paramKey}`;
  const useDynamicCalendars =
    widget === "select" && ui?.options_from === "connector_calendars";

  return (
    <div className="space-y-1">
      {renderWidget(widget, {
        inputId,
        value,
        onChange,
        disabled,
        placeholder,
        className,
        enumValues: property.enum,
        dynamicOptions: useDynamicCalendars ? dynamicSelectOptions : undefined,
        dynamicLoading: useDynamicCalendars ? dynamicSelectLoading : false,
      })}
      <FieldHints ui={ui} />
    </div>
  );
}

interface WidgetRenderProps {
  inputId: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  enumValues?: string[];
  dynamicOptions?: DynamicSelectOption[];
  dynamicLoading?: boolean;
}

function renderWidget(widget: WidgetType, props: WidgetRenderProps) {
  switch (widget) {
    case "select":
      return <SelectWidget {...props} />;
    case "textarea":
      return <TextareaWidget {...props} />;
    case "toggle":
      return <ToggleWidget {...props} />;
    case "number":
      return <NumberWidget {...props} />;
    case "date":
      return <DateWidget {...props} />;
    case "datetime":
      return <DateTimeWidget {...props} />;
    case "list":
      return <ListWidget {...props} />;
    case "text":
    default:
      return <TextWidget {...props} />;
  }
}

function TextWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  return (
    <Input
      id={inputId}
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      className={className}
    />
  );
}

function SelectWidget({
  inputId,
  value,
  onChange,
  disabled,
  placeholder,
  enumValues,
  className,
  dynamicOptions,
  dynamicLoading,
}: WidgetRenderProps) {
  const usingDynamic = dynamicOptions !== undefined;
  const opts = usingDynamic
    ? dynamicOptions
    : (enumValues ?? []).filter((opt) => opt !== "").map((v) => ({ value: v, label: v }));

  const showCustom =
    usingDynamic &&
    value !== "" &&
    !opts.some((o) => o.value === value);

  return (
    <select
      id={inputId}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled || dynamicLoading}
      className={`border-input bg-background ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50${className ? ` ${className}` : ""}`}
      data-testid={`select-${inputId}`}
    >
      <option value="">
        {dynamicLoading ? "Loading calendars…" : (placeholder ?? "Select…")}
      </option>
      {opts.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
      {showCustom && (
        <option value={value}>
          {value} (custom)
        </option>
      )}
    </select>
  );
}

function TextareaWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  return (
    <textarea
      id={inputId}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      rows={3}
      className={`border-input bg-background ring-ring/50 w-full rounded-md border px-3 py-2 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50${className ? ` ${className}` : ""}`}
      data-testid={`textarea-${inputId}`}
    />
  );
}

function ToggleWidget({ inputId, value, onChange, disabled, className }: WidgetRenderProps) {
  const checked = value === "true";
  return (
    <div className="flex items-center gap-2 py-1">
      <Switch
        id={inputId}
        checked={checked}
        onCheckedChange={(next) => onChange(String(next))}
        disabled={disabled}
        className={className}
      />
      <span className="text-muted-foreground text-sm" aria-hidden="true">
        {checked ? "Enabled" : "Disabled"}
      </span>
    </div>
  );
}

function NumberWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  return (
    <Input
      id={inputId}
      type="number"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      className={className}
    />
  );
}

function DateWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  return (
    <Input
      id={inputId}
      type="date"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      className={className}
    />
  );
}

function DateTimeWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  const localValue = toDatetimeLocalValue(value);

  return (
    <Input
      id={inputId}
      type="datetime-local"
      value={localValue}
      onChange={(e) => {
        const dtLocal = e.target.value;
        if (!dtLocal) {
          onChange("");
          return;
        }
        onChange(toRfc3339(dtLocal));
      }}
      disabled={disabled}
      placeholder={placeholder}
      className={className}
    />
  );
}

/** Parse a JSON array string into a string array, with fallback for non-JSON values. */
function parseListValue(value: string): string[] {
  if (!value) return [];
  if (value.startsWith("[")) {
    try {
      const parsed: unknown = JSON.parse(value);
      if (Array.isArray(parsed)) return parsed.map(String);
    } catch {
      // fall through
    }
  }
  return [value];
}

/** Serialize a string array into a JSON array string for form state. */
function serializeListValue(items: string[]): string {
  if (items.length === 0) return "";
  return JSON.stringify(items);
}

function ListWidget({ inputId, value, onChange, disabled, placeholder, className }: WidgetRenderProps) {
  const items = parseListValue(value);
  const nextId = useRef(0);
  const rowIds = useRef<number[]>([]);

  // Keep rowIds in sync with items length, assigning stable IDs to new rows.
  while (rowIds.current.length < items.length) {
    rowIds.current.push(nextId.current++);
  }
  if (rowIds.current.length > items.length) {
    rowIds.current = rowIds.current.slice(0, items.length);
  }

  function updateItem(index: number, newValue: string) {
    const next = [...items];
    next[index] = newValue;
    onChange(serializeListValue(next));
  }

  function removeItem(index: number) {
    const next = items.filter((_, i) => i !== index);
    rowIds.current = rowIds.current.filter((_, i) => i !== index);
    onChange(serializeListValue(next));
  }

  function addItem() {
    rowIds.current.push(nextId.current++);
    onChange(serializeListValue([...items, ""]));
  }

  return (
    <div className="w-full space-y-2" data-testid={`list-${inputId}`}>
      {items.map((item, index) => (
        <div key={rowIds.current[index]} className="flex items-center gap-2">
          <Input
            id={index === 0 ? inputId : undefined}
            type="text"
            value={item}
            onChange={(e) => updateItem(index, e.target.value)}
            disabled={disabled}
            placeholder={placeholder ?? `Item ${index + 1}`}
            className={className}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8 shrink-0"
            onClick={() => removeItem(index)}
            disabled={disabled}
            aria-label={`Remove item ${index + 1}`}
          >
            <X className="size-4" />
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="gap-1"
        onClick={addItem}
        disabled={disabled}
      >
        <Plus className="size-4" />
        Add item
      </Button>
    </div>
  );
}

/** Convert a datetime-local value ("YYYY-MM-DDTHH:mm") to an RFC 3339 string with local timezone offset. */
function toRfc3339(dtLocal: string): string {
  const d = new Date(dtLocal);
  if (isNaN(d.getTime())) return dtLocal;
  const offset = d.getTimezoneOffset();
  const sign = offset <= 0 ? "+" : "-";
  const absOffset = Math.abs(offset);
  const hours = String(Math.floor(absOffset / 60)).padStart(2, "0");
  const minutes = String(absOffset % 60).padStart(2, "0");
  return `${dtLocal}:00${sign}${hours}:${minutes}`;
}

/** Convert an RFC 3339 string to a datetime-local value ("YYYY-MM-DDTHH:mm") for the input element. */
function toDatetimeLocalValue(value: string): string {
  if (!value) return "";
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) return value;
  const d = new Date(value);
  if (isNaN(d.getTime())) return value;
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  const hrs = String(d.getHours()).padStart(2, "0");
  const mins = String(d.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hrs}:${mins}`;
}

/** Renders help_text and help_url hints below the input. */
function FieldHints({ ui }: { ui?: SchemaPropertyUI }) {
  const validHelpUrl =
    ui?.help_url && /^https?:\/\//i.test(ui.help_url) ? ui.help_url : null;
  if (!ui?.help_text && !validHelpUrl) return null;

  return (
    <div className="flex flex-wrap items-center gap-x-2">
      {ui?.help_text && (
        <p className="text-muted-foreground text-sm">{ui.help_text}</p>
      )}
      {validHelpUrl && (
        <a
          href={validHelpUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs underline"
        >
          Docs
          <ExternalLink className="size-3" />
        </a>
      )}
    </div>
  );
}
