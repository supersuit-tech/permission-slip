/**
 * permission-slip request --action <action_id> [--params '{}'] [--server <url>]
 *
 * Requests approval for an action. If a matching standing approval exists,
 * the action is auto-approved and executed immediately — the result is
 * returned inline. Otherwise, creates a pending approval and returns
 * an approval_id to poll via `permission-slip status`.
 */

import type { Command } from "commander";
import { ApiClient, PermissionSlipApiError } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { resolveServerUrl, isBuiltInDefaultServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";

export function requestCommand(program: Command): void {
  program
    .command("request")
    .description("Request approval for an action (auto-approves if a standing approval matches)")
    .requiredOption("--action <action_id>", "Action type (e.g. email.send)")
    .option("--params <json>", "Action parameters as JSON string", "{}")
    .option("--description <text>", "Human-readable description of the action")
    .option("--risk-level <level>", "Risk level: low, medium, high")
    .option(
      "--server <url>",
      "Permission Slip server URL (overrides PS_SERVER and config default_server)",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--request-id <id>", "Idempotency key — reuse across retries to prevent duplicate execution")
    .option("--payment-method-id <id>", "Payment method ID for payment-required actions")
    .option("--amount-cents <cents>", "Transaction amount in cents (required with --payment-method-id)", parseInt)
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      action: string;
      params: string;
      description?: string;
      riskLevel?: string;
      server?: string;
      agentId?: string;
      requestId?: string;
      paymentMethodId?: string;
      amountCents?: number;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = resolveServerUrl({ serverFlag: opts.server });
        let params: unknown;
        try {
          params = JSON.parse(opts.params);
        } catch {
          throw new Error(`--params must be valid JSON. Got: ${opts.params}`);
        }

        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });

        const context =
          opts.description || opts.riskLevel
            ? {
                description: opts.description,
                risk_level: opts.riskLevel,
              }
            : undefined;

        const result = await client.requestApproval(
          opts.action,
          params,
          context,
          { paymentMethodId: opts.paymentMethodId, amountCents: opts.amountCents },
          opts.requestId,
        );

        if (result.status === "approved") {
          // Auto-approved via standing approval — action already executed, result is inline.
          // Include executed flag so external tools know no further action is needed.
          output({ ...result, executed: true }, outputOpts);
        } else {
          // Pending — tell the user how to check the result.
          output(
            {
              ...result,
              next_step:
                "Approval requested. To check the result, run: " +
                `permission-slip status ${shellQuote(result.approval_id ?? "")}` +
                (!isBuiltInDefaultServerUrl(server) ? ` --server ${shellQuote(server)}` : ""),
            },
            outputOpts,
          );
        }
      } catch (err) {
        // Treat 409 duplicate_request_id as an idempotency success — the action
        // was already executed on a prior call with this request_id. Exit 0 so
        // external tools (AI agents, MCP wrappers) don't retry.
        if (
          err instanceof PermissionSlipApiError &&
          err.statusCode === 409 &&
          err.apiError.code === "duplicate_request_id"
        ) {
          output({ status: "duplicate", executed: true, request_id: opts.requestId }, outputOpts);
          return;
        }
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
