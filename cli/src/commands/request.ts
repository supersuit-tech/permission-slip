/**
 * permission-slip request --action <action_id> [--params '{}'] [--server <url>]
 *
 * Requests one-off approval for an action. Returns immediately by default
 * with the approval_id and a next_step hint. Pass --wait to block until
 * the approval is resolved.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";
import { pollUntilResolved, parseTimeout } from "../util/poll.js";

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
    .option("--wait", "Block until the approval is resolved (default: return immediately)")
    .option("--timeout <seconds>", "Max seconds to wait when using --wait (default: 120)", "120")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      action: string;
      params: string;
      description?: string;
      riskLevel?: string;
      server: string;
      agentId?: string;
      wait?: boolean;
      timeout: string;
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

        if (!opts.wait && opts.timeout !== "120") {
          process.stderr.write("Warning: --timeout has no effect without --wait\n");
        }
        if (!opts.wait) {
          // Default: return immediately with next_step hint.
          output(
            {
              ...result,
              next_step:
                "Approval requested. To wait for the result, run: " +
                `permission-slip status --wait ${shellQuote(result.approval_id)}` +
                (opts.server !== "https://app.permissionslip.dev" ? ` --server ${shellQuote(opts.server)}` : "") +
                " (omit --wait for a single status snapshot)",
            },
            outputOpts,
          );
          return;
        }

        // --wait: block until approval resolution.
        const timeoutSeconds = parseTimeout(opts.timeout);

        process.stderr.write(
          `Waiting for approval... (approve at ${result.approval_url})\n`,
        );

        const statusResult = await pollUntilResolved({
          approvalId: result.approval_id,
          client,
          timeoutSeconds,
          onPoll: ({ elapsed, timeout }) => {
            process.stderr.write(
              `Still waiting... ${elapsed}s / ${timeout}s\n`,
            );
          },
        });

        output(
          {
            ...statusResult,
            approval_url: result.approval_url,
          },
          outputOpts,
        );

        if (statusResult.timed_out) {
          process.exitCode = 2;
        }
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
