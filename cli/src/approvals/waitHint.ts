/**
 * In-band discovery for non-blocking approval waiting.
 *
 * Single source of truth for wait_hint / wait_command text shown when an
 * approval is still pending. Tuned for LLM readers — imperative, self-contained.
 */

/** Shown when status is `pending` on `request` and `status` output. */
export const WAIT_HINT =
  "Do NOT poll in a loop and do NOT block. Run the following command AS A DETACHED BACKGROUND PROCESS, then end your turn; you will be woken with the outcome when the human responds:";

/** Builds the `permission-slip watch` command string for a pending approval. */
export function buildWaitCommand(approvalId: string): string {
  return `permission-slip watch ${approvalId}`;
}

/** Fields merged into JSON output when an approval is still pending. */
export function pendingWaitFields(approvalId: string): {
  wait_hint: string;
  wait_command: string;
} {
  return {
    wait_hint: WAIT_HINT,
    wait_command: buildWaitCommand(approvalId),
  };
}

/** True when the approval is still awaiting human action. */
export function isPendingApprovalStatus(status: string): boolean {
  return status === "pending";
}
