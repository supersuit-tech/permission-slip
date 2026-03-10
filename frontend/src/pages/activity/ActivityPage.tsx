import { useState, useCallback } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Activity, ArrowLeft, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
} from "@/components/ui/table";
import { formatAbsoluteTime, formatRelativeTime } from "@/lib/utils";
import {
  type AuditEvent,
  type OutcomeFilter,
  ALL_OUTCOME_FILTERS,
  ACTION_EVENT_TYPES,
  AuditEventRow,
} from "@/lib/auditEvents";
import { useInfiniteAuditEvents } from "@/hooks/useInfiniteAuditEvents";
import type { AuditEventFilters } from "@/hooks/useAuditEvents";
import { useAgents } from "@/hooks/useAgents";
import { ActivityFilters } from "./ActivityFilters";
import { RetentionBanner } from "./RetentionBanner";
import { ActivityDetailSheet } from "./ActivityDetailSheet";

/** Set of valid outcome values for URL param validation. */
const VALID_OUTCOMES = new Set(
  ALL_OUTCOME_FILTERS.map((f) => f.value),
);

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Activity className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No activity found
      </p>
      <p className="text-muted-foreground max-w-xs text-xs">
        Try adjusting your filters or check back later. Approval decisions,
        standing approval executions, and payment charges will appear here.
      </p>
    </div>
  );
}

export function ActivityPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);

  const rawOutcome = searchParams.get("outcome") ?? "all";
  const outcomeFilter: OutcomeFilter = VALID_OUTCOMES.has(rawOutcome as OutcomeFilter)
    ? (rawOutcome as OutcomeFilter)
    : "all";
  const eventTypeFilter = searchParams.get("event_type") ?? "";
  const rawAgentId = searchParams.get("agent_id");
  const parsedAgentId = rawAgentId ? Number(rawAgentId) : NaN;
  const agentFilter = Number.isFinite(parsedAgentId) ? parsedAgentId : undefined;

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (value) {
          next.set(key, value);
        } else {
          next.delete(key);
        }
        return next;
      });
    },
    [setSearchParams],
  );

  // Default to action event types when no explicit event_type filter is set
  const filters: AuditEventFilters = {
    ...(outcomeFilter !== "all" && { outcome: outcomeFilter }),
    event_type: eventTypeFilter || ACTION_EVENT_TYPES,
    ...(agentFilter != null && { agent_id: agentFilter }),
  };

  const {
    events,
    retention,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    isLoading,
    error,
    refetch,
  } = useInfiniteAuditEvents(filters);

  const { agents } = useAgents();

  const handleRowClick = useCallback((event: AuditEvent) => {
    setSelectedEvent(event);
  }, []);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          to="/"
          className="text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Back to Dashboard"
        >
          <ArrowLeft className="size-5" />
        </Link>
        <h1 className="text-2xl font-bold tracking-tight">Activity Log</h1>
      </div>

      {/* Retention info */}
      {retention && <RetentionBanner retention={retention} />}

      {/* Filters */}
      <ActivityFilters
        outcomeFilter={outcomeFilter}
        onOutcomeChange={(v) =>
          setFilter("outcome", v === "all" ? undefined : v)
        }
        eventTypeFilter={eventTypeFilter}
        onEventTypeChange={(v) => setFilter("event_type", v || undefined)}
        agentFilter={agentFilter}
        onAgentChange={(v) =>
          setFilter("agent_id", v != null ? String(v) : undefined)
        }
        agents={agents}
      />

      {/* Content */}
      {isLoading ? (
        <div
          className="flex items-center justify-center py-16"
          role="status"
          aria-label="Loading activity"
        >
          <Loader2 className="text-muted-foreground size-6 animate-spin" />
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-destructive mb-2 text-sm">{error}</p>
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      ) : events.length === 0 ? (
        <EmptyState />
      ) : (
        <>
          <div className="-mx-3 overflow-x-auto sm:mx-0">
            <div className="min-w-[480px] overflow-hidden rounded-lg sm:min-w-0">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Timestamp
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
                    formatTimestamp={formatAbsoluteTime}
                    timestampTitle={formatRelativeTime(event.timestamp)}
                    actionMaxWidth="max-w-[300px]"
                    onClick={handleRowClick}
                  />
                ))}
              </TableBody>
            </Table>
            </div>
          </div>

          {/* Load More */}
          {hasNextPage && (
            <div className="flex justify-center">
              <Button
                variant="outline"
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
              >
                {isFetchingNextPage ? (
                  <>
                    <Loader2 className="mr-2 size-4 animate-spin" />
                    Loading...
                  </>
                ) : (
                  "Load More"
                )}
              </Button>
            </div>
          )}
        </>
      )}

      <ActivityDetailSheet
        event={selectedEvent}
        open={selectedEvent !== null}
        onOpenChange={(open) => { if (!open) setSelectedEvent(null); }}
      />
    </div>
  );
}
