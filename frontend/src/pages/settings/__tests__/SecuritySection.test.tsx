import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import type { Factor } from "@supabase/supabase-js";
import {
  setupAuthMocks,
  verifiedFactor,
} from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { SecuritySection } from "../SecuritySection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

/**
 * Sets up auth and API mocks for SecuritySection tests.
 * Pass `factors` to control what MFA factors are on the user (default: []).
 * AuthContext reads user.factors from React state, so factors must be set
 * on the session user rather than mocking supabase.auth.mfa.listFactors().
 */
function mockApiFetch({ factors = [] as Factor[] } = {}) {
  setupAuthMocks({ authenticated: true, factors });
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

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Two-Factor Authentication")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", () => {
    // SecuritySection starts in "loading" view before the first useEffect
    // fires and loadFactors() resolves. Even though listMfaFactors() now
    // resolves instantly from user state, the loading spinner is visible on
    // the initial render (before any effects run).
    mockApiFetch();

    render(<SecuritySection />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading security settings" }),
    ).toBeInTheDocument();
  });

  it("shows enrollment flow when no MFA factors exist", async () => {
    // No factors on user → SecuritySection should show the enrollment button.
    mockApiFetch({ factors: [] });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Set Up Authenticator App" }),
      ).toBeInTheDocument();
    });
  });

  it("shows enrolled factors with remove button", async () => {
    // User has a verified TOTP factor → SecuritySection should show it.
    mockApiFetch({ factors: [verifiedFactor] });

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Authenticator App")).toBeInTheDocument();
    });
    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
  });

  it("shows confirmation before removing MFA factor", async () => {
    mockApiFetch({ factors: [verifiedFactor] });
    const { supabase } = await import("../../../lib/supabaseClient");
    const user = userEvent.setup();

    render(<SecuritySection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    });

    // Click Remove - should show confirmation, not unenroll immediately.
    await user.click(screen.getByRole("button", { name: "Remove" }));

    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(vi.mocked(supabase.auth.mfa.unenroll)).not.toHaveBeenCalled();
  });

  it("cancels MFA removal when Cancel is clicked", async () => {
    mockApiFetch({ factors: [verifiedFactor] });
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
