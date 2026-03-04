/**
 * Tests for the useApproval hook — cache lookup, API fetch, and validation.
 */
import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { Text } from "react-native";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// --- Mocks ---

const mockClientGet = jest.fn();
let mockSession: { access_token: string } | null = null;

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockClientGet(...args) },
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({ session: mockSession }),
}));

import { useApproval } from "../useApproval";

// --- Helpers ---

const fakeApproval = {
  approval_id: "appr_abc123",
  agent_id: 1,
  action: { type: "test.action", parameters: {} },
  context: { risk_level: "low", details: {} },
  status: "pending",
  expires_at: "2026-12-31T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
};

let captured: ReturnType<typeof useApproval>;
let queryClient: QueryClient;

function TestHarness({ approvalId }: { approvalId: string }) {
  captured = useApproval(approvalId);
  return createElement(
    Text,
    null,
    `loading=${captured.isLoading} error=${captured.error}`,
  );
}

function renderWithQuery(approvalId: string) {
  queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return createElement(
    QueryClientProvider,
    { client: queryClient },
    createElement(TestHarness, { approvalId }),
  );
}

async function flush(ms = 50) {
  await act(async () => {
    await new Promise((r) => setTimeout(r, ms));
  });
}

// --- Tests ---

describe("useApproval", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.clearAllMocks();
    mockSession = { access_token: "test-token" };
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
    queryClient?.clear();
  });

  it("returns invalid format error for malformed approval IDs", async () => {
    await act(async () => {
      renderer = create(renderWithQuery("bad-id"));
    });
    await flush();

    expect(captured.error).toBe("Invalid approval ID format");
    expect(captured.isLoading).toBe(false);
    expect(captured.approval).toBeNull();
    expect(mockClientGet).not.toHaveBeenCalled();
  });

  it("fetches approval from API when not in cache", async () => {
    mockClientGet.mockResolvedValue({
      data: { data: [fakeApproval], has_more: false },
      error: undefined,
    });

    await act(async () => {
      renderer = create(renderWithQuery("appr_abc123"));
    });
    await flush();

    expect(mockClientGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: { Authorization: "Bearer test-token" },
      params: { query: { status: "all" } },
    });
    expect(captured.approval).toEqual(fakeApproval);
    expect(captured.error).toBeNull();
  });

  it("returns approval from React Query cache without API call", async () => {
    // Pre-populate the cache with approval list data
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(["approvals", "pending"], {
      data: [fakeApproval],
    });

    await act(async () => {
      renderer = create(
        createElement(
          QueryClientProvider,
          { client: queryClient },
          createElement(TestHarness, { approvalId: "appr_abc123" }),
        ),
      );
    });
    await flush();

    expect(captured.approval).toEqual(fakeApproval);
    expect(mockClientGet).not.toHaveBeenCalled();
  });

  it("returns error when approval is not found in API response", async () => {
    mockClientGet.mockResolvedValue({
      data: { data: [], has_more: false },
      error: undefined,
    });

    await act(async () => {
      renderer = create(renderWithQuery("appr_notfound"));
    });
    // useApproval has retry: 1 so React Query retries once before erroring.
    // Need enough time for the retry backoff to complete.
    await flush(500);
    await flush(500);
    await flush(500);

    expect(captured.approval).toBeNull();
    expect(captured.error).toBe("Approval not found");
  });

  it("does not fetch when auth session is missing", async () => {
    mockSession = null;

    await act(async () => {
      renderer = create(renderWithQuery("appr_abc123"));
    });
    await flush();

    expect(mockClientGet).not.toHaveBeenCalled();
  });
});
