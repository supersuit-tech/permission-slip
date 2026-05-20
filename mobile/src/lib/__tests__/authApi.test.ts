jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn(() => Promise.resolve(null)),
  setItemAsync: jest.fn(() => Promise.resolve()),
  deleteItemAsync: jest.fn(() => Promise.resolve()),
}));

jest.mock("../customHostConfig", () => ({
  getCustomHost: () => "https://stub.example/api",
  getGatewaySecret: () => null,
  PLACEHOLDER_API_BASE: "https://__permission_slip_no_host__.invalid/api",
}));

import { postAuth } from "../authApi";

describe("postAuth", () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    jest.useRealTimers();
  });

  it("returns a structured network_unreachable error when fetch rejects with a network failure", async () => {
    global.fetch = jest.fn(() => Promise.reject(new TypeError("Network request failed"))) as unknown as typeof fetch;

    const { data, error } = await postAuth("refresh", { refresh_token: "abc" });

    expect(data).toBeNull();
    expect(error).not.toBeNull();
    expect(error?.code).toBe("network_unreachable");
    expect(error?.status).toBe(0);
    expect(error?.message).toMatch(/unable to reach/i);
  });

  it("aborts the fetch after the timeout and returns network_unreachable", async () => {
    jest.useFakeTimers();

    const fetchMock = jest.fn(
      (_input: RequestInfo | URL, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener("abort", () => {
            const err = new Error("Aborted");
            err.name = "AbortError";
            reject(err);
          });
        }),
    );
    global.fetch = fetchMock as unknown as typeof fetch;

    const promise = postAuth("refresh", { refresh_token: "abc" });
    // Run all timers (including the 8s timeout) so the fetch is aborted.
    await jest.runAllTimersAsync();
    const { data, error } = await promise;

    expect(data).toBeNull();
    expect(error?.code).toBe("network_unreachable");
    expect(error?.message).toMatch(/server did not respond|unable to reach/i);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("returns success when fetch resolves with a valid token payload", async () => {
    const body = {
      access_token: "access-1",
      refresh_token: "refresh-1",
      expires_at: "2030-01-01T00:00:00.000Z",
    };
    global.fetch = jest.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    ) as unknown as typeof fetch;

    const { data, error } = await postAuth("login", {
      email: "a@b.co",
      password: "pw",
    });

    expect(error).toBeNull();
    expect(data?.access_token).toBe("access-1");
    expect(data?.refresh_token).toBe("refresh-1");
  });
});
