import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { mockAuth, setupAuthMocks } from "./fixtures";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import OnboardingPage from "../OnboardingPage";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("OnboardingPage", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    resetClientMocks();
  });

  it("renders username field and both buttons", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    expect(screen.getByText("Create account")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("calls signOut when Cancel is clicked", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByText("Cancel"));

    expect(mockAuth.signOut).toHaveBeenCalled();
  });

  it("submits username on form submit", async () => {
    mockPost.mockResolvedValue({ data: { id: "1", username: "alice", created_at: "2024-01-01" }, error: undefined });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "alice");
    await userEvent.click(screen.getByText("Create account"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/onboarding",
        expect.objectContaining({ body: { username: "alice" } })
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
    await userEvent.click(screen.getByText("Create account"));

    await waitFor(() => {
      expect(screen.getByText("Username already taken")).toBeInTheDocument();
    });
  });
});
