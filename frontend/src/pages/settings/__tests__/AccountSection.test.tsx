import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPatch,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { AccountSection } from "../AccountSection";

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

function mockApiFetch(profile = mockProfile) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/profile") {
      return Promise.resolve({ data: profile, response: { status: 200 } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("AccountSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders the account card with title", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Account")).toBeInTheDocument();
    });
    expect(
      screen.getByText("Manage your profile and contact information."),
    ).toBeInTheDocument();
  });

  it("shows read-only username field", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      const usernameInput = screen.getByLabelText("Username");
      expect(usernameInput).toBeDisabled();
      expect(usernameInput).toHaveValue("alice");
    });
  });

  it("shows read-only login email field", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      const loginEmailInput = screen.getByLabelText("Login email");
      expect(loginEmailInput).toBeDisabled();
    });
  });

  it("populates contact email from profile", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Contact email")).toHaveValue(
        "alice@example.com",
      );
    });
  });

  it("populates phone number from profile", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Phone number")).toHaveValue(
        "+15551234567",
      );
    });
  });

  it("shows empty fields when profile has no contact info", async () => {
    mockApiFetch({
      ...mockProfile,
      email: undefined as unknown as string,
      phone: undefined as unknown as string,
    });

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Contact email")).toHaveValue("");
    });
    expect(screen.getByLabelText("Phone number")).toHaveValue("");
  });

  it("disables Save button when form is not dirty", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Save Changes" }),
      ).toBeDisabled();
    });
  });

  it("enables Save button when email is changed", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Contact email")).toHaveValue(
        "alice@example.com",
      );
    });

    await user.clear(screen.getByLabelText("Contact email"));
    await user.type(
      screen.getByLabelText("Contact email"),
      "new@example.com",
    );

    expect(
      screen.getByRole("button", { name: "Save Changes" }),
    ).toBeEnabled();
  });

  it("shows Change button next to login email", async () => {
    mockApiFetch();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Change" }),
      ).toBeInTheDocument();
    });
  });

  it("opens email change dialog when Change button is clicked", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Change" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Change" }));

    await waitFor(() => {
      expect(screen.getByText("Change Email Address")).toBeInTheDocument();
    });
  });

  it("submits updated profile on save", async () => {
    mockApiFetch();
    mockPatch.mockResolvedValue({
      data: { ...mockProfile, email: "new@example.com" },
    });
    const user = userEvent.setup();

    render(<AccountSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Contact email")).toHaveValue(
        "alice@example.com",
      );
    });

    await user.clear(screen.getByLabelText("Contact email"));
    await user.type(
      screen.getByLabelText("Contact email"),
      "new@example.com",
    );
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith(
        "/v1/profile",
        expect.objectContaining({
          body: { email: "new@example.com", phone: "+15551234567" },
        }),
      );
    });
  });
});
