/**
 * permission-slip config [--server <url>]
 *
 * Shows all saved registrations or the registration for a specific server.
 */

import type { Command } from "commander";
import {
  loadRegistrations,
  findRegistration,
  CONFIG_DIR,
  REGISTRATIONS_FILE,
  PRIVATE_KEY_FILE,
  PUBLIC_KEY_FILE,
} from "../config/store.js";
import { keyPairExists } from "../auth/keys.js";
import { output, type OutputOptions } from "../output.js";

export function configCommand(program: Command): void {
  program
    .command("config")
    .description("Show saved configuration and registrations")
    .option("--server <url>", "Show registration for a specific server only")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action((opts: { server?: string; pretty?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const data = {
          config_dir: CONFIG_DIR,
          registrations_file: REGISTRATIONS_FILE,
          key: {
            private_key_file: PRIVATE_KEY_FILE,
            public_key_file: PUBLIC_KEY_FILE,
            exists: keyPairExists(),
          },
          registrations: opts.server
            ? (findRegistration(opts.server) ? [findRegistration(opts.server)] : [])
            : loadRegistrations(),
        };
        output(data, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
