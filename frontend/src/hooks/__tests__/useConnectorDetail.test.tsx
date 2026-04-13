import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useConnectorDetail } from "../useConnectorDetail";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockDetailResponse = {
  id: "github",
  name: "GitHub",
  description: "GitHub integration for repository management",
  actions: [
    {
      action_type: "github.create_issue",
      operation_type: "write",
      name: "Create Issue",
      description: "Create a new issue in a repository",
      risk_level: "low",
      parameters_schema: {
        type: "object",
        required: ["repo", "title"],
        properties: {
          repo: { type: "string" },
          title: { type: "string" },
        },
      },
    },
  ],
  required_credentials: [{ service: "github", auth_type: "api_key" }],
};

describe("useConnectorDetail", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("fetches connector details", async () => {
    mockGet.mockResolvedValue({ data: mockDetailResponse });

    const { result } = renderHook(() => useConnectorDetail("github"), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.connector).toEqual(mockDetailResponse);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/connectors/{connector_id}", {
      params: { path: { connector_id: "github" } },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("does not fetch for empty connector ID", () => {
    const { result } = renderHook(() => useConnectorDetail(""), {
      wrapper,
    });

    expect(result.current.isLoading).toBe(false);
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("sets error on fetch failure", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useConnectorDetail("github"), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.connector).toBeNull();
    expect(result.current.error).toBe(
      "Unable to load connector details. Please try again later.",
    );
  });
});
