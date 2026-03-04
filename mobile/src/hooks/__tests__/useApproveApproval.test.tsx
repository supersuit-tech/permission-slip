import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useApproveApproval } from "../useApproveApproval";
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
  approveApproval: ((id: string) => Promise<unknown>) | null;
  isPending: boolean;
  error: Error | null;
}

function createHookCapture() {
  const capture: HookCapture = { approveApproval: null, isPending: false, error: null };

  function Consumer() {
    const result = useApproveApproval();
    capture.approveApproval = result.approveApproval;
    capture.isPending = result.isPending;
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

describe("useApproveApproval", () => {
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

  it("calls POST /v1/approvals/{approval_id}/approve with correct params", async () => {
    mockPost.mockResolvedValue({
      data: {
        approval_id: "appr_abc123",
        status: "approved",
        approved_at: "2026-03-02T13:25:00Z",
        confirmation_code: "RK3-P7M",
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

    await waitFor(() => capture.approveApproval !== null);

    let result: unknown;
    await act(async () => {
      result = await capture.approveApproval!("appr_abc123");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/approve",
      {
        headers: { Authorization: expect.stringContaining("Bearer ") },
        params: { path: { approval_id: "appr_abc123" } },
      },
    );
    expect(result).toEqual(
      expect.objectContaining({ confirmation_code: "RK3-P7M" }),
    );
  });

  it("throws on API error", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "approval_expired", message: "Approval has expired" } },
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

    await waitFor(() => capture.approveApproval !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.approveApproval!("appr_expired");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Approval has expired");
  });

  it("throws when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.approveApproval !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.approveApproval!("appr_abc123");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Not authenticated");
    expect(mockPost).not.toHaveBeenCalled();
  });
});
