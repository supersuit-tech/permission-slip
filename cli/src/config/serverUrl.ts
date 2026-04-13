/**
 * Resolves the Permission Slip server URL from flag, env, config file, or built-in default.
 * Precedence: --server > PS_SERVER > config.json default_server > built-in.
 */

import { loadConfig } from "./store.js";

export const BUILT_IN_DEFAULT_SERVER = "https://app.permissionslip.dev";

export type ServerUrlSource = "flag" | "env" | "config" | "built-in";

export interface ResolvedServerUrl {
  url: string;
  source: ServerUrlSource;
}

export function resolveServerUrl(opts: { serverFlag?: string }): ResolvedServerUrl {
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
  return { url: BUILT_IN_DEFAULT_SERVER, source: "built-in" };
}

export function isBuiltInDefaultServerUrl(url: string): boolean {
  return url.replace(/\/+$/, "") === BUILT_IN_DEFAULT_SERVER;
}
