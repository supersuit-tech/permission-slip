/**
 * Notify-command helpers for the approval watcher.
 *
 * Default: auto-detect `openclaw` on PATH and fire a system event that wakes
 * the main OpenClaw session. Override with --notify-cmd using {id} and {status}
 * placeholders.
 */

import { execSync } from "node:child_process";

export const DEFAULT_NOTIFY_CMD_TEMPLATE =
  'openclaw system event --text "{message}" --mode now';

/** Replaces `{id}`, `{status}`, and `{message}` placeholders in a notify command template. */
export function expandNotifyCmd(
  template: string,
  approvalId: string,
  status: string,
  message?: string,
): string {
  const resolvedMessage = message ?? wakeMessage(approvalId, status);
  return template
    .replace(/\{id\}/g, approvalId)
    .replace(/\{status\}/g, status)
    .replace(/\{message\}/g, resolvedMessage);
}

/** Returns true when `openclaw` is available on PATH. */
export function isOpenclawOnPath(): boolean {
  try {
    execSync("command -v openclaw", { stdio: "ignore", shell: "/bin/sh" });
    return true;
  } catch {
    return false;
  }
}

/**
 * Resolves the notify command template: explicit override, or openclaw default
 * when detected, or null when nothing is available.
 */
export function resolveNotifyCmdTemplate(explicit?: string): string | null {
  if (explicit && explicit.trim().length > 0) {
    return explicit.trim();
  }
  if (isOpenclawOnPath()) {
    return DEFAULT_NOTIFY_CMD_TEMPLATE;
  }
  return null;
}

/** Human-readable wake text for a resolved approval (included in notify + final output). */
export function wakeMessage(approvalId: string, status: string): string {
  return `Permission Slip ${approvalId} resolved: ${status} — continue the task`;
}

/** Wake text when the approval expires before a human responds. */
export function expiredWakeMessage(approvalId: string): string {
  return `Permission Slip ${approvalId} expired unanswered`;
}

/** Wake text when the approval is missing (404 / deleted). */
export function notFoundWakeMessage(approvalId: string): string {
  return `Permission Slip ${approvalId} not found — may have been deleted`;
}
