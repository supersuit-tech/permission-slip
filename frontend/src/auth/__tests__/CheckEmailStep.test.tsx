import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { MemoryRouter } from "react-router-dom";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import CheckEmailStep from "../CheckEmailStep";

const defaultProps = {
  email: "test@example.com",
  onBack: vi.fn(),
  onResend: vi.fn().mockResolvedValue({ error: null }),
};

function renderCheckEmailStep(props = defaultProps) {
  return render(
    <MemoryRouter>
      <CookieConsentProvider>
        <CheckEmailStep {...props} />
      </CookieConsentProvider>
    </MemoryRouter>,
  );
}

describe("CheckEmailStep", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    defaultProps.onResend.mockResolvedValue({ error: null });
  });

  it("shows email, sign-in link message, and spam hint", () => {
    renderCheckEmailStep();
    expect(screen.getByText("Check your email")).toBeInTheDocument();
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
    expect(screen.getByText(/sign-in link/)).toBeInTheDocument();
    expect(screen.getByText(/spam folder/)).toBeInTheDocument();
    expect(screen.getByText("Back")).toBeInTheDocument();
  });

  it("does not show a code input", () => {
    renderCheckEmailStep();
    expect(screen.queryByLabelText("Code")).not.toBeInTheDocument();
  });

  it("calls onBack when back button is clicked", async () => {
    const onBack = vi.fn();
    renderCheckEmailStep({ ...defaultProps, onBack });

    await userEvent.click(screen.getByText("Back"));

    expect(onBack).toHaveBeenCalled();
  });

  it("shows resend email button", () => {
    renderCheckEmailStep();
    const btn = screen.getByRole("button", { name: "Resend email" });
    expect(btn).toBeInTheDocument();
    expect(btn).not.toBeDisabled();
  });

  it("calls onResend when resend button is clicked", async () => {
    const onResend = vi.fn().mockResolvedValue({ error: null });
    renderCheckEmailStep({ ...defaultProps, onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));

    expect(onResend).toHaveBeenCalled();
  });

  it("shows success message after resend", async () => {
    const onResend = vi.fn().mockResolvedValue({ error: null });
    renderCheckEmailStep({ ...defaultProps, onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => {
      expect(screen.getByText("Email resent.")).toBeInTheDocument();
    });
  });

  it("treats email rate limit as success on resend", async () => {
    const onResend = vi.fn().mockResolvedValue({
      error: new AuthError("Rate limit", 429, "over_email_send_rate_limit"),
    });
    renderCheckEmailStep({ ...defaultProps, onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => {
      expect(screen.getByText("Email resent.")).toBeInTheDocument();
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows error for non-rate-limit resend failures", async () => {
    const onResend = vi.fn().mockResolvedValue({
      error: new AuthError("Server error", 500, "unexpected_failure"),
    });
    renderCheckEmailStep({ ...defaultProps, onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
  });
});
