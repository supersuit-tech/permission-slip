import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { ExternalLink } from "lucide-react";
import type { SchemaProperty, SchemaPropertyUI, WidgetType } from "@/lib/parameterSchema";

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
}: ParameterFieldWidgetProps) {
  const ui = property["x-ui"];
  const widget: WidgetType = ui?.widget ?? "text";
  const placeholder = placeholderOverride ?? ui?.placeholder;
  const inputId = `param-${paramKey}`;

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

function SelectWidget({ inputId, value, onChange, disabled, placeholder, enumValues, className }: WidgetRenderProps) {
  return (
    <select
      id={inputId}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={`border-input bg-background ring-ring/50 flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50${className ? ` ${className}` : ""}`}
      data-testid={`select-${inputId}`}
    >
      <option value="">{placeholder ?? "Select…"}</option>
      {(enumValues ?? []).filter((opt) => opt !== "").map((opt) => (
        <option key={opt} value={opt}>
          {opt}
        </option>
      ))}
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

/** Renders help_text and help_url hints below the input. */
function FieldHints({ ui }: { ui?: SchemaPropertyUI }) {
  const validHelpUrl =
    ui?.help_url && /^https?:\/\//i.test(ui.help_url) ? ui.help_url : null;
  if (!ui?.help_text && !validHelpUrl) return null;

  return (
    <div className="flex flex-wrap items-center gap-x-2">
      {ui?.help_text && (
        <p className="text-muted-foreground text-xs">{ui.help_text}</p>
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
