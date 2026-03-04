/**
 * Hook that navigates to the ApprovalDetail screen when the user taps a push
 * notification. Extracts the approval_id from the notification data, fetches
 * the full approval from the API, and navigates using the root navigationRef.
 *
 * Handles three scenarios:
 * 1. App in foreground — notification response listener fires immediately
 * 2. App in background — listener fires when the app resumes
 * 3. Cold start — getLastNotificationResponseAsync picks up the launch notification
 *
 * If auth isn't ready when the tap fires (common on cold start), the response
 * is queued and processed once the user is authenticated.
 */
import { useCallback, useEffect, useRef } from "react";
import type { NotificationResponse } from "expo-notifications";
import * as Notifications from "expo-notifications";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { navigationRef } from "../navigation/RootNavigator";
import type { ApprovalSummary } from "./useApprovals";

/**
 * Extracts the approval_id from a notification response's data payload.
 * The backend sends `{ approval_id, url }` in the notification data.
 */
function extractApprovalId(response: NotificationResponse): string | null {
  const data = response.notification.request.content.data as
    | Record<string, unknown>
    | undefined;
  if (!data) return null;
  const id = data.approval_id;
  return typeof id === "string" && id.length > 0 ? id : null;
}

export function useNotificationNavigation() {
  const { authStatus, session } = useAuth();
  const accessTokenRef = useRef(session?.access_token);
  accessTokenRef.current = session?.access_token;

  // Track the last processed notification to avoid navigating twice for the
  // same tap (e.g. if the cold-start check and the listener both fire).
  const lastProcessedIdRef = useRef<string | null>(null);

  // Queue a pending notification response when auth isn't ready (cold start).
  const pendingResponseRef = useRef<NotificationResponse | null>(null);

  const navigateToApproval = useCallback(
    async (approvalId: string) => {
      const token = accessTokenRef.current;
      if (!token) return;

      if (__DEV__) {
        console.log(`[push] Fetching approval ${approvalId} for deep link...`);
      }

      // Fetch the approval from the list endpoint. We search "all" statuses
      // since the notification could arrive for any status transition.
      try {
        const { data, error } = await client.GET("/v1/approvals", {
          headers: { Authorization: `Bearer ${token}` },
          params: { query: { status: "all" } },
        });

        if (error || !data?.data) {
          if (__DEV__) {
            console.warn("[push] Failed to fetch approvals for deep link:", error);
          }
          return;
        }

        const approval = data.data.find(
          (a: ApprovalSummary) => a.approval_id === approvalId,
        );
        if (!approval) {
          if (__DEV__) {
            console.warn(`[push] Approval ${approvalId} not found in API response`);
          }
          return;
        }

        // The navigation container should be ready by the time auth is
        // available (it mounts as part of the initial render). If somehow
        // it's not, skip gracefully — the user can navigate manually.
        if (!navigationRef.isReady()) {
          if (__DEV__) {
            console.warn("[push] Navigation not ready — skipping deep link");
          }
          return;
        }

        if (__DEV__) {
          console.log(`[push] Navigating to ApprovalDetail for ${approvalId}`);
        }

        // Clear the app badge count since the user is actively engaging
        // with the notification content.
        Notifications.setBadgeCountAsync(0).catch(() => {
          // Best-effort — badge clearing is non-critical
        });

        navigationRef.navigate("ApprovalDetail", {
          approvalId: approval.approval_id,
          approval,
        });
      } catch {
        if (__DEV__) {
          console.warn("[push] Deep link navigation failed — user can navigate manually");
        }
      }
    },
    [],
  );

  const handleNotificationTap = useCallback(
    async (response: NotificationResponse) => {
      const approvalId = extractApprovalId(response);
      if (!approvalId) {
        if (__DEV__) {
          console.log("[push] Notification tapped but no approval_id in data — ignoring");
        }
        return;
      }

      // De-duplicate: don't navigate again for the same notification tap.
      const responseId =
        response.notification.request.identifier + response.actionIdentifier;
      if (lastProcessedIdRef.current === responseId) return;
      lastProcessedIdRef.current = responseId;

      if (__DEV__) {
        console.log(`[push] Notification tapped for approval ${approvalId}`);
      }

      const token = accessTokenRef.current;
      if (!token) {
        if (__DEV__) {
          console.log("[push] Auth not ready — queuing notification for later processing");
        }
        // Auth isn't ready yet (common on cold start). Queue the response
        // so it can be processed once authentication completes.
        pendingResponseRef.current = response;
        return;
      }

      await navigateToApproval(approvalId);
    },
    [navigateToApproval],
  );

  // Process any queued notification response once the user is authenticated.
  useEffect(() => {
    if (authStatus !== "authenticated") return;
    const pending = pendingResponseRef.current;
    if (!pending) return;
    pendingResponseRef.current = null;

    if (__DEV__) {
      console.log("[push] Auth ready — processing queued notification tap");
    }

    const approvalId = extractApprovalId(pending);
    if (approvalId) {
      navigateToApproval(approvalId);
    }
  }, [authStatus, navigateToApproval]);

  return { handleNotificationTap };
}
