import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { mockAuth, setupAuthMocks } from "./fixtures";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { supabase } from "../../lib/supabaseClient";
import OnboardingPage from "../OnboardingPage";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockUpdateUser = vi.mocked(supabase.auth.updateUser);

describe("OnboardingPage", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    resetClientMocks();
  });

  it("renders username field, password field, and both buttons", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByText("Create account")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("renders the terms of service agreement checkbox", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getAllByRole("checkbox")).toHaveLength(2);
    });
    expect(screen.getByText(/I agree to the/)).toBeInTheDocument();
  });

  it("renders the marketing opt-in checkbox", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByText(/Keep me in the loop/)).toBeInTheDocument();
    });
  });

  it("disables submit button until terms checkbox is checked and password meets minimum length", async () => {
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });

    const submitButton = screen.getByText("Create account");
    expect(submitButton).toBeDisabled();

    // Check terms — still disabled because password is empty
    const tosCheckbox = screen.getByLabelText(/I agree to the/);
    await userEvent.click(tosCheckbox);
    expect(submitButton).toBeDisabled();

    // Enter a password that meets the minimum length
    await userEvent.type(screen.getByLabelText("Password"), "securepass");
    expect(submitButton).toBeEnabled();
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

  it("submits username and sets password on form submit", async () => {
    mockPost.mockResolvedValue({ data: { id: "1", username: "alice", created_at: "2024-01-01" }, error: undefined });
    mockUpdateUser.mockResolvedValue({ data: { user: {} as never }, error: null });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "alice");
    await userEvent.type(screen.getByLabelText("Password"), "securepass");
    await userEvent.click(screen.getByLabelText(/I agree to the/));
    await userEvent.click(screen.getByText("Create account"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/onboarding",
        expect.objectContaining({ body: { username: "alice", marketing_opt_in: false } })
      );
    });
    await waitFor(() => {
      expect(mockUpdateUser).toHaveBeenCalledWith({ password: "securepass" });
    });
  });

  it("sends marketing_opt_in=true when checkbox is checked", async () => {
    mockPost.mockResolvedValue({ data: { id: "1", username: "bob", created_at: "2024-01-01" }, error: undefined });
    mockUpdateUser.mockResolvedValue({ data: { user: {} as never }, error: null });
    renderWithProviders(<OnboardingPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Username")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Username"), "bob");
    await userEvent.type(screen.getByLabelText("Password"), "securepass");
    await userEvent.click(screen.getByLabelText(/I agree to the/));
    await userEvent.click(screen.getByLabelText(/Keep me in the loop/));
    await userEvent.click(screen.getByText("Create account"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/onboarding",
        expect.objectContaining({ body: { username: "bob", marketing_opt_in: true } })
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
    await userEvent.type(screen.getByLabelText("Password"), "securepass");
    await userEvent.click(screen.getByLabelText(/I agree to the/));
    await userEvent.click(screen.getByText("Create account"));

    await waitFor(() => {
      expect(screen.getByText("Username already taken")).toBeInTheDocument();
    });
  });
});
