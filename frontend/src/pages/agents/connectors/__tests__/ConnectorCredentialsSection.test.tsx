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

const requiredCredentials = [
  { service: "github", auth_type: "api_key" as const },
];

const storedCredentials = {
  credentials: [
    {
      id: "cred_123",
      service: "github",
      label: "Personal Access Token",
      created_at: "2026-02-11T10:00:00Z",
    },
  ],
};

const emptyOAuthConnections = { connections: [] };

/** Route mockGet responses based on the API path. */
function setupMockGet(overrides: Record<string, unknown> = {}) {
  const routes: Record<string, unknown> = {
    "/v1/credentials": { data: { credentials: [] } },
    "/v1/oauth/connections": { data: emptyOAuthConnections },
    ...overrides,
  };
  mockGet.mockImplementation((path: string) => {
    const response = routes[path];
    if (response) return Promise.resolve(response);
    return Promise.resolve({ data: null });
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
      <ConnectorCredentialsSection requiredCredentials={[]} />,
    );
    expect(
      screen.getByText("This connector does not require any credentials."),
    ).toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockGet.mockReturnValue(new Promise(() => {}));
    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
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
        requiredCredentials={requiredCredentials}
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
        requiredCredentials={requiredCredentials}
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
        requiredCredentials={requiredCredentials}
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
        service: "github",
        created_at: "2026-02-20T10:00:00Z",
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
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
          service: "github",
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
        requiredCredentials={requiredCredentials}
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
        requiredCredentials={requiredCredentials}
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

  it("renders OAuth connect button for oauth2 auth type", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "notion",
            auth_type: "oauth2" as const,
            oauth_provider: "notion",
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Notion (OAuth)")).toBeInTheDocument();
    });
    expect(screen.getByText("Connect Notion")).toBeInTheDocument();
    expect(screen.getByText("Not connected")).toBeInTheDocument();
  });

  it("shows connected status for active OAuth connection", async () => {
    setupMockGet({
      "/v1/oauth/connections": {
        data: {
          connections: [
            {
              provider: "notion",
              status: "active",
              scopes: [],
              connected_at: "2026-03-01T10:00:00Z",
            },
          ],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "notion",
            auth_type: "oauth2" as const,
            oauth_provider: "notion",
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(screen.getByText("Connected via OAuth")).toBeInTheDocument();
    expect(screen.queryByText("Connect Notion")).not.toBeInTheDocument();
  });

  it("renders both OAuth and API key for dual-auth connectors", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "notion",
            auth_type: "oauth2" as const,
            oauth_provider: "notion",
          },
          {
            service: "notion_api_key",
            auth_type: "api_key" as const,
            instructions_url:
              "https://developers.notion.com/docs/create-a-notion-integration",
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Notion (OAuth)")).toBeInTheDocument();
    });
    expect(screen.getByText("notion_api_key")).toBeInTheDocument();
    expect(screen.getByText("Connect Notion")).toBeInTheDocument();
  });
});
