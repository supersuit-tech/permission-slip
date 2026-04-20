import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "../../../../auth/AuthContext";
import { ThemeProvider } from "../../../../components/ThemeContext";
import { Toaster } from "../../../../components/ui/sonner";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockPatch,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
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
    {
      action_type: "github.merge_pr",
      operation_type: "write",
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

const mockAgentResponse = {
  agent_id: 42,
  status: "registered",
  metadata: { name: "Test Agent" },
  created_at: "2026-02-10T10:00:00Z",
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
    mockPost.mockResolvedValue({ data: {} });
    mockPatch.mockResolvedValue({ data: {} });
    mockDelete.mockResolvedValue({ data: {} });
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state initially", () => {
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves
    renderPage();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders connector details after loading", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/agents/{agent_id}") {
        return Promise.resolve({ data: mockAgentResponse });
      }
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
      if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/instances") {
        return Promise.resolve({
          data: {
            data: [
              {
                connector_instance_id: "00000000-0000-0000-0000-000000000001",
                agent_id: 42,
                connector_id: "github",
                label: "GitHub",
                is_default: true,
                enabled_at: "2026-02-18T10:00:00Z",
              },
            ],
          },
        });
      }
      if (
        path ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential"
      ) {
        return Promise.resolve({
          data: {
            agent_id: 42,
            connector_id: "github",
            credential_id: "cred_123",
            oauth_connection_id: null,
          },
        });
      }
      if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential") {
        return Promise.resolve({
          data: {
            agent_id: 42,
            connector_id: "github",
            credential_id: "cred_123",
            oauth_connection_id: null,
          },
        });
      }
      if (path === "/v1/oauth/connections") {
        return Promise.resolve({ data: { connections: [] } });
      }
      if (path === "/v1/oauth/providers") {
        return Promise.resolve({ data: { providers: [] } });
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

    // Actions are behind a dialog — verify the trigger link is visible
    expect(
      screen.getByText("View all 2 available actions"),
    ).toBeInTheDocument();

    // Credentials section — content is behind "Manage credentials" modal
    const user = userEvent.setup();
    const manageBtn = await screen.findByRole("button", { name: /Manage credentials/i });
    await user.click(manageBtn);
    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });

    // Danger zone
    expect(screen.getByText("Danger Zone")).toBeInTheDocument();
    expect(screen.getByText("Disable")).toBeInTheDocument();
  });

  it("shows 'Agent not found' when user does not own the agent", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/agents/{agent_id}") {
        return Promise.resolve({ error: { message: "Not found" } });
      }
      // Connector detail is a public endpoint and still resolves,
      // but the ownership gate should prevent rendering it.
      if (path === "/v1/connectors/{connector_id}") {
        return Promise.resolve({ data: mockDetailResponse });
      }
      if (path === "/v1/agents/{agent_id}/connectors") {
        return Promise.resolve({ error: { message: "Not found" } });
      }
      if (path === "/v1/action-configurations") {
        return Promise.resolve({ data: mockActionConfigsResponse });
      }
      return Promise.resolve({ data: {} });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Agent not found.")).toBeInTheDocument();
    });
  });

  it("shows error when connector not found", async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === "/v1/agents/{agent_id}") {
        return Promise.resolve({ data: mockAgentResponse });
      }
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
      if (path === "/v1/agents/{agent_id}") {
        return Promise.resolve({ data: mockAgentResponse });
      }
      if (path === "/v1/connectors/{connector_id}") {
        return Promise.resolve({ data: mockDetailResponse });
      }
      if (path === "/v1/agents/{agent_id}/connectors") {
        return Promise.resolve({ data: mockAgentConnectorsResponse });
      }
      if (path === "/v1/credentials") {
        return Promise.resolve({ data: { credentials: [] } });
      }
      if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/instances") {
        return Promise.resolve({
          data: {
            data: [
              {
                connector_instance_id: "00000000-0000-0000-0000-000000000001",
                agent_id: 42,
                connector_id: "github",
                label: "GitHub",
                is_default: true,
                enabled_at: "2026-02-18T10:00:00Z",
              },
            ],
          },
        });
      }
      if (
        path ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential"
      ) {
        return Promise.resolve({
          data: {
            agent_id: 42,
            connector_id: "github",
            credential_id: null,
            oauth_connection_id: null,
          },
        });
      }
      if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential") {
        return Promise.resolve({
          data: {
            agent_id: 42,
            connector_id: "github",
            credential_id: null,
            oauth_connection_id: null,
          },
        });
      }
      if (path === "/v1/oauth/connections") {
        return Promise.resolve({ data: { connections: [] } });
      }
      if (path === "/v1/oauth/providers") {
        return Promise.resolve({ data: { providers: [] } });
      }
      return Promise.resolve({ data: {} });
    });

    const user = userEvent.setup();
    renderPage();

    const manageBtn = await screen.findByRole("button", { name: /Manage credentials/i });
    await user.click(manageBtn);

    await waitFor(() => {
      expect(screen.getByText("Not configured")).toBeInTheDocument();
    });
  });
});
