import { Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TableRow, TableCell } from "@/components/ui/table";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import { isPatternWrapper } from "@/lib/constraints";
import { StandingApprovalAccountRescopeCell } from "./StandingApprovalAccountRescopeCell";
import {
  connectorInstanceFromStandingApprovalId,
  resolveConnectorInstanceAccountLabel,
} from "./connectorInstanceAccount";

interface StandingApprovalRowProps {
  agentId: number;
  rule: StandingApproval;
  actions: ConnectorAction[];
  instances: AgentConnectorInstance[];
  showAccountColumn: boolean;
  onEdit: (rule: StandingApproval) => void;
  onRevoke: (rule: StandingApproval) => void;
  onChanged?: () => void;
}

function formatExpiresIn(expiresAt: string | null | undefined): string {
  if (expiresAt === null) return "Never";
  if (!expiresAt) return "\u2014";

  const exp = new Date(expiresAt);
  if (Number.isNaN(exp.getTime())) return "\u2014";

  const now = new Date();
  const diffMs = exp.getTime() - now.getTime();
  if (diffMs <= 0) return "Expired";

  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  if (diffHours < 24) return `${diffHours}h`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d`;
}

function statusBadgeVariant(
  status: StandingApproval["status"],
): "success-soft" | "secondary" | "outline" {
  if (status === "active") return "success-soft";
  if (status === "revoked") return "secondary";
  return "outline";
}

export function StandingApprovalRow({
  agentId,
  rule,
  actions,
  instances,
  showAccountColumn,
  onEdit,
  onRevoke,
  onChanged,
}: StandingApprovalRowProps) {
  const action = actions.find((a) => a.action_type === rule.action_type);
  const constraintEntries = Object.entries(
    (rule.constraints ?? {}) as Record<string, unknown>,
  ).filter(([key]) => key !== "$meta" && key !== "$data_window");
  const isInactive = rule.status !== "active";

  return (
    <TableRow
      className={
        isInactive
          ? "bg-muted/40 text-muted-foreground [&_td]:opacity-90"
          : undefined
      }
      data-status={rule.status}
    >
      <TableCell>
        <div>
          <p className="font-medium">{rule.name ?? rule.action_type}</p>
          {rule.description && (
            <p className="text-muted-foreground text-sm">{rule.description}</p>
          )}
        </div>
      </TableCell>
      <TableCell>
        <div>
          <p className="text-sm">{action?.name ?? rule.action_type}</p>
          <p className="text-muted-foreground font-mono text-xs">
            {rule.action_type}
          </p>
        </div>
      </TableCell>
      <TableCell>
        {constraintEntries.length > 0 ? (
          <div className="space-y-0.5">
            {constraintEntries.map(([key, value]) => (
              <ParameterPill key={key} name={key} value={value} />
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground text-xs">Match all</span>
        )}
      </TableCell>
      {showAccountColumn && (
        <TableCell>
          {rule.status === "active" ? (
            <StandingApprovalAccountRescopeCell
              agentId={agentId}
              rule={rule}
              instances={instances}
              onSuccess={onChanged}
            />
          ) : (
            <span className="text-sm">
              {resolveConnectorInstanceAccountLabel(
                connectorInstanceFromStandingApprovalId(
                  rule.connector_instance_id,
                ),
                instances,
              )}
            </span>
          )}
        </TableCell>
      )}
      <TableCell className="text-sm">{formatExpiresIn(rule.expires_at)}</TableCell>
      <TableCell>
        <Badge variant={statusBadgeVariant(rule.status)} className="font-normal">
          {rule.status}
        </Badge>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit(rule)}
            disabled={rule.status !== "active"}
            aria-label={`Edit ${rule.name ?? rule.action_type}`}
          >
            <Pencil className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => onRevoke(rule)}
            disabled={rule.status !== "active"}
            aria-label={`Revoke ${rule.name ?? rule.action_type}`}
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
