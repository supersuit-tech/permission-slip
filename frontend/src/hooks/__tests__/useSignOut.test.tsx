import { renderHook, act } from "@testing-library/react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { useSignOut } from "../useSignOut";

vi.mock("sonner");

describe("useSignOut", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    vi.mocked(toast.error).mockClear();
    wrapper = createAuthWrapper();
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/logout")) {
        return new Response(null, { status: 204 });
      }
      if (url.includes("/auth/refresh")) {
        return new Response(
          JSON.stringify({
            access_token: "t",
            refresh_token: "r",
            expires_at: new Date(Date.now() + 3600_000).toISOString(),
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      return new Response("{}", { status: 404 });
    }) as typeof fetch;
  });

  it("clears React Query cache on sign-out", async () => {
    const { result } = renderHook(
      () => ({ signOut: useSignOut(), queryClient: useQueryClient() }),
      { wrapper },
    );

    await settleAuthHydration();
    result.current.queryClient.setQueryData(["agent", 42], { agent_id: 42 });
    expect(result.current.queryClient.getQueryData(["agent", 42])).toBeDefined();

    await act(async () => {
      await result.current.signOut();
    });

    expect(result.current.queryClient.getQueryData(["agent", 42])).toBeUndefined();
  });

  it("shows toast when logout request fails", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/logout")) {
        return new Response(
          JSON.stringify({
            error: { code: "internal", message: "fail" },
          }),
          { status: 500, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.includes("/auth/refresh")) {
        return new Response(
          JSON.stringify({
            access_token: "t",
            refresh_token: "r",
            expires_at: new Date(Date.now() + 3600_000).toISOString(),
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      return new Response("{}", { status: 404 });
    }) as typeof fetch;

    const { result } = renderHook(() => useSignOut(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current();
    });

    expect(toast.error).toHaveBeenCalledWith(
      "Sign out failed. Please try again.",
    );
    consoleSpy.mockRestore();
  });

  it("does not show toast on successful logout", async () => {
    const { result } = renderHook(() => useSignOut(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current();
    });

    expect(toast.error).not.toHaveBeenCalled();
  });
});
