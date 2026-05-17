import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../auth/AuthContext";
import { useApprovals, type ApprovalSummary } from "../useApprovals";
import {
  mockSession,
  mockApproval,
  createQueryClient,
  waitFor,
} from "../../__test-utils__";

jest.mock("../../lib/authStorage");
jest.mock("../../lib/authApi");

import * as authStorage from "../../lib/authStorage";
import * as authApi from "../../lib/authApi";

// --- Mocks ---

const mockGet = jest.fn();

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockGet(...args) },
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

describe("useApprovals", () => {
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

  it("returns empty list and loading false when not authenticated", async () => {
    const { capture, Consumer } = createHookCapture();
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
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

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture("pending");
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
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

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createHookCapture("pending");
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
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

    const session = mockSession();
    bootstrapAuthenticated(session);

    const { Consumer } = createHookCapture("denied");
    currentQueryClient = createQueryClient();

    await act(async () => {
      currentRenderer = renderWithProviders(Consumer, currentQueryClient!);
    });

    await waitFor(() => mockGet.mock.calls.length > 0);

    expect(mockGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: expect.any(Object),
      params: { query: { status: "denied" } },
    });
  });
});
