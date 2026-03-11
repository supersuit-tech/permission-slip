import { renderHook, act } from "@testing-library/react";
import { AuthError } from "@supabase/supabase-js";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { mockAuth, setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { useSignOut } from "../useSignOut";
import { MFA_PENDING_ENROLLMENT_KEY } from "../../auth/mfaPendingEnrollment";

vi.mock("../../lib/supabaseClient");
vi.mock("sonner");

describe("useSignOut", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    vi.mocked(toast.error).mockClear();
    sessionStorage.clear();
    wrapper = createAuthWrapper();
  });

  it("calls signOut on the auth provider", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });

    const { result } = renderHook(() => useSignOut(), { wrapper });

    await act(async () => {
      await result.current();
    });

    expect(mockAuth.signOut).toHaveBeenCalled();
  });

  it("shows toast and logs error when signOut fails", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const authError = new AuthError("Sign out failed", 500);
    mockAuth.signOut.mockResolvedValue({ error: authError });

    const { result } = renderHook(() => useSignOut(), { wrapper });

    await act(async () => {
      await result.current();
    });

    expect(consoleSpy).toHaveBeenCalledWith("Sign out failed:", authError);
    expect(toast.error).toHaveBeenCalledWith(
      "Sign out failed. Please try again."
    );
    consoleSpy.mockRestore();
  });

  it("clears MFA pending enrollment from sessionStorage", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });
    sessionStorage.setItem(
      MFA_PENDING_ENROLLMENT_KEY,
      JSON.stringify({ userId: "user-123" })
    );

    const { result } = renderHook(() => useSignOut(), { wrapper });

    await act(async () => {
      await result.current();
    });

    expect(sessionStorage.getItem(MFA_PENDING_ENROLLMENT_KEY)).toBeNull();
  });

  it("does not show toast on success", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });

    const { result } = renderHook(() => useSignOut(), { wrapper });

    await act(async () => {
      await result.current();
    });

    expect(toast.error).not.toHaveBeenCalled();
  });

  it("clears React Query cache on sign-out", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });

    const { result } = renderHook(
      () => ({ signOut: useSignOut(), queryClient: useQueryClient() }),
      { wrapper },
    );

    // Seed the cache with data from the current user.
    result.current.queryClient.setQueryData(["agent", 42], { agent_id: 42 });
    expect(result.current.queryClient.getQueryData(["agent", 42])).toBeDefined();

    await act(async () => {
      await result.current.signOut();
    });

    // Cache should be empty after sign-out.
    expect(result.current.queryClient.getQueryData(["agent", 42])).toBeUndefined();
  });
});
