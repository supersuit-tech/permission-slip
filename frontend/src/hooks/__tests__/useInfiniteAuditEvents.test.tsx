import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { mockHookAuditResponse } from "../../lib/__tests__/auditEventFixtures";
import { useInfiniteAuditEvents } from "../useInfiniteAuditEvents";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const page1 = {
  data: mockHookAuditResponse.data,
  has_more: true,
  next_cursor: "cursor_page2",
};

const page2 = {
  data: [
    {
      event_type: "approval.approved" as const,
      timestamp: "2026-02-20T08:00:00Z",
      agent_id: 3,
      agent_metadata: { name: "Third Bot" },
      action: { type: "file.read", version: "1", parameters: { path: "/tmp" } },
      outcome: "approved" as const,
    },
  ],
  has_more: false,
};

describe("useInfiniteAuditEvents", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns empty events when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useInfiniteAuditEvents(), { wrapper });

    expect(result.current.events).toEqual([]);
    expect(result.current.hasNextPage).toBe(false);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches first page when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: page1 });

    const { result } = renderHook(() => useInfiniteAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.events).toEqual(page1.data);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 50 } },
    });
    expect(result.current.hasNextPage).toBe(true);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("fetches next page with cursor", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValueOnce({ data: page1 });

    const { result } = renderHook(() => useInfiniteAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    // Set up second page response
    mockGet.mockResolvedValueOnce({ data: page2 });
    result.current.fetchNextPage();

    await waitFor(() => {
      expect(result.current.events).toHaveLength(3);
    });

    // Verify second call used cursor
    const calls = mockGet.mock.calls;
    const secondCall = calls[1] as unknown[];
    expect(secondCall?.[0]).toBe("/v1/audit-events");
    const opts = secondCall?.[1] as { params?: { query?: { after?: string } } };
    expect(opts?.params?.query?.after).toBe("cursor_page2");

    expect(result.current.hasNextPage).toBe(false);
  });

  it("passes filters to API", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: page1 });

    const { result } = renderHook(
      () =>
        useInfiniteAuditEvents({
          outcome: "denied",
          agent_id: 42,
          event_type: "approval.denied",
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: {
        query: {
          limit: 50,
          outcome: "denied",
          agent_id: 42,
          event_type: "approval.denied",
        },
      },
    });
  });

  it("respects custom limit", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: page1 });

    const { result } = renderHook(
      () => useInfiniteAuditEvents({}, 25),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 25 } },
    });
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useInfiniteAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.events).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load activity. Please try again later.",
    );
  });
});
