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

const defaultProps = {
  email: "test@example.com",
  onVerify: vi.fn().mockResolvedValue({ error: null }),
  onBack: vi.fn(),
};

function renderOtpStep(props = defaultProps) {
  return render(
    <MemoryRouter>
      <CookieConsentProvider>
        <OtpStep {...props} />
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

  it("calls onVerify with entered code", async () => {
    const onVerify = vi.fn().mockResolvedValue({ error: null });
    renderOtpStep({ ...defaultProps, onVerify });

    await userEvent.type(screen.getByLabelText("Code"), "123456");
    await userEvent.click(screen.getByText("Verify"));

    expect(onVerify).toHaveBeenCalledWith("123456");
  });

  it("shows error on verification failure", async () => {
    const onVerify = vi.fn().mockResolvedValue({
      error: new AuthError("Token expired", 401, "otp_expired"),
    });
    renderOtpStep({ ...defaultProps, onVerify });

    await userEvent.type(screen.getByLabelText("Code"), "000000");
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
    renderOtpStep({ ...defaultProps, onBack });

    await userEvent.click(screen.getByText("Back"));

    expect(onBack).toHaveBeenCalled();
  });
});
