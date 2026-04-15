import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "../../../test-helpers";
import { ReviewPendingAgentDialog } from "../ReviewPendingAgentDialog";
import type { Agent } from "../../../hooks/useAgents";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

// Use a large buffer (1 hour) to prevent flaky tests in slow CI environments
const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();

const pendingAgent: Agent = {
  agent_id: 2,
  status: "pending",
  metadata: { name: "My Bot" },
  confirmation_code: "XK7M9-PQRST",
  expires_at: futureDate,
  request_count_30d: 0,
  created_at: "2026-02-01T00:00:00Z",
};

const pendingAgentNoName: Agent = {
  agent_id: 7,
  status: "pending",
  confirmation_code: "QR5ST-6UVWX",
  expires_at: futureDate,
  request_count_30d: 0,
  created_at: "2026-02-01T00:00:00Z",
};

const pendingAgentNoCode: Agent = {
  agent_id: 9,
  status: "pending",
  metadata: { name: "No Code Bot" },
  request_count_30d: 0,
  created_at: "2026-02-01T00:00:00Z",
};

describe("ReviewPendingAgentDialog", () => {
  it("renders dialog title and description when open", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.getByText("Complete Agent Registration")).toBeInTheDocument();
    expect(
      screen.getByText("Tell your agent to run this command to finish registration."),
    ).toBeInTheDocument();
  });

  it("displays agent name and ID", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.getByText("My Bot")).toBeInTheDocument();
    expect(screen.getByText("ID: 2")).toBeInTheDocument();
  });

  it("falls back to Agent <id> when name is missing", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgentNoName}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.getByText("Agent 7")).toBeInTheDocument();
    // ID subtitle is hidden when agent has no real name (fallback display already contains the ID)
    expect(screen.queryByText("ID: 7")).not.toBeInTheDocument();
  });

  it("displays confirmation code with copy button", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.getByText("XK7M9-PQRST")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Copy confirmation code" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Confirmation code (included in the command below)"),
    ).toBeInTheDocument();
  });

  it("does not render confirmation code section when code is absent", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgentNoCode}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(
      screen.queryByText("Confirmation code (included in the command below)"),
    ).not.toBeInTheDocument();
  });

  it("displays expiry countdown", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={true}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.getByText(/Expires in/)).toBeInTheDocument();
  });

  it("calls onOpenChange(false) when Close button is clicked", async () => {
    const onOpenChange = vi.fn();
    const user = userEvent.setup();

    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={true}
        onOpenChange={onOpenChange}
      />,
    );

    await user.click(screen.getByTestId("review-pending-close"));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("does not render content when closed", () => {
    renderWithProviders(
      <ReviewPendingAgentDialog
        agent={pendingAgent}
        open={false}
        onOpenChange={() => {}}
      />,
    );

    expect(screen.queryByText("Complete Agent Registration")).not.toBeInTheDocument();
  });
});
