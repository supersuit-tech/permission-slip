import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useApprovalDetail } from "@/hooks/useApprovalDetail";
import { useAgents } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import { ReviewApprovalDialog } from "@/pages/dashboard/ReviewApprovalDialog";

/**
 * Handles `/approve/:approvalId` URLs from email/SMS/push notifications.
 * Fetches the approval, resolves the agent name, and opens the review dialog.
 * Redirects to the dashboard when the dialog is closed.
 */
export function ApproveRedirectPage() {
  const { approvalId } = useParams<{ approvalId: string }>();
  const navigate = useNavigate();
  const { approval, isLoading, error } = useApprovalDetail(approvalId ?? null);
  const { agents } = useAgents();
  const [dialogOpen, setDialogOpen] = useState(false);

  // Open the dialog once the approval loads
  useEffect(() => {
    if (approval && !dialogOpen) {
      setDialogOpen(true);
    }
  }, [approval, dialogOpen]);

  const agentDisplayName = (() => {
    if (!approval) return "";
    const agent = agents.find((a) => a.agent_id === approval.agent_id);
    if (agent) return getAgentDisplayName(agent);
    return `Agent ${approval.agent_id}`;
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
        <p className="text-lg font-semibold">Approval not found</p>
        <p className="text-muted-foreground text-sm">
          This approval may have expired or already been handled.
        </p>
        <button
          type="button"
          onClick={() => navigate("/", { replace: true })}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Go to Dashboard
        </button>
      </div>
    );
  }

  if (isLoading || !approval) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4">
        <Loader2 className="text-muted-foreground size-6 animate-spin" />
        <p className="text-muted-foreground text-sm">Loading approval…</p>
      </div>
    );
  }

  return (
    <ReviewApprovalDialog
      approval={approval}
      agentDisplayName={agentDisplayName}
      open={dialogOpen}
      onOpenChange={handleOpenChange}
    />
  );
}
