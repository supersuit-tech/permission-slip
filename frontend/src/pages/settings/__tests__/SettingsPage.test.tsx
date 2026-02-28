import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, mockMfa } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { SettingsPage } from "../SettingsPage";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockProfile = {
  id: "user-123",
  username: "alice",
  email: "alice@example.com",
  phone: "+15551234567",
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
            { channel: "web-push", enabled: true },
            { channel: "sms", enabled: false },
          ],
        },
      });
    }
    if (url === "/v1/credentials") {
      return Promise.resolve({ data: { credentials: [] } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("SettingsPage", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders page title and back link", async () => {
    mockApiFetch();

    render(<SettingsPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Settings")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("link", { name: "Back to Dashboard" }),
    ).toHaveAttribute("href", "/");
  });

  it("renders all four sections", async () => {
    mockApiFetch();

    render(<SettingsPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Account")).toBeInTheDocument();
    });
    expect(screen.getByText("Security")).toBeInTheDocument();
    expect(screen.getByText("Notifications")).toBeInTheDocument();
    expect(screen.getByText("Credential Vault")).toBeInTheDocument();
  });
});
