/**
 * permission-slip request --action <action_id> [--instance <name_or_uuid>] [--params '{}'] [--server <url>]
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
import {
  mergeParamsWithConnectorInstance,
  parseAvailableInstances,
} from "../util/connectorInstance.js";
import { promptConnectorInstanceChoice } from "../util/promptConnectorInstance.js";

export function requestCommand(program: Command): void {
  program
    .command("request")
    .description("Request approval for an action (auto-approves if a standing approval matches)")
    .requiredOption("--action <action_id>", "Action type (e.g. email.send)")
    .option(
      "--instance <name_or_uuid>",
      "Connector instance (UUID or display name); merged into parameters as connector_instance",
    )
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
      instance?: string;
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

        // Snapshot before --instance merge so interactive retry can merge a picked UUID into the same base object.
        const paramsBeforeInstance: unknown = JSON.parse(opts.params);

        const instanceFlag = opts.instance?.trim();
        if (instanceFlag) {
          params = mergeParamsWithConnectorInstance(params, instanceFlag);
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

        const send = async (p: unknown) =>
          client.requestApproval(
            opts.action,
            p,
            context,
            { paymentMethodId: opts.paymentMethodId, amountCents: opts.amountCents },
            opts.requestId,
          );

        const tryRecoverConnectorInstance = async (
          err: PermissionSlipApiError,
        ): Promise<Awaited<ReturnType<ApiClient["requestApproval"]>> | null> => {
          if (err.statusCode !== 400 || err.apiError.code !== "connector_instance_required") {
            return null;
          }
          if (!process.stdin.isTTY || instanceFlag) {
            return null;
          }
          const list = parseAvailableInstances(err.apiError.details);
          if (list.length === 0) {
            return null;
          }
          const chosen = await promptConnectorInstanceChoice(list);
          if (!chosen) {
            return null;
          }
          return send(mergeParamsWithConnectorInstance(paramsBeforeInstance, chosen));
        };

        let result: Awaited<ReturnType<ApiClient["requestApproval"]>>;
        try {
          result = await send(params);
        } catch (firstErr) {
          if (firstErr instanceof PermissionSlipApiError) {
            const recovered = await tryRecoverConnectorInstance(firstErr);
            if (recovered !== null) {
              result = recovered;
            } else {
              throw firstErr;
            }
          } else {
            throw firstErr;
          }
        }

        if (result.status === "approved") {
          output({ ...result, executed: true }, outputOpts);
        } else {
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
