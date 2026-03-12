/**
 * permission-slip status [--server <url>]
 *
 * Shows the current registration state for the agent on the given server.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { findRegistration } from "../config/store.js";
import { output, type OutputOptions } from "../output.js";

export function statusCommand(program: Command): void {
  program
    .command("status")
    .description("Show current registration state")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });
        const result = await client.status();
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}

export function resolveAgentId(server: string, agentIdFlag?: string): number {
  if (agentIdFlag) {
    const id = parseInt(agentIdFlag, 10);
    if (isNaN(id)) throw new Error(`Invalid agent ID: ${agentIdFlag}`);
    return id;
  }
  const reg = findRegistration(server);
  if (!reg) {
    throw new Error(
      `No registration found for ${server}. ` +
      "Run 'permission-slip register --invite-code <code>' first.",
    );
  }
  return reg.agent_id;
}
