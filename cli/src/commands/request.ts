/**
 * permission-slip request --action <action_id> [--params '{}'] [--server <url>]
 *
 * Requests one-off approval for an action. Returns the approval ID and URL.
 * The user must approve on the dashboard and share a confirmation code.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";

export function requestCommand(program: Command): void {
  program
    .command("request")
    .description("Request approval for an action")
    .requiredOption("--action <action_id>", "Action type (e.g. email.send)")
    .option("--params <json>", "Action parameters as JSON string", "{}")
    .option("--description <text>", "Human-readable description of the action")
    .option("--risk-level <level>", "Risk level: low, medium, high")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Human-readable output instead of JSON")
    .action(async (opts: {
      action: string;
      params: string;
      description?: string;
      riskLevel?: string;
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        let params: unknown;
        try {
          params = JSON.parse(opts.params);
        } catch {
          throw new Error(`--params must be valid JSON. Got: ${opts.params}`);
        }

        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });

        const context =
          opts.description || opts.riskLevel
            ? {
                description: opts.description,
                risk_level: opts.riskLevel,
              }
            : undefined;

        const result = await client.requestApproval(opts.action, params, context);

        output(
          {
            ...result,
            next_step:
              "Wait for the user to approve on the dashboard, then run: " +
              `permission-slip execute --approval ${result.approval_id} --params '${opts.params}'`,
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
