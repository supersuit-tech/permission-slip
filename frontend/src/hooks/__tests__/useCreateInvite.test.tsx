import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useCreateInvite } from "../useCreateInvite";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockInviteResponse = {
  id: "ri_abc123",
  invite_code: "PS-ABCD-1234",
  status: "active",
  expires_at: "2026-02-19T12:15:00Z",
  created_at: "2026-02-19T12:00:00Z",
};

describe("useCreateInvite", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("creates an invite and returns the response", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: mockInviteResponse });

    const { result } = renderHook(() => useCreateInvite(), {
      wrapper,
    });

    let invite: Awaited<ReturnType<typeof result.current.createInvite>>;
    await act(async () => {
      invite = await result.current.createInvite();
    });

    expect(invite!).toEqual(mockInviteResponse);
    expect(result.current.isLoading).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useCreateInvite(), {
      wrapper,
    });

    await expect(result.current.createInvite()).rejects.toThrow(
      "Not authenticated"
    );
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useCreateInvite(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.createInvite();
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Server error");
  });

  it("sends correct request", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: mockInviteResponse });

    const { result } = renderHook(() => useCreateInvite(), {
      wrapper,
    });

    await act(async () => {
      await result.current.createInvite();
    });

    expect(mockPost).toHaveBeenCalledWith("/v1/registration-invites", {
      headers: { Authorization: "Bearer token" },
      body: {},
    });
  });
});
