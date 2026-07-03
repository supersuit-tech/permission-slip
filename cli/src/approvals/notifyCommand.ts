/**
 * Notify-command helpers for the approval watcher.
 *
 * Default: auto-detect `openclaw` on PATH and fire a system event that wakes
 * the main OpenClaw session. Override with --notify-cmd using {id}, {status},
 * {message}, and/or {session_key} placeholders.
 */

import { execSync } from "node:child_process";
import { shellQuote } from "../util/shell.js";

export const DEFAULT_NOTIFY_CMD_TEMPLATE =
  'openclaw system event --text "{message}" --mode now';

/**
 * Fails when a template references `{session_key}` but no session key was provided.
 */
export function validateNotifyCmdTemplate(
  template: string,
  sessionKey?: string,
): void {
  if (/\{session_key\}/.test(template) && !sessionKey) {
    throw new Error(
      "Notify command template contains {session_key} but no --session-key was provided.",
    );
  }
}

/**
 * Appends `--session-key` to the default openclaw template when a session key
 * is provided. Custom templates should use the `{session_key}` placeholder instead.
 */
export function applySessionKeyToDefaultTemplate(
  template: string,
  sessionKey?: string,
): string {
  if (!sessionKey || template !== DEFAULT_NOTIFY_CMD_TEMPLATE) {
    return template;
  }
  return `${template} --session-key ${shellQuote(sessionKey)}`;
}

/** Replaces `{id}`, `{status}`, `{message}`, and `{session_key}` placeholders in a notify command template. */
export function expandNotifyCmd(
  template: string,
  approvalId: string,
  status: string,
  message?: string,
  sessionKey?: string,
): string {
  validateNotifyCmdTemplate(template, sessionKey);
  const resolvedMessage = message ?? wakeMessage(approvalId, status);
  const sessionKeyReplacement = sessionKey ? shellQuote(sessionKey) : "";
  return template
    .replace(/\{id\}/g, approvalId)
    .replace(/\{status\}/g, status)
    .replace(/\{message\}/g, resolvedMessage)
    .replace(/\{session_key\}/g, sessionKeyReplacement);
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
