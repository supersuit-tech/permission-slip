import { Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TableRow, TableCell } from "@/components/ui/table";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import { ConstraintsSummary } from "@/pages/dashboard/ConstraintsSummary";
import { isBoilerplateStandingApprovalDescription } from "@/lib/standingApprovalConstraints";
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
  const constraints =
    rule.constraints && typeof rule.constraints === "object"
      ? (rule.constraints as Record<string, unknown>)
      : null;
  const isInactive = rule.status !== "active";
  const showDescription =
    rule.description &&
    !isBoilerplateStandingApprovalDescription(rule.description);

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
          {showDescription && (
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
        <ConstraintsSummary constraints={constraints} />
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
