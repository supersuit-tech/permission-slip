import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "../../../../auth/AuthContext";
import { ThemeProvider } from "../../../../components/ThemeContext";
import { Toaster } from "../../../../components/ui/sonner";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import { mockGet, resetClientMocks } from "../../../../api/__mocks__/client";
import { ConnectorConfigPage } from "../ConnectorConfigPage";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const mockDetailResponse = {
  id: "github",
  name: "GitHub",
  description: "GitHub integration for repository management",
  actions: [
    {
      action_type: "github.create_issue",
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
    {
      action_type: "github.merge_pr",
      name: "Merge Pull Request",
      description: "Merge an open pull request",
      risk_level: "high",
      parameters_schema: {
        type: "object",
        required: ["repo", "pr"],
        properties: {
          repo: { type: "string" },
          pr: { type: "integer" },
        },
      },
    },
  ],
  required_credentials: [{ service: "github", auth_type: "api_key" }],
};

const mockAgentConnectorsResponse = {
  data: [
    {
      id: "github",
      name: "GitHub",
      description: "GitHub integration",
      actions: ["github.create_issue", "github.merge_pr"],
      required_credentials: ["github"],
      enabled_at: "2026-02-18T10:00:00Z",
    },
  ],
};

const mockCredentialsResponse = {
  credentials: [
    {
      id: "cred_123",
      service: "github",
      label: "Personal Access Token",
      created_at: "2026-02-11T10:00:00Z",
    },
  ],
};

const mockActionConfigsResponse = {
  data: [],
};

function renderPage(route = "/agents/42/connectors/github") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={[route]}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <AuthProvider>
            <Routes>
              <Route
                path="/agents/:agentId/connectors/:connectorId"
                element={<ConnectorConfigPage />}
              />
            </Routes>
            <Toaster />
          </AuthProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("ConnectorConfigPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state initially", () => {
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves
    renderPage();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders connector details after loading", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/connectors/{connector_id}") {
        return Promise.resolve({ data: mockDetailResponse });
      }
      if (path === "/v1/agents/{agent_id}/connectors") {
        return Promise.resolve({ data: mockAgentConnectorsResponse });
      }
      if (path === "/v1/credentials") {
        return Promise.resolve({ data: mockCredentialsResponse });
      }
      if (path === "/v1/action-configurations") {
        return Promise.resolve({ data: mockActionConfigsResponse });
      }
      return Promise.resolve({ data: {} });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("GitHub")).toBeInTheDocument();
    });

    // Overview section
    expect(
      screen.getByText("GitHub integration for repository management"),
    ).toBeInTheDocument();

    // Actions section
    expect(screen.getByText("Create Issue")).toBeInTheDocument();
    expect(screen.getByText("Merge Pull Request")).toBeInTheDocument();
    expect(screen.getByText("Low")).toBeInTheDocument();
    expect(screen.getByText("High")).toBeInTheDocument();

    // Credentials section (loads asynchronously via useCredentials)
    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });

    // Danger zone
    expect(screen.getByText("Danger Zone")).toBeInTheDocument();
    expect(screen.getByText("Disable")).toBeInTheDocument();
  });

  it("shows error when connector not found", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/connectors/{connector_id}") {
        return Promise.reject(new Error("Not found"));
      }
      if (path === "/v1/agents/{agent_id}/connectors") {
        return Promise.resolve({ data: mockAgentConnectorsResponse });
      }
      return Promise.resolve({ data: {} });
    });

    renderPage();

    await waitFor(() => {
      expect(
        screen.getByText(
          "Unable to load connector details. Please try again later.",
        ),
      ).toBeInTheDocument();
    });

    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("shows credentials as not configured when none stored", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/connectors/{connector_id}") {
        return Promise.resolve({ data: mockDetailResponse });
      }
      if (path === "/v1/agents/{agent_id}/connectors") {
        return Promise.resolve({ data: mockAgentConnectorsResponse });
      }
      if (path === "/v1/credentials") {
        return Promise.resolve({ data: { credentials: [] } });
      }
      return Promise.resolve({ data: {} });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Not configured")).toBeInTheDocument();
    });
  });
});
