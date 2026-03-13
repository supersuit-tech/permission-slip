/**
 * permission-slip request --action <action_id> [--params '{}'] [--server <url>]
 *
 * Requests one-off approval for an action. Returns the approval ID and URL.
 * Once the approver approves on the dashboard, the action executes automatically
 * — there is no separate execute step for one-off approvals.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";

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
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
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
              "Your request is pending approval. Once approved, the action will execute automatically — no further action is needed from you. " +
              "To check the outcome, run: " +
              `permission-slip request-status --approval-id ${shellQuote(result.approval_id)}` +
              (opts.server !== "https://app.permissionslip.dev" ? ` --server ${shellQuote(opts.server)}` : ""),
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
