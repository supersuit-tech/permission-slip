import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useUpdateNotificationPreferences } from "../useUpdateNotificationPreferences";
import { mockSession, createQueryClient, waitFor } from "../../__test-utils__";

jest.mock("../../lib/authStorage");
jest.mock("../../lib/authApi");

import * as authStorage from "../../lib/authStorage";
import * as authApi from "../../lib/authApi";

// --- Mocks ---

const mockPut = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { PUT: (...args: unknown[]) => mockPut(...args) },
}));

function bootstrapAuthenticated(session: ReturnType<typeof mockSession>) {
  authStorage.getStoredRefreshToken.mockResolvedValue("mock-refresh");
  authApi.postAuth.mockImplementation(async (path: string) => {
    if (path === "refresh") {
      return {
        data: {
          access_token: session.access_token,
          refresh_token: "mock-refresh-2",
          expires_at: session.expires_at,
        },
        error: null,
      };
    }
    return { data: null, error: null };
  });
}

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
    authStorage.getStoredRefreshToken.mockResolvedValue(null);
    authApi.postAuth.mockResolvedValue({ data: null, error: null });
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

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

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

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
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
