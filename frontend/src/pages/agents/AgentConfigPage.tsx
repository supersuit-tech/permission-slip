import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAgent } from "@/hooks/useAgent";
import { useAgentConnectors } from "@/hooks/useAgentConnectors";
import { AgentInfoSection } from "./AgentInfoSection";
import { AgentConnectorsSection } from "./AgentConnectorsSection";

import { AgentPaymentMethodSection } from "./AgentPaymentMethodSection";
import { DeactivateSection } from "./DeactivateSection";

export function AgentConfigPage() {
  const { agentId: rawId } = useParams<{ agentId: string }>();
  const agentId = Number(rawId);

  const { agent, isLoading, error, refetch } = useAgent(agentId);
  const {
    connectors,
    isLoading: connectorsLoading,
    error: connectorsError,
  } = useAgentConnectors(agentId);

  if (isNaN(agentId) || agentId <= 0) {
    return (
      <div className="space-y-4">
        <BackLink />
        <p className="text-destructive text-sm">Invalid agent ID.</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <BackLink />
        <div className="flex items-center justify-center py-12" role="status" aria-label="Loading agent">
          <Loader2 className="text-muted-foreground size-6 animate-spin" aria-hidden="true" />
        </div>
      </div>
    );
  }

  if (error || !agent) {
    return (
      <div className="space-y-4">
        <BackLink />
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <p className="text-destructive mb-2 text-sm">
            {error ?? "Agent not found."}
          </p>
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <BackLink />
      <AgentInfoSection agent={agent} />
      <AgentConnectorsSection
        agentId={agentId}
        connectors={connectors}
        isLoading={connectorsLoading}
        error={connectorsError}
      />
      <AgentPaymentMethodSection agentId={agentId} />
      {agent.status !== "deactivated" && (
        <DeactivateSection agentId={agent.agent_id} />
      )}
    </div>
  );
}

function BackLink() {
  return (
    <Link
      to="/"
      className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm transition-colors"
    >
      <ArrowLeft className="size-4" />
      Back to Dashboard
    </Link>
  );
}
