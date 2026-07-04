import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useDenyStandingApprovalRequest } from "../useDenyStandingApprovalRequest";

vi.mock("../../api/client");

describe("useDenyStandingApprovalRequest", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("throws API error message on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: {
        error: {
          code: "conflict",
          message: "Standing approval request is no longer pending",
        },
      },
    });

    const { result } = renderHook(() => useDenyStandingApprovalRequest(), {
      wrapper,
    });

    await settleAuthHydration();
    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.denyRequest("sar_test");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Standing approval request is no longer pending");
  });
});
