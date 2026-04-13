/**
 * permission-slip config [--server <url>]
 * permission-slip config set <key> <value>
 * permission-slip config unset <key>
 *
 * Shows saved configuration and registrations, or updates ~/.permission-slip/config.json.
 */

import type { Command } from "commander";
import {
  loadRegistrations,
  findRegistration,
  CONFIG_DIR,
  CONFIG_FILE,
  REGISTRATIONS_FILE,
  loadConfig,
  saveConfig,
  unsetConfigKey,
  PRIVATE_KEY_FILE,
  PUBLIC_KEY_FILE,
} from "../config/store.js";
import { resolveServerUrl } from "../config/serverUrl.js";
import { keyPairExists } from "../auth/keys.js";
import { output, type OutputOptions } from "../output.js";

const CONFIG_KEYS = ["default_server"] as const;
type ConfigKey = (typeof CONFIG_KEYS)[number];

function assertConfigKey(key: string): asserts key is ConfigKey {
  if (!CONFIG_KEYS.includes(key as ConfigKey)) {
    throw new Error(
      `Unknown config key: ${key}. Supported keys: ${CONFIG_KEYS.join(", ")}`,
    );
  }
}

export function configCommand(program: Command): void {
  const configCmd = program
    .command("config")
    .description("Show or update saved configuration and registrations");

  configCmd
    .command("set")
    .description("Persist a preference (e.g. default_server for the API URL)")
    .argument("<key>", `Preference key (${CONFIG_KEYS.join(", ")})`)
    .argument("<value>", "Value to store")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action((key: string, value: string, cmdOpts: { pretty?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: cmdOpts.pretty ?? false };
      try {
        assertConfigKey(key);
        if (key === "default_server") {
          saveConfig({ default_server: value.trim() });
        }
        output({ ok: true, key, value: value.trim() }, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });

  configCmd
    .command("unset")
    .description("Remove a preference (reverts to built-in defaults / env)")
    .argument("<key>", `Preference key (${CONFIG_KEYS.join(", ")})`)
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action((key: string, cmdOpts: { pretty?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: cmdOpts.pretty ?? false };
      try {
        assertConfigKey(key);
        unsetConfigKey(key);
        output({ ok: true, key }, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });

  configCmd
    .option("--server <url>", "Show registration for a specific server only")
    .option("--pretty", "Pretty-printed JSON (default is compact JSON)")
    .action((opts: { server?: string; pretty?: boolean }) => {
      const outputOpts: OutputOptions = { pretty: opts.pretty ?? false };
      try {
        const resolved = resolveServerUrl({});
        const regForServer = opts.server ? findRegistration(opts.server) : undefined;
        const data = {
          config_dir: CONFIG_DIR,
          preferences_file: CONFIG_FILE,
          preferences: loadConfig(),
          default_server: {
            resolved: resolved.url,
            source: resolved.source,
          },
          registrations_file: REGISTRATIONS_FILE,
          key: {
            private_key_file: PRIVATE_KEY_FILE,
            public_key_file: PUBLIC_KEY_FILE,
            exists: keyPairExists(),
          },
          registrations: opts.server ? (regForServer ? [regForServer] : []) : loadRegistrations(),
        };
        output(data, outputOpts);
      } catch (err) {
        output({ error: err instanceof Error ? err.message : String(err) }, outputOpts);
        process.exit(1);
      }
    });
}
