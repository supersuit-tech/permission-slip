/**
 * permission-slip execute --approval <id> [--params '{}'] [--server <url>]
 *   or
 * permission-slip execute --configuration <id> [--params '{}'] [--server <url>]
 *
 * Executes an approved action using either:
 *  - A one-off approval token (requires --approval to get a token first)
 *  - A standing approval configuration (--configuration)
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";

export function executeCommand(program: Command): void {
  program
    .command("execute")
    .description("Execute an approved action")
    .option("--approval <id>", "Approval ID from a 'request' command")
    .option("--token <token>", "Execution token (obtained via approval verification)")
    .option("--configuration <id>", "Standing approval configuration ID")
    .option("--action <action_id>", "Action type (required with --token)")
    .option("--params <json>", "Action parameters as JSON string", "{}")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Human-readable output instead of JSON")
    .action(async (opts: {
      approval?: string;
      token?: string;
      configuration?: string;
      action?: string;
      params: string;
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        if (!opts.token && !opts.configuration) {
          throw new Error(
            "Provide either --token <token> (one-off approval) or --configuration <id> (standing approval).",
          );
        }

        let params: unknown;
        try {
          params = JSON.parse(opts.params);
        } catch {
          throw new Error(`--params must be valid JSON. Got: ${opts.params}`);
        }

        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });

        let result: unknown;
        if (opts.token) {
          result = await client.execute(
            { token: opts.token },
            opts.action,
            params,
          );
        } else if (opts.configuration) {
          result = await client.execute(
            { configuration_id: opts.configuration },
            undefined,
            params,
          );
        }

        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
