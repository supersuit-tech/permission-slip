import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useDenyApproval } from "../useDenyApproval";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useDenyApproval", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends deny request with correct path and params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: { approval_id: "approval-abc", status: "denied", denied_at: "2026-02-21T12:00:00Z" },
    });

    const { result } = renderHook(() => useDenyApproval(), {
      wrapper,
    });

    await act(async () => {
      await result.current.denyApproval("approval-abc");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/deny",
      {
        headers: { Authorization: "Bearer token" },
        params: { path: { approval_id: "approval-abc" } },
      },
    );
    expect(result.current.isPending).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useDenyApproval(), {
      wrapper,
    });

    await expect(
      result.current.denyApproval("approval-abc"),
    ).rejects.toThrow("Not authenticated");
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useDenyApproval(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.denyApproval("approval-abc");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to deny request");
  });

  it("passes approval ID via path params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: {} });

    const { result } = renderHook(() => useDenyApproval(), {
      wrapper,
    });

    await act(async () => {
      await result.current.denyApproval("other-approval-id");
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/approvals/{approval_id}/deny",
      expect.objectContaining({
        params: { path: { approval_id: "other-approval-id" } },
      }),
    );
  });
});
