/**
 * permission-slip verify --code <confirmation_code> [--server <url>]
 *
 * Completes the registration flow by submitting the confirmation code
 * that the user sees on their dashboard after reviewing the pending agent.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { findRegistration, saveRegistration } from "../config/store.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";

export function verifyCommand(program: Command): void {
  program
    .command("verify")
    .description("Complete registration with the confirmation code from the dashboard")
    .requiredOption("--code <confirmation_code>", "Confirmation code from the dashboard")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      code: string;
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        let agentId: number;
        if (opts.agentId) {
          agentId = parseInt(opts.agentId, 10);
          if (isNaN(agentId)) {
            throw new Error(`Invalid agent ID: ${opts.agentId}`);
          }
        } else {
          const reg = findRegistration(opts.server);
          if (!reg) {
            throw new Error(
              `No registration found for ${opts.server}. ` +
              "Run 'permission-slip register --invite-code <code>' first, " +
              "or pass --agent-id explicitly.",
            );
          }
          agentId = reg.agent_id;
        }

        const client = new ApiClient({ serverUrl: opts.server, agentId });
        const result = await client.verify(agentId, opts.code);

        // Update the registration with the confirmed timestamp
        saveRegistration({
          server: opts.server,
          agent_id: agentId,
          registered_at: result.registered_at,
        });

        output(
          {
            status: result.status,
            agent_id: agentId,
            registered_at: result.registered_at,
            next_step: opts.server === "https://app.permissionslip.dev"
              ? "Run: permission-slip capabilities"
              : `Run: permission-slip capabilities --server ${shellQuote(opts.server)}`,
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
