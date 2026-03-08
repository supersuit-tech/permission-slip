import client from "@/api/client";

/**
 * Calls POST /billing/activate to confirm a Stripe checkout session and
 * activate the paid plan. This bypasses the webhook path, ensuring the
 * upgrade completes even when webhooks are delayed or misconfigured.
 *
 * Failures are silently swallowed — the caller should refetch the billing
 * plan afterward to check the actual status.
 */
export async function activateUpgrade(
  sessionId: string,
  accessToken: string,
): Promise<void> {
  try {
    await client.POST("/v1/billing/activate", {
      headers: { Authorization: `Bearer ${accessToken}` },
      body: { session_id: sessionId },
    });
  } catch {
    // Best-effort — if this fails, the webhook may still activate the plan.
  }
}
