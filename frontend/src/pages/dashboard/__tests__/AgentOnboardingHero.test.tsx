import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AgentOnboardingHero } from "../AgentOnboardingHero";

describe("AgentOnboardingHero", () => {
  it("renders headline and description", () => {
    render(<AgentOnboardingHero onRegisterAgent={vi.fn()} />);

    expect(
      screen.getByText("Control what Openclaw can do"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Connect Openclaw to get started/),
    ).toBeInTheDocument();
  });

  it("renders three onboarding steps", () => {
    render(<AgentOnboardingHero onRegisterAgent={vi.fn()} />);

    expect(screen.getAllByText("Connect Openclaw").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Set permissions")).toBeInTheDocument();
    expect(screen.getByText("Monitor activity")).toBeInTheDocument();
  });

  it("calls onRegisterAgent when CTA is clicked", async () => {
    const user = userEvent.setup();
    const onRegisterAgent = vi.fn();
    render(<AgentOnboardingHero onRegisterAgent={onRegisterAgent} />);

    await user.click(
      screen.getByRole("button", { name: "Connect Openclaw" }),
    );

    expect(onRegisterAgent).toHaveBeenCalledOnce();
  });
});
