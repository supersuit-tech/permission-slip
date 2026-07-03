/**
 * permission-slip webhook set|status|clear
 *
 * Register the OpenClaw gateway hooks URL + token for server-push approval wakes.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";

export function webhookCommand(program: Command): void {
  const webhookCmd = program
    .command("webhook")
    .description("Configure OpenClaw gateway push wakes for approval resolution");

  webhookCmd
    .command("set")
    .description("Register hooks URL and token (runs a test wake)")
    .requiredOption("--url <url>", "Gateway hooks base URL (private network only)")
    .requiredOption("--token <token>", "Hooks bearer token from OpenClaw config")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      url: string;
      token: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.setAgentWebhook(opts.url, opts.token);
        output(result, outputOpts);
        if (result.test && !result.test.success) {
          process.exit(1);
        }
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });

  webhookCmd
    .command("status")
    .description("Show webhook config and optionally fire a test wake")
    .option("--test", "Fire a test wake to verify delivery")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      test?: boolean;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.getAgentWebhook(Boolean(opts.test));
        output(result, outputOpts);
        if (opts.test && result.test && !result.test.success) {
          process.exit(1);
        }
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });

  webhookCmd
    .command("clear")
    .description("Remove webhook configuration")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.clearAgentWebhook();
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
