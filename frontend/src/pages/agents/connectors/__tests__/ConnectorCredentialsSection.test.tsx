import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ConnectorCredentialsSection } from "../ConnectorCredentialsSection";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const apiKeyCredentials = [
  { service: "github_pat", auth_type: "api_key" as const },
];

const oauthCredentials = [
  {
    service: "github",
    auth_type: "oauth2" as const,
    oauth_provider: "github",
    oauth_scopes: ["repo"],
  },
];

const mixedCredentials = [
  {
    service: "github",
    auth_type: "oauth2" as const,
    oauth_provider: "github",
    oauth_scopes: ["repo"],
  },
  {
    service: "github_pat",
    auth_type: "api_key" as const,
    instructions_url:
      "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens",
  },
];

const storedCredentials = {
  credentials: [
    {
      id: "cred_123",
      service: "github_pat",
      label: "Personal Access Token",
      created_at: "2026-02-11T10:00:00Z",
    },
  ],
};

/**
 * Helper to mock GET responses by endpoint path.
 * Returns empty data for unrecognized paths.
 */
function setupMockGet(overrides: Record<string, unknown> = {}) {
  const defaults: Record<string, unknown> = {
    "/v1/credentials": { data: { credentials: [] } },
    "/v1/oauth/connections": { data: { connections: [] } },
    "/v1/oauth/providers": { data: { providers: [] } },
    ...overrides,
  };
  mockGet.mockImplementation((path: string, ..._args: unknown[]) => {
    // Match the agent connector credential path pattern
    if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential") {
      return Promise.resolve({
        data: { agent_id: 42, connector_id: "github", credential_id: null, oauth_connection_id: null },
      });
    }
    const match = defaults[path];
    if (match) return Promise.resolve(match);
    return Promise.resolve({ data: {} });
  });
}

/** Open the "Manage credentials" modal */
async function openManageModal(user: ReturnType<typeof userEvent.setup>) {
  const btn = await screen.findByRole("button", { name: /Manage credentials/i });
  await user.click(btn);
}

describe("ConnectorCredentialsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("does not show manage button when no credentials required", () => {
    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={[]}
      />,
    );
    expect(
      screen.queryByRole("button", { name: /Manage credentials/i }),
    ).not.toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockGet.mockReturnValue(new Promise(() => {}));
    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );
    expect(screen.getByText("Credentials")).toBeInTheDocument();
  });

  it("shows connected status with stored credentials", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/credentials": { data: storedCredentials },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    expect(screen.getByText("Add Another")).toBeInTheDocument();
  });

  it("shows not configured status when no credentials stored", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Not configured")).toBeInTheDocument();
    });
    expect(screen.getByText("Connect")).toBeInTheDocument();
  });

  it("opens Add Credential dialog", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));

    expect(screen.getByText("Add Credential")).toBeInTheDocument();
    expect(screen.getByLabelText("API Key")).toBeInTheDocument();
  });

  it("stores credential through Add dialog", async () => {
    const user = userEvent.setup();
    setupMockGet();
    mockPost.mockResolvedValue({
      data: {
        id: "cred_new",
        service: "github_pat",
        created_at: "2026-02-20T10:00:00Z",
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));
    await user.type(screen.getByLabelText("API Key"), "ghp_test_key");
    await user.click(screen.getByText("Store Credential"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith("/v1/credentials", {
        headers: { Authorization: "Bearer token" },
        body: {
          service: "github_pat",
          credentials: { api_key: "ghp_test_key" },
        },
      });
    });
  });

  it("opens Remove Credential dialog", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/credentials": { data: storedCredentials },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Remove credential Personal Access Token"),
    );

    expect(screen.getByText("Remove Credential")).toBeInTheDocument();
    expect(
      screen.getByText(/This will permanently delete/),
    ).toBeInTheDocument();
  });

  it("deletes credential through Remove dialog", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/credentials": { data: storedCredentials },
    });
    mockDelete.mockResolvedValue({
      data: { id: "cred_123", deleted_at: "2026-02-20T10:00:00Z" },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Remove credential Personal Access Token"),
    );
    await user.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/credentials/{credential_id}",
        {
          headers: { Authorization: "Bearer token" },
          params: { path: { credential_id: "cred_123" } },
        },
      );
    });
  });

  it("renders basic auth fields for basic auth type", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="jira"
        requiredCredentials={[
          { service: "jira", auth_type: "basic" as const },
        ]}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));

    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByLabelText("Password / API Token")).toBeInTheDocument();
  });

  it("shows OAuth connect button for oauth2 credential", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={oauthCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    });
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    expect(screen.getByText("Recommended")).toBeInTheDocument();
  });

  it("shows OAuth credential row for Slack oauth2 auth type", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/oauth/providers": {
        data: {
          providers: [{ id: "slack", has_credentials: true }],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2" as const,
            oauth_provider: "slack",
            oauth_scopes: ["chat:write", "channels:read"],
          },
        ]}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("OAuth")).toBeInTheDocument();
    });
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    expect(screen.getByText("Not connected")).toBeInTheDocument();
  });

  it("shows OAuth connected status when user has connection", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/oauth/connections": {
        data: {
          connections: [
            {
              provider: "slack",
              status: "active",
              scopes: ["chat:write"],
              connected_at: "2026-03-01T10:00:00Z",
            },
          ],
        },
      },
      "/v1/oauth/providers": {
        data: {
          providers: [{ id: "slack", has_credentials: true }],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2" as const,
            oauth_provider: "slack",
            oauth_scopes: ["chat:write"],
          },
        ]}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText(/^Connected/)).toBeInTheDocument();
    });
    expect(
      screen.getByLabelText("Disconnect Slack"),
    ).toBeInTheDocument();
  });

  it("shows re-authorization state for expired connection", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/oauth/connections": {
        data: {
          connections: [
            {
              provider: "slack",
              status: "needs_reauth",
              scopes: ["chat:write"],
              connected_at: "2026-01-01T10:00:00Z",
            },
          ],
        },
      },
      "/v1/oauth/providers": {
        data: { providers: [{ id: "slack", has_credentials: true }] },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[{
          service: "slack",
          auth_type: "oauth2" as const,
          oauth_provider: "slack",
        }]}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Re-authorization required")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: /Re-authorize/ })).toBeInTheDocument();
  });

  it("shows both OAuth and API key options for mixed credentials", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="github"
        requiredCredentials={mixedCredentials}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    });
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    expect(screen.getByText("Alternative")).toBeInTheDocument();
    // Service name should be human-readable, not raw ID
    expect(
      screen.getByText("GitHub Personal Access Token"),
    ).toBeInTheDocument();
  });
});
