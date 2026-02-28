import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockDelete, resetClientMocks } from "../../../api/__mocks__/client";
import { DangerZoneSection } from "../DangerZoneSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

describe("DangerZoneSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
    setupAuthMocks({ authenticated: true });
  });

  it("renders delete account button", () => {
    render(<DangerZoneSection />, { wrapper });

    expect(screen.getByText("Danger Zone")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Delete Account" }),
    ).toBeInTheDocument();
  });

  it("opens confirmation dialog when clicking delete", async () => {
    const user = userEvent.setup();

    render(<DangerZoneSection />, { wrapper });

    await user.click(screen.getByRole("button", { name: "Delete Account" }));

    await waitFor(() => {
      expect(screen.getByText("Are you sure?")).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/Type/)).toBeInTheDocument();
  });

  it("disables submit until DELETE is typed", async () => {
    const user = userEvent.setup();

    render(<DangerZoneSection />, { wrapper });

    await user.click(screen.getByRole("button", { name: "Delete Account" }));

    await waitFor(() => {
      expect(screen.getByText("Are you sure?")).toBeInTheDocument();
    });

    const submitButton = screen.getByRole("button", { name: "Delete My Account" });
    expect(submitButton).toBeDisabled();

    await user.type(screen.getByLabelText(/Type/), "DELETE");

    expect(submitButton).toBeEnabled();
  });

  it("calls delete API and signs out on confirmation", async () => {
    const user = userEvent.setup();
    mockDelete.mockResolvedValue({ data: { deleted: true }, error: null });

    render(<DangerZoneSection />, { wrapper });

    await user.click(screen.getByRole("button", { name: "Delete Account" }));

    await waitFor(() => {
      expect(screen.getByText("Are you sure?")).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText(/Type/), "DELETE");
    await user.click(screen.getByRole("button", { name: "Delete My Account" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/profile",
        expect.objectContaining({
          body: { confirmation: "DELETE" },
        }),
      );
    });
  });
});
