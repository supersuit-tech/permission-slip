import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockDelete, resetClientMocks } from "../../api/__mocks__/client";
import { useDisableAgentConnector } from "../useDisableAgentConnector";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useDisableAgentConnector", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends disable request with correct path and params", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: {
        agent_id: 42,
        connector_id: "gmail",
        disabled_at: "2026-02-18T12:00:00Z",
        revoked_standing_approvals: 2,
      },
    });

    const { result } = renderHook(() => useDisableAgentConnector(), {
      wrapper,
    });

    let response: unknown;
    await act(async () => {
      response = await result.current.disableConnector({
        agentId: 42,
        connectorId: "gmail",
      });
    });

    expect(mockDelete).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}",
      {
        headers: { Authorization: "Bearer token" },
        params: {
          path: { agent_id: 42, connector_id: "gmail" },
          query: {},
        },
      },
    );
    expect(response).toEqual({
      agent_id: 42,
      connector_id: "gmail",
      disabled_at: "2026-02-18T12:00:00Z",
      revoked_standing_approvals: 2,
    });
    expect(result.current.isLoading).toBe(false);
  });

  it("sends delete_credentials=true when deleteCredentials is set", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: {
        agent_id: 42,
        connector_id: "github",
        disabled_at: "2026-02-18T12:00:00Z",
        revoked_standing_approvals: 0,
      },
    });

    const { result } = renderHook(() => useDisableAgentConnector(), {
      wrapper,
    });

    await act(async () => {
      await result.current.disableConnector({
        agentId: 42,
        connectorId: "github",
        deleteCredentials: true,
      });
    });

    expect(mockDelete).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}",
      {
        headers: { Authorization: "Bearer token" },
        params: {
          path: { agent_id: 42, connector_id: "github" },
          query: { delete_credentials: true },
        },
      },
    );
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useDisableAgentConnector(), {
      wrapper,
    });

    await expect(
      result.current.disableConnector({ agentId: 42, connectorId: "gmail" }),
    ).rejects.toThrow("Not authenticated");
  });

  it("throws on server error", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: undefined,
      error: {
        error: {
          code: "agent_connector_not_found",
          message: "Connector not enabled for this agent",
        },
      },
    });

    const { result } = renderHook(() => useDisableAgentConnector(), {
      wrapper,
    });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.disableConnector({
          agentId: 42,
          connectorId: "gmail",
        });
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to disable connector");
  });
});
