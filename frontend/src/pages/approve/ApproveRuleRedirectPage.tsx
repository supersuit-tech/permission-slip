import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useStandingApprovalRequestDetail } from "@/hooks/useStandingApprovalRequestDetail";
import { useAgents } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import { ReviewStandingApprovalRequestDialog } from "@/pages/dashboard/ReviewStandingApprovalRequestDialog";

export function ApproveRuleRedirectPage() {
  const { requestId } = useParams<{ requestId: string }>();
  const navigate = useNavigate();
  const { request, isLoading, error } = useStandingApprovalRequestDetail(requestId ?? null);
  const { agents } = useAgents();
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    if (request && !dialogOpen) {
      setDialogOpen(true);
    }
  }, [request, dialogOpen]);

  const agentDisplayName = (() => {
    if (!request) return "";
    const agent = agents.find((a) => a.agent_id === request.agent_id);
    if (agent) return getAgentDisplayName(agent);
    return `Agent ${request.agent_id}`;
  })();

  function handleOpenChange(nextOpen: boolean) {
    setDialogOpen(nextOpen);
    if (!nextOpen) {
      navigate("/", { replace: true });
    }
  }

  if (error) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
        <p className="text-lg font-semibold">Rule proposal not found</p>
        <button
          type="button"
          onClick={() => navigate("/", { replace: true })}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground"
        >
          Go to Dashboard
        </button>
      </div>
    );
  }

  if (isLoading || !request) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4">
        <Loader2 className="text-muted-foreground size-6 animate-spin" />
        <p className="text-muted-foreground text-sm">Loading rule proposal…</p>
      </div>
    );
  }

  return (
    <ReviewStandingApprovalRequestDialog
      request={request}
      agentDisplayName={agentDisplayName}
      open={dialogOpen}
      onOpenChange={handleOpenChange}
    />
  );
}
