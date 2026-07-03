/**
 * In-band discovery for non-blocking approval waiting.
 *
 * Single source of truth for wait_hint / wait_command text shown when an
 * approval is still pending. Tuned for LLM readers — imperative, self-contained.
 */

/** Shown when status is `pending` on `request` and `status` output. */
export const WAIT_HINT =
  "Do NOT poll in a loop and do NOT block. Run the following command AS A DETACHED BACKGROUND PROCESS, then end your turn; you will be woken with the outcome when the human responds. If your wake channel routes by session (e.g. OpenClaw), append --session-key <your session key> to the watch command so the wake targets the session that opened the approval, not the default session:";

/** Shown when a push wake webhook is registered — watcher is optional. */
export const PUSH_WAKE_WAIT_HINT =
  "A push wake webhook is configured — the server will wake your OpenClaw gateway when the human responds (watcher optional). You may still run wait_command as a detached fallback. If your wake channel routes by session, append --session-key <your session key> to the watch command:";

/** Builds the `permission-slip watch` command string for a pending approval. */
export function buildWaitCommand(approvalId: string): string {
  return `permission-slip watch ${approvalId}`;
}

/** Fields merged into JSON output when an approval is still pending. */
export function pendingWaitFields(
  approvalId: string,
  pushWakeConfigured = false,
): {
  wait_hint: string;
  wait_command: string;
} {
  return {
    wait_hint: pushWakeConfigured ? PUSH_WAKE_WAIT_HINT : WAIT_HINT,
    wait_command: buildWaitCommand(approvalId),
  };
}

/** True when the approval is still awaiting human action. */
export function isPendingApprovalStatus(status: string): boolean {
  return status === "pending";
}
