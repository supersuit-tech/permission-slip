import type { ComponentProps } from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { MemoryRouter } from "react-router-dom";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import OtpStep from "../OtpStep";

vi.mock("../dev", () => ({
  fetchOtpFromMailpit: vi.fn().mockResolvedValue(null),
}));

const defaultProps: Pick<
  ComponentProps<typeof OtpStep>,
  "email" | "onVerify" | "onBack" | "onResend"
> = {
  email: "test@example.com",
  onVerify: vi.fn().mockResolvedValue({ error: null }),
  onBack: vi.fn(),
  onResend: vi.fn().mockResolvedValue({ error: null }),
};

function renderOtpStep(overrides: Partial<ComponentProps<typeof OtpStep>> = {}) {
  return render(
    <MemoryRouter>
      <CookieConsentProvider>
        <OtpStep {...defaultProps} {...overrides} />
      </CookieConsentProvider>
    </MemoryRouter>,
  );
}

describe("OtpStep", () => {
  it("shows email and code field", () => {
    renderOtpStep();
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
    expect(screen.getByLabelText("Code")).toBeInTheDocument();
    expect(screen.getByText("Verify")).toBeInTheDocument();
    expect(screen.getByText("Back")).toBeInTheDocument();
  });

  it("disables verify until full email OTP length is entered", async () => {
    renderOtpStep();
    const verify = screen.getByRole("button", { name: "Verify" });
    expect(verify).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Code"), "1234567");
    expect(verify).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Code"), "8");
    expect(verify).not.toBeDisabled();
  });

  it("calls onVerify with entered code", async () => {
    const onVerify = vi.fn().mockResolvedValue({ error: null });
    renderOtpStep({ onVerify });

    await userEvent.type(screen.getByLabelText("Code"), "12345678");
    await userEvent.click(screen.getByText("Verify"));

    expect(onVerify).toHaveBeenCalledWith("12345678");
  });

  it("shows error on verification failure", async () => {
    const onVerify = vi.fn().mockResolvedValue({
      error: new AuthError("Token expired", 401, "otp_expired"),
    });
    renderOtpStep({ onVerify });

    await userEvent.type(screen.getByLabelText("Code"), "00000000");
    await userEvent.click(screen.getByText("Verify"));

    await waitFor(() => {
      expect(
        screen.getByText(
          "Your code has expired. Please request a new one.",
          { exact: false }
        )
      ).toBeInTheDocument();
    });
  });

  it("calls onBack when back button is clicked", async () => {
    const onBack = vi.fn();
    renderOtpStep({ onBack });

    await userEvent.click(screen.getByText("Back"));

    expect(onBack).toHaveBeenCalled();
  });

  it("shows resend button", () => {
    renderOtpStep();
    const btn = screen.getByRole("button", { name: "Resend code" });
    expect(btn).toBeInTheDocument();
    expect(btn).not.toBeDisabled();
  });

  it("does not show password link when onUsePassword is omitted", () => {
    renderOtpStep();
    expect(
      screen.queryByRole("button", { name: "or sign in with password" }),
    ).not.toBeInTheDocument();
  });

  it("calls onUsePassword when password link is clicked", async () => {
    const onUsePassword = vi.fn();
    renderOtpStep({ onUsePassword });

    await userEvent.click(
      screen.getByRole("button", { name: "or sign in with password" }),
    );

    expect(onUsePassword).toHaveBeenCalled();
  });

  it("calls onResend when resend button is clicked", async () => {
    const onResend = vi.fn().mockResolvedValue({ error: null });
    renderOtpStep({ onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend code" }));

    expect(onResend).toHaveBeenCalled();
  });

  it("shows success message after resend", async () => {
    const onResend = vi.fn().mockResolvedValue({ error: null });
    renderOtpStep({ onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend code" }));

    await waitFor(() => {
      expect(screen.getByText("Code resent.")).toBeInTheDocument();
    });
  });

  it("treats email rate limit as success on resend", async () => {
    const onResend = vi.fn().mockResolvedValue({
      error: new AuthError("Rate limit", 429, "over_email_send_rate_limit"),
    });
    renderOtpStep({ onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend code" }));

    await waitFor(() => {
      expect(screen.getByText("Code resent.")).toBeInTheDocument();
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows error for non-rate-limit resend failures", async () => {
    const onResend = vi.fn().mockResolvedValue({
      error: new AuthError("Server error", 500, "unexpected_failure"),
    });
    renderOtpStep({ onResend });

    await userEvent.click(screen.getByRole("button", { name: "Resend code" }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
  });
});
