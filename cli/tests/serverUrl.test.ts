/**
 * Tests for server URL resolution (flag > PS_SERVER > config).
 */

import {
  loadConfig,
  saveConfig,
  unsetConfigKey,
} from "../src/config/store.js";
import {
  tryResolveServerUrl,
  requireServerUrl,
  ServerUrlRequiredError,
} from "../src/config/serverUrl.js";

beforeEach(() => {
  delete process.env["PS_SERVER"];
  unsetConfigKey("default_server");
});

describe("tryResolveServerUrl", () => {
  const prevServer = process.env["PS_SERVER"];

  afterEach(() => {
    if (prevServer === undefined) {
      delete process.env["PS_SERVER"];
    } else {
      process.env["PS_SERVER"] = prevServer;
    }
    unsetConfigKey("default_server");
  });

  it("returns null when nothing is set", () => {
    delete process.env["PS_SERVER"];
    expect(loadConfig().default_server).toBeUndefined();
    expect(tryResolveServerUrl({})).toBeNull();
  });

  it("uses default_server from config when no flag or env", () => {
    delete process.env["PS_SERVER"];
    saveConfig({ default_server: "https://pi.local:8080" });
    const r = tryResolveServerUrl({});
    expect(r?.url).toBe("https://pi.local:8080");
    expect(r?.source).toBe("config");
  });

  it("prefers PS_SERVER over config file", () => {
    saveConfig({ default_server: "https://from-config.dev" });
    process.env["PS_SERVER"] = "https://from-env.dev";
    const r = tryResolveServerUrl({});
    expect(r?.url).toBe("https://from-env.dev");
    expect(r?.source).toBe("env");
  });

  it("prefers --server flag over env and config", () => {
    saveConfig({ default_server: "https://from-config.dev" });
    process.env["PS_SERVER"] = "https://from-env.dev";
    const r = tryResolveServerUrl({ serverFlag: "https://from-flag.dev" });
    expect(r?.url).toBe("https://from-flag.dev");
    expect(r?.source).toBe("flag");
  });

  it("trims PS_SERVER whitespace", () => {
    process.env["PS_SERVER"] = "  https://trimmed.dev  ";
    const r = tryResolveServerUrl({});
    expect(r?.url).toBe("https://trimmed.dev");
    expect(r?.source).toBe("env");
  });
});

describe("requireServerUrl", () => {
  const prevServer = process.env["PS_SERVER"];

  afterEach(() => {
    if (prevServer === undefined) {
      delete process.env["PS_SERVER"];
    } else {
      process.env["PS_SERVER"] = prevServer;
    }
    unsetConfigKey("default_server");
  });

  it("throws ServerUrlRequiredError when unset", () => {
    delete process.env["PS_SERVER"];
    expect(() => requireServerUrl({})).toThrow(ServerUrlRequiredError);
  });

  it("returns resolved URL when config has default_server", () => {
    saveConfig({ default_server: "https://ok.example/api" });
    const r = requireServerUrl({});
    expect(r.url).toBe("https://ok.example/api");
    expect(r.source).toBe("config");
  });
});
