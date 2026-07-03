/**
 * Core polling loop for `permission-slip watch`.
 *
 * Polls approval status until a terminal state, expiry, or not-found.
 * Fires the notify command on every exit path. Tolerates transient poll errors.
 */

import { exec } from "node:child_process";
import { promisify } from "node:util";
import {
  expandNotifyCmd,
  expiredWakeMessage,
  notFoundWakeMessage,
  wakeMessage,
} from "./notifyCommand.js";
import { isPendingApprovalStatus } from "./waitHint.js";

const execAsync = promisify(exec);

export type PollResult =
  | { kind: "status"; status: string; expiresAt?: string }
  | { kind: "not_found" }
  | { kind: "error"; message: string };

export interface WatchLoopOptions {
  approvalId: string;
  expiresAt: Date;
  intervalMs: number;
  notifyCmdTemplate: string;
  sessionKey?: string;
  poll: () => Promise<PollResult>;
  sleep?: (ms: number) => Promise<void>;
  now?: () => Date;
  runNotify?: (cmd: string) => Promise<void>;
}

export interface WatchLoopResult {
  approval_id: string;
  status: string;
  wake_message: string;
  notified: boolean;
}

const defaultSleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

const defaultRunNotify = async (cmd: string): Promise<void> => {
  await execAsync(cmd, { shell: "/bin/sh" });
};

function parseExpiresAt(iso: string | undefined, fallback: Date): Date {
  if (!iso) {
    return fallback;
  }
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) {
    return fallback;
  }
  return parsed;
}

async function fireNotify(
  template: string,
  approvalId: string,
  status: string,
  message: string,
  sessionKey: string | undefined,
  runNotify: (cmd: string) => Promise<void>,
): Promise<boolean> {
  const cmd = expandNotifyCmd(template, approvalId, status, message, sessionKey);
  try {
    await runNotify(cmd);
    return true;
  } catch {
    return false;
  }
}

/**
 * Runs the watcher until a terminal outcome. Always returns (never loops forever).
 */
export async function runWatchLoop(opts: WatchLoopOptions): Promise<WatchLoopResult> {
  const sleep = opts.sleep ?? defaultSleep;
  const now = opts.now ?? (() => new Date());
  const runNotify = opts.runNotify ?? defaultRunNotify;

  let effectiveExpiresAt = opts.expiresAt;

  while (true) {
    const pollResult = await opts.poll();

    if (pollResult.kind === "not_found") {
      const wake_message = notFoundWakeMessage(opts.approvalId);
      const notified = await fireNotify(
        opts.notifyCmdTemplate,
        opts.approvalId,
        "not_found",
        wake_message,
        opts.sessionKey,
        runNotify,
      );
      return {
        approval_id: opts.approvalId,
        status: "not_found",
        wake_message,
        notified,
      };
    }

    if (pollResult.kind === "error") {
      if (now().getTime() >= effectiveExpiresAt.getTime()) {
        break;
      }
      await sleep(opts.intervalMs);
      continue;
    }

    effectiveExpiresAt = parseExpiresAt(pollResult.expiresAt, effectiveExpiresAt);

    if (!isPendingApprovalStatus(pollResult.status)) {
      const wake_message = wakeMessage(opts.approvalId, pollResult.status);
      const notified = await fireNotify(
        opts.notifyCmdTemplate,
        opts.approvalId,
        pollResult.status,
        wake_message,
        opts.sessionKey,
        runNotify,
      );
      return {
        approval_id: opts.approvalId,
        status: pollResult.status,
        wake_message,
        notified,
      };
    }

    if (now().getTime() >= effectiveExpiresAt.getTime()) {
      break;
    }

    await sleep(opts.intervalMs);
  }

  const wake_message = expiredWakeMessage(opts.approvalId);
  const notified = await fireNotify(
    opts.notifyCmdTemplate,
    opts.approvalId,
    "expired",
    wake_message,
    opts.sessionKey,
    runNotify,
  );
  return {
    approval_id: opts.approvalId,
    status: "expired",
    wake_message,
    notified,
  };
}
