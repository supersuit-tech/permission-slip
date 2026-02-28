import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../test-helpers";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import {
  mockGet,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { AgentConnectorsSection } from "../AgentConnectorsSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockConnectors = [
  {
    id: "gmail",
    name: "Gmail",
    description: "Send and manage emails via Gmail API",
    actions: ["email.send", "email.read"],
    required_credentials: ["gmail"],
    enabled_at: "2026-02-18T10:00:00Z",
  },
  {
    id: "github",
    name: "GitHub",
    description: "GitHub integration",
    actions: ["github.create_issue"],
    required_credentials: ["github"],
    enabled_at: "2026-02-19T10:00:00Z",
  },
];

describe("AgentConnectorsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state", () => {
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

  it("shows error state", () => {
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error="Something went wrong"
      />,
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("shows empty state with Add Connector CTA", () => {
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={[]}
        isLoading={false}
        error={null}
      />,
    );
    expect(screen.getByText("No connectors enabled")).toBeInTheDocument();
    // One in the header, one in the empty state
    const addButtons = screen.getAllByText("Add Connector");
    expect(addButtons.length).toBeGreaterThanOrEqual(2);
  });

  it("renders connector rows with Configure and Remove buttons", () => {
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockConnectors}
        isLoading={false}
        error={null}
      />,
    );
    expect(screen.getByText("Gmail")).toBeInTheDocument();
    expect(screen.getByText("GitHub")).toBeInTheDocument();
    expect(screen.getAllByText("Configure")).toHaveLength(2);
    expect(screen.getAllByText("Remove")).toHaveLength(2);
  });

  it("opens Add Connector dialog", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            id: "slack",
            name: "Slack",
            description: "Slack messaging",
            actions: ["slack.send_message"],
            required_credentials: ["slack"],
          },
        ],
      },
    });

    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockConnectors}
        isLoading={false}
        error={null}
      />,
    );

    const addButton = screen.getAllByText("Add Connector")[0];
    if (addButton) {
      await user.click(addButton);
    }
    expect(
      screen.getByText(
        "Enable a connector for this agent to allow it to submit actions from external services.",
      ),
    ).toBeInTheDocument();
  });

  it("opens Remove confirmation dialog", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <AgentConnectorsSection
        agentId={42}
        connectors={mockConnectors}
        isLoading={false}
        error={null}
      />,
    );

    const removeButtons = screen.getAllByText("Remove");
    const firstRemove = removeButtons[0];
    if (firstRemove) {
      await user.click(firstRemove);
    }
    expect(screen.getByText("Remove Gmail")).toBeInTheDocument();
  });
});
