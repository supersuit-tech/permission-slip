import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useNotificationPreferences } from "../useNotificationPreferences";
import { mockSession, createQueryClient, waitFor } from "../../__test-utils__";

jest.mock("../../lib/authStorage");
jest.mock("../../lib/authApi");

import * as authStorage from "../../lib/authStorage";
import * as authApi from "../../lib/authApi";

const mockAuthStorage = jest.mocked(authStorage);
const mockAuthApi = jest.mocked(authApi);

// --- Mocks ---

const mockGet = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockGet(...args) },
}));

function bootstrapAuthenticated(session: ReturnType<typeof mockSession>) {
  mockAuthStorage.getStoredRefreshToken.mockResolvedValue("mock-refresh");
  mockAuthApi.postAuth.mockImplementation(async (path: string) => {
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
  preferences: Array<{ channel: string; enabled: boolean; available: boolean }>;
  isLoading: boolean;
  error: string | null;
}

function createHookCapture() {
  const capture: HookCapture = {
    preferences: [],
    isLoading: true,
    error: null,
  };

  function Consumer() {
    const result = useNotificationPreferences();
    capture.preferences = result.preferences;
    capture.isLoading = result.isLoading;
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

describe("useNotificationPreferences", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthStorage.getStoredRefreshToken.mockResolvedValue(null);
    mockAuthApi.postAuth.mockResolvedValue({ data: null, error: null });
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

  it("returns empty preferences when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.isLoading === false);

    expect(capture.preferences).toEqual([]);
    expect(capture.error).toBeNull();
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("fetches preferences when authenticated", async () => {
    const mockPrefs = [
      { channel: "email", enabled: true, available: true },
      { channel: "mobile-push", enabled: true, available: true },
    ];
    mockGet.mockResolvedValue({
      data: { preferences: mockPrefs },
      error: undefined,
    });

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.preferences.length > 0);

    expect(mockGet).toHaveBeenCalledWith(
      "/v1/profile/notification-preferences",
      {
        headers: { Authorization: expect.stringContaining("Bearer ") },
      },
    );
    expect(capture.preferences).toHaveLength(2);
    expect(capture.preferences[0]?.channel).toBe("email");
    expect(capture.preferences[1]?.channel).toBe("mobile-push");
    expect(capture.error).toBeNull();
  });

  it("returns error message on API failure", async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: {
        error: { code: "internal_error", message: "Something went wrong" },
      },
    });

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.error !== null);

    expect(capture.error).toBe("Something went wrong");
    expect(capture.preferences).toEqual([]);
  });
});
