/**
 * permission-slip connectors [--server <url>] [--id <connector_id>]
 *
 * Lists available connectors (public endpoint, no auth required).
 * With --id, shows details for a specific connector.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { output, type OutputOptions } from "../output.js";

export function connectorsCommand(program: Command): void {
  program
    .command("connectors")
    .description("List available connectors (public — no registration required)")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--id <connector_id>", "Get details for a specific connector")
    .option("--pretty", "Human-readable output instead of JSON")
    .action(async (opts: {
      server: string;
      id?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const client = new ApiClient({
          serverUrl: opts.server,
          agentId: 0, // unused for public endpoints
        });
        const result = opts.id
          ? await client.connector(opts.id)
          : await client.connectors();
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
