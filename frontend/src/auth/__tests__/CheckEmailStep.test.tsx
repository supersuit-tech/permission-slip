import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { MemoryRouter } from "react-router-dom";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import CheckEmailStep from "../CheckEmailStep";

const defaultProps = {
  email: "test@example.com",
  onBack: vi.fn(),
  onResend: vi.fn().mockResolvedValue({ error: null }),
  resendCooldownSeconds: 0,
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

  it("shows resend email button when cooldown is 0", () => {
    renderCheckEmailStep({ ...defaultProps, resendCooldownSeconds: 0 });
    const btn = screen.getByRole("button", { name: "Resend email" });
    expect(btn).toBeInTheDocument();
    expect(btn).not.toBeDisabled();
  });

  it("disables resend button and shows countdown during cooldown", () => {
    renderCheckEmailStep({ ...defaultProps, resendCooldownSeconds: 30 });
    const btn = screen.getByRole("button", { name: "Resend email in 30s (on cooldown)" });
    expect(btn).toBeInTheDocument();
    expect(btn).toBeDisabled();
    expect(btn).toHaveTextContent("30s");
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

  it("shows context-specific error when rate-limited", async () => {
    const onResend = vi.fn().mockResolvedValue({
      error: new AuthError("Rate limit", 429, "over_email_send_rate_limit"),
    });
    renderCheckEmailStep({ ...defaultProps, onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => {
      expect(
        screen.getByText("Too many sign-in emails sent.", { exact: false })
      ).toBeInTheDocument();
      expect(
        screen.getByText("already received a link", { exact: false })
      ).toBeInTheDocument();
    });
  });

  it("clears success banner when cooldown expires", async () => {
    const onResend = vi.fn().mockResolvedValue({ error: null });
    const { rerender } = renderCheckEmailStep({ ...defaultProps, onResend, resendCooldownSeconds: 0 });

    await userEvent.click(screen.getByRole("button", { name: "Resend email" }));
    await waitFor(() => {
      expect(screen.getByText("Email resent.")).toBeInTheDocument();
    });

    rerender(
      <MemoryRouter>
        <CookieConsentProvider>
          <CheckEmailStep {...defaultProps} onResend={onResend} resendCooldownSeconds={5} />
        </CookieConsentProvider>
      </MemoryRouter>
    );
    rerender(
      <MemoryRouter>
        <CookieConsentProvider>
          <CheckEmailStep {...defaultProps} onResend={onResend} resendCooldownSeconds={0} />
        </CookieConsentProvider>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.queryByText("Email resent.")).not.toBeInTheDocument();
    });
  });
});
