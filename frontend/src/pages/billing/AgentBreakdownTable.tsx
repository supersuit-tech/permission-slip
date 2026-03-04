import { useMemo } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents } from "@/hooks/useAgents";
import { formatCents } from "./formatters";

interface AgentBreakdownTableProps {
  byAgent: Record<string, number>;
  totalRequests: number;
  showCost: boolean;
  costCents: number;
}

export function AgentBreakdownTable({
  byAgent,
  totalRequests,
  showCost,
  costCents,
}: AgentBreakdownTableProps) {
  const { agents } = useAgents();

  const agentNameMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const agent of agents) {
      const meta = agent.metadata as Record<string, unknown> | undefined;
      const name =
        typeof meta?.name === "string" ? meta.name : `Agent ${agent.agent_id}`;
      map.set(String(agent.agent_id), name);
    }
    return map;
  }, [agents]);

  const sorted = Object.entries(byAgent).sort(([, a], [, b]) => b - a);
  const maxCount = sorted.length > 0 ? (sorted[0]?.[1] ?? 0) : 0;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Agent</TableHead>
          <TableHead className="text-right">Requests</TableHead>
          <TableHead className="text-right">% of total</TableHead>
          {showCost && <TableHead className="text-right">Est. cost</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {sorted.map(([agentId, count]) => {
          const pct =
            totalRequests > 0
              ? ((count / totalRequests) * 100).toFixed(1)
              : "0";
          const barWidth =
            maxCount > 0 ? Math.max(2, (count / maxCount) * 100) : 0;
          const agentCost =
            totalRequests > 0
              ? Math.round((count / totalRequests) * costCents)
              : 0;

          return (
            <TableRow key={agentId}>
              <TableCell className="font-medium">
                <div className="space-y-1">
                  <Link
                    to={`/agents/${agentId}`}
                    className="text-primary hover:underline"
                  >
                    {agentNameMap.get(agentId) ?? `Agent ${agentId}`}
                  </Link>
                  <div className="h-1 w-full max-w-[120px] overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary/40"
                      style={{ width: `${barWidth}%` }}
                    />
                  </div>
                </div>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {count.toLocaleString()}
              </TableCell>
              <TableCell className="text-right tabular-nums text-muted-foreground">
                {pct}%
              </TableCell>
              {showCost && (
                <TableCell className="text-right tabular-nums text-muted-foreground">
                  {formatCents(agentCost)}
                </TableCell>
              )}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
