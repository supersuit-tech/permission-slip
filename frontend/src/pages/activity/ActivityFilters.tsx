import { Button } from "@/components/ui/button";
import { type OutcomeFilter, ALL_OUTCOME_FILTERS } from "@/lib/auditEvents";
import { getAgentDisplayName } from "@/lib/agents";
import type { Agent } from "@/hooks/useAgents";

/** All event types the API supports. */
export const EVENT_TYPES = [
  { label: "All Types", value: "" },
  { label: "Approved", value: "approval.approved" },
  { label: "Denied", value: "approval.denied" },
  { label: "Cancelled", value: "approval.cancelled" },
  { label: "Standing Executed", value: "standing_approval.executed" },
  { label: "Agent Registered", value: "agent.registered" },
  { label: "Agent Deactivated", value: "agent.deactivated" },
] as const;

interface ActivityFiltersProps {
  outcomeFilter: OutcomeFilter;
  onOutcomeChange: (v: OutcomeFilter) => void;
  eventTypeFilter: string;
  onEventTypeChange: (v: string) => void;
  agentFilter: number | undefined;
  onAgentChange: (v: number | undefined) => void;
  agents: Agent[];
}

export function ActivityFilters({
  outcomeFilter,
  onOutcomeChange,
  eventTypeFilter,
  onEventTypeChange,
  agentFilter,
  onAgentChange,
  agents,
}: ActivityFiltersProps) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:gap-6">
      {/* Agent filter */}
      <div className="flex items-center gap-2">
        <label
          htmlFor="agent-filter"
          className="text-sm font-medium whitespace-nowrap"
        >
          Agent
        </label>
        <select
          id="agent-filter"
          value={agentFilter ?? ""}
          onChange={(e) => {
            const val = e.target.value;
            onAgentChange(val ? Number(val) : undefined);
          }}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
        >
          <option value="">All Agents</option>
          {agents.map((a) => (
            <option key={a.agent_id} value={a.agent_id}>
              {getAgentDisplayName(a)}
            </option>
          ))}
        </select>
      </div>

      {/* Event type filter */}
      <div className="flex items-center gap-2">
        <label
          htmlFor="event-type-filter"
          className="text-sm font-medium whitespace-nowrap"
        >
          Event Type
        </label>
        <select
          id="event-type-filter"
          value={eventTypeFilter}
          onChange={(e) => onEventTypeChange(e.target.value)}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm"
        >
          {EVENT_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      {/* Outcome filter tabs */}
      <div
        className="flex flex-wrap gap-1"
        role="tablist"
        aria-label="Filter activity by outcome"
      >
        {ALL_OUTCOME_FILTERS.map((f) => (
          <Button
            key={f.value}
            variant={outcomeFilter === f.value ? "default" : "ghost"}
            size="sm"
            role="tab"
            aria-selected={outcomeFilter === f.value}
            onClick={() => onOutcomeChange(f.value)}
          >
            {f.label}
          </Button>
        ))}
      </div>
    </div>
  );
}
