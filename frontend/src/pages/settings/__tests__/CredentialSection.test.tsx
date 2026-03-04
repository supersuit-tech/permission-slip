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
import { CredentialSection } from "../CredentialSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const freePlanResponse = {
  plan: {
    id: "free",
    name: "Free",
    max_requests_per_month: 1000,
    max_agents: 3,
    max_standing_approvals: 5,
    max_credentials: 5,
    audit_retention_days: 7,
  },
  subscription: { status: "active", can_upgrade: true, can_downgrade: false },
  usage: { requests: 10, agents: 2, standing_approvals: 1, credentials: 0 },
};

function mockApiFetch(credentials: unknown[] = [], billingPlan = freePlanResponse) {
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
    if (url === "/v1/credentials") {
      return Promise.resolve({ data: { credentials } });
    }
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: billingPlan });
    }
    return Promise.resolve({ data: null });
  });
}

describe("CredentialSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders the credential vault card", async () => {
    mockApiFetch();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Credential Vault")).toBeInTheDocument();
    });
  });

  it("shows empty state when no credentials exist", async () => {
    mockApiFetch([]);

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/No credentials stored yet/),
      ).toBeInTheDocument();
    });
  });

  it("shows loading state", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<CredentialSection />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading credentials" }),
    ).toBeInTheDocument();
  });

  it("renders credentials when they exist", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "Gmail",
        label: "OAuth2",
        created_at: "2026-01-15T00:00:00Z",
      },
      {
        id: "cred-2",
        service: "Stripe",
        label: "API Key",
        created_at: "2026-02-01T00:00:00Z",
      },
    ]);

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });
    expect(screen.getByText("Stripe")).toBeInTheDocument();
    expect(screen.getByText(/OAuth2/)).toBeInTheDocument();
    expect(screen.getByText(/API Key/)).toBeInTheDocument();
  });

  it("shows 'Credential' as fallback when label is missing", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "Slack",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Slack")).toBeInTheDocument();
    });
    expect(screen.getByText(/Credential · Added/)).toBeInTheDocument();
  });

  it("shows Active badge for each credential", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "Gmail",
        label: "OAuth2",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Active")).toBeInTheDocument();
    });
  });

  it("shows Add Credential button", async () => {
    mockApiFetch();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Add Credential" }),
      ).toBeInTheDocument();
    });
  });

  it("opens Add Credential dialog when button is clicked", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Add Credential" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Add Credential" }));

    await waitFor(() => {
      expect(screen.getByLabelText("Service")).toBeInTheDocument();
    });
  });

  it("shows delete button for each credential", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "GitHub",
        label: "Token",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Delete GitHub credential" }),
      ).toBeInTheDocument();
    });
  });

  it("shows confirmation before deleting credential", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "GitHub",
        label: "Token",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);
    const user = userEvent.setup();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Delete GitHub credential" }),
      ).toBeInTheDocument();
    });

    // Click trash icon - should show confirmation, not delete immediately.
    await user.click(
      screen.getByRole("button", { name: "Delete GitHub credential" }),
    );

    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(mockDelete).not.toHaveBeenCalled();
  });

  it("cancels deletion when Cancel is clicked", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "GitHub",
        label: "Token",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);
    const user = userEvent.setup();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Delete GitHub credential" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: "Delete GitHub credential" }),
    );
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    // Should be back to trash icon.
    expect(
      screen.getByRole("button", { name: "Delete GitHub credential" }),
    ).toBeInTheDocument();
  });

  it("calls DELETE endpoint after confirmation", async () => {
    mockApiFetch([
      {
        id: "cred-1",
        service: "GitHub",
        label: "Token",
        created_at: "2026-01-15T00:00:00Z",
      },
    ]);
    mockDelete.mockResolvedValue({
      data: { id: "cred-1", deleted_at: "2026-02-28T00:00:00Z" },
    });
    const user = userEvent.setup();

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Delete GitHub credential" }),
      ).toBeInTheDocument();
    });

    // Click trash, then confirm.
    await user.click(
      screen.getByRole("button", { name: "Delete GitHub credential" }),
    );
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/credentials/{credential_id}",
        expect.objectContaining({
          params: { path: { credential_id: "cred-1" } },
        }),
      );
    });
  });

  it("shows credential count with limit badge", async () => {
    const planWith2Creds = {
      ...freePlanResponse,
      usage: { ...freePlanResponse.usage, credentials: 2 },
    };
    mockApiFetch(
      [
        {
          id: "cred-1",
          service: "GitHub",
          label: "Token",
          created_at: "2026-01-15T00:00:00Z",
        },
        {
          id: "cred-2",
          service: "Slack",
          label: "Bot",
          created_at: "2026-01-16T00:00:00Z",
        },
      ],
      planWith2Creds,
    );

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("2 / 5 credentials")).toBeInTheDocument();
    });
  });

  it("shows upgrade prompt when at credential limit", async () => {
    const atLimitPlan = {
      ...freePlanResponse,
      usage: { ...freePlanResponse.usage, credentials: 5 },
    };
    mockApiFetch(
      [
        { id: "cred-1", service: "S1", created_at: "2026-01-15T00:00:00Z" },
        { id: "cred-2", service: "S2", created_at: "2026-01-15T00:00:00Z" },
        { id: "cred-3", service: "S3", created_at: "2026-01-15T00:00:00Z" },
        { id: "cred-4", service: "S4", created_at: "2026-01-15T00:00:00Z" },
        { id: "cred-5", service: "S5", created_at: "2026-01-15T00:00:00Z" },
      ],
      atLimitPlan,
    );

    render(<CredentialSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/Upgrade to store more credentials/),
      ).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", { name: "Add Credential" }),
    ).not.toBeInTheDocument();
  });
});
