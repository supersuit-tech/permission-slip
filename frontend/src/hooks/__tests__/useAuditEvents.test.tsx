import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { mockHookAuditResponse } from "../../lib/__tests__/auditEventFixtures";
import { useAuditEvents } from "../useAuditEvents";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useAuditEvents", () => {
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

    const { result } = renderHook(() => useAuditEvents(), { wrapper });

    expect(result.current.events).toEqual([]);
    expect(result.current.hasMore).toBe(false);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches audit events when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(() => useAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.events).toEqual(mockHookAuditResponse.data);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 20 } },
    });
    expect(result.current.hasMore).toBe(true);
    expect(result.current.nextCursor).toBe("cursor_abc");
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("passes outcome filter to API", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(
      () => useAuditEvents({ outcome: "denied" }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 20, outcome: "denied" } },
    });
  });

  it("passes agent_id filter to API", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(
      () => useAuditEvents({ agent_id: 42 }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 20, agent_id: 42 } },
    });
  });

  it("passes event_type filter to API", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(
      () => useAuditEvents({ event_type: "approval.approved" }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 20, event_type: "approval.approved" } },
    });
  });

  it("respects custom limit", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(
      () => useAuditEvents({}, 5),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/audit-events", {
      headers: { Authorization: "Bearer token" },
      params: { query: { limit: 5 } },
    });
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.events).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load activity. Please try again later.",
    );
  });

  it("sets error on API error response", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.events).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load activity. Please try again later.",
    );
  });

  it("provides a refetch function", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockHookAuditResponse });

    const { result } = renderHook(() => useAuditEvents(), { wrapper });

    await waitFor(() => {
      expect(result.current.events).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledTimes(1);

    await result.current.refetch();

    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});
