const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api/v1";

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
    await fetch(`${API_BASE}/billing/activate`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ session_id: sessionId }),
    });
  } catch {
    // Best-effort — if this fails, the webhook may still activate the plan.
  }
}
