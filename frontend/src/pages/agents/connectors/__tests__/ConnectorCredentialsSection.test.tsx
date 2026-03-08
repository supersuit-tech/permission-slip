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

/**
 * Helper to mock GET responses by endpoint path.
 * Returns empty data for unrecognized paths.
 */
function setupMockGet(overrides: Record<string, unknown> = {}) {
  const defaults: Record<string, unknown> = {
    "/v1/credentials": { data: { credentials: [] } },
    "/v1/oauth/connections": { data: { connections: [] } },
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

  it("renders OAuth credential row with Connect button when not connected", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "square",
            auth_type: "oauth2" as const,
            oauth_provider: "square",
            oauth_scopes: ["ORDERS_READ", "PAYMENTS_READ"],
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Not configured")).toBeInTheDocument();
    });
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    expect(screen.getByText("Connect Square")).toBeInTheDocument();
  });

  it("renders OAuth credential row as connected when OAuth connection is active", async () => {
    setupMockGet({
      "/v1/oauth/connections": {
        data: {
          connections: [
            {
              provider: "square",
              status: "active",
              scopes: ["ORDERS_READ", "PAYMENTS_READ"],
              connected_at: "2026-03-01T12:00:00Z",
            },
          ],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "square",
            auth_type: "oauth2" as const,
            oauth_provider: "square",
            oauth_scopes: ["ORDERS_READ", "PAYMENTS_READ"],
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(
      screen.getByText((content) => content.includes("2 scope")),
    ).toBeInTheDocument();
  });

  it("shows re-authorize button when OAuth connection needs re-auth", async () => {
    setupMockGet({
      "/v1/oauth/connections": {
        data: {
          connections: [
            {
              provider: "square",
              status: "needs_reauth",
              scopes: ["ORDERS_READ"],
              connected_at: "2026-03-01T12:00:00Z",
            },
          ],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "square",
            auth_type: "oauth2" as const,
            oauth_provider: "square",
            oauth_scopes: ["ORDERS_READ"],
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Needs re-auth")).toBeInTheDocument();
    });
    expect(screen.getByText("Re-authorize")).toBeInTheDocument();
    expect(
      screen.getByText("Connection expired — please re-authorize"),
    ).toBeInTheDocument();
  });

  it("renders both OAuth and API key credentials for dual-auth connectors", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "square",
            auth_type: "oauth2" as const,
            oauth_provider: "square",
            oauth_scopes: ["ORDERS_READ"],
          },
          {
            service: "square_api_key",
            auth_type: "api_key" as const,
            instructions_url:
              "https://developer.squareup.com/docs/build-basics/access-tokens",
          },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect Square")).toBeInTheDocument();
    });
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    expect(screen.getByText("API Key")).toBeInTheDocument();
    // UX: "or" divider between auth methods
    expect(screen.getByText("or")).toBeInTheDocument();
    // UX: "Recommended" badge on OAuth row
    expect(screen.getByText("Recommended")).toBeInTheDocument();
  });

  it("displays friendly service name instead of raw ID for API key rows", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          {
            service: "square_api_key",
            auth_type: "api_key" as const,
          },
        ]}
      />,
    );

    await waitFor(() => {
      // "square_api_key" should render as "Square"
      expect(screen.getByText("Square")).toBeInTheDocument();
    });
    // Raw service ID should not appear
    expect(screen.queryByText("square_api_key")).not.toBeInTheDocument();
  });

  it("displays auth type badges for credential rows", async () => {
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("API Key")).toBeInTheDocument();
    });
  });
});
