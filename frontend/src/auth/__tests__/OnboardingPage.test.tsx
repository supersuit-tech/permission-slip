import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { setupAuthMocks } from "./fixtures";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import OnboardingPage from "../OnboardingPage";

vi.mock("../../api/client");

describe("OnboardingPage", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    resetClientMocks();
  });

  it("renders username field and continue button", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("renders the terms of service agreement checkbox", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getAllByRole("checkbox")).toHaveLength(2);
    });
    expect(screen.getByText(/I confirm I am authorized/)).toBeInTheDocument();
  });

  it("renders the marketing opt-in checkbox", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByText(/Keep me in the loop/)).toBeInTheDocument();
    });
  });

  it("disables continue until terms checkbox is checked", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });

    const continueButton = screen.getByRole("button", { name: "Continue" });
    expect(continueButton).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Username"), "alice");
    expect(continueButton).toBeDisabled();

    await userEvent.click(screen.getByLabelText(/I confirm I am authorized/));
    expect(continueButton).toBeEnabled();
  });

  it("calls logout when Cancel is clicked", async () => {
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByText("Cancel"));

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/logout"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("submits username on form submit", async () => {
    mockPost.mockResolvedValue({
      data: { id: "1", username: "alice", created_at: "2024-01-01" },
      error: undefined,
    });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "alice");
    await userEvent.click(screen.getByLabelText(/I confirm I am authorized/));
    await userEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/onboarding",
        expect.objectContaining({
          body: { username: "alice", marketing_opt_in: false },
        })
      );
    });
  });

  it("sends marketing_opt_in=true when checkbox is checked", async () => {
    mockPost.mockResolvedValue({
      data: { id: "1", username: "bob", created_at: "2024-01-01" },
      error: undefined,
    });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "bob");
    await userEvent.click(screen.getByLabelText(/I confirm I am authorized/));
    await userEvent.click(screen.getByLabelText(/Keep me in the loop/));
    await userEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/onboarding",
        expect.objectContaining({
          body: { username: "bob", marketing_opt_in: true },
        })
      );
    });
  });

  it("shows error when API returns an error", async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { message: "Username already taken" } },
    });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "taken");
    await userEvent.click(screen.getByLabelText(/I confirm I am authorized/));
    await userEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => {
      expect(screen.getByText("Username already taken")).toBeInTheDocument();
    });
  });
});
