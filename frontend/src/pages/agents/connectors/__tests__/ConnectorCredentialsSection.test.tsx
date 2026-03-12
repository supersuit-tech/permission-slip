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
  mockGet.mockImplementation((path: string) => {
    const match = defaults[path];
    if (match) return Promise.resolve(match);
    return Promise.resolve({ data: {} });
  });
}

describe("ConnectorCredentialsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows no credentials required message when empty", () => {
    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={[]}
      />,
    );
    expect(
      screen.getByText("This connector does not require any credentials."),
    ).toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockGet.mockReturnValue(new Promise(() => {}));
    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );
    expect(screen.getByText("Credentials")).toBeInTheDocument();
  });

  it("shows connected status with stored credentials", async () => {
    setupMockGet({
      "/v1/credentials": { data: storedCredentials },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    expect(screen.getByText("Add Another")).toBeInTheDocument();
  });

  it("shows not configured status when no credentials stored", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

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
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

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
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

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
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

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
        connectorId="github"
        requiredCredentials={apiKeyCredentials}
      />,
    );

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
        connectorId="jira"
        requiredCredentials={[
          { service: "jira", auth_type: "basic" as const },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));

    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByLabelText("Password / API Token")).toBeInTheDocument();
  });

  it("shows OAuth connect button for oauth2 credential", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={oauthCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    });
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    expect(screen.getByText("Recommended")).toBeInTheDocument();
  });

  it("shows OAuth credential row for Slack oauth2 auth type", async () => {
    setupMockGet({
      "/v1/oauth/providers": {
        data: {
          providers: [{ id: "slack", has_credentials: true }],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
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

    await waitFor(() => {
      expect(screen.getByText("OAuth")).toBeInTheDocument();
    });
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    expect(screen.getByText("Not connected")).toBeInTheDocument();
  });

  it("shows OAuth connected status when user has connection", async () => {
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

    await waitFor(() => {
      expect(screen.getByText(/^Connected/)).toBeInTheDocument();
    });
    expect(
      screen.getByLabelText("Disconnect Slack"),
    ).toBeInTheDocument();
  });

  it("shows both OAuth and API key options for mixed credentials", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        connectorId="github"
        requiredCredentials={mixedCredentials}
      />,
    );

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
