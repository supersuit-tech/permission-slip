import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { UpgradeSuccessBanner } from "../UpgradeSuccessBanner";

describe("UpgradeSuccessBanner", () => {
  it("shows confirmed message when upgraded is true", () => {
    render(<UpgradeSuccessBanner onDismiss={() => {}} upgraded={true} />);

    expect(screen.getByText("Welcome to Pay-as-you-go!")).toBeInTheDocument();
    expect(
      screen.getByText(/unlimited agents, credentials, and standing approvals/),
    ).toBeInTheDocument();
  });

  it("shows activating message when upgraded is false", () => {
    render(<UpgradeSuccessBanner onDismiss={() => {}} upgraded={false} />);

    expect(screen.getByText("Activating your upgrade…")).toBeInTheDocument();
    expect(
      screen.getByText(/Your payment was received/),
    ).toBeInTheDocument();
  });

  it("has accessible status role", () => {
    render(<UpgradeSuccessBanner onDismiss={() => {}} upgraded={false} />);

    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("calls onDismiss when close button is clicked", async () => {
    const onDismiss = vi.fn();
    const user = userEvent.setup();

    render(<UpgradeSuccessBanner onDismiss={onDismiss} upgraded={true} />);

    await user.click(
      screen.getByRole("button", { name: "Dismiss success message" }),
    );
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("shows spinner icon when pending", () => {
    const { container } = render(
      <UpgradeSuccessBanner onDismiss={() => {}} upgraded={false} />,
    );

    // Loader2 has animate-spin class
    const spinner = container.querySelector(".animate-spin");
    expect(spinner).toBeInTheDocument();
  });

  it("shows checkmark icon when confirmed", () => {
    const { container } = render(
      <UpgradeSuccessBanner onDismiss={() => {}} upgraded={true} />,
    );

    // CheckCircle2 should be present (no spinner)
    const spinner = container.querySelector(".animate-spin");
    expect(spinner).not.toBeInTheDocument();
  });
});
