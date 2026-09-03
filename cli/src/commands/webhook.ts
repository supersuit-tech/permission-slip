/**
 * permission-slip webhook set|status|clear
 *
 * Register a push-wake webhook (OpenClaw gateway hooks or Grok Bot Cursor webhook)
 * for server-push approval wakes.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { resolveAgentId } from "./status.js";
import { requireServerUrl } from "../config/serverUrl.js";
import { output, type OutputOptions } from "../output.js";

export function webhookCommand(program: Command): void {
  const webhookCmd = program
    .command("webhook")
    .description("Configure push wakes for approval resolution (OpenClaw or Grok Bot)");

  webhookCmd
    .command("set")
    .description("Register webhook URL and token (runs a test wake)")
    .requiredOption("--url <url>", "OpenClaw hooks base URL, or Grok Bot https://api2.cursor.sh/automations/webhook/… URL")
    .requiredOption("--token <token>", "OpenClaw hooks bearer token, or Grok Bot Authorization header value")
    .option("--provider <provider>", "Wake adapter: openclaw (default) or grokbot", "openclaw")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      url: string;
      token: string;
      provider?: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.setAgentWebhook(opts.url, opts.token, opts.provider ?? "openclaw");
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
