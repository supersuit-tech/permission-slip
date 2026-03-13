/**
 * permission-slip request-status --approval-id <id> [--server <url>]
 *
 * Checks the status of a previously submitted approval request.
 * Returns the approval status plus execution outcome once the action
 * has been auto-executed on approval.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";

export function requestStatusCommand(program: Command): void {
  program
    .command("request-status")
    .description("Check the status of an approval request")
    .requiredOption("--approval-id <id>", "Approval ID returned by the request command")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      approvalId: string;
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });
        const result = await client.approvalStatus(opts.approvalId);
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
