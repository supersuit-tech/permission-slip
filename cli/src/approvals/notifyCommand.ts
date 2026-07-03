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
 * Default notify template when `--session-key` is set.
 *
 * OpenClaw treats `--mode next-heartbeat` with `--session-key` as an immediate
 * targeted wake (heartbeat intent `immediate`) that bypasses the runner's not-due
 * gate. `--mode now --session-key` can return RPC ok without resuming an
 * idle/yielded session — see permission-slip#1365.
 */
export const DEFAULT_NOTIFY_CMD_TEMPLATE_WITH_SESSION_KEY =
  'openclaw system event --text "{message}" --mode next-heartbeat --session-key {session_key}';

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
 * Switches the default openclaw template to the session-key variant when a key
 * is provided. Custom templates should use the `{session_key}` placeholder instead.
 */
export function applySessionKeyToDefaultTemplate(
  template: string,
  sessionKey?: string,
): string {
  if (!sessionKey || template !== DEFAULT_NOTIFY_CMD_TEMPLATE) {
    return template;
  }
  return DEFAULT_NOTIFY_CMD_TEMPLATE_WITH_SESSION_KEY;
}

/** True when the template is one of the built-in openclaw defaults. */
export function isDefaultOpenclawNotifyTemplate(template: string): boolean {
  return (
    template === DEFAULT_NOTIFY_CMD_TEMPLATE ||
    template === DEFAULT_NOTIFY_CMD_TEMPLATE_WITH_SESSION_KEY
  );
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
