import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthError } from "@supabase/supabase-js";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../test-helpers";
import {
  aalResponse,
  mockAuth,
  mockMfa,
  setupAuthMocks,
} from "../auth/__tests__/fixtures";
import App from "../App";

vi.mock("../lib/supabaseClient");
vi.mock("sonner");

// Mock useProfile so App tests don't need a real API server.
// When authenticated, simulate a profile existing (no onboarding needed).
vi.mock("../hooks/useProfile", () => ({
  useProfile: () => ({
    profile: { id: "user-123", username: "testuser", marketing_opt_in: false, created_at: "2024-01-01" },
    needsOnboarding: false,
    isLoading: false,
  }),
}));

describe("App", () => {
  beforeEach(() => {
    setupAuthMocks();
  });

  it("shows loading state initially", () => {
    // Override setupAuthMocks: don't fire callback so auth stays in "loading"
    mockAuth.onAuthStateChange.mockReturnValue({
      data: {
        subscription: { id: "test", callback: vi.fn(), unsubscribe: vi.fn() },
      },
    });
    renderWithProviders(<App />);
    expect(screen.getByRole("status", { name: "Loading" })).toBeInTheDocument();
  });

  it("shows login page when not authenticated", async () => {
    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByText("Permission Slip")).toBeInTheDocument();
    });
  });

  it("shows dashboard when authenticated", async () => {
    setupAuthMocks({ authenticated: true });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByText("Permission Slip")).toBeInTheDocument();
    });
    // Without API mocks, agent queries fail so Dashboard shows the agents card
    expect(screen.getByText("Registered Agents")).toBeInTheDocument();
  });

  it("shows user menu avatar when authenticated", async () => {
    setupAuthMocks({ authenticated: true });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });
  });

  it("calls signOut via user menu", async () => {
    setupAuthMocks({ authenticated: true });
    mockAuth.signOut.mockResolvedValue({ error: null });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));
    expect(mockAuth.signOut).toHaveBeenCalled();
  });

  it("shows toast error when signOut fails via user menu", async () => {
    setupAuthMocks({ authenticated: true });
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockAuth.signOut.mockResolvedValue({
      error: new AuthError("Sign out failed", 500),
    });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Sign out failed. Please try again."
      );
    });
    consoleSpy.mockRestore();
  });

  describe("MFA gating", () => {
    beforeEach(() => {
      setupAuthMocks({ authenticated: true });
      mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
        aalResponse("aal1", "aal2")
      );
    });

    it("shows MfaChallengePage when authStatus is mfa_required", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        expect(
          screen.getByLabelText("Authenticator Code")
        ).toBeInTheDocument();
      });
      expect(screen.queryByText("Recent Activity")).not.toBeInTheDocument();
    });

    it("does not show dashboard content when MFA is required", async () => {
      renderWithProviders(<App />);

      await waitFor(() => {
        expect(
          screen.getByText(
            "Enter the 6-digit code from your authenticator app to continue."
          )
        ).toBeInTheDocument();
      });
      expect(screen.queryByLabelText("User menu")).not.toBeInTheDocument();
    });
  });
});
