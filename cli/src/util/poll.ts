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
  /** Fixed polling interval in seconds. When set, disables exponential backoff. */
  fixedIntervalSeconds?: number;
  /** Called after each poll while still pending. Use for progress messages. */
  onPoll?: (info: { status: string; elapsed: number; timeout: number }) => void;
}

const TERMINAL_STATUSES = new Set(["approved", "denied", "cancelled", "expired"]);

const DEFAULT_TIMEOUT = 120;
const MIN_TIMEOUT = 1;
const MAX_TIMEOUT = 86400;

/** Parses and clamps a timeout string to a valid range. Warns on invalid input. */
export function parseTimeout(
  value: string | undefined,
  warn: (msg: string) => void = (msg) => process.stderr.write(msg),
  options?: { flagName?: string; defaultTimeout?: number },
): number {
  const flagName = options?.flagName ?? "--timeout";
  const defaultVal = options?.defaultTimeout ?? DEFAULT_TIMEOUT;
  const parsed = Number(value);
  if (value !== undefined && (isNaN(parsed) || parsed <= 0)) {
    warn(`Warning: invalid ${flagName} value "${value}", using default ${defaultVal}s\n`);
    return defaultVal;
  }
  return Math.max(MIN_TIMEOUT, Math.min(parsed || defaultVal, MAX_TIMEOUT));
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isTransientError(err: unknown): boolean {
  if (err instanceof PermissionSlipApiError) {
    return err.apiError.retryable || err.statusCode === 429 || err.statusCode >= 500;
  }
  // Native fetch throws TypeError for network-level failures
  // (ECONNREFUSED, DNS errors, aborted connections) — treat as transient.
  if (err instanceof TypeError) {
    return true;
  }
  return false;
}

/**
 * Polls `GET /approvals/{id}/status` with gentle backoff until the approval
 * reaches a terminal state (approved/denied/cancelled/expired) or the timeout
 * elapses. Returns the last status result. Transient errors — API errors
 * (429, 5xx, retryable) and native network failures (TypeError from fetch) —
 * are silently retried on the next interval instead of aborting the wait.
 */
export async function pollUntilResolved(
  opts: PollOptions,
): Promise<ApprovalStatusResult & { timed_out?: boolean }> {
  const start = Date.now();
  const deadline = start + opts.timeoutSeconds * 1000;
  const useFixedInterval = opts.fixedIntervalSeconds !== undefined;
  let interval = useFixedInterval ? Math.max(1, opts.fixedIntervalSeconds!) * 1000 : 2000;
  const maxInterval = 5000; // cap at 5 seconds (only used with backoff)

  while (Date.now() < deadline) {
    try {
      const result = await opts.client.approvalStatus(opts.approvalId);

      if (TERMINAL_STATUSES.has(result.status)) {
        return result;
      }

      opts.onPoll?.({
        status: result.status,
        elapsed: Math.round((Date.now() - start) / 1000),
        timeout: opts.timeoutSeconds,
      });
    } catch (err) {
      if (!isTransientError(err)) throw err;
      // Transient error — continue to next poll interval.
    }

    const remaining = deadline - Date.now();
    if (remaining <= 0) break;

    await sleep(Math.min(interval, remaining));
    if (!useFixedInterval) {
      interval = Math.min(Math.ceil(interval * 1.5), maxInterval);
    }
  }

  // Final check after timeout — also protected from transient errors.
  try {
    const final = await opts.client.approvalStatus(opts.approvalId);

    if (TERMINAL_STATUSES.has(final.status)) {
      return final;
    }

    return { ...final, timed_out: true };
  } catch (err) {
    if (!isTransientError(err)) throw err;
    // Transient error on final check — report timeout with minimal info.
    return {
      approval_id: opts.approvalId,
      status: "pending",
      expires_at: "",
      created_at: "",
      timed_out: true,
    };
  }
}
