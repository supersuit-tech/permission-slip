/**
 * Manages ~/.permission-slip/ — config directory, preferences, and registrations.
 *
 * User preferences:
 *   ~/.permission-slip/config.json
 *
 * Registrations are stored as:
 *   ~/.permission-slip/registrations.json
 *
 * Key files are stored as:
 *   ~/.ssh/permission_slip_agent        (private key, OpenSSH format)
 *   ~/.ssh/permission_slip_agent.pub    (public key)
 */

import fs from "node:fs";
import os from "node:os";
import path from "node:path";

export interface Registration {
  server: string;
  agent_id: number;
  registered_at: string;
}

interface RegistrationsFile {
  registrations: Registration[];
}

// Paths can be overridden via environment variables for testing.
export const CONFIG_DIR =
  process.env["PS_CLI_TEST_CONFIG_DIR"] ??
  path.join(os.homedir(), ".permission-slip");
export const REGISTRATIONS_FILE = path.join(CONFIG_DIR, "registrations.json");
export const CONFIG_FILE = path.join(CONFIG_DIR, "config.json");
export const SSH_DIR =
  process.env["PS_CLI_TEST_SSH_DIR"] ?? path.join(os.homedir(), ".ssh");
export const PRIVATE_KEY_FILE =
  process.env["PS_CLI_TEST_PRIVATE_KEY"] ??
  path.join(SSH_DIR, "permission_slip_agent");
export const PUBLIC_KEY_FILE =
  process.env["PS_CLI_TEST_PUBLIC_KEY"] ??
  path.join(SSH_DIR, "permission_slip_agent.pub");

export function ensureConfigDir(): void {
  if (!fs.existsSync(CONFIG_DIR)) {
    fs.mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
  }
}

export function loadRegistrations(): Registration[] {
  if (!fs.existsSync(REGISTRATIONS_FILE)) {
    return [];
  }
  try {
    const raw = fs.readFileSync(REGISTRATIONS_FILE, "utf-8");
    const data = JSON.parse(raw) as RegistrationsFile;
    return data.registrations ?? [];
  } catch {
    return [];
  }
}

export function saveRegistration(reg: Registration): void {
  ensureConfigDir();
  // Normalize: strip trailing slashes so lookups are consistent regardless of
  // whether the caller passed "https://host" or "https://host/".
  const normalized: Registration = { ...reg, server: reg.server.replace(/\/+$/, "") };
  const existing = loadRegistrations();
  // Upsert: replace existing entry for same server+agent_id if present
  const idx = existing.findIndex(
    (r) => r.server === normalized.server && r.agent_id === normalized.agent_id,
  );
  if (idx >= 0) {
    existing[idx] = normalized;
  } else {
    existing.push(normalized);
  }
  const data: RegistrationsFile = { registrations: existing };
  fs.writeFileSync(REGISTRATIONS_FILE, JSON.stringify(data, null, 2) + "\n", {
    mode: 0o600,
  });
}

/**
 * Find a registration for the given server. If multiple registrations exist
 * for the same server (different agent IDs), returns the most recently
 * registered one.
 */
export function findRegistration(server: string): Registration | undefined {
  const normalizedServer = server.replace(/\/+$/, "");
  const regs = loadRegistrations();
  const matching = regs.filter((r) => r.server === normalizedServer);
  if (matching.length === 0) return undefined;
  return matching.sort(
    (a, b) =>
      new Date(b.registered_at).getTime() - new Date(a.registered_at).getTime(),
  )[0];
}

/** Stored in ~/.permission-slip/config.json — extend with new optional fields as needed. */
export interface CliConfigFile {
  default_server?: string;
}

export function normalizeServerUrl(url: string): string {
  return url.replace(/\/+$/, "");
}

function assertValidServerUrl(url: string): void {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    throw new Error(`Invalid server URL: ${url}`);
  }
  if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
    throw new Error(
      `Server URL must use http or https (got ${parsed.protocol}).`,
    );
  }
}

export function loadConfig(): CliConfigFile {
  if (!fs.existsSync(CONFIG_FILE)) {
    return {};
  }
  try {
    const raw = fs.readFileSync(CONFIG_FILE, "utf-8");
    const data = JSON.parse(raw) as CliConfigFile;
    return typeof data === "object" && data !== null && !Array.isArray(data)
      ? data
      : {};
  } catch {
    return {};
  }
}

export function saveConfig(partial: CliConfigFile): void {
  ensureConfigDir();
  const current = loadConfig();
  const next: CliConfigFile = { ...current, ...partial };
  if (next.default_server !== undefined) {
    next.default_server = normalizeServerUrl(next.default_server);
    assertValidServerUrl(next.default_server);
  }
  fs.writeFileSync(CONFIG_FILE, JSON.stringify(next, null, 2) + "\n", {
    mode: 0o600,
  });
}

export function unsetConfigKey(key: keyof CliConfigFile): void {
  if (!fs.existsSync(CONFIG_FILE)) {
    return;
  }
  const current = loadConfig();
  if (!(key in current)) {
    return;
  }
  const next = { ...current };
  delete next[key];
  if (Object.keys(next).length === 0) {
    fs.unlinkSync(CONFIG_FILE);
    return;
  }
  fs.writeFileSync(CONFIG_FILE, JSON.stringify(next, null, 2) + "\n", {
    mode: 0o600,
  });
}
