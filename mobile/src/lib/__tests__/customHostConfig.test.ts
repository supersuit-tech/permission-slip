jest.mock("expo-secure-store", () => {
  const store = new Map<string, string>();
  return {
    getItemAsync: jest.fn((key: string) =>
      Promise.resolve(store.get(key) ?? null),
    ),
    setItemAsync: jest.fn((key: string, value: string) => {
      store.set(key, value);
      return Promise.resolve();
    }),
    deleteItemAsync: jest.fn((key: string) => {
      store.delete(key);
      return Promise.resolve();
    }),
    __store: store,
  };
});

import * as SecureStore from "expo-secure-store";
import {
  getCustomHost,
  loadCustomHostConfig,
  normalizeApiBase,
  setCustomHostConfig,
} from "../customHostConfig";

const store = (SecureStore as unknown as { __store: Map<string, string> }).__store;

describe("normalizeApiBase", () => {
  it("appends /api when the URL is just the origin", () => {
    expect(normalizeApiBase("https://permission-slip.example.com")).toBe(
      "https://permission-slip.example.com/api",
    );
  });

  it("strips a trailing slash before appending /api", () => {
    expect(normalizeApiBase("https://permission-slip.example.com/")).toBe(
      "https://permission-slip.example.com/api",
    );
  });

  it("leaves a URL that already ends in /api alone", () => {
    expect(normalizeApiBase("https://permission-slip.example.com/api")).toBe(
      "https://permission-slip.example.com/api",
    );
  });

  it("strips a trailing slash after /api", () => {
    expect(normalizeApiBase("https://permission-slip.example.com/api/")).toBe(
      "https://permission-slip.example.com/api",
    );
  });

  it("strips a trailing /v1 segment", () => {
    expect(normalizeApiBase("https://permission-slip.example.com/api/v1")).toBe(
      "https://permission-slip.example.com/api",
    );
  });

  it("preserves a custom path and still ensures /api is appended", () => {
    expect(normalizeApiBase("https://example.com/mounted-at")).toBe(
      "https://example.com/mounted-at/api",
    );
  });

  it("returns null for empty, whitespace, or null input", () => {
    expect(normalizeApiBase(null)).toBeNull();
    expect(normalizeApiBase(undefined)).toBeNull();
    expect(normalizeApiBase("")).toBeNull();
    expect(normalizeApiBase("   ")).toBeNull();
  });

  it("is idempotent", () => {
    const once = normalizeApiBase("https://permission-slip.example.com");
    const twice = normalizeApiBase(once);
    expect(twice).toBe(once);
  });
});

describe("setCustomHostConfig", () => {
  beforeEach(() => {
    store.clear();
    jest.clearAllMocks();
  });

  it("normalizes the URL before persisting", async () => {
    await setCustomHostConfig("https://permission-slip.example.com");
    expect(store.get("custom_host_url")).toBe(
      "https://permission-slip.example.com/api",
    );
    expect(getCustomHost()).toBe("https://permission-slip.example.com/api");
  });

  it("deletes the stored host when given an empty value", async () => {
    await setCustomHostConfig("https://permission-slip.example.com");
    await setCustomHostConfig("");
    expect(store.has("custom_host_url")).toBe(false);
    expect(getCustomHost()).toBeNull();
  });
});

describe("loadCustomHostConfig", () => {
  beforeEach(() => {
    store.clear();
    jest.clearAllMocks();
  });

  it("normalizes legacy un-normalized values on read", async () => {
    // Simulate a value saved by an older app version (no /api suffix).
    store.set("custom_host_url", "https://permission-slip.example.com");
    await loadCustomHostConfig();
    expect(getCustomHost()).toBe("https://permission-slip.example.com/api");
  });
});
