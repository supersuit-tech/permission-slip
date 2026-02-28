import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { usePendingAgents } from "../usePendingAgents";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockAgentsResponse = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      last_active_at: "2026-02-19T10:00:00Z",
      request_count_30d: 42,
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      agent_id: 2,
      status: "pending",
      confirmation_code: "XK7-M9P",
      expires_at: "2026-12-31T23:59:59Z",
      request_count_30d: 0,
      created_at: "2026-02-01T00:00:00Z",
    },
    {
      agent_id: 3,
      status: "pending",
      confirmation_code: "AB3-Z4Q",
      expires_at: "2026-12-31T23:59:59Z",
      request_count_30d: 0,
      created_at: "2026-02-02T00:00:00Z",
    },
    {
      agent_id: 4,
      status: "deactivated",
      created_at: "2025-12-01T00:00:00Z",
    },
  ],
  has_more: false,
};

describe("usePendingAgents", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns empty array when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    expect(result.current.pendingAgents).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("returns only pending agents when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockAgentsResponse });

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    await waitFor(() => {
      expect(result.current.pendingAgents).toHaveLength(2);
    });

    expect(result.current.pendingAgents[0]?.agent_id).toBe(2);
    expect(result.current.pendingAgents[1]?.agent_id).toBe(3);
    expect(result.current.isLoading).toBe(false);
  });

  it("returns empty array when no agents are pending", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            agent_id: 1,
            status: "registered",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_more: false,
      },
    });

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.pendingAgents).toEqual([]);
  });

  it("excludes expired pending agents", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            agent_id: 2,
            status: "pending",
            confirmation_code: "XK7-M9P",
            expires_at: "2020-01-01T00:00:00Z", // expired
            request_count_30d: 0,
            created_at: "2026-02-01T00:00:00Z",
          },
          {
            agent_id: 3,
            status: "pending",
            confirmation_code: "AB3-Z4Q",
            expires_at: "2099-12-31T23:59:59Z", // still active
            request_count_30d: 0,
            created_at: "2026-02-02T00:00:00Z",
          },
        ],
        has_more: false,
      },
    });

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    await waitFor(() => {
      expect(result.current.pendingAgents).toHaveLength(1);
    });

    expect(result.current.pendingAgents[0]?.agent_id).toBe(3);
  });

  it("includes pending agents without expiration time", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            agent_id: 4,
            status: "pending",
            confirmation_code: "NO-EXP",
            expires_at: null, // no expiration
            request_count_30d: 0,
            created_at: "2026-02-03T00:00:00Z",
          },
        ],
        has_more: false,
      },
    });

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    await waitFor(() => {
      expect(result.current.pendingAgents).toHaveLength(1);
    });

    expect(result.current.pendingAgents[0]?.agent_id).toBe(4);
  });
  it("returns empty array on API error", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => usePendingAgents(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.pendingAgents).toEqual([]);
  });
});
