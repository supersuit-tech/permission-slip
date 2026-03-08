import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockDelete,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { ConnectedAccountsSection } from "../ConnectedAccountsSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockApiFetch(
  connections: unknown[] = [],
  providers: unknown[] = [],
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/oauth/connections") {
      return Promise.resolve({ data: { connections } });
    }
    if (url === "/v1/oauth/providers") {
      return Promise.resolve({ data: { providers } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("ConnectedAccountsSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders the section title", async () => {
    mockApiFetch();

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Connected Accounts")).toBeInTheDocument();
    });
  });

  it("shows loading state", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<ConnectedAccountsSection />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading connected accounts" }),
    ).toBeInTheDocument();
  });

  it("shows empty state when no connections or providers", async () => {
    mockApiFetch([], []);

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/No OAuth providers are configured yet/),
      ).toBeInTheDocument();
    });
  });

  it("renders connected accounts", async () => {
    mockApiFetch(
      [
        {
          provider: "google",
          scopes: ["openid", "https://www.googleapis.com/auth/gmail.send"],
          status: "active",
          connected_at: "2026-03-05T10:00:00Z",
        },
      ],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Google")).toBeInTheDocument();
    });
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getByText(/2 scopes granted/)).toBeInTheDocument();
  });

  it("shows needs re-auth badge and re-authorize button", async () => {
    mockApiFetch(
      [
        {
          provider: "microsoft",
          scopes: ["openid"],
          status: "needs_reauth",
          connected_at: "2026-03-05T10:00:00Z",
        },
      ],
      [
        { id: "microsoft", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Needs Re-auth")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Re-authorize" }),
    ).toBeInTheDocument();
  });

  it("shows connect button for available providers", async () => {
    mockApiFetch(
      [],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
        { id: "microsoft", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Connect Google" }),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Connect Microsoft" }),
    ).toBeInTheDocument();
  });

  it("does not show connect button for already-connected providers", async () => {
    mockApiFetch(
      [
        {
          provider: "google",
          scopes: ["openid"],
          status: "active",
          connected_at: "2026-03-05T10:00:00Z",
        },
      ],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
        { id: "microsoft", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Google")).toBeInTheDocument();
    });
    // Google is already connected, should not show "Connect Google"
    expect(
      screen.queryByRole("button", { name: "Connect Google" }),
    ).not.toBeInTheDocument();
    // Microsoft is not connected, should show connect button
    expect(
      screen.getByRole("button", { name: "Connect Microsoft" }),
    ).toBeInTheDocument();
  });

  it("shows disconnect confirmation", async () => {
    mockApiFetch(
      [
        {
          provider: "google",
          scopes: ["openid"],
          status: "active",
          connected_at: "2026-03-05T10:00:00Z",
        },
      ],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );
    const user = userEvent.setup();

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Disconnect Google" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: "Disconnect Google" }),
    );

    expect(
      screen.getByRole("button", { name: "Disconnect" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("calls DELETE endpoint after disconnect confirmation", async () => {
    mockApiFetch(
      [
        {
          provider: "google",
          scopes: ["openid"],
          status: "active",
          connected_at: "2026-03-05T10:00:00Z",
        },
      ],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );
    mockDelete.mockResolvedValue({
      data: { provider: "google", disconnected_at: "2026-03-05T15:00:00Z" },
    });
    const user = userEvent.setup();

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Disconnect Google" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: "Disconnect Google" }),
    );
    await user.click(screen.getByRole("button", { name: "Disconnect" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/oauth/connections/{provider}",
        expect.objectContaining({
          params: { path: { provider: "google" } },
        }),
      );
    });
  });

  it("renders Linear connection with correct label", async () => {
    mockApiFetch(
      [
        {
          provider: "linear",
          scopes: ["read", "write"],
          status: "active",
          connected_at: "2026-03-07T10:00:00Z",
        },
      ],
      [
        { id: "linear", scopes: ["read", "write"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Linear")).toBeInTheDocument();
    });
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.getByText(/2 scopes granted/)).toBeInTheDocument();
  });

  it("shows connect button for Linear provider", async () => {
    mockApiFetch(
      [],
      [
        { id: "linear", scopes: ["read", "write"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Connect Linear" }),
      ).toBeInTheDocument();
    });
  });

  it("shows connection count badge", async () => {
    mockApiFetch(
      [
        {
          provider: "google",
          scopes: ["openid"],
          status: "active",
          connected_at: "2026-03-05T10:00:00Z",
        },
        {
          provider: "microsoft",
          scopes: ["openid"],
          status: "active",
          connected_at: "2026-03-05T11:00:00Z",
        },
      ],
      [
        { id: "google", scopes: ["openid"], source: "built_in", has_credentials: true },
        { id: "microsoft", scopes: ["openid"], source: "built_in", has_credentials: true },
      ],
    );

    render(<ConnectedAccountsSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("2")).toBeInTheDocument();
    });
  });
});
