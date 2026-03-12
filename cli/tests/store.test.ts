/**
 * Tests for the config store module.
 *
 * Uses PS_CLI_TEST_CONFIG_DIR (set by global setup) to isolate from ~/.permission-slip.
 */

import {
  loadRegistrations,
  saveRegistration,
  findRegistration,
  CONFIG_DIR,
  REGISTRATIONS_FILE,
} from "../src/config/store.js";

describe("CONFIG_DIR / REGISTRATIONS_FILE", () => {
  it("uses the test temp dir (not ~/.permission-slip)", () => {
    expect(CONFIG_DIR).toContain("ps-cli-test-");
    expect(REGISTRATIONS_FILE).toContain("ps-cli-test-");
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
      api_base: "https://example.permissionslip.dev/api/v1",
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
      api_base: "https://example.permissionslip.dev/api/v1",
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
      api_base: "https://other.permissionslip.dev/api/v1",
      agent_id: 100,
      registered_at: "2026-01-01T00:00:00Z",
    });
    const afterCount = loadRegistrations().length;
    expect(afterCount).toBe(beforeCount + 1);
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
