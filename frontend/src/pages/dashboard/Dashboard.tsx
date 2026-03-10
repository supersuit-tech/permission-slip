import { useState } from "react";
import { useApprovalEvents } from "@/hooks/useApprovalEvents";
import { useAgents } from "@/hooks/useAgents";
import { useApprovals } from "@/hooks/useApprovals";
import { useStandingApprovals } from "@/hooks/useStandingApprovals";
import { useAuditEvents } from "@/hooks/useAuditEvents";
import { NotificationBanner } from "./NotificationBanner";
import { PendingApprovalsCard } from "./PendingApprovalsCard";
import { RecentActivityCard } from "./RecentActivityCard";
import { RegisteredAgentsCard } from "./RegisteredAgentsCard";
import { StandingApprovalsCard } from "./StandingApprovalsCard";
import { AgentOnboardingHero } from "./AgentOnboardingHero";
import { InviteCodeDialog } from "./InviteCodeDialog";

export function Dashboard() {
  useApprovalEvents();
  const { agents, isLoading, error } = useAgents();
  const { approvals } = useApprovals();
  const { standingApprovals } = useStandingApprovals();
  const { events } = useAuditEvents();
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);

  const showOnboarding = !isLoading && !error && agents.length === 0;
  const hasActiveAgents = agents.some((a) => a.status === "registered");
  const hasActivity = events.length > 0;
  const hasPendingApprovals = approvals.length > 0;

  // Progressive disclosure: show cards only when relevant to the user's journey.
  // Standing Approvals is a power-user feature — only show it once the user
  // has activity or has already created one.
  const showPendingApprovals = hasActiveAgents || hasPendingApprovals;
  const showActivity = hasActivity;
  const showStandingApprovals =
    standingApprovals.length > 0 || hasActivity;

  // Only prompt for notifications after the user has an active agent
  const showNotificationBanner = hasActiveAgents;

  return (
    <div className="space-y-6">
      {showNotificationBanner && <NotificationBanner />}
      {showOnboarding ? (
        <>
          <AgentOnboardingHero
            onRegisterAgent={() => setInviteDialogOpen(true)}
          />
          <InviteCodeDialog
            open={inviteDialogOpen}
            onOpenChange={setInviteDialogOpen}
          />
        </>
      ) : (
        <>
          <RegisteredAgentsCard />
          {showPendingApprovals && <PendingApprovalsCard />}
          {showActivity && <RecentActivityCard />}
          {showStandingApprovals && <StandingApprovalsCard />}
        </>
      )}
    </div>
  );
}
