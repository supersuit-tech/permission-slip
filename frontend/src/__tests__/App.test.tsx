import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../test-helpers";
import { makeTestAccessToken, setupAuthMocks } from "../auth/__tests__/fixtures";
import App from "../App";

vi.mock("sonner");

vi.mock("../hooks/useProfile", () => ({
  useProfile: () => ({
    profile: {
      id: "user-123",
      username: "testuser",
      marketing_opt_in: false,
      created_at: "2024-01-01",
    },
    needsOnboarding: false,
    isLoading: false,
  }),
}));

describe("App", () => {
  beforeEach(() => {
    setupAuthMocks({ authenticated: false });
  });

  it("shows loading state while session refresh is pending", async () => {
    localStorage.setItem("ps_refresh_token", "pending-rt");
    let resolveRefresh!: (r: Response) => void;
    const refreshPromise = new Promise<Response>((resolve) => {
      resolveRefresh = resolve;
    });
    globalThis.fetch = vi.fn(() => refreshPromise) as typeof fetch;

    renderWithProviders(<App />);
    expect(screen.getByRole("status", { name: "Loading" })).toBeInTheDocument();

    resolveRefresh!(
      new Response(
        JSON.stringify({
          access_token: makeTestAccessToken("user-123", "t@e.com"),
          refresh_token: "rt2",
          expires_at: new Date(Date.now() + 3600_000).toISOString(),
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    await waitFor(() => {
      expect(screen.queryByRole("status", { name: "Loading" })).not.toBeInTheDocument();
    });
  });

  it("shows login page when not authenticated", async () => {
    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByText("Permission Slip")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("shows dashboard when authenticated", async () => {
    setupAuthMocks({ authenticated: true });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByText("Permission Slip")).toBeInTheDocument();
    });
    expect(screen.getByText("Registered Agents")).toBeInTheDocument();
  });

  it("shows user menu avatar when authenticated", async () => {
    setupAuthMocks({ authenticated: true });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });
  });

  it("calls logout via user menu", async () => {
    setupAuthMocks({ authenticated: true });

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/logout"),
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("shows toast error when signOut fails via user menu", async () => {
    setupAuthMocks({ authenticated: true, logoutStatus: 500 });
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    renderWithProviders(<App />);
    await waitFor(() => {
      expect(screen.getByLabelText("User menu")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByLabelText("User menu"));
    await userEvent.click(screen.getByText("Sign Out"));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Sign out failed. Please try again."
      );
    });
    consoleSpy.mockRestore();
  });
});
