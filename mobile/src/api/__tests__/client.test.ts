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

import { customHostMiddleware } from "../client";
import { setCustomHostConfig, clearCustomHostConfig } from "../../lib/customHostConfig";

const PLACEHOLDER = "https://__permission_slip_no_host__.invalid/api";

async function runMiddleware(req: Request): Promise<Request> {
  // openapi-fetch's onRequest signature: ({ request, schemaPath, params, id, options }) => Request | undefined
  // We only need `request` for this middleware. The other fields are unused.
  const result = await customHostMiddleware.onRequest!({
    request: req,
    schemaPath: "",
    params: {},
    id: "",
    options: {} as never,
  } as never);
  return (result as Request) ?? req;
}

describe("customHostMiddleware", () => {
  afterEach(async () => {
    await clearCustomHostConfig();
  });

  it("preserves the request body when rewriting the URL", async () => {
    // Regression test: passing a Request as the second arg of `new Request(url, init)`
    // does NOT copy the body in React Native's whatwg-fetch polyfill. Self-hosted
    // POST/PUT requests were arriving at the server with empty bodies, causing
    // 400s on push-subscriptions and notification-preferences toggles.
    await setCustomHostConfig("https://my-host.example.com/api");

    const payload = JSON.stringify({
      type: "expo",
      expo_token: "ExponentPushToken[abc123]",
    });
    const original = new Request(`${PLACEHOLDER}/v1/push-subscriptions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: payload,
    });

    const modified = await runMiddleware(original);

    expect(modified.url).toBe("https://my-host.example.com/api/v1/push-subscriptions");
    expect(modified.method).toBe("POST");
    expect(await modified.text()).toBe(payload);
  });

  it("preserves headers and method when rewriting", async () => {
    await setCustomHostConfig("https://my-host.example.com/api");

    const original = new Request(`${PLACEHOLDER}/v1/profile/notification-preferences`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer abc",
      },
      body: JSON.stringify({ preferences: [{ channel: "mobile-push", enabled: false }] }),
    });

    const modified = await runMiddleware(original);

    expect(modified.method).toBe("PUT");
    expect(modified.headers.get("Authorization")).toBe("Bearer abc");
    expect(modified.headers.get("Content-Type")).toBe("application/json");
  });

  it("does not touch the request when no custom host is set", async () => {
    const original = new Request(`${PLACEHOLDER}/v1/profile`, {
      method: "GET",
      headers: { Authorization: "Bearer abc" },
    });

    const modified = await runMiddleware(original);

    expect(modified.url).toBe(`${PLACEHOLDER}/v1/profile`);
  });

  it("does not attempt to read a body from GET requests", async () => {
    await setCustomHostConfig("https://my-host.example.com/api");

    const original = new Request(`${PLACEHOLDER}/v1/profile`, {
      method: "GET",
      headers: { Authorization: "Bearer abc" },
    });

    const modified = await runMiddleware(original);

    expect(modified.url).toBe("https://my-host.example.com/api/v1/profile");
    expect(modified.method).toBe("GET");
  });
});
