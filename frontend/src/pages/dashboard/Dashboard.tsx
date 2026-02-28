import { useApprovalEvents } from "@/hooks/useApprovalEvents";
import { NotificationBanner } from "./NotificationBanner";
import { PendingApprovalsCard } from "./PendingApprovalsCard";
import { RecentActivityCard } from "./RecentActivityCard";
import { RegisteredAgentsCard } from "./RegisteredAgentsCard";
import { StandingApprovalsCard } from "./StandingApprovalsCard";

export function Dashboard() {
  // Connect to SSE for real-time approval updates. New approvals
  // appear instantly instead of waiting for the 30-second poll cycle.
  useApprovalEvents();

  return (
    <div className="space-y-6">
      <NotificationBanner />
      <PendingApprovalsCard />
      <StandingApprovalsCard />
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <RecentActivityCard />
        <RegisteredAgentsCard />
      </div>
    </div>
  );
}
