import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../test-helpers";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPut,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { AgentConnectorsSection } from "../AgentConnectorsSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockEnabledConnectors = [
  {
    id: "gmail",
    name: "Gmail",
    description: "Send and manage emails via Gmail API",
    actions: ["email.send", "email.read"],
    required_credentials: ["gmail"],
    enabled_at: "2026-02-18T10:00:00Z",
  },
];

function mockAllConnectors(connectors = [
  {
    id: "gmail",
    name: "Gmail",
    description: "Send and manage emails via Gmail API",
    actions: ["email.send", "email.read"],
    required_credentials: ["gmail"],
  },
  {
    id: "github",
    name: "GitHub",
    description: "GitHub integration",
    actions: ["github.create_issue"],
    required_credentials: ["github"],
  },
  {
    id: "slack",
    name: "Slack",
    description: "Slack messaging",
    actions: ["slack.send_message"],
    required_credentials: ["slack"],
  },
]) {
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/connectors") {
      return Promise.resolve({ data: { data: connectors } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("AgentConnectorsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state", () => {
    mockAllConnectors();
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={true}
        error={null}
      />,
    );
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("shows error state", async () => {
    mockAllConnectors();
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error="Something went wrong"
      />,
    );
    await waitFor(() => {
      expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    });
  });

  it("shows all available connectors as cards", async () => {
    mockAllConnectors();
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockEnabledConnectors}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });
    expect(screen.getByText("GitHub")).toBeInTheDocument();
    expect(screen.getByText("Slack")).toBeInTheDocument();
  });

  it("shows Enabled badge on enabled connectors", async () => {
    mockAllConnectors();
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockEnabledConnectors}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });
    // Only Gmail is enabled
    expect(screen.getAllByText("Enabled")).toHaveLength(1);
  });

  it("shows empty state when no connectors available", async () => {
    mockAllConnectors([]);
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(
        screen.getByText("No connectors are available yet."),
      ).toBeInTheDocument();
    });
  });

  it("filters connectors by search", async () => {
    const user = userEvent.setup();
    // Need 4+ connectors to show search
    mockAllConnectors([
      { id: "gmail", name: "Gmail", description: "Email", actions: ["a"], required_credentials: [] },
      { id: "github", name: "GitHub", description: "Code", actions: ["b"], required_credentials: [] },
      { id: "slack", name: "Slack", description: "Chat", actions: ["c"], required_credentials: [] },
      { id: "stripe", name: "Stripe", description: "Payments", actions: ["d"], required_credentials: [] },
    ]);
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText("Search connectors...");
    await user.type(searchInput, "git");

    expect(screen.getByText("GitHub")).toBeInTheDocument();
    expect(screen.queryByText("Gmail")).not.toBeInTheDocument();
    expect(screen.queryByText("Slack")).not.toBeInTheDocument();
  });

  it("shows no-match message when search has no results", async () => {
    const user = userEvent.setup();
    mockAllConnectors([
      { id: "gmail", name: "Gmail", description: "Email", actions: ["a"], required_credentials: [] },
      { id: "github", name: "GitHub", description: "Code", actions: ["b"], required_credentials: [] },
      { id: "slack", name: "Slack", description: "Chat", actions: ["c"], required_credentials: [] },
      { id: "stripe", name: "Stripe", description: "Payments", actions: ["d"], required_credentials: [] },
    ]);
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText("Search connectors...");
    await user.type(searchInput, "zzz-no-match");

    expect(screen.getByText(/No connectors match/)).toBeInTheDocument();
  });

  it("enables and navigates when clicking a non-enabled connector", async () => {
    const user = userEvent.setup();
    mockAllConnectors();
    mockPut.mockResolvedValue({ data: {}, error: undefined });

    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockEnabledConnectors}
        isLoading={false}
        error={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("GitHub")).toBeInTheDocument();
    });

    // GitHub is not enabled — clicking it should call PUT to enable
    await user.click(screen.getByText("GitHub").closest("button")!);

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        "/v1/agents/{agent_id}/connectors/{connector_id}",
        expect.objectContaining({
          params: { path: { agent_id: 42, connector_id: "github" } },
        }),
      );
    });
  });
});
