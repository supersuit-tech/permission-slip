import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useConnectors } from "../useConnectors";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockConnectorsResponse = {
  data: [
    {
      id: "github",
      name: "GitHub",
      description: "GitHub integration for repository management",
      actions: ["github.create_issue", "github.merge_pr"],
      required_credentials: ["github"],
    },
    {
      id: "gmail",
      name: "Gmail",
      description: "Send and manage emails via Gmail API",
      actions: ["email.send", "email.read"],
      required_credentials: ["gmail"],
    },
  ],
};

describe("useConnectors", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("fetches all available connectors", async () => {
    mockGet.mockResolvedValue({ data: mockConnectorsResponse });

    const { result } = renderHook(() => useConnectors(), { wrapper });

    await waitFor(() => {
      expect(result.current.connectors).toEqual(mockConnectorsResponse.data);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/connectors");
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("returns empty array on error", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useConnectors(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.connectors).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load connectors. Please try again later.",
    );
  });

  it("returns empty array when API returns error response", async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "server_error", message: "Internal error" } },
    });

    const { result } = renderHook(() => useConnectors(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.connectors).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load connectors. Please try again later.",
    );
  });
});
