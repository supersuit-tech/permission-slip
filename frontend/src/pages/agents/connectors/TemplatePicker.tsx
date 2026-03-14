import { Check, FileText, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import { isPatternWrapper } from "@/lib/constraints";

interface TemplatePickerProps {
  templates: ActionConfigTemplate[];
  isLoading: boolean;
  actionType: string;
  onSelect: (template: ActionConfigTemplate) => void;
  disabled?: boolean;
  selectedTemplateId?: string | null;
}

export function TemplatePicker({
  templates,
  isLoading,
  actionType,
  onSelect,
  disabled,
  selectedTemplateId,
}: TemplatePickerProps) {
  const filtered = templates.filter((t) => t.action_type === actionType);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-2">
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
        <span className="text-muted-foreground text-xs">
          Loading templates...
        </span>
      </div>
    );
  }

  if (filtered.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="text-muted-foreground text-xs">
        Start from a template to pre-fill the form, or configure from scratch
        below.
      </p>
      <div className="grid gap-2">
        {filtered.map((template) => (
          <TemplateCard
            key={template.id}
            template={template}
            onSelect={onSelect}
            disabled={disabled}
            isApplied={selectedTemplateId === template.id}
          />
        ))}
      </div>
    </div>
  );
}

function TemplateCard({
  template,
  onSelect,
  disabled,
  isApplied,
}: {
  template: ActionConfigTemplate;
  onSelect: (template: ActionConfigTemplate) => void;
  disabled?: boolean;
  isApplied?: boolean;
}) {
  const paramEntries = Object.entries(template.parameters);

  return (
    <button
      type="button"
      onClick={() => onSelect(template)}
      disabled={disabled}
      className={`flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 ${
        isApplied
          ? "border-primary bg-primary/5 ring-1 ring-primary/20"
          : "border-input hover:bg-accent"
      }`}
    >
      {isApplied ? (
        <Check className="text-primary mt-0.5 size-4 shrink-0" />
      ) : (
        <FileText className="text-muted-foreground mt-0.5 size-4 shrink-0" />
      )}
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium">{template.name}</p>
          {isApplied && (
            <Badge variant="secondary" className="text-xs">
              Applied
            </Badge>
          )}
        </div>
        {template.description && (
          <p className="text-muted-foreground text-xs">
            {template.description}
          </p>
        )}
        {paramEntries.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {paramEntries.map(([key, value]) => (
              <TemplateParamBadge key={key} name={key} value={value} />
            ))}
          </div>
        )}
      </div>
    </button>
  );
}

function TemplateParamBadge({
  name,
  value,
}: {
  name: string;
  value: unknown;
}) {
  const isWildcard = value === "*";
  const isPattern = isPatternWrapper(value);
  const displayValue = isPattern ? value.$pattern : String(value);

  return (
    <span className="inline-flex items-center gap-1 text-xs">
      <span className="text-muted-foreground font-mono">{name}:</span>
      <Badge
        variant={isWildcard || isPattern ? "outline" : "secondary"}
        className={`font-mono text-xs ${isWildcard ? "border-dashed" : ""} ${isPattern ? "border-dashed italic" : ""}`}
      >
        {displayValue}
      </Badge>
    </span>
  );
}
