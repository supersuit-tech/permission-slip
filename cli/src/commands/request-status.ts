/**
 * permission-slip request-status --approval-id <id> [--server <url>]
 *
 * Checks the status of a previously submitted approval request. Always
 * returns immediately with the current status.
 *
 * Deprecated: prefer `permission-slip status <approval_id>` instead.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { resolveServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";

export function requestStatusCommand(program: Command): void {
  program
    .command("request-status")
    .description("Check approval status (deprecated: use 'status <approval_id>' instead)")
    .requiredOption("--approval-id <id>", "Approval ID returned by the request command")
    .option(
      "--server <url>",
      "Permission Slip server URL (overrides PS_SERVER and config default_server)",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      approvalId: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = resolveServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });

        const result = await client.approvalStatus(opts.approvalId);
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
