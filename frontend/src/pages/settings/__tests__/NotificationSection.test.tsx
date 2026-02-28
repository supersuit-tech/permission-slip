import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPut,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { NotificationSection } from "../NotificationSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const allEnabled = [
  { channel: "email", enabled: true },
  { channel: "web-push", enabled: true },
  { channel: "sms", enabled: true },
];

interface MockProfile {
  id: string;
  username: string;
  email?: string | null;
  phone?: string | null;
  created_at: string;
}

const profileWithContact: MockProfile = {
  id: "user-123",
  username: "alice",
  email: "alice@example.com",
  phone: "+15551234567",
  created_at: "2026-01-01T00:00:00Z",
};

const profileNoContact: MockProfile = {
  id: "user-123",
  username: "alice",
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

  it("renders all three channels", async () => {
    mockApiFetch();

    render(<NotificationSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email")).toBeInTheDocument();
    });
    expect(screen.getByText("Web Push")).toBeInTheDocument();
    expect(screen.getByText("SMS")).toBeInTheDocument();
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
      { channel: "email", enabled: true },
      { channel: "web-push", enabled: false },
      { channel: "sms", enabled: true },
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
      expect(enabledButtons).toHaveLength(2);
      expect(disabledButtons).toHaveLength(1);
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
      { channel: "email", enabled: false },
      { channel: "web-push", enabled: true },
      { channel: "sms", enabled: false },
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
});
