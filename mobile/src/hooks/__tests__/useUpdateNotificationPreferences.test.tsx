import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useUpdateNotificationPreferences } from "../useUpdateNotificationPreferences";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import { mockSession, createQueryClient, waitFor } from "../../__test-utils__";

// --- Mocks ---

const mockPut = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { PUT: (...args: unknown[]) => mockPut(...args) },
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
  updatePreferences: (
    prefs: Array<{ channel: string; enabled: boolean }>,
  ) => Promise<unknown>;
  isUpdating: boolean;
  error: Error | null;
}

function createHookCapture() {
  const capture: HookCapture = {
    updatePreferences: async () => {},
    isUpdating: false,
    error: null,
  };

  function Consumer() {
    const result = useUpdateNotificationPreferences();
    capture.updatePreferences = result.updatePreferences as HookCapture["updatePreferences"];
    capture.isUpdating = result.isUpdating;
    capture.error = result.error;
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

describe("useUpdateNotificationPreferences", () => {
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

  it("calls PUT with correct payload when authenticated", async () => {
    mockPut.mockResolvedValue({
      data: {
        preferences: [
          { channel: "mobile-push", enabled: false, available: true },
        ],
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

    // Call updatePreferences
    await act(async () => {
      await capture.updatePreferences([
        { channel: "mobile-push", enabled: false },
      ]);
    });

    expect(mockPut).toHaveBeenCalledWith(
      "/v1/profile/notification-preferences",
      {
        headers: { Authorization: expect.stringContaining("Bearer ") },
        body: {
          preferences: [{ channel: "mobile-push", enabled: false }],
        },
      },
    );
  });

  it("throws error when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.isUpdating === false);

    let thrownError: Error | null = null;
    await act(async () => {
      try {
        await capture.updatePreferences([
          { channel: "mobile-push", enabled: false },
        ]);
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeTruthy();
    expect(thrownError!.message).toBe("Not authenticated");
    expect(mockPut).not.toHaveBeenCalled();
  });

  it("throws error on API failure", async () => {
    mockPut.mockResolvedValue({
      data: undefined,
      error: {
        error: { code: "internal_error", message: "Update failed" },
      },
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

    let thrownError: Error | null = null;
    await act(async () => {
      try {
        await capture.updatePreferences([
          { channel: "mobile-push", enabled: false },
        ]);
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeTruthy();
    expect(thrownError!.message).toBe("Update failed");
  });
});
