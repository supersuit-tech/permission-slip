import { screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../test-helpers";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { AgentCredentialsSection } from "../AgentCredentialsSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockConnectors = [
  {
    id: "gmail",
    name: "Gmail",
    description: "Send and manage emails via Gmail API",
    status: "untested" as const,
    actions: ["email.send", "email.read"],
    required_credentials: ["gmail"],
    enabled_at: "2026-02-18T10:00:00Z",
  },
  {
    id: "github",
    name: "GitHub",
    description: "GitHub integration",
    status: "tested" as const,
    actions: ["github.create_issue"],
    required_credentials: ["github"],
    enabled_at: "2026-02-19T10:00:00Z",
  },
];

describe("AgentCredentialsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state", () => {
    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={[]}
        isLoading={true}
        error={null}
      />,
    );
    expect(
      screen.getByRole("status", { name: "Loading credentials" }),
    ).toBeInTheDocument();
  });

  it("shows error state", () => {
    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error="Something went wrong"
      />,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("shows empty state when no connectors are enabled", () => {
    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error={null}
      />,
    );
    expect(
      screen.getByText(
        /No connectors enabled — add connectors above to manage credentials/,
      ),
    ).toBeInTheDocument();
  });

  it("shows credential status for each required service", async () => {
    mockGet.mockResolvedValue({
      data: {
        credentials: [
          {
            id: "cred_123",
            service: "github",
            label: "My GitHub PAT",
            created_at: "2026-02-11T10:00:00Z",
          },
        ],
      },
    });

    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={mockConnectors}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("1 of 2 connected")).toBeInTheDocument();
    });

    expect(screen.getByText("github")).toBeInTheDocument();
    expect(screen.getByText("gmail")).toBeInTheDocument();
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getByText("Needs credentials")).toBeInTheDocument();
    expect(screen.getByText("Manage")).toBeInTheDocument();
    expect(screen.getByText("Connect")).toBeInTheDocument();
  });

  it("shows all connected when all credentials are present", async () => {
    mockGet.mockResolvedValue({
      data: {
        credentials: [
          {
            id: "cred_123",
            service: "github",
            created_at: "2026-02-11T10:00:00Z",
          },
          {
            id: "cred_456",
            service: "gmail",
            created_at: "2026-02-12T10:00:00Z",
          },
        ],
      },
    });

    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={mockConnectors}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("2 of 2 connected")).toBeInTheDocument();
    });
  });

  it("shows message when connectors require no credentials", () => {
    const noCredConnectors = [
      {
        id: "webhook",
        name: "Webhook",
        description: "Simple webhook",
        status: "untested" as const,
        actions: ["webhook.send"],
        required_credentials: [] as string[],
        enabled_at: "2026-02-18T10:00:00Z",
      },
    ];

    renderWithProviders(
      <AgentCredentialsSection
        agentId={42}
        connectors={noCredConnectors}
        isLoading={false}
        error={null}
      />,
    );

    expect(
      screen.getByText(
        "Enabled connectors do not require any credentials.",
      ),
    ).toBeInTheDocument();
  });
});
