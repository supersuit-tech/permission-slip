/**
 * permission-slip request-bulk --action <type> --actions '<json-array>'
 *
 * Submits N same-type actions as one bulk approval (one notification for the user).
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";
import { shellQuote } from "../util/shell.js";

interface BulkActionItem {
  request_id?: string;
  parameters: unknown;
  description?: string;
  risk_level?: string;
}

export function requestBulkCommand(program: Command): void {
  program
    .command("request-bulk")
    .description(
      "Request bulk approval for N same-type actions (one notification for the user)",
    )
    .requiredOption("--action <action_id>", "Shared action type for all items")
    .requiredOption(
      "--actions <json>",
      "JSON array of action items: [{\"parameters\":{...},\"description\":\"...\"}, ...]",
    )
    .option(
      "--server <url>",
      "Permission Slip server URL — required unless PS_SERVER or config default_server is set",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      action: string;
      actions: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        let items: BulkActionItem[];
        try {
          items = JSON.parse(opts.actions) as BulkActionItem[];
        } catch {
          throw new Error(`--actions must be a valid JSON array. Got: ${opts.actions}`);
        }
        if (!Array.isArray(items) || items.length < 2) {
          throw new Error("--actions must be a JSON array with at least 2 items");
        }

        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });

        const bodyItems = items.map((item) => ({
          request_id: item.request_id,
          action: {
            type: opts.action,
            parameters: item.parameters ?? {},
          },
          context: {
            ...(item.description ? { description: item.description } : {}),
            ...(item.risk_level ? { risk_level: item.risk_level } : {}),
          },
        }));

        const result = await client.requestBulkApproval(bodyItems);

        output(
          {
            ...result,
            next_step:
              result.status === "pending" || result.status === "partial"
                ? "Bulk approval submitted. Poll: " +
                  `permission-slip status --group ${shellQuote(result.bulk_group_id ?? "")} --server ${shellQuote(server)}`
                : undefined,
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
