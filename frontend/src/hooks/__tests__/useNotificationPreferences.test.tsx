import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useNotificationPreferences } from "../useNotificationPreferences";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockPreferencesResponse = {
  preferences: [
    { channel: "email", enabled: true, available: true },
    { channel: "web-push", enabled: true, available: true },
    { channel: "sms", enabled: false, available: true },
  ],
};

describe("useNotificationPreferences", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns empty preferences when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper,
    });

    expect(result.current.preferences).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches preferences when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockPreferencesResponse });

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.preferences).toEqual(
        mockPreferencesResponse.preferences,
      );
    });

    expect(mockGet).toHaveBeenCalledWith(
      "/v1/profile/notification-preferences",
      { headers: { Authorization: "Bearer token" } },
    );
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useNotificationPreferences(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.preferences).toEqual([]);
    expect(result.current.error).not.toBeNull();
  });
});
