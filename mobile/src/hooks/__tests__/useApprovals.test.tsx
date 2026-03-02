import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useApprovals, type ApprovalSummary } from "../useApprovals";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";

// --- Mocks ---

const mockGet = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockGet(...args) },
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

function mockSession(): Session {
  const payload = btoa(JSON.stringify({ aal: "aal1" }));
  return {
    access_token: `header.${payload}.signature`,
    refresh_token: "mock-refresh",
    expires_in: 3600,
    expires_at: Date.now() / 1000 + 3600,
    token_type: "bearer",
    user: {
      id: "user-1",
      email: "test@example.com",
      app_metadata: {},
      user_metadata: {},
      aud: "authenticated",
      created_at: new Date().toISOString(),
      factors: [],
    },
  } as Session;
}

const mockApproval: ApprovalSummary = {
  approval_id: "appr_abc123",
  agent_id: 42,
  action: {
    type: "email.send",
    version: "1",
    parameters: { to: ["user@example.com"], subject: "Hello" },
  },
  context: {
    description: "Send welcome email to new user",
    risk_level: "low",
  },
  status: "pending",
  expires_at: "2026-03-02T13:25:00Z",
  created_at: "2026-03-02T13:20:00Z",
};

interface HookCapture {
  approvals: ApprovalSummary[];
  isLoading: boolean;
  error: string | null;
}

function createHookCapture(status: "pending" | "approved" | "denied" = "pending") {
  const capture: HookCapture = { approvals: [], isLoading: true, error: null };

  function Consumer() {
    const result = useApprovals(status);
    capture.approvals = result.approvals;
    capture.isLoading = result.isLoading;
    capture.error = result.error;
    return null;
  }

  return { capture, Consumer };
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

function renderWithProviders(Consumer: React.ComponentType) {
  const queryClient = createQueryClient();
  return create(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(AuthProvider, null, createElement(Consumer)),
    ),
  );
}

/**
 * Wait for condition to become true, flushing React updates each iteration.
 * Avoids infinite timer loops by using real timers with small real delays.
 */
async function waitFor(
  predicate: () => boolean,
  { timeout = 3000, interval = 10 } = {},
) {
  const start = Date.now();
  while (!predicate()) {
    if (Date.now() - start > timeout) {
      throw new Error("waitFor timed out");
    }
    await act(async () => {
      await new Promise((r) => setTimeout(r, interval));
    });
  }
}

// --- Tests ---

let currentRenderer: ReactTestRenderer | null = null;

describe("useApprovals", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    if (currentRenderer) {
      await act(async () => {
        currentRenderer!.unmount();
      });
      currentRenderer = null;
    }
  });

  it("returns empty list and loading false when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer);
    });

    await waitFor(() => capture.isLoading === false);

    expect(capture.approvals).toEqual([]);
    expect(capture.error).toBeNull();
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("fetches pending approvals when authenticated", async () => {
    mockGet.mockResolvedValue({
      data: { data: [mockApproval], has_more: false },
      error: undefined,
    });

    const { capture, Consumer } = createHookCapture("pending");

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer);
    });

    await waitFor(() => capture.isLoading === false);

    // Authenticate
    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.approvals.length > 0);

    expect(mockGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: { Authorization: expect.stringContaining("Bearer ") },
      params: { query: { status: "pending" } },
    });
    expect(capture.approvals).toHaveLength(1);
    expect(capture.approvals[0]?.approval_id).toBe("appr_abc123");
    expect(capture.error).toBeNull();
  });

  it("returns error message on API failure", async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { capture, Consumer } = createHookCapture("pending");

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer);
    });

    await waitFor(() => capture.isLoading === false);

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.error !== null);

    expect(capture.error).toBe("Server error");
    expect(capture.approvals).toEqual([]);
  });

  it("passes the correct status filter to the API", async () => {
    mockGet.mockResolvedValue({
      data: { data: [], has_more: false },
      error: undefined,
    });

    const { Consumer } = createHookCapture("denied");

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer);
    });

    await waitFor(() => mockGet.mock.calls.length === 0);

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => mockGet.mock.calls.length > 0);

    expect(mockGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: expect.any(Object),
      params: { query: { status: "denied" } },
    });
  });
});
