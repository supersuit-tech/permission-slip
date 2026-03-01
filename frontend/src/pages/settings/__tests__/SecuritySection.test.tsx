import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  setupAuthMocks,
  mockMfa,
  verifiedFactor,
} from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { SecuritySection } from "../SecuritySection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockApiFetch() {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/profile") {
      return Promise.resolve({
        data: {
          id: "user-123",
          username: "alice",
          marketing_opt_in: false,
          created_at: "2026-01-01T00:00:00Z",
        },
        response: { status: 200 },
      });
    }
    return Promise.resolve({ data: null });
  });
}

describe("SecuritySection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders the security card with title", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [], totp: [] },
      error: null,
    });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Security")).toBeInTheDocument();
    });
    expect(
      screen.getByText("Manage your account security and authentication methods."),
    ).toBeInTheDocument();
  });

  it("shows Two-Factor Authentication heading", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [], totp: [] },
      error: null,
    });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Two-Factor Authentication")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", () => {
    mockApiFetch();
    mockMfa.listFactors.mockReturnValue(new Promise(() => {}));

    render(<SecuritySection />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading security settings" }),
    ).toBeInTheDocument();
  });

  it("shows enrollment flow when no MFA factors exist", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [], totp: [] },
      error: null,
    });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Set Up Authenticator App" }),
      ).toBeInTheDocument();
    });
  });

  it("shows enrolled factors with remove button", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [verifiedFactor], totp: [verifiedFactor] },
      error: null,
    });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Authenticator App")).toBeInTheDocument();
    });
    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
  });

  it("shows error state with retry button", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: null,
      error: { message: "Network error" },
    });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Failed to load security settings."),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Try Again" })).toBeInTheDocument();
  });

  it("shows confirmation before removing MFA factor", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [verifiedFactor], totp: [verifiedFactor] },
      error: null,
    });
    const user = userEvent.setup();

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    });

    // Click Remove - should show confirmation, not unenroll immediately.
    await user.click(screen.getByRole("button", { name: "Remove" }));

    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(mockMfa.unenroll).not.toHaveBeenCalled();
  });

  it("cancels MFA removal when Cancel is clicked", async () => {
    mockApiFetch();
    mockMfa.listFactors.mockResolvedValue({
      data: { all: [verifiedFactor], totp: [verifiedFactor] },
      error: null,
    });
    const user = userEvent.setup();

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Remove" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    // Should be back to the Remove button.
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
  });
});
