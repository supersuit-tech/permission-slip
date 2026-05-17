import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useApproveApproval } from "../useApproveApproval";
import { mockSession, createQueryClient, waitFor } from "../../__test-utils__";

jest.mock("../../lib/authStorage");
jest.mock("../../lib/authApi");

import * as authStorage from "../../lib/authStorage";
import * as authApi from "../../lib/authApi";

const mockAuthStorage = jest.mocked(authStorage);
const mockAuthApi = jest.mocked(authApi);

// --- Mocks ---

const mockPost = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { POST: (...args: unknown[]) => mockPost(...args) },
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

  it("calls POST /v1/approvals/{approval_id}/approve with correct params", async () => {
    mockPost.mockResolvedValue({
      data: {
        approval_id: "appr_abc123",
        status: "approved",
        approved_at: "2026-03-02T13:25:00Z",
        confirmation_code: "RK3P7-MNPQR",
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
      expect.objectContaining({ confirmation_code: "RK3P7-MNPQR" }),
    );
  });

  it("throws on API error", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "approval_expired", message: "Approval has expired" } },
    });

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
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
