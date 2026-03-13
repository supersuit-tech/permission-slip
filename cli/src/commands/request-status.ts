/**
 * permission-slip request-status --approval-id <id> [--server <url>]
 *
 * Checks the status of a previously submitted approval request. By default,
 * blocks until the approval reaches a terminal state. Pass --no-wait for a
 * single status check.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";
import { pollUntilResolved } from "../util/poll.js";

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
    .option("--no-wait", "Return immediately without waiting for resolution")
    .option("--timeout <seconds>", "Max seconds to wait for resolution (default: 120)", "120")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      approvalId: string;
      server: string;
      agentId?: string;
      wait: boolean;
      timeout: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });

        if (!opts.wait) {
          // Single check, no waiting.
          const result = await client.approvalStatus(opts.approvalId);
          output(result, outputOpts);
          return;
        }

        // Wait for terminal state.
        const timeoutSeconds = Math.max(1, Math.min(Number(opts.timeout) || 120, 86400));

        process.stderr.write(`Waiting for approval ${opts.approvalId}...\n`);

        const result = await pollUntilResolved({
          approvalId: opts.approvalId,
          client,
          timeoutSeconds,
          onPoll: ({ elapsed, timeout }) => {
            process.stderr.write(
              `Still waiting... ${elapsed}s / ${timeout}s\n`,
            );
          },
        });

        output(result, outputOpts);

        if (result.timed_out) {
          process.exit(2);
        }
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
