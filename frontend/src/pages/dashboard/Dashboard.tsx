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
import { AgentConfigHero } from "./AgentConfigHero";
import { InviteCodeDialog } from "./InviteCodeDialog";
import { useUnconfiguredAgent } from "@/hooks/useUnconfiguredAgent";

export function Dashboard() {
  useApprovalEvents();
  const { agents, isLoading, error } = useAgents();
  const { approvals, isLoading: approvalsLoading } = useApprovals();
  const { standingApprovals, isLoading: standingLoading } =
    useStandingApprovals();
  const { events, isLoading: eventsLoading } = useAuditEvents();
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const { isUnconfigured, agentId: unconfiguredAgentId } =
    useUnconfiguredAgent(agents, isLoading);

  const showOnboarding = !isLoading && !error && agents.length === 0;
  const hasActiveAgents = agents.some((a) => a.status === "registered");
  const hasActivity = events.length > 0;
  const hasPendingApprovals = approvals.length > 0;

  // Progressive disclosure: show cards only when relevant to the user's journey.
  // While a hook is still fetching we don't yet know whether data exists,
  // so default to showing the card to avoid jarring pop-in for returning users.
  const showPendingApprovals =
    approvalsLoading || hasActiveAgents || hasPendingApprovals;
  const showActivity = eventsLoading || hasActivity;
  const showStandingApprovals =
    standingLoading || standingApprovals.length > 0 || hasActivity;

  // Only prompt for notifications after the user has an active, configured agent
  const showNotificationBanner = hasActiveAgents && !isUnconfigured;

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
      ) : isUnconfigured ? (
        <>
          <AgentConfigHero agentId={unconfiguredAgentId} />
          <RegisteredAgentsCard />
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
