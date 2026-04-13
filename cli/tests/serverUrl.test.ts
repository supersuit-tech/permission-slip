/**
 * Tests for server URL resolution (flag > PS_SERVER > config > built-in).
 */

import {
  loadConfig,
  saveConfig,
  unsetConfigKey,
} from "../src/config/store.js";
import {
  resolveServerUrl,
  BUILT_IN_DEFAULT_SERVER,
  isBuiltInDefaultServerUrl,
} from "../src/config/serverUrl.js";

describe("resolveServerUrl", () => {
  const prevServer = process.env["PS_SERVER"];

  afterEach(() => {
    if (prevServer === undefined) {
      delete process.env["PS_SERVER"];
    } else {
      process.env["PS_SERVER"] = prevServer;
    }
    unsetConfigKey("default_server");
  });

  it("uses built-in default when nothing else is set", () => {
    delete process.env["PS_SERVER"];
    expect(loadConfig().default_server).toBeUndefined();
    const r = resolveServerUrl({});
    expect(r.url).toBe(BUILT_IN_DEFAULT_SERVER);
    expect(r.source).toBe("built-in");
  });

  it("uses default_server from config when no flag or env", () => {
    delete process.env["PS_SERVER"];
    saveConfig({ default_server: "https://pi.local:8080" });
    const r = resolveServerUrl({});
    expect(r.url).toBe("https://pi.local:8080");
    expect(r.source).toBe("config");
  });

  it("prefers PS_SERVER over config file", () => {
    saveConfig({ default_server: "https://from-config.dev" });
    process.env["PS_SERVER"] = "https://from-env.dev";
    const r = resolveServerUrl({});
    expect(r.url).toBe("https://from-env.dev");
    expect(r.source).toBe("env");
  });

  it("prefers --server flag over env and config", () => {
    saveConfig({ default_server: "https://from-config.dev" });
    process.env["PS_SERVER"] = "https://from-env.dev";
    const r = resolveServerUrl({ serverFlag: "https://from-flag.dev" });
    expect(r.url).toBe("https://from-flag.dev");
    expect(r.source).toBe("flag");
  });

  it("trims PS_SERVER whitespace", () => {
    process.env["PS_SERVER"] = "  https://trimmed.dev  ";
    const r = resolveServerUrl({});
    expect(r.url).toBe("https://trimmed.dev");
    expect(r.source).toBe("env");
  });
});

describe("isBuiltInDefaultServerUrl", () => {
  it("matches the canonical host with or without trailing slash", () => {
    expect(isBuiltInDefaultServerUrl("https://app.permissionslip.dev")).toBe(true);
    expect(isBuiltInDefaultServerUrl("https://app.permissionslip.dev/")).toBe(true);
    expect(isBuiltInDefaultServerUrl("https://other.dev")).toBe(false);
  });
});
