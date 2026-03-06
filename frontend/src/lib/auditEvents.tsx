import {
  CheckCircle2,
  XCircle,
  Zap,
  Bot,
  Power,
  Ban,
  Clock,
  TimerOff,
  CreditCard,
  HelpCircle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { getAgentDisplayName } from "@/lib/agents";
import type { components } from "@/api/schema";

export type AuditEvent = components["schemas"]["AuditEvent"];

export type OutcomeFilter = AuditEvent["outcome"] | "all";

/** Subset of outcome filters shown on the dashboard card. */
export const OUTCOME_FILTERS: { label: string; value: OutcomeFilter }[] = [
  { label: "All", value: "all" },
  { label: "Approved", value: "approved" },
  { label: "Denied", value: "denied" },
  { label: "Auto-executed", value: "auto_executed" },
];

interface OutcomeStyle {
  label: string;
  variant: "default" | "secondary" | "destructive" | "outline";
  Icon: typeof CheckCircle2;
}

export const OUTCOME_CONFIG: Record<string, OutcomeStyle> = {
  approved: { label: "Approved", variant: "default", Icon: CheckCircle2 },
  denied: { label: "Denied", variant: "destructive", Icon: XCircle },
  cancelled: { label: "Cancelled", variant: "outline", Icon: Ban },
  auto_executed: { label: "Auto-executed", variant: "secondary", Icon: Zap },
  registered: { label: "Registered", variant: "default", Icon: Bot },
  deactivated: { label: "Deactivated", variant: "destructive", Icon: Power },
  pending: { label: "Pending", variant: "outline", Icon: Clock },
  expired: { label: "Expired", variant: "secondary", Icon: TimerOff },
  charged: { label: "Charged", variant: "default", Icon: CreditCard },
};

/** All outcome values (derived from OUTCOME_CONFIG) plus "all", for the full activity page. */
export const ALL_OUTCOME_FILTERS: { label: string; value: OutcomeFilter }[] = [
  { label: "All", value: "all" },
  ...Object.entries(OUTCOME_CONFIG).map(([value, { label }]) => ({
    label,
    value: value as OutcomeFilter,
  })),
];

const FALLBACK_OUTCOME: OutcomeStyle = {
  label: "Unknown",
  variant: "secondary",
  Icon: HelpCircle,
};

export function getAuditAgentName(event: AuditEvent): string {
  return getAgentDisplayName({
    agent_id: event.agent_id,
    metadata: event.agent_metadata,
  });
}

const MAX_PARAM_LENGTH = 100;

function truncate(value: string, max: number): string {
  return value.length > max ? `${value.slice(0, max)}…` : value;
}

export function getActionSummary(event: AuditEvent): string {
  // action is typed as `unknown` in the generated schema but the server
  // always stores it as a JSON object (or null for lifecycle events)
  const action = event.action as Record<string, unknown> | null | undefined;
  if (!action) {
    if (event.event_type === "agent.registered") {
      if (event.outcome === "pending") return "Registration pending";
      if (event.outcome === "expired") return "Registration expired";
      return "Agent registered";
    }
    if (event.event_type === "agent.deactivated") return "Agent deactivated";
    return event.event_type;
  }

  const actionType = typeof action.type === "string" ? action.type : "";

  // Payment method charged events include amount and card metadata.
  if (event.event_type === "payment_method.charged") {
    const cents = typeof action.amount_cents === "number" ? action.amount_cents : 0;
    const currency = typeof action.currency === "string" ? action.currency.toUpperCase() : "USD";
    const amount = (cents / 100).toFixed(2);
    const brand = typeof action.brand === "string" ? action.brand : "";
    const last4 = typeof action.last4 === "string" ? action.last4 : "";
    const cardLabel = brand && last4 ? ` (${brand} ••${last4})` : "";
    return `${actionType} — ${currency} $${amount}${cardLabel}`;
  }

  const params = action.parameters as Record<string, unknown> | undefined;

  if (params) {
    const summary = Object.entries(params)
      .slice(0, 2)
      .map(([key, val]) => {
        const v = typeof val === "string" ? val : JSON.stringify(val);
        return `${key}: ${truncate(v, MAX_PARAM_LENGTH)}`;
      })
      .join(", ");
    if (summary) return `${actionType} (${summary})`;
  }

  return actionType || event.event_type;
}

export function OutcomeBadge({ outcome }: { outcome: string }) {
  const config = OUTCOME_CONFIG[outcome] ?? FALLBACK_OUTCOME;
  const { Icon } = config;
  return (
    <Badge variant={config.variant} className="gap-1 rounded-full">
      <Icon className="size-3" />
      {config.label}
    </Badge>
  );
}

/**
 * Shared table row for rendering a single audit event.
 * Accepts a `formatTimestamp` prop so callers can choose relative vs absolute display.
 */
export function AuditEventRow({
  event,
  formatTimestamp,
  timestampTitle,
  actionMaxWidth = "max-w-[200px]",
}: {
  event: AuditEvent;
  formatTimestamp: (ts: string) => string;
  timestampTitle?: string;
  actionMaxWidth?: string;
}) {
  return (
    <TableRow>
      <TableCell className="whitespace-nowrap text-xs">
        {timestampTitle ? (
          <span title={timestampTitle}>
            {formatTimestamp(event.timestamp)}
          </span>
        ) : (
          formatTimestamp(event.timestamp)
        )}
      </TableCell>
      <TableCell className="font-medium">
        {getAuditAgentName(event)}
      </TableCell>
      <TableCell className={`${actionMaxWidth} truncate text-sm`}>
        {getActionSummary(event)}
      </TableCell>
      <TableCell>
        <OutcomeBadge outcome={event.outcome} />
      </TableCell>
    </TableRow>
  );
}
