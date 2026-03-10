import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { AgentConfigHero } from "../AgentConfigHero";

function renderWithRouter(agentId: number) {
  return render(
    <MemoryRouter>
      <AgentConfigHero agentId={agentId} />
    </MemoryRouter>,
  );
}

describe("AgentConfigHero", () => {
  it("renders headline and description", () => {
    renderWithRouter(42);

    expect(
      screen.getByText(/Your agent is ready.*now give it superpowers/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Connect services like GitHub, Gmail, or Slack/),
    ).toBeInTheDocument();
  });

  it("renders two configuration steps", () => {
    renderWithRouter(42);

    expect(screen.getByText("Add a connector")).toBeInTheDocument();
    expect(screen.getByText("Set permissions")).toBeInTheDocument();
  });

  it("links CTA to the agent config page", () => {
    renderWithRouter(42);

    const link = screen.getByRole("link", { name: "Configure Your Agent" });
    expect(link).toHaveAttribute("href", "/agents/42");
  });
});
