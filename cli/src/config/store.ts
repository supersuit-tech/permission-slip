/**
 * Manages ~/.permission-slip/ — config directory and registrations file.
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
  const existing = loadRegistrations();
  // Upsert: replace existing entry for same server+agent_id if present
  const idx = existing.findIndex(
    (r) => r.server === reg.server && r.agent_id === reg.agent_id,
  );
  if (idx >= 0) {
    existing[idx] = reg;
  } else {
    existing.push(reg);
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
  const regs = loadRegistrations();
  const matching = regs.filter((r) => r.server === server);
  if (matching.length === 0) return undefined;
  return matching.sort(
    (a, b) =>
      new Date(b.registered_at).getTime() - new Date(a.registered_at).getTime(),
  )[0];
}
