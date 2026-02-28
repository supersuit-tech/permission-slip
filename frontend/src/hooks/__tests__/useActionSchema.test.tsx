import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useActionSchema } from "../useActionSchema";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockConnectorDetail = {
  id: "github",
  name: "GitHub",
  description: "GitHub integration",
  actions: [
    {
      action_type: "github.create_issue",
      name: "Create Issue",
      description: "Create a new issue",
      risk_level: "low",
      parameters_schema: {
        type: "object",
        required: ["owner", "repo", "title"],
        properties: {
          owner: { type: "string", description: "Repository owner" },
          repo: { type: "string", description: "Repository name" },
          title: { type: "string", description: "Issue title" },
        },
      },
    },
    {
      action_type: "github.merge_pr",
      name: "Merge Pull Request",
      risk_level: "medium",
    },
  ],
  required_credentials: [{ service: "github", auth_type: "api_key" }],
};

describe("useActionSchema", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns schema and action name for known action", async () => {
    mockGet.mockResolvedValue({ data: mockConnectorDetail });

    const { result } = renderHook(
      () => useActionSchema("github.create_issue"),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.schema).not.toBeNull();
    });

    expect(result.current.actionName).toBe("Create Issue");
    expect(result.current.schema?.required).toEqual([
      "owner",
      "repo",
      "title",
    ]);
    expect(result.current.schema?.properties?.owner?.description).toBe(
      "Repository owner",
    );
    expect(result.current.isLoading).toBe(false);
  });

  it("returns null schema when action has no parameters_schema", async () => {
    mockGet.mockResolvedValue({ data: mockConnectorDetail });

    const { result } = renderHook(
      () => useActionSchema("github.merge_pr"),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.actionName).toBe("Merge Pull Request");
    });

    expect(result.current.schema).toBeNull();
  });

  it("returns nulls when connector fetch fails", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(
      () => useActionSchema("github.create_issue"),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.schema).toBeNull();
    expect(result.current.actionName).toBeNull();
  });

  it("extracts connector ID from action type", async () => {
    mockGet.mockResolvedValue({ data: mockConnectorDetail });

    renderHook(() => useActionSchema("github.create_issue"), { wrapper });

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledWith(
        "/v1/connectors/{connector_id}",
        { params: { path: { connector_id: "github" } } },
      );
    });
  });

  it("returns nulls for unknown action type within known connector", async () => {
    mockGet.mockResolvedValue({ data: mockConnectorDetail });

    const { result } = renderHook(
      () => useActionSchema("github.unknown_action"),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.schema).toBeNull();
    expect(result.current.actionName).toBeNull();
  });
});
