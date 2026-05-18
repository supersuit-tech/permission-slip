/**
 * Resolves the Permission Slip server URL from flag, env, or config file.
 * Precedence: --server > PS_SERVER > config.json default_server.
 * There is no built-in default — self-hosted deployments must supply a host.
 */

import { loadConfig } from "./store.js";

export type ServerUrlSource = "flag" | "env" | "config";

export interface ResolvedServerUrl {
  url: string;
  source: ServerUrlSource;
}

export class ServerUrlRequiredError extends Error {
  constructor() {
    super(
      "Permission Slip server URL is required. Set it using one of:\n" +
        "  --server <url>           (highest precedence)\n" +
        "  PS_SERVER=<url>          (environment variable)\n" +
        "  permission-slip config set default_server <url>  (saved to ~/.permission-slip/config.json)",
    );
    this.name = "ServerUrlRequiredError";
  }
}

/**
 * Returns the resolved server URL, or null when none of flag, env, or
 * config default_server is set.
 */
export function tryResolveServerUrl(opts: { serverFlag?: string }): ResolvedServerUrl | null {
  if (opts.serverFlag !== undefined && opts.serverFlag !== "") {
    return { url: opts.serverFlag, source: "flag" };
  }
  const fromEnv = process.env["PS_SERVER"]?.trim();
  if (fromEnv) {
    return { url: fromEnv, source: "env" };
  }
  const cfg = loadConfig();
  const fromFile = cfg.default_server?.trim();
  if (fromFile) {
    return { url: fromFile, source: "config" };
  }
  return null;
}

/** Same as tryResolveServerUrl but throws ServerUrlRequiredError when unset. */
export function requireServerUrl(opts: { serverFlag?: string }): ResolvedServerUrl {
  const r = tryResolveServerUrl(opts);
  if (!r) {
    throw new ServerUrlRequiredError();
  }
  return r;
}
