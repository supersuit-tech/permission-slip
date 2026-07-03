/**
 * permission-slip watch <approval_id>
 *
 * Polls approval status in a detached background process and fires a notify
 * command (default: openclaw system event) when the approval reaches a terminal
 * state, expires, or is not found. Designed so agents end their turn instead of
 * busy-polling inside the session.
 */

import type { Command } from "commander";
import { ApiClient, PermissionSlipApiError } from "../api/client.js";
import {
  applySessionKeyToDefaultTemplate,
  resolveNotifyCmdTemplate,
  validateNotifyCmdTemplate,
} from "../approvals/notifyCommand.js";
import { runWatchLoop, type PollResult } from "../approvals/watchLoop.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";
import { parseDurationToSeconds } from "../util/parseDuration.js";
import { resolveAgentId } from "./status.js";

const WATCH_HELP =
  "Poll a pending approval in the background and wake your session when it resolves. " +
  "Run this as a detached background process after `request` returns pending — do NOT poll " +
  "`status` in a loop inside your turn. The watcher exits on approval/denial/cancel, expiry, " +
  "or if the approval is deleted (404), firing a notify command each time (default: " +
  "`openclaw system event` when openclaw is on PATH). With `--session-key`, the default notify " +
  "uses `--mode next-heartbeat` for a reliable targeted wake. JSON output includes " +
  "`notify_attempts` because gateway RPC ok does not guarantee the session resumed. " +
  "One watcher per approval; N pending approvals means N small background processes.";

export function watchCommand(program: Command): void {
  program
    .command("watch")
    .description(WATCH_HELP)
    .argument("<approval_id>", "Approval ID returned by `request`")
    .option(
      "--notify-cmd <cmd>",
      "Shell command to run on exit; use {id}, {status}, {message}, and/or {session_key} placeholders " +
        "(default: openclaw system event when openclaw is on PATH)",
    )
    .option(
      "--session-key <key>",
      "OpenClaw session key to wake when the approval resolves (targets the session that spawned the watcher)",
    )
    .option("--interval <duration>", "Poll interval (e.g. 5s, 30s)", "5s")
    .option(
      "--expires-at <iso>",
      "Approval expiry time (ISO 8601); fetched from status on first poll if omitted",
    )
    .option(
      "--server <url>",
      "Permission Slip server URL — required unless PS_SERVER or config default_server is set",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (
      approvalId: string,
      opts: {
        notifyCmd?: string;
        sessionKey?: string;
        interval: string;
        expiresAt?: string;
        server?: string;
        agentId?: string;
        pretty?: boolean;
      },
    ) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const baseTemplate = resolveNotifyCmdTemplate(opts.notifyCmd);
        if (!baseTemplate) {
          throw new Error(
            "No notify command available. Install openclaw on PATH or pass --notify-cmd " +
              "with {id} and {status} placeholders.",
          );
        }
        validateNotifyCmdTemplate(baseTemplate, opts.sessionKey);
        const notifyTemplate = applySessionKeyToDefaultTemplate(
          baseTemplate,
          opts.sessionKey,
        );

        const intervalMs = parseDurationToSeconds(opts.interval) * 1000;
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });

        const poll = async (): Promise<PollResult> => {
          try {
            const result = await client.approvalStatus(approvalId);
            return {
              kind: "status",
              status: result.status,
              expiresAt: result.expires_at,
            };
          } catch (err) {
            if (
              err instanceof PermissionSlipApiError &&
              err.statusCode === 404
            ) {
              return { kind: "not_found" };
            }
            const message = err instanceof Error ? err.message : String(err);
            return { kind: "error", message };
          }
        };

        let expiresAt: Date;
        if (opts.expiresAt) {
          const parsed = new Date(opts.expiresAt);
          if (Number.isNaN(parsed.getTime())) {
            throw new Error(`Invalid --expires-at: ${opts.expiresAt}`);
          }
          expiresAt = parsed;
        } else {
          const first = await poll();
          if (first.kind === "status" && first.expiresAt) {
            expiresAt = new Date(first.expiresAt);
            if (Number.isNaN(expiresAt.getTime())) {
              expiresAt = new Date(Date.now() + 24 * 3600_000);
            }
          } else {
            expiresAt = new Date(Date.now() + 24 * 3600_000);
          }
        }

        const result = await runWatchLoop({
          approvalId,
          expiresAt,
          intervalMs,
          notifyCmdTemplate: notifyTemplate,
          sessionKey: opts.sessionKey,
          poll,
        });
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
