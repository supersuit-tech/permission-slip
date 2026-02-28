import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPut, resetClientMocks } from "../../api/__mocks__/client";
import { useEnableAgentConnector } from "../useEnableAgentConnector";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useEnableAgentConnector", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends enable request with correct path and params", async () => {
    setupAuthMocks({ authenticated: true });
    mockPut.mockResolvedValue({
      data: {
        agent_id: 42,
        connector_id: "gmail",
        enabled_at: "2026-02-18T10:00:00Z",
      },
    });

    const { result } = renderHook(() => useEnableAgentConnector(), {
      wrapper,
    });

    await act(async () => {
      await result.current.enableConnector({
        agentId: 42,
        connectorId: "gmail",
      });
    });

    expect(mockPut).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}",
      {
        headers: { Authorization: "Bearer token" },
        params: {
          path: { agent_id: 42, connector_id: "gmail" },
        },
      },
    );
    expect(result.current.isLoading).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useEnableAgentConnector(), {
      wrapper,
    });

    await expect(
      result.current.enableConnector({ agentId: 42, connectorId: "gmail" }),
    ).rejects.toThrow("Not authenticated");
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPut.mockResolvedValue({
      data: undefined,
      error: {
        error: {
          code: "connector_not_found",
          message: "Connector not found",
        },
      },
    });

    const { result } = renderHook(() => useEnableAgentConnector(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.enableConnector({
          agentId: 42,
          connectorId: "invalid",
        });
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to enable connector");
  });
});
