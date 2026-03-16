/**
 * permission-slip request --action <action_id> [--params '{}'] [--server <url>]
 *
 * Requests one-off approval for an action. Returns immediately by default
 * with the approval_id and a next_step hint. Pass --wait to block until
 * the approval is resolved, or --poll to poll with exponential backoff
 * (2s → 4s → … → 512s) until resolved or timed out (ideal for agents).
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";
import { pollUntilResolved, parseTimeout } from "../util/poll.js";

const DEFAULT_POLL_INTERVAL = 5;
const MIN_POLL_INTERVAL = 1;
const MAX_POLL_INTERVAL = 300;

const DEFAULT_POLL_TIMEOUT = 3600;

/** Parses and clamps --poll-interval to a valid range. */
function parsePollInterval(value: string): number {
  const parsed = Number(value);
  if (isNaN(parsed) || parsed <= 0) {
    process.stderr.write(
      `Warning: invalid --poll-interval value "${value}", using default ${DEFAULT_POLL_INTERVAL}s\n`,
    );
    return DEFAULT_POLL_INTERVAL;
  }
  const clamped = Math.max(MIN_POLL_INTERVAL, Math.min(parsed, MAX_POLL_INTERVAL));
  if (clamped !== parsed) {
    process.stderr.write(
      `Warning: --poll-interval value "${value}" out of range [${MIN_POLL_INTERVAL}, ${MAX_POLL_INTERVAL}], using ${clamped}s\n`,
    );
  }
  return clamped;
}

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
    .option("--poll", "Poll with exponential backoff until resolved (default: return immediately)")
    .option("--poll-interval <seconds>", "Use fixed polling interval instead of exponential backoff")
    .option("--poll-timeout <seconds>", `Max seconds to poll before giving up (default: ${DEFAULT_POLL_TIMEOUT})`, String(DEFAULT_POLL_TIMEOUT))
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
      poll?: boolean;
      pollInterval?: string;
      pollTimeout: string;
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

        if (opts.wait && opts.poll) {
          throw new Error("--wait and --poll are mutually exclusive. Use one or the other.");
        }

        if (!opts.wait && opts.timeout !== "120") {
          process.stderr.write("Warning: --timeout has no effect without --wait\n");
        }
        if (!opts.poll && opts.pollInterval !== undefined) {
          process.stderr.write("Warning: --poll-interval has no effect without --poll\n");
        }
        if (!opts.poll && opts.pollTimeout !== String(DEFAULT_POLL_TIMEOUT)) {
          process.stderr.write("Warning: --poll-timeout has no effect without --poll\n");
        }

        const result = await client.requestApproval(opts.action, params, context);

        if (!opts.wait && !opts.poll) {
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

        if (opts.poll) {
          // --poll: exponential backoff by default, or fixed interval with --poll-interval.
          const timeoutSeconds = parseTimeout(
            opts.pollTimeout,
            undefined,
            { flagName: "--poll-timeout", defaultTimeout: DEFAULT_POLL_TIMEOUT },
          );

          const fixedIntervalSeconds = opts.pollInterval !== undefined
            ? parsePollInterval(opts.pollInterval)
            : undefined;

          const intervalDesc = fixedIntervalSeconds !== undefined
            ? `every ${fixedIntervalSeconds}s`
            : "with exponential backoff";

          process.stderr.write(
            `Polling for approval ${intervalDesc} (timeout ${timeoutSeconds}s)... Approve at ${result.approval_url}\n`,
          );

          const statusResult = await pollUntilResolved({
            approvalId: result.approval_id,
            client,
            timeoutSeconds,
            fixedIntervalSeconds,
            onPoll: ({ elapsed, timeout }) => {
              process.stderr.write(
                `Polling... ${elapsed}s / ${timeout}s\n`,
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
