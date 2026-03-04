import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPatch,
  mockPut,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { NotificationSection } from "../NotificationSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const allEnabled = [
  { channel: "email", enabled: true, available: true },
  { channel: "web-push", enabled: true, available: true },
  { channel: "sms", enabled: true, available: true },
  { channel: "mobile-push", enabled: true, available: true },
];

const smsGated = [
  { channel: "email", enabled: true, available: true },
  { channel: "web-push", enabled: true, available: true },
  { channel: "sms", enabled: false, available: false },
  { channel: "mobile-push", enabled: true, available: true },
];

interface MockProfile {
  id: string;
  username: string;
  email?: string | null;
  phone?: string | null;
  marketing_opt_in: boolean;
  created_at: string;
}

const profileWithContact: MockProfile = {
  id: "user-123",
  username: "alice",
  email: "alice@example.com",
  phone: "+15551234567",
  marketing_opt_in: false,
  created_at: "2026-01-01T00:00:00Z",
};

const profileNoContact: MockProfile = {
  id: "user-123",
  username: "alice",
  marketing_opt_in: false,
  created_at: "2026-01-01T00:00:00Z",
};

function mockApiFetch(
  profile = profileWithContact,
  preferences = allEnabled,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/profile") {
      return Promise.resolve({ data: profile, response: { status: 200 } });
    }
    if (url === "/v1/profile/notification-preferences") {
      return Promise.resolve({ data: { preferences } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("NotificationSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders the notification card with title", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Notifications")).toBeInTheDocument();
    });
  });

  it("renders all four channels", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    expect(screen.getByText("Web Push")).toBeInTheDocument();
    expect(screen.getByText("SMS")).toBeInTheDocument();
    expect(screen.getByText("Mobile Push")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<NotificationSection />, { wrapper });

    expect(
      screen.getByRole("status", {
        name: "Loading notification preferences",
      }),
    ).toBeInTheDocument();
  });

  it("shows enabled/disabled state for each channel", async () => {
    const prefs = [
      { channel: "email", enabled: true, available: true },
      { channel: "web-push", enabled: false, available: true },
      { channel: "sms", enabled: true, available: true },
      { channel: "mobile-push", enabled: true, available: true },
    ];
    mockApiFetch(profileWithContact, prefs);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      const buttons = screen.getAllByRole("button");
      const enabledButtons = buttons.filter(
        (b) => b.textContent === "Enabled",
      );
      const disabledButtons = buttons.filter(
        (b) => b.textContent === "Disabled",
      );
      // 3 channels enabled (email, sms, mobile-push) + 2 disabled (web-push, product updates)
      expect(enabledButtons).toHaveLength(3);
      expect(disabledButtons).toHaveLength(2);
    });
  });

  it("shows warning when email is missing and email notifications enabled", async () => {
    mockApiFetch(profileNoContact, allEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(
          "Add a contact email above to receive email notifications.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows warning when phone is missing and SMS notifications enabled", async () => {
    mockApiFetch(profileNoContact, allEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(
          "Add a phone number above to receive SMS notifications.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("does not show warnings when contact info is present", async () => {
    mockApiFetch(profileWithContact, allEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/Add a contact email/),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Add a phone number/),
    ).not.toBeInTheDocument();
  });

  it("does not show warning for disabled channel even if contact is missing", async () => {
    const prefs = [
      { channel: "email", enabled: false, available: true },
      { channel: "web-push", enabled: true, available: true },
      { channel: "sms", enabled: false, available: true },
      { channel: "mobile-push", enabled: true, available: true },
    ];
    mockApiFetch(profileNoContact, prefs);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/Add a contact email/),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Add a phone number/),
    ).not.toBeInTheDocument();
  });

  it("calls updatePreferences when toggle is clicked", async () => {
    mockApiFetch();
    mockPut.mockResolvedValue({
      data: {
        preferences: [
          { channel: "email", enabled: false },
          { channel: "web-push", enabled: true },
          { channel: "sms", enabled: true },
          { channel: "mobile-push", enabled: true },
        ],
      },
    });
    const user = userEvent.setup();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });

    // Click the first "Enabled" button (email)
    const enabledButtons = screen.getAllByRole("button").filter(
      (b) => b.textContent === "Enabled",
    );
    await user.click(enabledButtons[0]!);

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        "/v1/profile/notification-preferences",
        expect.objectContaining({
          body: {
            preferences: [{ channel: "email", enabled: false }],
          },
        }),
      );
    });
  });

  it("can toggle mobile-push channel", async () => {
    mockApiFetch();
    mockPut.mockResolvedValue({
      data: {
        preferences: allEnabled.map((p) =>
          p.channel === "mobile-push" ? { ...p, enabled: false } : p,
        ),
      },
    });
    const user = userEvent.setup();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Mobile Push")).toBeInTheDocument();
    });

    // Find the Mobile Push row's Enabled button and click it
    const enabledButtons = screen.getAllByRole("button").filter(
      (b) => b.textContent === "Enabled",
    );
    // Mobile Push is the last enabled channel button (after email, web-push, sms)
    const mobilePushBtn = enabledButtons[enabledButtons.length - 1]!;
    await user.click(mobilePushBtn);

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        "/v1/profile/notification-preferences",
        expect.objectContaining({
          body: {
            preferences: [{ channel: "mobile-push", enabled: false }],
          },
        }),
      );
    });
  });

  it("renders the product updates section", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Product updates")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Occasional emails about new features/),
    ).toBeInTheDocument();
  });

  it("calls updateProfile when product updates toggle is clicked", async () => {
    mockApiFetch();
    mockPatch.mockResolvedValue({
      data: { ...profileWithContact, marketing_opt_in: true },
    });
    const user = userEvent.setup();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Product updates")).toBeInTheDocument();
    });

    // The product updates button is the last "Disabled" button
    const disabledButtons = screen.getAllByRole("button").filter(
      (b) => b.textContent === "Disabled",
    );
    const productUpdatesBtn = disabledButtons[disabledButtons.length - 1]!;
    await user.click(productUpdatesBtn);

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith(
        "/v1/profile",
        expect.objectContaining({
          body: { marketing_opt_in: true },
        }),
      );
    });
  });

  it("shows warning when product updates enabled but email is missing", async () => {
    const profileOptedIn = { ...profileNoContact, marketing_opt_in: true };
    mockApiFetch(profileOptedIn, allEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(
          "Add a contact email above to receive product update emails.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows SMS as gated with upgrade link on free plan", async () => {
    mockApiFetch(profileWithContact, smsGated);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Available on paid plan"),
      ).toBeInTheDocument();
    });
    const upgradeLink = screen.getByRole("link", {
      name: /Available on paid plan/,
    });
    expect(upgradeLink).toHaveAttribute("href", "/billing");
  });

  it("does not show toggle button for plan-gated SMS", async () => {
    mockApiFetch(profileWithContact, smsGated);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("SMS")).toBeInTheDocument();
    });

    // Should have 3 enabled buttons (email, web-push, mobile-push)
    // SMS should not have a toggle button
    const enabledButtons = screen.getAllByRole("button").filter(
      (b) => b.textContent === "Enabled",
    );
    expect(enabledButtons).toHaveLength(3);
  });

  it("does not show missing phone warning for plan-gated SMS", async () => {
    mockApiFetch(profileNoContact, smsGated);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("SMS")).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/Add a phone number/),
    ).not.toBeInTheDocument();
  });
});
