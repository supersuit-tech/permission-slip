import { useState } from "react";
import { useApprovalEvents } from "@/hooks/useApprovalEvents";
import { useAgents } from "@/hooks/useAgents";
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
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);

  const showOnboarding = !isLoading && !error && agents.length === 0;

  return (
    <div className="space-y-6">
      <NotificationBanner />
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
          <PendingApprovalsCard />
          <StandingApprovalsCard />
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <RecentActivityCard />
            <RegisteredAgentsCard />
          </div>
        </>
      )}
    </div>
  );
}
