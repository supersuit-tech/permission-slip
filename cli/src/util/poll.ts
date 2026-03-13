/**
 * Polls an approval's status until it reaches a terminal state or timeout.
 */

import type { ApiClient } from "../api/client.js";

/** The shape returned by ApiClient.approvalStatus(). */
export interface ApprovalStatusResult {
  approval_id: string;
  status: string;
  expires_at: string;
  created_at: string;
  execution_status?: string;
  execution_result?: unknown;
}

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

/**
 * Polls `GET /approvals/{id}/status` with gentle backoff until the approval
 * reaches a terminal state (approved/denied/cancelled/expired) or the timeout
 * elapses. Returns the last status result.
 */
export async function pollUntilResolved(
  opts: PollOptions,
): Promise<ApprovalStatusResult & { timed_out?: boolean }> {
  const deadline = Date.now() + opts.timeoutSeconds * 1000;
  let interval = 2000; // start at 2 seconds
  const maxInterval = 5000; // cap at 5 seconds

  while (Date.now() < deadline) {
    const result = (await opts.client.approvalStatus(
      opts.approvalId,
    )) as ApprovalStatusResult;

    if (TERMINAL_STATUSES.has(result.status)) {
      return result;
    }

    const remaining = deadline - Date.now();
    if (remaining <= 0) break;

    await sleep(Math.min(interval, remaining));
    interval = Math.min(Math.ceil(interval * 1.5), maxInterval);
  }

  // Final check after timeout.
  const final = (await opts.client.approvalStatus(
    opts.approvalId,
  )) as ApprovalStatusResult;

  if (TERMINAL_STATUSES.has(final.status)) {
    return final;
  }

  return { ...final, timed_out: true };
}
