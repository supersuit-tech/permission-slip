import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthError } from "@supabase/supabase-js";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import {
  aalResponse,
  mockAuth,
  mockMfa,
  mockSession,
  setupAuthMocks,
  verifiedFactor,
} from "./fixtures";
import MfaChallengePage from "../MfaChallengePage";

vi.mock("../../lib/supabaseClient");
vi.mock("sonner");

describe("MfaChallengePage", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    // Simulate a user who has enrolled MFA but hasn't completed the second
    // factor yet — this is what triggers the MFA challenge page.
    mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
      aalResponse("aal1", "aal2")
    );
  });

  it("renders the authenticator code form", async () => {
    renderWithProviders(<MfaChallengePage />);
    await waitFor(() => {
      expect(
        screen.getByLabelText("Authenticator Code")
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Verify" })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sign Out" })
    ).toBeInTheDocument();
  });

  it("submits the TOTP code via verifyMfa", async () => {
    // verifyMfa reads from user.factors in React state; set up the factor
    // on the user via setupAuthMocks instead of mocking listFactors.
    setupAuthMocks({ authenticated: true, factors: [verifiedFactor] });
    mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
      aalResponse("aal1", "aal2")
    );
    mockMfa.challengeAndVerify.mockResolvedValue({
      data: mockSession,
      error: null,
    });

    renderWithProviders(<MfaChallengePage />);

    const input = await screen.findByLabelText("Authenticator Code");
    await userEvent.type(input, "123456");
    await userEvent.click(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() => {
      expect(mockMfa.challengeAndVerify).toHaveBeenCalledWith({
        factorId: "factor-1",
        code: "123456",
      });
    });
  });

  it("shows error message on verification failure", async () => {
    // verifyMfa reads from user.factors in React state; set up the factor
    // on the user via setupAuthMocks instead of mocking listFactors.
    setupAuthMocks({ authenticated: true, factors: [verifiedFactor] });
    mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
      aalResponse("aal1", "aal2")
    );
    mockMfa.challengeAndVerify.mockResolvedValue({
      data: null,
      error: new AuthError("Invalid code", 400, "mfa_verification_failed"),
    });

    renderWithProviders(<MfaChallengePage />);

    const input = await screen.findByLabelText("Authenticator Code");
    await userEvent.type(input, "000000");
    await userEvent.click(screen.getByRole("button", { name: "Verify" }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
  });

  it("calls signOut when Sign Out button is clicked", async () => {
    mockAuth.signOut.mockResolvedValue({ error: null });

    renderWithProviders(<MfaChallengePage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Sign Out" })
      ).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: "Sign Out" }));

    await waitFor(() => {
      expect(mockAuth.signOut).toHaveBeenCalled();
    });
  });

  it("shows toast error when signOut fails", async () => {
    const consoleSpy = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    mockAuth.signOut.mockResolvedValue({
      error: new AuthError("Sign out failed", 500),
    });

    renderWithProviders(<MfaChallengePage />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Sign Out" })
      ).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: "Sign Out" }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Sign out failed. Please try again."
      );
    });
    consoleSpy.mockRestore();
  });

  it("strips non-digit characters from input", async () => {
    renderWithProviders(<MfaChallengePage />);

    const input = await screen.findByLabelText("Authenticator Code");
    await userEvent.type(input, "12ab34");

    expect(input).toHaveValue("1234");
  });
});
