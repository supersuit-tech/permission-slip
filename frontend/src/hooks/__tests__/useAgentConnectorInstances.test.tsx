import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import {
  mockGet,
  mockPost,
  mockPatch,
  mockDelete,
  resetClientMocks,
} from "../../api/__mocks__/client";
import {
  useAgentConnectorInstances,
  useCreateAgentConnectorInstance,
  useRenameAgentConnectorInstance,
  useSetDefaultAgentConnectorInstance,
  useDeleteAgentConnectorInstance,
} from "../useAgentConnectorInstances";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useAgentConnectorInstances", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
    setupAuthMocks({ authenticated: true });
  });

  it("lists instances", async () => {
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            connector_instance_id: "11111111-1111-1111-1111-111111111111",
            agent_id: 1,
            connector_id: "slack",
            label: "Main",
            is_default: true,
            enabled_at: "2026-01-01T00:00:00Z",
          },
        ],
      },
    });

    const { result } = renderHook(
      () => useAgentConnectorInstances(1, "slack"),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.instances).toHaveLength(1);
    });
    expect(result.current.instances[0]?.label).toBe("Main");
    expect(mockGet).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
      expect.objectContaining({
        params: { path: { agent_id: 1, connector_id: "slack" } },
      }),
    );
  });

  it("create posts label", async () => {
    mockPost.mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useCreateAgentConnectorInstance(), {
      wrapper,
    });

    await act(async () => {
      await result.current.create({
        agentId: 1,
        connectorId: "slack",
        label: "Sales",
      });
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
      expect.objectContaining({
        body: { label: "Sales" },
        params: { path: { agent_id: 1, connector_id: "slack" } },
      }),
    );
  });

  it("setDefault patches is_default", async () => {
    mockPatch.mockResolvedValue({ data: {} });
    const { result } = renderHook(
      () => useSetDefaultAgentConnectorInstance(),
      { wrapper },
    );

    await act(async () => {
      await result.current.setDefault({
        agentId: 1,
        connectorId: "slack",
        instanceId: "22222222-2222-2222-2222-222222222222",
      });
    });

    expect(mockPatch).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
      expect.objectContaining({
        body: { is_default: true },
        params: {
          path: {
            agent_id: 1,
            connector_id: "slack",
            instance_id: "22222222-2222-2222-2222-222222222222",
          },
        },
      }),
    );
  });

  it("rename patches label", async () => {
    mockPatch.mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useRenameAgentConnectorInstance(), {
      wrapper,
    });

    await act(async () => {
      await result.current.rename({
        agentId: 1,
        connectorId: "slack",
        instanceId: "11111111-1111-1111-1111-111111111111",
        label: "Renamed",
      });
    });

    expect(mockPatch).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
      expect.objectContaining({
        body: { label: "Renamed" },
      }),
    );
  });

  it("deleteInstance calls DELETE", async () => {
    mockDelete.mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useDeleteAgentConnectorInstance(), {
      wrapper,
    });

    await act(async () => {
      await result.current.deleteInstance({
        agentId: 1,
        connectorId: "slack",
        instanceId: "22222222-2222-2222-2222-222222222222",
      });
    });

    expect(mockDelete).toHaveBeenCalledWith(
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
      expect.objectContaining({
        params: {
          path: {
            agent_id: 1,
            connector_id: "slack",
            instance_id: "22222222-2222-2222-2222-222222222222",
          },
        },
      }),
    );
  });
});
