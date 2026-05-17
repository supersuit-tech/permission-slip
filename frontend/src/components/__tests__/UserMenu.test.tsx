import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../test-helpers";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { UserMenu } from "../UserMenu";

vi.mock("sonner");
vi.mock("../../hooks/useProfile");

import { useProfile } from "../../hooks/useProfile";
const mockUseProfile = vi.mocked(useProfile);

describe("UserMenu", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: true });
    vi.mocked(toast.error).mockClear();
    mockUseProfile.mockReturnValue({
      profile: null,
      needsOnboarding: false,
      isLoading: false,
    });
    localStorage.removeItem("permission-slip-theme");
    document.documentElement.classList.remove("dark");
  });

  it("renders the avatar trigger", async () => {
    renderWithProviders(<UserMenu />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });
  });

  it("shows user email in the dropdown", async () => {
    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("shows username in the nav trigger when profile is loaded", async () => {
    mockUseProfile.mockReturnValue({
      profile: {
        id: "user-123",
        username: "janedoe",
        marketing_opt_in: false,
        created_at: "2024-01-01T00:00:00Z",
      },
      needsOnboarding: false,
      isLoading: false,
    });

    renderWithProviders(<UserMenu />);
    const trigger = screen.getByLabelText("User menu");
    expect(trigger).toHaveTextContent("janedoe");
  });

  it("shows username and email when profile is loaded", async () => {
    mockUseProfile.mockReturnValue({
      profile: {
        id: "user-123",
        username: "janedoe",
        marketing_opt_in: false,
        created_at: "2024-01-01T00:00:00Z",
      },
      needsOnboarding: false,
      isLoading: false,
    });

    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    expect(screen.getAllByText("janedoe").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("does not render username when profile is null", async () => {
    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    expect(screen.queryByText("janedoe")).not.toBeInTheDocument();
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("shows all menu items", async () => {
    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    expect(screen.getByText("Profile")).toBeInTheDocument();
    expect(screen.getByText("Security")).toBeInTheDocument();
    expect(screen.getByText("Billing")).toBeInTheDocument();
    expect(screen.getByText("Sign Out")).toBeInTheDocument();
  });

  it("does not show a theme toggle (moved to Profile)", async () => {
    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    expect(screen.queryByText(/dark mode/i)).not.toBeInTheDocument();
  });

  it("calls logout when Sign Out is clicked", async () => {
    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/logout"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("shows toast and logs error when signOut fails", async () => {
    setupAuthMocks({ authenticated: true, logoutStatus: 500 });
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    renderWithProviders(<UserMenu />);
    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Sign out failed. Please try again."
      );
    });
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });
});
