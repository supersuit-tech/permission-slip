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

// Default fixture: SMS not configured on server, so excluded from response.
const allEnabled = [
  { channel: "email", enabled: true, available: true },
  { channel: "mobile-push", enabled: true, available: true },
];

// Fixture: SMS configured on server and included.
const withSmsEnabled = [
  { channel: "email", enabled: true, available: true },
  { channel: "mobile-push", enabled: true, available: true },
  { channel: "sms", enabled: true, available: true },
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

const defaultTypePreferences = [
  { notification_type: "standing_execution" as const, enabled: true },
];

function mockApiFetch(
  profile = profileWithContact,
  preferences = allEnabled,
  typePreferences = defaultTypePreferences,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/profile") {
      return Promise.resolve({ data: profile, response: { status: 200 } });
    }
    if (url === "/v1/profile/notification-preferences") {
      return Promise.resolve({ data: { preferences } });
    }
    if (url === "/v1/profile/notification-type-preferences") {
      return Promise.resolve({ data: { preferences: typePreferences } });
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

  it("renders channels returned by server (no SMS when not configured)", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    expect(screen.queryByText("Web Push")).not.toBeInTheDocument();
    expect(screen.getByText("Mobile Push")).toBeInTheDocument();
    expect(screen.queryByText("SMS")).not.toBeInTheDocument();
  });

  it("renders SMS channel when configured on server", async () => {
    mockApiFetch(profileWithContact, withSmsEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("SMS")).toBeInTheDocument();
    });
    expect(screen.getByText("Email")).toBeInTheDocument();
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
      { channel: "mobile-push", enabled: true, available: true },
    ];
    mockApiFetch(profileWithContact, prefs);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      const switches = screen.getAllByRole("switch");
      const checked = switches.filter(
        (s) => s.getAttribute("data-state") === "checked",
      );
      const unchecked = switches.filter(
        (s) => s.getAttribute("data-state") === "unchecked",
      );
      // email + mobile-push + standing execution enabled + product updates unchecked
      expect(checked).toHaveLength(3);
      expect(unchecked).toHaveLength(1);
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

  it("does not show phone warning when SMS is not configured", async () => {
    mockApiFetch(profileNoContact, allEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    // SMS is excluded from response when not configured, so no phone warning.
    expect(screen.queryByText("SMS")).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Add a phone number/),
    ).not.toBeInTheDocument();
  });

  it("shows phone warning when SMS is configured and phone is missing", async () => {
    mockApiFetch(profileNoContact, withSmsEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("SMS")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Add a phone number/),
    ).toBeInTheDocument();
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

    // Click the Email switch by accessible name
    await user.click(screen.getByRole("switch", { name: /email notifications/i }));

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

    // Find the Mobile Push switch by accessible name
    await user.click(screen.getByRole("switch", { name: /mobile push notifications/i }));

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

    // Find the Product updates switch by accessible name
    await user.click(screen.getByRole("switch", { name: /product updates notifications/i }));

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

  it("shows toggle for SMS when configured", async () => {
    mockApiFetch(profileWithContact, withSmsEnabled);

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("SMS")).toBeInTheDocument();
    });

    // SMS should have a toggle (available=true), not "Coming soon"
    expect(screen.getByRole("switch", { name: /sms notifications/i })).toBeInTheDocument();
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
  });

  it("renders Notify me about and auto-approval toggle", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Notify me about")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("switch", { name: /auto-approval execution notifications/i }),
    ).toBeInTheDocument();
  });

  it("calls update for notification type when auto-approval toggle is clicked", async () => {
    mockApiFetch();
    mockPut.mockImplementation((url: string) => {
      if (url === "/v1/profile/notification-type-preferences") {
        return Promise.resolve({
          data: {
            preferences: [
              { notification_type: "standing_execution", enabled: false },
            ],
          },
        });
      }
      return Promise.resolve({ data: { preferences: [] } });
    });
    const user = userEvent.setup();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("switch", { name: /auto-approval execution notifications/i }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("switch", { name: /auto-approval execution notifications/i }),
    );

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        "/v1/profile/notification-type-preferences",
        expect.objectContaining({
          body: {
            preferences: [{ notification_type: "standing_execution", enabled: false }],
          },
        }),
      );
    });
  });
});
