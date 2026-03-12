/**
 * permission-slip register --invite-code <code> [--server <url>] [--name <name>]
 *
 * Generates (or reuses) an Ed25519 key pair, then registers with the server
 * using the provided invite code. Saves the result to
 * ~/.permission-slip/registrations.json.
 *
 * After registration the server requires a verification step — the user will
 * see a confirmation code on their dashboard and must relay it to the agent
 * (via `permission-slip verify --code <code>`).
 */

import type { Command } from "commander";
import { generateKeyPair, keyPairExists, displayPath } from "../auth/keys.js";
import { ApiClient } from "../api/client.js";
import { REGISTRATION_AGENT_ID } from "../auth/signing.js";
import { saveRegistration } from "../config/store.js";
import { output, type OutputOptions } from "../output.js";

export function registerCommand(program: Command): void {
  program
    .command("register")
    .description("Generate keys and register with a Permission Slip server")
    .requiredOption("--invite-code <code>", "Invite code from the dashboard")
    .option(
      "--server <url>",
      "Permission Slip server URL (default: https://app.permissionslip.dev)",
      "https://app.permissionslip.dev",
    )
    .option("--name <name>", "Agent name shown in the dashboard", "Agent")
    .option("--version <version>", "Agent version metadata", "1.0.0")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action(async (opts: {
      inviteCode: string;
      server: string;
      name: string;
      version: string;
      pretty?: boolean;
    }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        // Key generation
        const hadKey = keyPairExists();
        const kp = generateKeyPair(false);

        if (!hadKey && outputOpts.pretty) {
          console.error(
            `Generated new Ed25519 key pair at ${displayPath(kp.privateKeyFile)}`,
          );
        } else if (hadKey && outputOpts.pretty) {
          console.error(
            `Reusing existing key pair at ${displayPath(kp.privateKeyFile)}`,
          );
        }

        // Register
        const client = new ApiClient({
          serverUrl: opts.server,
          agentId: REGISTRATION_AGENT_ID,
        });

        const result = await client.register(
          opts.inviteCode,
          kp.publicKey,
          opts.name,
          opts.version,
        );

        // Save a partial registration (will be completed after verify)
        saveRegistration({
          server: opts.server,
          agent_id: result.agent_id,
          registered_at: new Date().toISOString(),
        });

        output(
          {
            agent_id: result.agent_id,
            expires_at: result.expires_at,
            verification_required: result.verification_required,
            key_file: displayPath(kp.privateKeyFile),
            next_step: `Run: permission-slip verify --code <confirmation_code> --server ${opts.server}`,
          },
          outputOpts,
        );
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
