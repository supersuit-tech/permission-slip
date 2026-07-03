/**
 * permission-slip pending [--resolved-since <RFC3339>] [--server <url>]
 *
 * Heartbeat sweep: lists pending approvals and those resolved since a timestamp.
 * Run on every OpenClaw heartbeat to catch approvals the push webhook may have missed.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";

const RESOLVED_HINT =
  "One or more approvals resolved since your last sweep — fetch each with `permission-slip status <approval_id>` and continue the task.";

export function pendingCommand(program: Command): void {
  program
    .command("pending")
    .description(
      "List pending approvals and recently resolved ones (heartbeat sweep backstop)",
    )
    .option(
      "--resolved-since <rfc3339>",
      "Include terminal approvals resolved on or after this timestamp (default: 24h ago)",
    )
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      resolvedSince?: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.agentApprovalSweep(opts.resolvedSince);
        const payload =
          result.resolved.length > 0
            ? { ...result, wait_hint: RESOLVED_HINT }
            : result;
        output(payload, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
