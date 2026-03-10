import { useState } from "react";
import { Link } from "react-router-dom";
import { Loader2, Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
} from "@/components/ui/table";
import { formatRelativeTime } from "@/lib/utils";
import {
  type OutcomeFilter,
  OUTCOME_FILTERS,
  AuditEventRow,
} from "@/lib/auditEvents";
import {
  useAuditEvents,
  type AuditEventFilters,
} from "@/hooks/useAuditEvents";

function OutcomeFilterTabs({
  value,
  onChange,
}: {
  value: OutcomeFilter;
  onChange: (v: OutcomeFilter) => void;
}) {
  return (
    <div
      className="flex flex-wrap gap-1"
      role="tablist"
      aria-label="Filter activity by outcome"
    >
      {OUTCOME_FILTERS.map((f) => (
        <Button
          key={f.value}
          variant={value === f.value ? "default" : "ghost"}
          size="sm"
          role="tab"
          aria-selected={value === f.value}
          onClick={() => onChange(f.value)}
        >
          {f.label}
        </Button>
      ))}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <Activity className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No activity yet
      </p>
      <p className="text-muted-foreground max-w-xs text-xs">
        Every approval, denial, and agent action is logged here so you always
        have a complete audit trail.
      </p>
    </div>
  );
}

export function RecentActivityCard() {
  const [outcomeFilter, setOutcomeFilter] = useState<OutcomeFilter>("all");

  const filters: AuditEventFilters =
    outcomeFilter !== "all" ? { outcome: outcomeFilter } : {};

  const { events, hasMore, isLoading, error, refetch } = useAuditEvents(filters);

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>Recent Activity</CardTitle>
          <OutcomeFilterTabs
            value={outcomeFilter}
            onChange={setOutcomeFilter}
          />
        </div>
      </CardHeader>
      <CardContent className="px-3 md:px-6">
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading activity"
          >
            <Loader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <p className="text-destructive mb-2 text-sm">{error}</p>
            <Button variant="ghost" size="sm" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        ) : events.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="overflow-hidden rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    When
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Agent
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Action
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Outcome
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {events.map((event) => (
                  <AuditEventRow
                    key={`${event.timestamp}-${event.agent_id}-${event.event_type}`}
                    event={event}
                    formatTimestamp={formatRelativeTime}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
      {hasMore && (
        <div className="px-6 pb-4 text-center">
          <Link
            to="/activity"
            className="text-sm font-medium text-primary hover:underline"
          >
            View All Activity
          </Link>
        </div>
      )}
    </Card>
  );
}
