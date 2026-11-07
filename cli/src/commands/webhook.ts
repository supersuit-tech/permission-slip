/**
 * permission-slip webhook set|status|clear
 *
 * Register the wake token and provider. The destination URL is passed per
 * approval request via --callback-url, and to a test wake via status --test --url.
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
    .description("Register the wake token and provider. Pass the destination with request --callback-url.")
    .requiredOption("--token <token>", "OpenClaw hooks bearer token, or Grok Bot Authorization header value")
    .option("--provider <provider>", "Wake adapter: openclaw (default) or grokbot", "openclaw")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
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
        const result = await client.setAgentWebhook(opts.token, opts.provider ?? "openclaw");
        output(result, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });

  webhookCmd
    .command("status")
    .description("Show wake token registration and optionally fire a test wake")
    .option("--test", "Fire a test wake to the URL given by --url")
    .option("--url <url>", "Destination for --test (current hooks base or Grok Bot webhook)")
    .option("--server <url>", "Permission Slip server URL")
    .option("--agent-id <id>", "Agent ID")
    .option("--pretty", "Pretty-printed JSON")
    .action(async (opts: {
      test?: boolean;
      url?: string;
      server?: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        if (opts.test && !opts.url?.trim()) {
          throw new Error("--url is required with --test");
        }
        const { url: server } = requireServerUrl({ serverFlag: opts.server });
        const agentId = resolveAgentId(server, opts.agentId);
        const client = new ApiClient({ serverUrl: server, agentId });
        const result = await client.getAgentWebhook(opts.test ? opts.url : undefined);
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
