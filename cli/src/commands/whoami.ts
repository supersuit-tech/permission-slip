/**
 * permission-slip whoami [--server <url>]
 *
 * Shows agent identity: local registration info and live status from the server.
 */

import type { Command } from "commander";
import { ApiClient } from "../api/client.js";
import { loadRegistrations, findRegistration } from "../config/store.js";
import { keyPairExists, readPublicKey } from "../auth/keys.js";
import { output, type OutputOptions } from "../output.js";

export function whoamiCommand(program: Command): void {
  program
    .command("whoami")
    .description("Show agent identity and registration info")
    .option(
      "--server <url>",
      "Permission Slip server URL",
      "https://app.permissionslip.dev",
    )
    .option("--agent-id <id>", "Agent ID (auto-detected from saved registration)")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      server: string;
      agentId?: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const hasKey = keyPairExists();
        let publicKey: string | null = null;
        if (hasKey) {
          try {
            publicKey = readPublicKey();
          } catch {
            // ignore
          }
        }

        const registrations = loadRegistrations();
        const reg = findRegistration(opts.server);
        let agentId: number | undefined;
        if (opts.agentId) {
          agentId = parseInt(opts.agentId, 10);
        } else if (reg) {
          agentId = reg.agent_id;
        }

        let liveStatus: unknown = null;
        if (agentId !== undefined) {
          try {
            const client = new ApiClient({ serverUrl: opts.server, agentId });
            liveStatus = await client.status();
          } catch {
            liveStatus = null;
          }
        }

        output(
          {
            key: { exists: hasKey, public_key: publicKey },
            registrations,
            current_server: {
              server: opts.server,
              registration: reg ?? null,
              live_status: liveStatus,
            },
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
