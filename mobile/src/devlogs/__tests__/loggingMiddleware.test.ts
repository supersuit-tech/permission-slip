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
  };
});

import { setDevModeEnabled } from "../../lib/devModeConfig";
import { loggingMiddleware } from "../loggingMiddleware";
import {
  __getDevLogsForTests,
  __resetDevLogsForTests,
} from "../devLogsStore";

async function runRequest(
  request: Request,
  response: Response,
  id = "req-1",
): Promise<void> {
  const ctx = {
    request,
    schemaPath: "",
    params: {},
    id,
    options: {} as never,
  } as never;
  await loggingMiddleware.onRequest!(ctx);
  await loggingMiddleware.onResponse!({ ...(ctx as object), response } as never);
}

async function runRequestError(
  request: Request,
  error: unknown,
  id = "req-err",
): Promise<void> {
  const ctx = {
    request,
    schemaPath: "",
    params: {},
    id,
    options: {} as never,
  } as never;
  await loggingMiddleware.onRequest!(ctx);
  await loggingMiddleware.onError!({ ...(ctx as object), error } as never);
}

describe("loggingMiddleware", () => {
  beforeEach(async () => {
    __resetDevLogsForTests();
    await setDevModeEnabled(false);
  });

  it("captures nothing when dev mode is off", async () => {
    await runRequest(
      new Request("https://x/api/v1/profile", { method: "GET" }),
      new Response("{}", { status: 200 }),
    );
    expect(__getDevLogsForTests()).toHaveLength(0);
  });

  it("captures method, url, status, and body when dev mode is on", async () => {
    await setDevModeEnabled(true);
    await runRequest(
      new Request("https://my-host/api/v1/profile/notification-preferences", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ preferences: [] }),
      }),
      new Response(
        JSON.stringify({ error: { code: "invalid_request", message: "x" } }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    );
    const entries = __getDevLogsForTests();
    expect(entries).toHaveLength(1);
    const entry = entries[0]!;
    expect(entry.method).toBe("PUT");
    expect(entry.url).toBe(
      "https://my-host/api/v1/profile/notification-preferences",
    );
    expect(entry.status).toBe(400);
    expect(entry.isError).toBe(true);
    expect(entry.body).toContain("invalid_request");
  });

  it("does not drain the body for the downstream caller", async () => {
    await setDevModeEnabled(true);
    const original = new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
    await runRequest(
      new Request("https://x/api/v1/profile", { method: "GET" }),
      original,
    );
    // Downstream caller must still be able to read the body.
    await expect(original.text()).resolves.toBe(JSON.stringify({ ok: true }));
  });

  it("captures transport errors via onError", async () => {
    await setDevModeEnabled(true);
    await runRequestError(
      new Request("https://x/api/v1/profile", { method: "GET" }),
      new Error("Network request failed"),
    );
    const entries = __getDevLogsForTests();
    expect(entries).toHaveLength(1);
    expect(entries[0]!.status).toBeNull();
    expect(entries[0]!.isError).toBe(true);
    expect(entries[0]!.body).toBe("Network request failed");
  });
});
