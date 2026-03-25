import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useApprovalDetail } from "../useApprovalDetail";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockApprovalResponse = {
  approval_id: "apr_123",
  agent_id: 1,
  action: { type: "email.send", parameters: { to: "a@b.com" } },
  context: { risk_level: "low" },
  status: "approved",
  execution_status: "success",
  execution_result: { message_id: "msg_456" },
  expires_at: "2026-02-21T12:00:00Z",
  created_at: "2026-02-20T11:00:00Z",
};

describe("useApprovalDetail", () => {
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

  it("does not fetch when approvalId is null", () => {
    setupAuthMocks({ authenticated: true });

    const { result } = renderHook(() => useApprovalDetail(null), { wrapper });

    expect(result.current.approval).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("does not fetch when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useApprovalDetail("apr_123"), {
      wrapper,
    });

    expect(result.current.approval).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("fetches approval details when authenticated with an ID", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockApprovalResponse });

    const { result } = renderHook(() => useApprovalDetail("apr_123"), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.approval).toEqual(mockApprovalResponse);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/approvals/{approval_id}", {
      headers: { Authorization: "Bearer token" },
      params: { path: { approval_id: "apr_123" } },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("returns error state on API failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      error: { message: "Not found" },
      response: { status: 404 },
    });

    const { result } = renderHook(() => useApprovalDetail("apr_bad"), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.error).toBe(
        "Unable to load approval details.",
      );
    });
    expect(result.current.approval).toBeNull();
  });
});
