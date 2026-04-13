/**
 * Tests for the config store module.
 *
 * Uses PS_CLI_TEST_CONFIG_DIR (set by global setup) to isolate from ~/.permission-slip.
 */

import {
  loadRegistrations,
  saveRegistration,
  findRegistration,
  loadConfig,
  saveConfig,
  unsetConfigKey,
  CONFIG_DIR,
  REGISTRATIONS_FILE,
  CONFIG_FILE,
} from "../src/config/store.js";
import fs from "node:fs";

describe("CONFIG_DIR / REGISTRATIONS_FILE", () => {
  it("uses the test temp dir (not ~/.permission-slip)", () => {
    expect(CONFIG_DIR).toContain("ps-cli-test-");
    expect(REGISTRATIONS_FILE).toContain("ps-cli-test-");
    expect(CONFIG_FILE).toContain("ps-cli-test-");
  });
});

describe("loadRegistrations", () => {
  it("returns empty array when file does not exist", () => {
    const regs = loadRegistrations();
    expect(Array.isArray(regs)).toBe(true);
  });
});

describe("saveRegistration / loadRegistrations", () => {
  it("saves and loads a registration", () => {
    saveRegistration({
      server: "https://example.permissionslip.dev",
      agent_id: 99,
      registered_at: "2026-01-01T00:00:00Z",
    });

    const regs = loadRegistrations();
    const found = regs.find((r) => r.server === "https://example.permissionslip.dev");
    expect(found).toBeDefined();
    expect(found?.agent_id).toBe(99);
  });

  it("upserts when same server+agent_id exists", () => {
    saveRegistration({
      server: "https://example.permissionslip.dev",
      agent_id: 99,
      registered_at: "2026-02-01T00:00:00Z",
    });

    const regs = loadRegistrations();
    const matching = regs.filter(
      (r) => r.server === "https://example.permissionslip.dev" && r.agent_id === 99,
    );
    expect(matching).toHaveLength(1);
    expect(matching[0]?.registered_at).toBe("2026-02-01T00:00:00Z");
  });

  it("appends a new entry for a different server", () => {
    const beforeCount = loadRegistrations().length;
    saveRegistration({
      server: "https://other.permissionslip.dev",
      agent_id: 100,
      registered_at: "2026-01-01T00:00:00Z",
    });
    const afterCount = loadRegistrations().length;
    expect(afterCount).toBe(beforeCount + 1);
  });
});

describe("loadConfig / saveConfig / unsetConfigKey", () => {
  afterEach(() => {
    unsetConfigKey("default_server");
    if (fs.existsSync(CONFIG_FILE)) {
      fs.unlinkSync(CONFIG_FILE);
    }
  });

  it("returns empty object when config file is missing", () => {
    expect(loadConfig()).toEqual({});
  });

  it("saves and loads default_server", () => {
    saveConfig({ default_server: "https://cfg.example.dev" });
    expect(loadConfig().default_server).toBe("https://cfg.example.dev");
  });

  it("normalizes trailing slashes on default_server", () => {
    saveConfig({ default_server: "https://cfg.example.dev/" });
    expect(loadConfig().default_server).toBe("https://cfg.example.dev");
  });

  it("rejects invalid default_server URLs", () => {
    expect(() => saveConfig({ default_server: "not-a-url" })).toThrow(/Invalid server URL/);
  });

  it("unset removes key and deletes file when empty", () => {
    saveConfig({ default_server: "https://cfg.example.dev" });
    unsetConfigKey("default_server");
    expect(loadConfig()).toEqual({});
    expect(fs.existsSync(CONFIG_FILE)).toBe(false);
  });
});

describe("findRegistration", () => {
  it("returns the registration for a given server", () => {
    const reg = findRegistration("https://example.permissionslip.dev");
    expect(reg).toBeDefined();
    expect(reg?.agent_id).toBe(99);
  });

  it("returns undefined for an unknown server", () => {
    const reg = findRegistration("https://unknown.permissionslip.dev");
    expect(reg).toBeUndefined();
  });
});

describe("URL normalization", () => {
  it("strips trailing slashes on save and finds by URL without slash", () => {
    saveRegistration({
      server: "https://slash.permissionslip.dev/",
      agent_id: 200,
      registered_at: "2026-01-01T00:00:00Z",
    });
    const reg = findRegistration("https://slash.permissionslip.dev");
    expect(reg).toBeDefined();
    expect(reg?.agent_id).toBe(200);
  });

  it("finds by URL with trailing slash when saved without", () => {
    saveRegistration({
      server: "https://noslash.permissionslip.dev",
      agent_id: 201,
      registered_at: "2026-01-01T00:00:00Z",
    });
    const reg = findRegistration("https://noslash.permissionslip.dev/");
    expect(reg).toBeDefined();
    expect(reg?.agent_id).toBe(201);
  });

  it("deduplicates registrations saved with and without trailing slash", () => {
    saveRegistration({
      server: "https://dedup.permissionslip.dev/",
      agent_id: 202,
      registered_at: "2026-01-01T00:00:00Z",
    });
    saveRegistration({
      server: "https://dedup.permissionslip.dev",
      agent_id: 202,
      registered_at: "2026-02-01T00:00:00Z",
    });
    const regs = loadRegistrations().filter((r) => r.server === "https://dedup.permissionslip.dev");
    expect(regs).toHaveLength(1);
    expect(regs[0]?.registered_at).toBe("2026-02-01T00:00:00Z");
  });
});
