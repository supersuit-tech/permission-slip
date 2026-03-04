import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useApproveApproval } from "../useApproveApproval";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockApproveResponse = {
  approval_id: "approval-abc",
  status: "approved",
  approved_at: "2026-02-21T12:00:00Z",
  execution_status: "success" as const,
  execution_result: { data: "ok" },
};

describe("useApproveApproval", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends approve request with correct path and params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: mockApproveResponse });

    const { result } = renderHook(() => useApproveApproval(), {
      wrapper,
    });

    await act(async () => {
      await result.current.approveApproval("approval-abc");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/approve",
      {
        headers: { Authorization: "Bearer token" },
        params: { path: { approval_id: "approval-abc" } },
      },
    );
    expect(result.current.isPending).toBe(false);
  });

  it("returns the approval data from the response", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: mockApproveResponse });

    const { result } = renderHook(() => useApproveApproval(), {
      wrapper,
    });

    let response: Awaited<ReturnType<typeof result.current.approveApproval>>;
    await act(async () => {
      response = await result.current.approveApproval("approval-abc");
    });

    expect(response!.approval_id).toBe("approval-abc");
    expect(response!.status).toBe("approved");
    expect(response!.execution_status).toBe("success");
    expect(response!.execution_result).toEqual({ data: "ok" });
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useApproveApproval(), {
      wrapper,
    });

    await expect(
      result.current.approveApproval("approval-abc"),
    ).rejects.toThrow("Not authenticated");
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useApproveApproval(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.approveApproval("approval-abc");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to approve request");
  });

  it("passes approval ID via path params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: mockApproveResponse });

    const { result } = renderHook(() => useApproveApproval(), {
      wrapper,
    });

    await act(async () => {
      await result.current.approveApproval("some-other-id");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/approve",
      expect.objectContaining({
        params: { path: { approval_id: "some-other-id" } },
      }),
    );
  });
});
