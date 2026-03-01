import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { useProfile } from "../useProfile";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useProfile", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns null profile when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useProfile(), {
      wrapper,
    });

    expect(result.current.profile).toBeNull();
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches profile when authenticated", async () => {
    setupAuthMocks({ authenticated: true });

    const mockProfile = {
      id: "user-123",
      username: "janedoe",
      marketing_opt_in: false,
      created_at: "2024-01-01T00:00:00Z",
    };
    mockGet.mockResolvedValue({
      data: mockProfile,
      error: undefined,
      response: { status: 200 },
    });

    const { result } = renderHook(() => useProfile(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.profile).toEqual(mockProfile);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/profile", {
      headers: { Authorization: "Bearer token" },
    });
    expect(result.current.needsOnboarding).toBe(false);
  });

  it("returns needsOnboarding on 404", async () => {
    setupAuthMocks({ authenticated: true });

    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "profile_not_found", message: "Profile not found" } },
      response: { status: 404 },
    });

    const { result } = renderHook(() => useProfile(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.profile).toBeNull();
    expect(result.current.needsOnboarding).toBe(true);
  });

  it("returns null profile on fetch error", async () => {
    setupAuthMocks({ authenticated: true });

    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useProfile(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.profile).toBeNull();
    expect(result.current.needsOnboarding).toBe(false);
  });
});
