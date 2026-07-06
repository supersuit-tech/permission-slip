import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useDenyAllApprovals } from "../useDenyAllApprovals";

vi.mock("../../api/client");

describe("useDenyAllApprovals", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("calls deny-all endpoint and returns denied count", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: { denied_count: 2, skipped_count: 0 },
    });

    const { result } = renderHook(() => useDenyAllApprovals(), { wrapper });

    await settleAuthHydration();
    let response;
    await act(async () => {
      response = await result.current.denyAllApprovals();
    });

    expect(mockPost).toHaveBeenCalledWith("/v1/approvals/deny-all", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
    });
    expect(response).toEqual({ denied_count: 2, skipped_count: 0 });
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useDenyAllApprovals(), { wrapper });

    await settleAuthHydration();
    await expect(result.current.denyAllApprovals()).rejects.toThrow(
      "Not authenticated",
    );
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useDenyAllApprovals(), { wrapper });

    await settleAuthHydration();
    await expect(result.current.denyAllApprovals()).rejects.toThrow("Server error");
  });
});
