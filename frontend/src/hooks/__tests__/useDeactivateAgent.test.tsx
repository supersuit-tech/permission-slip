import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useDeactivateAgent } from "../useDeactivateAgent";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useDeactivateAgent", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends deactivate request with correct path and params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: { agent_id: 42, status: "deactivated" },
    });

    const { result } = renderHook(() => useDeactivateAgent(), {
      wrapper,
    });

    await act(async () => {
      await result.current.deactivateAgent(42);
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/deactivate",
      {
        headers: { Authorization: "Bearer token" },
        params: { path: { agent_id: 42 } },
      },
    );
    expect(result.current.isLoading).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useDeactivateAgent(), {
      wrapper,
    });

    await expect(result.current.deactivateAgent(42)).rejects.toThrow(
      "Not authenticated"
    );
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: "agent_not_found", message: "Agent not found" } },
    });

    const { result } = renderHook(() => useDeactivateAgent(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.deactivateAgent(42);
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to deactivate agent");
  });

  it("passes numeric agent ID via path params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({ data: {} });

    const { result } = renderHook(() => useDeactivateAgent(), {
      wrapper,
    });

    await act(async () => {
      await result.current.deactivateAgent(999);
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/deactivate",
      expect.objectContaining({
        params: { path: { agent_id: 999 } },
      }),
    );
  });
});
