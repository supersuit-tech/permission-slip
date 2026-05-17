import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPatch, resetClientMocks } from "../../api/__mocks__/client";
import { useUpdateProfile } from "../useUpdateProfile";

vi.mock("../../api/client");

const mockUpdatedProfile = {
  id: "user-123",
  username: "alice",
  email: "new@example.com",
  phone: "+15551234567",
  marketing_opt_in: false,
  created_at: "2026-01-01T00:00:00Z",
};

describe("useUpdateProfile", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("updates profile contact fields", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({ data: mockUpdatedProfile });

    const { result } = renderHook(() => useUpdateProfile(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current.updateProfile({
        email: "new@example.com",
        phone: "+15551234567",
      });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/profile", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      body: { email: "new@example.com", phone: "+15551234567" },
    });
  });

  it("supports partial update (email only)", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: { ...mockUpdatedProfile, phone: null },
    });

    const { result } = renderHook(() => useUpdateProfile(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current.updateProfile({ email: "new@example.com" });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/profile", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      body: { email: "new@example.com" },
    });
  });

  it("throws on failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: undefined,
      error: {
        error: { code: "internal_error", message: "Server error" },
      },
    });

    const { result } = renderHook(() => useUpdateProfile(), { wrapper });

    await settleAuthHydration();
    await expect(
      act(async () => {
        await result.current.updateProfile({ email: "new@example.com" });
      }),
    ).rejects.toThrow("Failed to update profile");
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useUpdateProfile(), { wrapper });

    await settleAuthHydration();
    await expect(
      act(async () => {
        await result.current.updateProfile({ email: "new@example.com" });
      }),
    ).rejects.toThrow("Not authenticated");
  });
});
