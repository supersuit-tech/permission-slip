import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { MemoryRouter } from "react-router-dom";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import PasswordStep from "../PasswordStep";

const defaultProps = {
  email: "test@example.com",
  onSubmit: vi.fn().mockResolvedValue({ error: null }),
  onBack: vi.fn(),
};

function renderPasswordStep(props = defaultProps) {
  return render(
    <MemoryRouter>
      <CookieConsentProvider>
        <PasswordStep {...props} />
      </CookieConsentProvider>
    </MemoryRouter>,
  );
}

describe("PasswordStep", () => {
  it("shows email and password field", () => {
    renderPasswordStep();
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByText("Sign In")).toBeInTheDocument();
    expect(screen.getByText("Back")).toBeInTheDocument();
  });

  it("calls onSubmit with entered password", async () => {
    const onSubmit = vi.fn().mockResolvedValue({ error: null });
    renderPasswordStep({ ...defaultProps, onSubmit });

    await userEvent.type(screen.getByLabelText("Password"), "my-secret-pw");
    await userEvent.click(screen.getByText("Sign In"));

    expect(onSubmit).toHaveBeenCalledWith("my-secret-pw");
  });

  it("shows error on authentication failure", async () => {
    const onSubmit = vi.fn().mockResolvedValue({
      error: new AuthError(
        "Invalid credentials",
        400,
        "invalid_credentials",
      ),
    });
    renderPasswordStep({ ...defaultProps, onSubmit });

    await userEvent.type(screen.getByLabelText("Password"), "wrong-password");
    await userEvent.click(screen.getByText("Sign In"));

    await waitFor(() => {
      expect(
        screen.getByText("Invalid email or password. Please try again.", {
          exact: false,
        }),
      ).toBeInTheDocument();
    });
  });

  it("calls onBack when back button is clicked", async () => {
    const onBack = vi.fn();
    renderPasswordStep({ ...defaultProps, onBack });

    await userEvent.click(screen.getByText("Back"));

    expect(onBack).toHaveBeenCalled();
  });

  it("disables submit when password is empty", async () => {
    const onSubmit = vi.fn().mockResolvedValue({ error: null });
    renderPasswordStep({ ...defaultProps, onSubmit });

    // Password field is empty by default; the required attribute prevents form submission
    expect(screen.getByLabelText("Password")).toHaveValue("");
    await userEvent.click(screen.getByText("Sign In"));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
