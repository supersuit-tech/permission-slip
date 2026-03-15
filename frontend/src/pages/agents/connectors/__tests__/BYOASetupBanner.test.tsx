import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ConnectorCredentialsSection } from "../ConnectorCredentialsSection";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const salesforceOAuth = [
  {
    service: "salesforce",
    auth_type: "oauth2" as const,
    oauth_provider: "salesforce",
    oauth_scopes: ["api", "refresh_token"],
  },
];

function setupMockGet(overrides: Record<string, unknown> = {}) {
  const defaults: Record<string, unknown> = {
    "/v1/credentials": { data: { credentials: [] } },
    "/v1/oauth/connections": { data: { connections: [] } },
    "/v1/oauth/providers": {
      data: {
        providers: [
          { id: "salesforce", has_credentials: false, source: "manifest" },
        ],
      },
    },
    ...overrides,
  };
  mockGet.mockImplementation((path: string, ..._args: unknown[]) => {
    if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential") {
      return Promise.resolve({
        data: {
          agent_id: 42,
          connector_id: "salesforce",
          credential_id: null,
          oauth_connection_id: null,
        },
      });
    }
    const match = defaults[path];
    if (match) return Promise.resolve(match);
    return Promise.resolve({ data: {} });
  });
}

async function openManageModal(user: ReturnType<typeof userEvent.setup>) {
  const btn = await screen.findByRole("button", {
    name: /Manage credentials/i,
  });
  await user.click(btn);
}

describe("BYOASetupBanner", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows BYOA setup banner when provider has_credentials is false", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(
        screen.getByText("OAuth app setup required"),
      ).toBeInTheDocument();
    });

    // Verify step-by-step guide is shown
    expect(screen.getByText("Create an OAuth app")).toBeInTheDocument();
    expect(
      screen.getByText("Enter your Client ID and Client Secret"),
    ).toBeInTheDocument();
    expect(screen.getByText("Connect your account")).toBeInTheDocument();

    // Connect button should be disabled
    expect(
      screen.getByRole("button", { name: /Connect Salesforce/ }),
    ).toBeDisabled();
  });

  it("shows developer console link for known providers", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(
        screen.getByText("OAuth app setup required"),
      ).toBeInTheDocument();
    });

    const link = screen.getByRole("link", {
      name: /Open Salesforce developer console/,
    });
    expect(link).toHaveAttribute(
      "href",
      "https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_oauth_and_connected_apps.htm",
    );
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("opens BYOAConfigDialog from the banner", async () => {
    const user = userEvent.setup();
    setupMockGet();

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(
        screen.getByText("OAuth app setup required"),
      ).toBeInTheDocument();
    });

    // Click the configure credentials button in the banner
    await user.click(
      screen.getByRole("button", { name: /Configure credentials/ }),
    );

    // BYOAConfigDialog should appear
    expect(
      screen.getByText("Configure Salesforce OAuth App"),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Client ID")).toBeInTheDocument();
    expect(screen.getByLabelText("Client Secret")).toBeInTheDocument();
  });

  it("saves BYOA credentials through the config dialog", async () => {
    const user = userEvent.setup();
    setupMockGet();
    mockPost.mockResolvedValue({
      data: {
        provider: "salesforce",
        created_at: "2026-03-15T10:00:00Z",
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(
        screen.getByText("OAuth app setup required"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: /Configure credentials/ }),
    );

    await user.type(screen.getByLabelText("Client ID"), "test-client-id");
    await user.type(
      screen.getByLabelText("Client Secret"),
      "test-client-secret",
    );
    await user.click(screen.getByRole("button", { name: "Save Credentials" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith("/v1/oauth/provider-configs", {
        headers: { Authorization: "Bearer token" },
        body: {
          provider: "salesforce",
          client_id: "test-client-id",
          client_secret: "test-client-secret",
        },
      });
    });

    // Dialog should close after successful save
    await waitFor(() => {
      expect(
        screen.queryByText("Configure Salesforce OAuth App"),
      ).not.toBeInTheDocument();
    });
  });

  it("hides banner and enables Connect after saving credentials", async () => {
    const user = userEvent.setup();

    // Track calls to the providers endpoint so we can switch the response
    // after the BYOA config save triggers a React Query invalidation.
    let providerCallCount = 0;
    const defaults: Record<string, unknown> = {
      "/v1/credentials": { data: { credentials: [] } },
      "/v1/oauth/connections": { data: { connections: [] } },
      "/v1/oauth/provider-configs": { data: { configs: [] } },
    };
    mockGet.mockImplementation((path: string, ..._args: unknown[]) => {
      if (
        path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential"
      ) {
        return Promise.resolve({
          data: {
            agent_id: 42,
            connector_id: "salesforce",
            credential_id: null,
            oauth_connection_id: null,
          },
        });
      }
      if (path === "/v1/oauth/providers") {
        providerCallCount++;
        // After the first fetch, return has_credentials: true
        // (simulating React Query refetch after invalidation)
        const hasCredentials = providerCallCount > 1;
        return Promise.resolve({
          data: {
            providers: [
              {
                id: "salesforce",
                has_credentials: hasCredentials,
                source: hasCredentials ? "byoa" : "manifest",
              },
            ],
          },
        });
      }
      const match = defaults[path];
      if (match) return Promise.resolve(match);
      return Promise.resolve({ data: {} });
    });

    mockPost.mockResolvedValue({
      data: {
        provider: "salesforce",
        created_at: "2026-03-15T10:00:00Z",
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    // Banner should be visible initially
    await waitFor(() => {
      expect(
        screen.getByText("OAuth app setup required"),
      ).toBeInTheDocument();
    });

    // Save credentials through the dialog
    await user.click(
      screen.getByRole("button", { name: /Configure credentials/ }),
    );
    await user.type(screen.getByLabelText("Client ID"), "my-client-id");
    await user.type(screen.getByLabelText("Client Secret"), "my-secret");
    await user.click(screen.getByRole("button", { name: "Save Credentials" }));

    // After save + React Query invalidation, banner should disappear
    // and Connect button should be enabled
    await waitFor(() => {
      expect(
        screen.queryByText("OAuth app setup required"),
      ).not.toBeInTheDocument();
    });

    const connectBtn = screen.getByRole("button", {
      name: /Connect Salesforce/,
    });
    expect(connectBtn).not.toBeDisabled();
  });

  it("does not show banner when provider has credentials", async () => {
    const user = userEvent.setup();
    setupMockGet({
      "/v1/oauth/providers": {
        data: {
          providers: [
            { id: "salesforce", has_credentials: true, source: "byoa" },
          ],
        },
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        agentId={42}
        connectorId="salesforce"
        requiredCredentials={salesforceOAuth}
      />,
    );

    await openManageModal(user);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Connect Salesforce/ }),
      ).toBeInTheDocument();
    });

    // Banner should NOT be shown
    expect(
      screen.queryByText("OAuth app setup required"),
    ).not.toBeInTheDocument();
  });
});
