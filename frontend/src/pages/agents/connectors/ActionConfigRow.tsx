import { CheckCircle2, Circle, Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TableRow, TableCell } from "@/components/ui/table";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { isPatternWrapper, WILDCARD_ACTION_TYPE } from "./ActionConfigFormFields";

interface ActionConfigRowProps {
  config: ActionConfiguration;
  actions: ConnectorAction[];
  credentials: CredentialSummary[];
  onEdit: (config: ActionConfiguration) => void;
  onDelete: (config: ActionConfiguration) => void;
}

export function ActionConfigRow({
  config,
  actions,
  credentials,
  onEdit,
  onDelete,
}: ActionConfigRowProps) {
  const isWildcardConfig = config.action_type === WILDCARD_ACTION_TYPE;
  const action = isWildcardConfig
    ? null
    : actions.find((a) => a.action_type === config.action_type);
  const credential = config.credential_id
    ? credentials.find((c) => c.id === config.credential_id)
    : null;

  const paramEntries = Object.entries(config.parameters);

  return (
    <TableRow>
      <TableCell>
        <div>
          <p className="font-medium">{config.name}</p>
          {config.description && (
            <p className="text-muted-foreground text-xs">
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
        {credential ? (
          <div className="flex items-center gap-1.5">
            <CheckCircle2 className="size-3.5 text-green-600 dark:text-green-400" />
            <span className="text-sm">
              {credential.label ?? credential.service}
            </span>
          </div>
        ) : config.credential_id ? (
          <div className="flex items-center gap-1.5">
            <CheckCircle2 className="text-muted-foreground size-3.5" />
            <span className="text-muted-foreground text-sm">Assigned</span>
          </div>
        ) : (
          <div className="flex items-center gap-1.5">
            <Circle className="text-muted-foreground size-3.5" />
            <span className="text-muted-foreground text-sm">Not assigned</span>
          </div>
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
      <Badge
        variant="secondary"
        className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
      >
        Active
      </Badge>
    );
  }
  return (
    <Badge variant="secondary" className="text-muted-foreground">
      Disabled
    </Badge>
  );
}
