import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPut, resetClientMocks } from "../../api/__mocks__/client";
import { useUpdateNotificationPreferences } from "../useUpdateNotificationPreferences";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockUpdatedResponse = {
  preferences: [
    { channel: "email", enabled: false, available: true },
    { channel: "web-push", enabled: true, available: true },
    { channel: "sms", enabled: true, available: true },
  ],
};

describe("useUpdateNotificationPreferences", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("updates notification preferences", async () => {
    setupAuthMocks({ authenticated: true });
    mockPut.mockResolvedValue({ data: mockUpdatedResponse });

    const { result } = renderHook(
      () => useUpdateNotificationPreferences(),
      { wrapper },
    );

    await act(async () => {
      await result.current.updatePreferences([
        { channel: "email", enabled: false },
      ]);
    });

    expect(mockPut).toHaveBeenCalledWith(
      "/v1/profile/notification-preferences",
      {
        headers: { Authorization: "Bearer token" },
        body: { preferences: [{ channel: "email", enabled: false }] },
      },
    );
  });

  it("throws on failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockPut.mockResolvedValue({
      data: undefined,
      error: {
        error: { code: "internal_error", message: "Server error" },
      },
    });

    const { result } = renderHook(
      () => useUpdateNotificationPreferences(),
      { wrapper },
    );

    await expect(
      act(async () => {
        await result.current.updatePreferences([
          { channel: "email", enabled: false },
        ]);
      }),
    ).rejects.toThrow("Failed to update notification preferences");
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(
      () => useUpdateNotificationPreferences(),
      { wrapper },
    );

    await expect(
      act(async () => {
        await result.current.updatePreferences([
          { channel: "email", enabled: true },
        ]);
      }),
    ).rejects.toThrow("Not authenticated");
  });
});
