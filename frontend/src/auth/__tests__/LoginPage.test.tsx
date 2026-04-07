import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { renderWithProviders } from "../../test-helpers";
import { mockAuth, mockUser, mockSession, setupAuthMocks } from "./fixtures";
import LoginPage from "../LoginPage";

vi.mock("../../lib/supabaseClient");
vi.mock("../dev", () => ({
  fetchOtpFromMailpit: vi.fn().mockResolvedValue(null),
}));

describe("LoginPage", () => {
  beforeEach(() => {
    setupAuthMocks();
  });

  it("shows email step initially", async () => {
    renderWithProviders(<LoginPage />);
    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Code")).not.toBeInTheDocument();
  });

  it("transitions to OTP step after successful code send", async () => {
    mockAuth.signInWithOtp.mockResolvedValue({
      data: { user: null, session: null },
      error: null,
    });

    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(screen.getByLabelText("Code")).toBeInTheDocument();
    });
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("transitions to OTP step even on email rate limit", async () => {
    mockAuth.signInWithOtp.mockResolvedValue({
      data: { user: null, session: null },
      error: new AuthError("Rate limit", 429, "over_email_send_rate_limit"),
    });

    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(screen.getByLabelText("Code")).toBeInTheDocument();
    });
  });

  it("returns to email step when back is clicked", async () => {
    mockAuth.signInWithOtp.mockResolvedValue({
      data: { user: null, session: null },
      error: null,
    });

    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(screen.getByLabelText("Code")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText("Back"));

    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Code")).not.toBeInTheDocument();
  });

  it("calls verifyOtp with correct email and code", async () => {
    mockAuth.signInWithOtp.mockResolvedValue({
      data: { user: null, session: null },
      error: null,
    });
    mockAuth.verifyOtp.mockResolvedValue({
      data: { session: mockSession, user: mockUser },
      error: null,
    });

    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(screen.getByLabelText("Code")).toBeInTheDocument();
    });
    await userEvent.type(screen.getByLabelText("Code"), "12345678");
    await userEvent.click(screen.getByText("Verify"));

    expect(mockAuth.verifyOtp).toHaveBeenCalledWith({
      email: "test@example.com",
      token: "12345678",
      type: "email",
    });
  });

  it("stays on email step when send fails with non-rate-limit error", async () => {
    mockAuth.signInWithOtp.mockResolvedValue({
      data: { user: null, session: null },
      error: new AuthError("Server error", 500, "unexpected_failure"),
    });

    renderWithProviders(<LoginPage />);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(screen.getByLabelText("Email")).toBeInTheDocument();
    });
    expect(screen.queryByLabelText("Code")).not.toBeInTheDocument();
  });
});
