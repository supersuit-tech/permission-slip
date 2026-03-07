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

/** Set up mockGet to handle both credentials and OAuth connections endpoints. */
function setupMockGet(
  credentialsResponse: unknown = { data: { credentials: [] } },
  oauthConnectionsResponse: unknown = {
    data: { connections: [] },
  },
) {
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/oauth/connections") {
      return Promise.resolve(oauthConnectionsResponse);
    }
    // Default: credentials endpoint
    return Promise.resolve(credentialsResponse);
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
    setupMockGet({ data: storedCredentials });

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
    setupMockGet({ data: storedCredentials });

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
    setupMockGet({ data: storedCredentials });
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

  it("shows OAuth credential row with connect button", async () => {
    setupMockGet(
      { data: { credentials: [] } },
      { data: { connections: [] } },
    );

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "linear_oauth",
            auth_type: "oauth2" as const,
            oauth_provider: "linear",
            oauth_scopes: ["read", "write"],
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Linear OAuth")).toBeInTheDocument();
    });
    expect(screen.getByText("Not connected")).toBeInTheDocument();
    expect(screen.getByText("Connect Linear")).toBeInTheDocument();
  });

  it("shows OAuth as recommended when both auth types available", async () => {
    setupMockGet(
      { data: { credentials: [] } },
      { data: { connections: [] } },
    );

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "linear_oauth",
            auth_type: "oauth2" as const,
            oauth_provider: "linear",
            oauth_scopes: ["read", "write"],
          },
          {
            service: "linear",
            auth_type: "api_key" as const,
            instructions_url: "https://linear.app/docs",
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Recommended")).toBeInTheDocument();
    });
    expect(screen.getByText("Alternative")).toBeInTheDocument();
  });

  it("shows connected status for active OAuth connection", async () => {
    setupMockGet(
      { data: { credentials: [] } },
      {
        data: {
          connections: [
            {
              provider: "linear",
              status: "active",
              scopes: ["read", "write"],
              connected_at: "2026-03-01T10:00:00Z",
            },
          ],
        },
      },
    );

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "linear_oauth",
            auth_type: "oauth2" as const,
            oauth_provider: "linear",
            oauth_scopes: ["read", "write"],
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(screen.getByText("2 scopes")).toBeInTheDocument();
  });
});
