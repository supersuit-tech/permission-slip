import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { EmailChangeDialog } from "../EmailChangeDialog";

vi.mock("sonner");

describe("EmailChangeDialog", () => {
  const onOpenChange = vi.fn();

  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    vi.mocked(toast.error).mockClear();
    onOpenChange.mockClear();
  });

  async function renderOpenDialog() {
    renderWithProviders(
      <EmailChangeDialog open={true} onOpenChange={onOpenChange} />
    );
    await waitFor(() => {
      expect(screen.getByLabelText("Current email")).toHaveValue("test@example.com");
    });
  }

  it("shows the current email as disabled", async () => {
    await renderOpenDialog();
    const currentEmailInput = screen.getByLabelText("Current email");
    expect(currentEmailInput).toHaveValue("test@example.com");
    expect(currentEmailInput).toBeDisabled();
  });

  it("shows the new email input", async () => {
    await renderOpenDialog();
    expect(screen.getByLabelText("New email")).toBeInTheDocument();
  });

  it("disables submit button when new email is empty", async () => {
    await renderOpenDialog();
    expect(screen.getByRole("button", { name: "Send Confirmation" })).toBeDisabled();
  });

  it("enables submit button when new email is entered", async () => {
    await renderOpenDialog();
    await userEvent.type(screen.getByLabelText("New email"), "new@example.com");
    expect(screen.getByRole("button", { name: "Send Confirmation" })).toBeEnabled();
  });

  it("shows error when entering the same email", async () => {
    await renderOpenDialog();
    await userEvent.type(screen.getByLabelText("New email"), "test@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send Confirmation" }));

    expect(toast.error).toHaveBeenCalledWith("That's already your current email address.");
  });

  it("shows error toast when email change is not available", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    await renderOpenDialog();
    await userEvent.type(screen.getByLabelText("New email"), "new@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send Confirmation" }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("Email change is not available.");
    });
    consoleSpy.mockRestore();
  });

  it("calls onOpenChange when Cancel is clicked", async () => {
    await renderOpenDialog();
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
