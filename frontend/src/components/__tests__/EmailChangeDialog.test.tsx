import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { AuthError } from "@supabase/supabase-js";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { mockAuth, mockUser, setupAuthMocks } from "../../auth/__tests__/fixtures";
import { EmailChangeDialog } from "../EmailChangeDialog";

vi.mock("../../lib/supabaseClient");
vi.mock("sonner");

describe("EmailChangeDialog", () => {
  const onOpenChange = vi.fn();

  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    // The auto-mock may not include updateUser; ensure it's a mock function.
    if (!mockAuth.updateUser) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (mockAuth as any).updateUser = vi.fn();
    }
    vi.mocked(mockAuth.updateUser).mockReset();
    vi.mocked(toast.error).mockClear();
    onOpenChange.mockClear();
  });

  it("shows the current email as disabled", () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    const currentEmailInput = screen.getByLabelText("Current email");
    expect(currentEmailInput).toHaveValue("test@example.com");
    expect(currentEmailInput).toBeDisabled();
  });

  it("shows the new email input", () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    expect(screen.getByLabelText("New email")).toBeInTheDocument();
  });

  it("disables submit button when new email is empty", () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    expect(screen.getByRole("button", { name: "Send Confirmation" })).toBeDisabled();
  });

  it("enables submit button when new email is entered", async () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await userEvent.type(screen.getByLabelText("New email"), "new@example.com");
    expect(screen.getByRole("button", { name: "Send Confirmation" })).toBeEnabled();
  });

  it("shows error when entering the same email", async () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await userEvent.type(screen.getByLabelText("New email"), "test@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send Confirmation" }));

    expect(toast.error).toHaveBeenCalledWith("That's already your current email address.");
    expect(mockAuth.updateUser).not.toHaveBeenCalled();
  });

  it("calls updateUser and shows success on valid submission", async () => {
    mockAuth.updateUser.mockResolvedValue({
      data: { user: mockUser },
      error: null,
    });

    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await userEvent.type(screen.getByLabelText("New email"), "new@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send Confirmation" }));

    await waitFor(() => {
      expect(mockAuth.updateUser).toHaveBeenCalledWith({ email: "new@example.com" });
    });

    // Should show the confirmation message
    await waitFor(() => {
      expect(screen.getByText(/confirmation link has been sent/i)).toBeInTheDocument();
    });
  });

  it("shows error toast when updateUser fails", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const authError = new AuthError("Rate limit exceeded", 429);
    mockAuth.updateUser.mockResolvedValue({
      data: { user: null },
      error: authError,
    });

    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await userEvent.type(screen.getByLabelText("New email"), "new@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send Confirmation" }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("Rate limit exceeded");
    });
    consoleSpy.mockRestore();
  });

  it("calls onOpenChange when Cancel is clicked", async () => {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not render when closed", () => {
    renderWithProviders(
      <EmailChangeDialog open={false} onOpenChange={onOpenChange} />
    );
    expect(screen.queryByText("Change Email Address")).not.toBeInTheDocument();
  });
});
