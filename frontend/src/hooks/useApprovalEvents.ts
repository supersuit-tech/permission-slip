import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";

/**
 * Resolves the SSE endpoint URL for approval events.
 * Uses the same base URL logic as the API client.
 */
function resolveSSEUrl(): string {
  const envUrl = import.meta.env.VITE_API_BASE_URL;
  if (envUrl) {
    // Strip trailing /v1 and slashes, then add the SSE path
    const base = envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
    return `${base}/v1/approvals/events`;
  }
  return "/api/v1/approvals/events";
}

/**
 * Connects to the approval events SSE endpoint and invalidates
 * the approvals query cache when new events arrive. This replaces
 * 5-second polling with instant updates.
 *
 * Events:
 *  - approval_created: a new approval was requested by an agent
 *  - approval_resolved: an approval was approved or denied
 *  - approval_cancelled: an agent cancelled an approval
 *
 * The hook is resilient to disconnections — EventSource automatically
 * reconnects with exponential backoff (browser built-in). The
 * `connected` event confirms the stream is live.
 */
export function useApprovalEvents() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.access_token;
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    // Guard for environments without EventSource (e.g. jsdom in tests,
    // or very old browsers). The 30-second poll fallback in useApprovals
    // still works.
    if (!accessToken || typeof EventSource === "undefined") return;

    // EventSource doesn't support custom headers, so we pass the token
    // as a query parameter. The backend SSE endpoint accepts both
    // Authorization header and ?token= query parameter.
    const url = `${resolveSSEUrl()}?token=${encodeURIComponent(accessToken)}`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    const invalidateApprovals = () => {
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
    };

    const onApprovalCreated = () => {
      invalidateApprovals();
      toast.info("New approval request", {
        description: "An agent is waiting for your review.",
      });
    };

    es.addEventListener("approval_created", onApprovalCreated);
    es.addEventListener("approval_resolved", invalidateApprovals);
    es.addEventListener("approval_cancelled", invalidateApprovals);

    // Clean up on unmount or token change.
    return () => {
      es.removeEventListener("approval_created", onApprovalCreated);
      es.removeEventListener("approval_resolved", invalidateApprovals);
      es.removeEventListener("approval_cancelled", invalidateApprovals);
      es.close();
      eventSourceRef.current = null;
    };
  }, [accessToken, queryClient]);
}
