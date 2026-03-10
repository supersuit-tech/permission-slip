import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { MemoryRouter } from "react-router-dom";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import EmailStep from "../EmailStep";

type OnSubmit = (email: string) => Promise<{ error: AuthError | null }>;

function renderEmailStep(onSubmit: OnSubmit) {
  return render(
    <MemoryRouter>
      <CookieConsentProvider>
        <EmailStep onSubmit={onSubmit} />
      </CookieConsentProvider>
    </MemoryRouter>,
  );
}

describe("EmailStep", () => {
  it("renders email field and continue button", () => {
    renderEmailStep(vi.fn<OnSubmit>());
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByText("Continue")).toBeInTheDocument();
  });

  it("calls onSubmit with entered email", async () => {
    const onSubmit = vi.fn<OnSubmit>().mockResolvedValue({ error: null });
    renderEmailStep(onSubmit);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    expect(onSubmit).toHaveBeenCalledWith("test@example.com");
  });

  it("shows error on failure", async () => {
    const onSubmit = vi.fn<OnSubmit>().mockResolvedValue({
      error: new AuthError(
        "Rate limit",
        429,
        "over_email_send_rate_limit"
      ),
    });
    renderEmailStep(onSubmit);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(
        screen.getByText(
          "Too many login emails sent. Please wait",
          { exact: false }
        )
      ).toBeInTheDocument();
    });
  });

  it("does not submit when email is empty", async () => {
    const onSubmit = vi.fn<OnSubmit>();
    renderEmailStep(onSubmit);

    await userEvent.click(screen.getByText("Continue"));

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("shows generic error for unknown errors", async () => {
    const onSubmit = vi.fn<OnSubmit>().mockResolvedValue({
      error: new AuthError("Some internal detail", 500),
    });
    renderEmailStep(onSubmit);

    await userEvent.type(screen.getByLabelText("Email"), "test@example.com");
    await userEvent.click(screen.getByText("Continue"));

    await waitFor(() => {
      expect(
        screen.getByText("Something went wrong. Please try again.", {
          exact: false,
        })
      ).toBeInTheDocument();
    });
  });
});
