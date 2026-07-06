import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useApproveStandingApprovalRequest } from "../useApproveStandingApprovalRequest";

vi.mock("../../api/client");

describe("useApproveStandingApprovalRequest", () => {
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
          code: "invalid_request",
          message: "Cannot create standing approval: action type is not registered for connector",
        },
      },
    });

    const { result } = renderHook(() => useApproveStandingApprovalRequest(), {
      wrapper,
    });

    await settleAuthHydration();
    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.approveRequest("sar_test");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe(
      "Cannot create standing approval: action type is not registered for connector",
    );
  });
});
