/**
 * permission-slip status [<approval_id>] [--server <url>]
 *
 * Without an approval_id: shows the current registration state.
 * With an approval_id: checks the status of a previously submitted approval
 * request. Returns the current status and, if approved, the execution result.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { findRegistration } from "../config/store.js";
import { output, type OutputOptions } from "../output.js";
import { pollUntilResolved, parseTimeout } from "../util/poll.js";

export function statusCommand(program: Command): void {
  program
    .command("status")
    .description("Show registration state, or check approval status with an approval_id")
    .argument("[approval_id]", "Approval ID to check (omit for registration status)")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--wait", "Block until the approval reaches a terminal state")
    .option("--timeout <seconds>", "Max seconds to wait when using --wait (default: 120)", "120")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (approvalId: string | undefined, opts: {
      server: string;
      agentId?: string;
      wait?: boolean;
      timeout: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const agentId = resolveAgentId(opts.server, opts.agentId);
        const client = new ApiClient({ serverUrl: opts.server, agentId });

        if (!approvalId) {
          // No approval_id: show registration state.
          const result = await client.status();
          output(result, outputOpts);
          return;
        }

        // Approval status check.
        if (opts.wait) {
          const timeoutSeconds = parseTimeout(opts.timeout);
          process.stderr.write(`Waiting for approval ${approvalId}...\n`);
          const result = await pollUntilResolved({
            approvalId,
            client,
            timeoutSeconds,
            onPoll: ({ elapsed, timeout }) => {
              process.stderr.write(`Still waiting... ${elapsed}s / ${timeout}s\n`);
            },
          });
          output(result, outputOpts);
          if (result.timed_out) {
            process.exitCode = 2;
          }
        } else {
          // Single check, return immediately.
          const result = await client.approvalStatus(approvalId);
          output(result, outputOpts);
        }
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
