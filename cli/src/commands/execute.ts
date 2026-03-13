/**
 * permission-slip execute --configuration <id> [--params '{}'] [--server <url>]
 *
 * Executes an action using a standing approval configuration.
 *
 * Standing approvals are pre-approved recurring action configurations. Use
 * `permission-slip capabilities` to see available standing approvals and
 * their configuration IDs.
 *
 * NOTE: For one-off approvals (created via `permission-slip request`), the
 * server auto-executes the action when the approver approves it — you do NOT
 * need to call `execute`. Use `permission-slip request-status` to check the
 * outcome instead.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";

export function executeCommand(program: Command): void {
  program
    .command("execute")
    .description("Execute an action using a standing approval (one-off approvals auto-execute on approval)")
    .option("--token <token>", "Execution token (advanced — most agents should use --configuration instead)")
    .option("--configuration <id>", "Standing approval configuration ID")
    .option("--action <action_id>", "Action type (required with --token)")
    .option("--params <json>", "Action parameters as JSON string", "{}")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
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
        if (opts.token && !opts.action) {
          throw new Error("--action <action_id> is required when using --token.");
        }

        let params: unknown;
        try {
          params = JSON.parse(opts.params);
        } catch {
          throw new Error(`--params must be valid JSON. Got: ${opts.params}`);
        }

        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });

        const result: unknown = opts.token
          ? await client.execute({ token: opts.token }, opts.action, params)
          : await client.execute({ configuration_id: opts.configuration! }, undefined, params);

        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
