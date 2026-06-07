import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useAgents } from "@/hooks/useAgents";
import { useApprovalBulkGroup } from "@/hooks/useApprovalBulkGroup";
import { getAgentDisplayName } from "@/lib/agents";
import { ReviewBulkApprovalDialog } from "@/pages/dashboard/ReviewBulkApprovalDialog";

export function ApproveGroupRedirectPage() {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const { data: group, isLoading, error } = useApprovalBulkGroup(groupId ?? undefined);
  const { agents } = useAgents();
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    if (group && !dialogOpen) {
      setDialogOpen(true);
    }
  }, [group, dialogOpen]);

  const agentDisplayName = (() => {
    if (!group) return "";
    const agent = agents.find((a) => a.agent_id === group.agent_id);
    if (agent) return getAgentDisplayName(agent);
    return `Agent ${group.agent_id}`;
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
        <p className="text-lg font-semibold">Bulk approval not found</p>
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

  if (isLoading || !group || !groupId) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4">
        <Loader2 className="text-muted-foreground size-6 animate-spin" />
        <p className="text-muted-foreground text-sm">Loading bulk approval…</p>
      </div>
    );
  }

  return (
    <ReviewBulkApprovalDialog
      bulkGroupId={groupId}
      agentDisplayName={agentDisplayName}
      open={dialogOpen}
      onOpenChange={handleOpenChange}
    />
  );
}
