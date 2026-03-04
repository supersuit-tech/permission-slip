import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useRegisterPushToken, unregisterPushTokenDirect } from "../useRegisterPushToken";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import { mockSession, createQueryClient, waitFor } from "../../__test-utils__";

// --- Mocks ---

const mockPost = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { POST: (...args: unknown[]) => mockPost(...args) },
}));

const authMocks = {
  authChangeCallback: null as
    | ((event: AuthChangeEvent, session: Session | null) => void)
    | null,
};

jest.mock("../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      onAuthStateChange: jest.fn(
        (cb: (event: AuthChangeEvent, session: Session | null) => void) => {
          authMocks.authChangeCallback = cb;
          Promise.resolve().then(() =>
            cb("INITIAL_SESSION" as AuthChangeEvent, null),
          );
          return { data: { subscription: { unsubscribe: jest.fn() } } };
        },
      ),
      signInWithOtp: jest.fn(),
      verifyOtp: jest.fn(),
      signOut: jest.fn(),
      mfa: {
        getAuthenticatorAssuranceLevel: jest.fn().mockResolvedValue({
          data: { currentLevel: "aal1", nextLevel: "aal1" },
          error: null,
        }),
      },
    },
  },
}));

// --- Helpers ---

interface HookCapture {
  registerToken: ((token: string) => Promise<unknown>) | null;
  unregisterToken: ((token: string) => Promise<unknown>) | null;
  isRegistering: boolean;
  isUnregistering: boolean;
  registerError: string | null;
  unregisterError: string | null;
}

function createHookCapture() {
  const capture: HookCapture = {
    registerToken: null,
    unregisterToken: null,
    isRegistering: false,
    isUnregistering: false,
    registerError: null,
    unregisterError: null,
  };

  function Consumer() {
    const result = useRegisterPushToken();
    capture.registerToken = result.registerToken;
    capture.unregisterToken = result.unregisterToken;
    capture.isRegistering = result.isRegistering;
    capture.isUnregistering = result.isUnregistering;
    capture.registerError = result.registerError;
    capture.unregisterError = result.unregisterError;
    return null;
  }

  return { capture, Consumer };
}

function renderWithProviders(Consumer: React.ComponentType, qc: QueryClient) {
  return create(
    createElement(
      QueryClientProvider,
      { client: qc },
      createElement(AuthProvider, null, createElement(Consumer)),
    ),
  );
}

// --- Tests ---

let currentRenderer: ReactTestRenderer | null = null;
let currentQueryClient: QueryClient | null = null;

describe("useRegisterPushToken", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    if (currentQueryClient) {
      currentQueryClient.cancelQueries();
      currentQueryClient.clear();
      currentQueryClient = null;
    }
    if (currentRenderer) {
      await act(async () => {
        currentRenderer!.unmount();
      });
      currentRenderer = null;
    }
  });

  it("registers an Expo push token via POST /v1/push-subscriptions", async () => {
    mockPost.mockResolvedValue({
      data: {
        id: 1,
        channel: "mobile-push",
        expo_token: "ExponentPushToken[abc123]",
        created_at: "2026-03-04T00:00:00Z",
      },
      error: undefined,
    });

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    // Authenticate
    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.registerToken !== null);

    let result: unknown;
    await act(async () => {
      result = await capture.registerToken!("ExponentPushToken[abc123]");
    });

    expect(mockPost).toHaveBeenCalledWith("/v1/push-subscriptions", {
      headers: { Authorization: expect.stringContaining("Bearer ") },
      body: { type: "expo", expo_token: "ExponentPushToken[abc123]" },
    });
    expect(result).toEqual(
      expect.objectContaining({ expo_token: "ExponentPushToken[abc123]" }),
    );
  });

  it("unregisters an Expo push token via POST /v1/push-subscriptions/unregister", async () => {
    mockPost.mockResolvedValue({
      data: { message: "Push token unregistered" },
      error: undefined,
    });

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.unregisterToken !== null);

    await act(async () => {
      await capture.unregisterToken!("ExponentPushToken[abc123]");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/push-subscriptions/unregister",
      {
        headers: { Authorization: expect.stringContaining("Bearer ") },
        body: { expo_token: "ExponentPushToken[abc123]" },
      },
    );
  });

  it("throws when not authenticated (register)", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.registerToken !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.registerToken!("ExponentPushToken[abc123]");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Not authenticated");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("throws when not authenticated (unregister)", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.unregisterToken !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.unregisterToken!("ExponentPushToken[abc123]");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Not authenticated");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("throws with API error message on register failure", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "invalid_token", message: "Invalid push token format" } },
    });

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.registerToken !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.registerToken!("bad-token");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Invalid push token format");
  });

  it("throws with API error message on unregister failure", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "not_found", message: "Token not found" } },
    });

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.unregisterToken !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.unregisterToken!("ExponentPushToken[abc123]");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Token not found");
  });
});

describe("unregisterPushTokenDirect", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("calls POST /v1/push-subscriptions/unregister with the provided access token", async () => {
    mockPost.mockResolvedValue({ data: {}, error: undefined });

    await unregisterPushTokenDirect("ExponentPushToken[abc]", "my-access-token");

    expect(mockPost).toHaveBeenCalledWith("/v1/push-subscriptions/unregister", {
      headers: { Authorization: "Bearer my-access-token" },
      body: { expo_token: "ExponentPushToken[abc]" },
    });
  });

  it("does not throw on API error", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "not_found", message: "Token not found" } },
    });

    // Should not throw
    await expect(
      unregisterPushTokenDirect("ExponentPushToken[abc]", "tok"),
    ).resolves.toBeUndefined();
  });

  it("does not throw on network error", async () => {
    mockPost.mockRejectedValue(new Error("Network error"));

    // Should not throw
    await expect(
      unregisterPushTokenDirect("ExponentPushToken[abc]", "tok"),
    ).resolves.toBeUndefined();
  });
});
