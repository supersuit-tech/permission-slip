import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useDenyApproval } from "../useDenyApproval";
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
  denyApproval: ((id: string) => Promise<void>) | null;
  isPending: boolean;
  error: Error | null;
}

function createHookCapture() {
  const capture: HookCapture = { denyApproval: null, isPending: false, error: null };

  function Consumer() {
    const result = useDenyApproval();
    capture.denyApproval = result.denyApproval;
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

describe("useDenyApproval", () => {
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

  it("calls POST /v1/approvals/{approval_id}/deny with correct params", async () => {
    mockPost.mockResolvedValue({
      data: {
        approval_id: "appr_abc123",
        status: "denied",
        denied_at: "2026-03-02T13:25:00Z",
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

    await waitFor(() => capture.denyApproval !== null);

    await act(async () => {
      await capture.denyApproval!("appr_abc123");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/deny",
      {
        headers: { Authorization: expect.stringContaining("Bearer ") },
        params: { path: { approval_id: "appr_abc123" } },
      },
    );
  });

  it("throws on API error", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "approval_already_resolved", message: "Already resolved" } },
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

    await waitFor(() => capture.denyApproval !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.denyApproval!("appr_resolved");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Already resolved");
  });

  it("throws when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => capture.denyApproval !== null);

    let thrownError: Error | undefined;
    await act(async () => {
      try {
        await capture.denyApproval!("appr_abc123");
      } catch (e) {
        thrownError = e as Error;
      }
    });

    expect(thrownError).toBeDefined();
    expect(thrownError!.message).toBe("Not authenticated");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("invalidates approvals queries on success", async () => {
    mockPost.mockResolvedValue({ data: { approval_id: "appr_1", status: "denied" }, error: undefined });

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();
    const invalidateSpy = jest.spyOn(currentQueryClient, "invalidateQueries");

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    const session = mockSession();
    await act(async () => {
      authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    await waitFor(() => capture.denyApproval !== null);

    await act(async () => {
      await capture.denyApproval!("appr_1");
    });

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["approvals"] });
  });
});
