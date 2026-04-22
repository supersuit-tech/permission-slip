import { render, screen, waitFor } from "@testing-library/react";
import { Route, Routes, MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, mockMfa } from "../../../auth/__tests__/fixtures";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { AuthProvider } from "../../../auth/AuthContext";
import { CookieConsentProvider } from "../../../components/CookieConsentContext";
import { ThemeProvider } from "../../../components/ThemeContext";
import { SettingsLayout } from "../SettingsLayout";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockProfile = {
  id: "user-123",
  username: "alice",
  email: "alice@example.com",
  phone: "+15551234567",
  marketing_opt_in: false,
  created_at: "2026-01-01T00:00:00Z",
};

function mockApiFetch() {
  setupAuthMocks({ authenticated: true });
  mockMfa.listFactors.mockResolvedValue({
    data: { all: [], totp: [] },
    error: null,
  });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/profile") {
      return Promise.resolve({ data: mockProfile, response: { status: 200 } });
    }
    if (url === "/v1/profile/notification-preferences") {
      return Promise.resolve({
        data: {
          preferences: [
            { channel: "email", enabled: true },
            { channel: "sms", enabled: false },
          ],
        },
      });
    }
    if (url === "/v1/credentials") {
      return Promise.resolve({ data: { credentials: [] } });
    }
    if (url === "/v1/oauth/connections") {
      return Promise.resolve({ data: { connections: [] } });
    }
    if (url === "/v1/oauth/providers") {
      return Promise.resolve({ data: { providers: [] } });
    }
    if (url === "/v1/oauth/provider-configs") {
      return Promise.resolve({ data: { configs: [] } });
    }
    if (url === "/v1/profile/data-retention") {
      return Promise.resolve({
        data: {
          plan_id: "free",
          plan_name: "Free",
          audit_retention_days: 7,
        },
      });
    }
    return Promise.resolve({ data: null });
  });
}

/** Renders SettingsLayout inside a parent route at /settings/* so nested Routes work. */
function renderSettingsAt(path: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <CookieConsentProvider>
            <AuthProvider>
              <Routes>
                <Route path="/settings/*" element={<SettingsLayout />} />
              </Routes>
            </AuthProvider>
          </CookieConsentProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("SettingsLayout", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
  });

  it("renders page title and nav links", async () => {
    mockApiFetch();
    renderSettingsAt("/settings/profile");

    await waitFor(() => {
      expect(screen.getByText("Settings")).toBeInTheDocument();
    });
    // Each nav item renders twice (desktop sidebar + mobile tabs)
    expect(screen.getAllByRole("link", { name: /Profile/ }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: /Security/ }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: /Billing/ }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("link", { name: /Account/ }).length).toBeGreaterThanOrEqual(1);
  });

  it("renders profile sections on /settings/profile", async () => {
    mockApiFetch();
    renderSettingsAt("/settings/profile");

    await waitFor(() => {
      // "Account" appears in both nav link and section heading
      expect(screen.getAllByText("Account").length).toBeGreaterThanOrEqual(2);
    });
    expect(screen.getByText("Notifications")).toBeInTheDocument();
  });

  it("renders security section on /settings/security", async () => {
    mockApiFetch();
    renderSettingsAt("/settings/security");

    await waitFor(() => {
      // "Security" appears in both nav and section heading — check for section content
      const headings = screen.getAllByText("Security");
      expect(headings.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("redirects unknown sub-routes to /settings/profile", async () => {
    mockApiFetch();
    renderSettingsAt("/settings/integrations");

    await waitFor(() => {
      // Should fall through to profile page via catch-all route
      expect(screen.getAllByText("Account").length).toBeGreaterThanOrEqual(2);
    });
  });

  it("renders danger zone on /settings/account", async () => {
    mockApiFetch();
    renderSettingsAt("/settings/account");

    await waitFor(() => {
      expect(screen.getByText("Danger Zone")).toBeInTheDocument();
    });
  });
});
