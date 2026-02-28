import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPatch, resetClientMocks } from "../../api/__mocks__/client";
import { useUpdateAgent } from "../useUpdateAgent";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockUpdatedAgent = {
  agent_id: 42,
  status: "registered",
  metadata: { name: "New Name" },
  created_at: "2026-02-11T13:15:00Z",
};

describe("useUpdateAgent", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("updates agent metadata", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({ data: mockUpdatedAgent });

    const { result } = renderHook(() => useUpdateAgent(), { wrapper });

    await act(async () => {
      await result.current.updateAgent({
        agentId: 42,
        metadata: { name: "New Name" },
      });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/agents/{agent_id}", {
      headers: { Authorization: "Bearer token" },
      params: { path: { agent_id: 42 } },
      body: { metadata: { name: "New Name" } },
    });
  });

  it("throws on failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: undefined,
      error: { error: { code: "agent_not_found", message: "Agent not found" } },
    });

    const { result } = renderHook(() => useUpdateAgent(), { wrapper });

    await expect(
      act(async () => {
        await result.current.updateAgent({
          agentId: 999,
          metadata: { name: "Test" },
        });
      }),
    ).rejects.toThrow("Failed to update agent");
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useUpdateAgent(), { wrapper });

    await expect(
      act(async () => {
        await result.current.updateAgent({
          agentId: 42,
          metadata: { name: "Test" },
        });
      }),
    ).rejects.toThrow("Not authenticated");
  });
});
