import { useState } from "react";
import { Bot } from "lucide-react";
import { usePendingAgents } from "@/hooks/usePendingAgents";
import { getAgentDisplayName } from "@/lib/agents";
import { CopyableCode } from "@/components/CopyableCode";
import { CountdownBadge } from "@/pages/dashboard/approval-components";
import { ReviewPendingAgentDialog } from "@/pages/dashboard/ReviewPendingAgentDialog";
import type { Agent } from "@/hooks/useAgents";

function PendingAgentBanner({ agent }: { agent: Agent }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const name = getAgentDisplayName(agent);

  return (
    <>
      <div
        role="button"
        tabIndex={0}
        onClick={() => setDialogOpen(true)}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") setDialogOpen(true); }}
        className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-left text-sm text-amber-900 shadow-sm transition-colors hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-950/50 dark:text-amber-200 dark:hover:bg-amber-950/70"
        aria-label={`Pending agent registration: ${name}`}
      >
        <Bot className="size-5 shrink-0" />
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
          <span className="font-medium">
            New agent registration: <span className="font-semibold">{name}</span>
          </span>
          {agent.confirmation_code && (
            <span className="inline-flex items-center gap-1.5 text-xs">
              Code: <CopyableCode
                code={agent.confirmation_code}
                className="rounded bg-white/50 px-1.5 py-0.5 font-bold tracking-wider dark:bg-black/20"
              />
            </span>
          )}
          {agent.expires_at && (
            <CountdownBadge expiresAt={agent.expires_at} />
          )}
        </div>
        <span className="shrink-0 text-xs font-medium underline underline-offset-2 opacity-75">
          Review
        </span>
      </div>

      <ReviewPendingAgentDialog
        agent={agent}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </>
  );
}

export function PendingAgentBanners() {
  const { pendingAgents, isLoading } = usePendingAgents();

  if (isLoading || pendingAgents.length === 0) {
    return null;
  }

  return (
    <div className="mb-4 space-y-2" role="status" aria-label="Pending agent registrations">
      {pendingAgents.map((agent) => (
        <PendingAgentBanner key={agent.agent_id} agent={agent} />
      ))}
    </div>
  );
}
