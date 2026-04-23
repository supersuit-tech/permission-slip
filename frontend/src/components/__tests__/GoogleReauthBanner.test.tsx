import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { GoogleReauthBanner } from "../GoogleReauthBanner";

const saasMode = vi.hoisted(() => ({ isSaas: true }));

vi.mock("@/lib/saas", () => ({
  get isSaas() {
    return saasMode.isSaas;
  },
}));

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

function mockConnections(connections: unknown[]) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/oauth/connections") {
      return Promise.resolve({ data: { connections } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("GoogleReauthBanner", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    sessionStorage.clear();
    saasMode.isSaas = true;
    wrapper = createAuthWrapper(["/"]);
  });

  it("renders nothing for self-hosted when Google needs reauth", async () => {
    saasMode.isSaas = false;
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    const { container } = render(<GoogleReauthBanner />, { wrapper });
    await waitFor(() => {
      expect(container.firstChild).toBeNull();
    });
  });

  it("renders nothing when there are no Google connections", async () => {
    mockConnections([]);
    const { container } = render(<GoogleReauthBanner />, { wrapper });
    // Wait one tick to let the query settle.
    await waitFor(() => {
      expect(container.firstChild).toBeNull();
    });
  });

  it("renders nothing when Google connection is active", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "active",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    const { container } = render(<GoogleReauthBanner />, { wrapper });
    await waitFor(() => {
      expect(container.firstChild).toBeNull();
    });
  });

  it("does not render for non-google connection needing reauth", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "microsoft",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    const { container } = render(<GoogleReauthBanner />, { wrapper });
    await waitFor(() => {
      expect(container.firstChild).toBeNull();
    });
  });

  it("renders the reconnect banner when a google connection needs reauth", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    render(<GoogleReauthBanner />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Your Google connection needs to be reconnected"),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /reconnect google/i }),
    ).toBeInTheDocument();
  });

  it("shows a plural title when multiple google connections need reauth", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
      {
        id: "conn-2",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    render(<GoogleReauthBanner />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("2 Google connections need to be reconnected"),
      ).toBeInTheDocument();
    });
  });

  it("opens the beta notice dialog when the reconnect button is clicked", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    const user = userEvent.setup();
    render(<GoogleReauthBanner />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /reconnect google/i }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: /reconnect google/i }),
    );

    expect(
      screen.getByText("Your Google account must be added to the beta"),
    ).toBeInTheDocument();
  });

  it("can be dismissed per session", async () => {
    mockConnections([
      {
        id: "conn-1",
        provider: "google",
        scopes: ["openid"],
        status: "needs_reauth",
        connected_at: "2026-03-05T10:00:00Z",
      },
    ]);
    const user = userEvent.setup();
    render(<GoogleReauthBanner />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Your Google connection needs to be reconnected"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: /dismiss until next session/i }),
    );

    await waitFor(() => {
      expect(
        screen.queryByText("Your Google connection needs to be reconnected"),
      ).not.toBeInTheDocument();
    });
  });
});
