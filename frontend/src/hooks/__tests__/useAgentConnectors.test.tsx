import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useAgentConnectors } from "../useAgentConnectors";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockConnectorsResponse = {
  data: [
    {
      id: "gmail",
      name: "Gmail",
      description: "Send and manage emails via Gmail API",
      actions: ["email.send", "email.read"],
      required_credentials: ["gmail"],
      enabled_at: "2026-02-18T10:00:00Z",
    },
  ],
};

describe("useAgentConnectors", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns empty connectors when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useAgentConnectors(42), { wrapper });

    expect(result.current.connectors).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches connectors when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockConnectorsResponse });

    const { result } = renderHook(() => useAgentConnectors(42), { wrapper });

    await waitFor(() => {
      expect(result.current.connectors).toEqual(
        mockConnectorsResponse.data,
      );
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/agents/{agent_id}/connectors", {
      headers: { Authorization: "Bearer token" },
      params: { path: { agent_id: 42 } },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useAgentConnectors(42), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.connectors).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load connectors. Please try again later.",
    );
  });

  it("does not fetch for invalid agent ID", () => {
    setupAuthMocks({ authenticated: true });

    const { result } = renderHook(() => useAgentConnectors(0), { wrapper });

    expect(result.current.isLoading).toBe(false);
    expect(mockGet).not.toHaveBeenCalled();
  });
});
