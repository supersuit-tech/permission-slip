/**
 * Changelog parsing and "since last use" tracking for agent discoverability.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { loadConfig, saveConfig } from "./config/store.js";

export interface ChangelogEntry {
  version: string;
  date?: string;
  body: string;
}

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const CHANGELOG_PATH = path.join(__dirname, "..", "CHANGELOG.md");

/** Current CLI version from package.json (bundled at build). */
export function currentCliVersion(): string {
  try {
    const pkgPath = path.join(__dirname, "..", "package.json");
    const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf-8")) as { version?: string };
    return pkg.version ?? "0.0.0";
  } catch {
    return "0.0.0";
  }
}

/** Parse CHANGELOG.md into versioned entries (## [x.y.z] headers). */
export function parseChangelog(content: string): ChangelogEntry[] {
  const entries: ChangelogEntry[] = [];
  const sections = content.split(/^## \[/m).slice(1);
  for (const section of sections) {
    const headerEnd = section.indexOf("]");
    if (headerEnd === -1) continue;
    const header = section.slice(0, headerEnd);
    const [version, datePart] = header.split("] - ");
    const body = section.slice(headerEnd + 1).trim();
    entries.push({
      version: version.trim(),
      date: datePart?.trim(),
      body,
    });
  }
  return entries;
}

export function loadChangelogEntries(): ChangelogEntry[] {
  if (!fs.existsSync(CHANGELOG_PATH)) {
    return [];
  }
  return parseChangelog(fs.readFileSync(CHANGELOG_PATH, "utf-8"));
}

/** Semver-ish compare: returns negative if a < b, 0 if equal, positive if a > b. */
export function compareVersions(a: string, b: string): number {
  const pa = a.split(".").map((n) => parseInt(n, 10) || 0);
  const pb = b.split(".").map((n) => parseInt(n, 10) || 0);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const da = pa[i] ?? 0;
    const db = pb[i] ?? 0;
    if (da !== db) return da - db;
  }
  return 0;
}

export function entriesSinceVersion(
  entries: ChangelogEntry[],
  sinceVersion: string | undefined,
): ChangelogEntry[] {
  if (!sinceVersion) {
    return entries;
  }
  return entries.filter((e) => compareVersions(e.version, sinceVersion) > 0);
}

export function getLastChangelogVersion(): string | undefined {
  return loadConfig().last_changelog_version;
}

export function markChangelogRead(version?: string): void {
  saveConfig({ last_changelog_version: version ?? currentCliVersion() });
}

/** Short stderr notice for agents — skipped when empty or PS_CLI_NO_CHANGELOG=1. */
export function printUnreadChangelogNotice(): void {
  if (process.env["PS_CLI_NO_CHANGELOG"] === "1") {
    return;
  }
  const entries = loadChangelogEntries();
  const unread = entriesSinceVersion(entries, getLastChangelogVersion());
  if (unread.length === 0) {
    return;
  }
  const versions = unread.map((e) => e.version).join(", ");
  console.error(
    `[permission-slip] CLI updates since your last session (${versions}). ` +
      "Run: permission-slip changelog",
  );
}

export function readChangelogFile(): string {
  if (!fs.existsSync(CHANGELOG_PATH)) {
    return "";
  }
  return fs.readFileSync(CHANGELOG_PATH, "utf-8");
}
