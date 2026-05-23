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
  isDevModeEnabled,
  loadDevModeConfig,
  setDevModeEnabled,
  subscribeDevMode,
} from "../devModeConfig";

const mockStore = (SecureStore as unknown as { __store: Map<string, string> })
  .__store;

describe("devModeConfig", () => {
  beforeEach(async () => {
    mockStore.clear();
    await setDevModeEnabled(false);
  });

  it("defaults to disabled when nothing is stored", async () => {
    mockStore.clear();
    await loadDevModeConfig();
    expect(isDevModeEnabled()).toBe(false);
  });

  it("loads the persisted true value on startup", async () => {
    mockStore.set("developer_mode_enabled", "1");
    await loadDevModeConfig();
    expect(isDevModeEnabled()).toBe(true);
  });

  it("persists toggles and notifies subscribers", async () => {
    let calls = 0;
    const unsub = subscribeDevMode(() => {
      calls += 1;
    });

    await setDevModeEnabled(true);
    expect(isDevModeEnabled()).toBe(true);
    expect(mockStore.get("developer_mode_enabled")).toBe("1");

    await setDevModeEnabled(false);
    expect(isDevModeEnabled()).toBe(false);
    expect(mockStore.has("developer_mode_enabled")).toBe(false);

    expect(calls).toBe(2);
    unsub();
  });
});
