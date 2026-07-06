/**
 * permission-slip auto-approve request — propose a standing approval rule for human review.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";

export function autoApproveRequestCommand(program: Command): void {
  const autoApprove = program
    .command("auto-approve")
    .description("Propose auto-approve (standing approval) rules");

  autoApprove
    .command("request")
    .description("Request creation of a new auto-approve rule (requires approval in web or mobile)")
    .requiredOption("--action-type <type>", "Action type (e.g. email.send)")
    .requiredOption("--constraints <json>", "Constraints JSON (use $meta for verified sender/recipient rules on supported actions)")
    .option("--action-version <version>", "Action version (digits only)", "1")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID (from saved registration)")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      actionType: string;
      constraints: string;
      actionVersion?: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        let constraints: unknown;
        try {
          constraints = JSON.parse(opts.constraints);
        } catch {
          throw new Error(`--constraints must be valid JSON. Got: ${opts.constraints}`);
        }

        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });

        const result = await client.requestStandingApproval({
          action_type: opts.actionType,
          action_version: opts.actionVersion,
          constraints,
        });

        output(
          {
            ...result,
            hint: "Approve or deny this rule proposal from the Permission Slip web or mobile app.",
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
