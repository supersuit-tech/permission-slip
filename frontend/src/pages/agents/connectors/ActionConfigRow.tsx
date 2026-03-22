import { Ban, Check, Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TableRow, TableCell } from "@/components/ui/table";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { isPatternWrapper } from "@/lib/constraints";
import { WILDCARD_ACTION_TYPE } from "./ActionConfigFormFields";

interface ActionConfigRowProps {
  config: ActionConfiguration;
  actions: ConnectorAction[];
  onEdit: (config: ActionConfiguration) => void;
  onDelete: (config: ActionConfiguration) => void;
}

export function ActionConfigRow({
  config,
  actions,
  onEdit,
  onDelete,
}: ActionConfigRowProps) {
  const isWildcardConfig = config.action_type === WILDCARD_ACTION_TYPE;
  const action = isWildcardConfig
    ? null
    : actions.find((a) => a.action_type === config.action_type);

  const paramEntries = Object.entries(config.parameters);
  const isDisabled = config.status === "disabled";

  return (
    <TableRow
      className={
        isDisabled
          ? "bg-muted/40 text-muted-foreground [&_td]:opacity-90"
          : undefined
      }
      data-status={config.status}
    >
      <TableCell>
        <div>
          <p className="font-medium">{config.name}</p>
          {config.description && (
            <p className="text-muted-foreground text-sm">
              {config.description}
            </p>
          )}
        </div>
      </TableCell>
      <TableCell>
        {isWildcardConfig ? (
          <div>
            <Badge className="bg-primary/10 text-primary border-primary/20 border">
              All Actions
            </Badge>
            <p className="text-muted-foreground mt-0.5 font-mono text-xs">*</p>
          </div>
        ) : (
          <div>
            <p className="text-sm">{action?.name ?? config.action_type}</p>
            <p className="text-muted-foreground font-mono text-xs">
              {config.action_type}
            </p>
          </div>
        )}
      </TableCell>
      <TableCell>
        {isWildcardConfig ? (
          <span className="text-muted-foreground text-xs italic">
            All parameters — agent chooses freely
          </span>
        ) : paramEntries.length > 0 ? (
          <div className="space-y-0.5">
            {paramEntries.map(([key, value]) => (
              <ParameterPill key={key} name={key} value={value} />
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground text-xs">No parameters</span>
        )}
      </TableCell>
      <TableCell>
        <StatusBadge status={config.status} />
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit(config)}
            aria-label={`Edit ${config.name}`}
          >
            <Pencil className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => onDelete(config)}
            aria-label={`Delete ${config.name}`}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

function ParameterPill({ name, value }: { name: string; value: unknown }) {
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

function StatusBadge({ status }: { status: "active" | "disabled" }) {
  if (status === "active") {
    return (
      <Badge variant="success-soft" className="gap-1 pr-2">
        <Check className="size-3 shrink-0" aria-hidden />
        Active
      </Badge>
    );
  }
  return (
    <Badge variant="destructive-soft" className="gap-1 pr-2">
      <Ban className="size-3 shrink-0" aria-hidden />
      Disabled
    </Badge>
  );
}
