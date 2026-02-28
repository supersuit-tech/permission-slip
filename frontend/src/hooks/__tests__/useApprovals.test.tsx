import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useApprovals } from "../useApprovals";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockApprovalsResponse = {
  data: [
    {
      approval_id: "appr_abc123",
      agent_id: 1,
      action: {
        type: "email.send",
        version: "1",
        parameters: { to: ["bob@example.com"], subject: "Hello" },
      },
      context: { description: "Send email", risk_level: "low" },
      status: "pending",
      expires_at: "2026-02-21T10:05:00Z",
      created_at: "2026-02-21T10:00:00Z",
    },
  ],
  has_more: false,
};

describe("useApprovals", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns empty approvals when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useApprovals(), { wrapper });

    expect(result.current.approvals).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches pending approvals when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockApprovalsResponse });

    const { result } = renderHook(() => useApprovals(), { wrapper });

    await waitFor(() => {
      expect(result.current.approvals).toEqual(mockApprovalsResponse.data);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: { Authorization: "Bearer token" },
      params: { query: { status: "pending" } },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useApprovals(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.approvals).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load approvals. Please try again later.",
    );
  });

  it("sets error on API error response", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useApprovals(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.approvals).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load approvals. Please try again later.",
    );
  });

  it("provides a refetch function", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockApprovalsResponse });

    const { result } = renderHook(() => useApprovals(), { wrapper });

    await waitFor(() => {
      expect(result.current.approvals).toHaveLength(1);
    });

    expect(mockGet).toHaveBeenCalledTimes(1);

    await result.current.refetch();

    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});
