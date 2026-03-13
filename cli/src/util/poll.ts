/**
 * Polls an approval's status until it reaches a terminal state or timeout.
 */

import {
  type ApiClient,
  type ApprovalStatusResult,
  PermissionSlipApiError,
} from "../api/client.js";

export type { ApprovalStatusResult };

export interface PollOptions {
  approvalId: string;
  client: ApiClient;
  /** Maximum seconds to wait before returning. */
  timeoutSeconds: number;
}

const TERMINAL_STATUSES = new Set(["approved", "denied", "cancelled", "expired"]);

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isTransientError(err: unknown): boolean {
  if (err instanceof PermissionSlipApiError) {
    return err.apiError.retryable || err.statusCode === 429 || err.statusCode >= 500;
  }
  return false;
}

/**
 * Polls `GET /approvals/{id}/status` with gentle backoff until the approval
 * reaches a terminal state (approved/denied/cancelled/expired) or the timeout
 * elapses. Returns the last status result. Transient API errors (429, 5xx) are
 * silently retried on the next interval instead of aborting the wait.
 */
export async function pollUntilResolved(
  opts: PollOptions,
): Promise<ApprovalStatusResult & { timed_out?: boolean }> {
  const deadline = Date.now() + opts.timeoutSeconds * 1000;
  let interval = 2000; // start at 2 seconds
  const maxInterval = 5000; // cap at 5 seconds

  while (Date.now() < deadline) {
    try {
      const result = await opts.client.approvalStatus(opts.approvalId);

      if (TERMINAL_STATUSES.has(result.status)) {
        return result;
      }
    } catch (err) {
      if (!isTransientError(err)) throw err;
      // Transient error — continue to next poll interval.
    }

    const remaining = deadline - Date.now();
    if (remaining <= 0) break;

    await sleep(Math.min(interval, remaining));
    interval = Math.min(Math.ceil(interval * 1.5), maxInterval);
  }

  // Final check after timeout.
  const final = await opts.client.approvalStatus(opts.approvalId);

  if (TERMINAL_STATUSES.has(final.status)) {
    return final;
  }

  return { ...final, timed_out: true };
}
