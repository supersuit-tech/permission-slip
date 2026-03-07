import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPost,
  mockDelete,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { OAuthProviderSection } from "../OAuthProviderSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockApiFetch(
  providers: unknown[] = [],
  configs: unknown[] = [],
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/oauth/providers") {
      return Promise.resolve({ data: { providers } });
    }
    if (url === "/v1/oauth/provider-configs") {
      return Promise.resolve({ data: { configs } });
    }
    return Promise.resolve({ data: null });
  });
}

describe("OAuthProviderSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("does not render when no BYOA configs or unconfigured providers", async () => {
    // All providers have credentials, no BYOA configs
    mockApiFetch(
      [{ id: "google", scopes: ["openid"], source: "built_in", has_credentials: true }],
      [],
    );

    const { container } = render(<OAuthProviderSection />, { wrapper });

    // Wait for data to load, then verify section is not rendered
    await waitFor(() => {
      expect(mockGet).toHaveBeenCalled();
    });
    // Give React time to settle after data loads
    await new Promise((r) => setTimeout(r, 100));
    expect(container.querySelector("[class*='card']")).not.toBeInTheDocument();
  });

  it("renders section with unconfigured providers", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "manifest", has_credentials: false },
      ],
      [],
    );

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("OAuth App Credentials")).toBeInTheDocument();
    });
    expect(screen.getByText("Salesforce")).toBeInTheDocument();
    expect(
      screen.getByText(/Needs OAuth client credentials/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Configure" }),
    ).toBeInTheDocument();
  });

  it("renders BYOA configs", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "byoa", has_credentials: true },
      ],
      [
        {
          provider: "salesforce",
          created_at: "2026-03-05T12:00:00Z",
          updated_at: "2026-03-05T12:00:00Z",
        },
      ],
    );

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Salesforce")).toBeInTheDocument();
    });
    expect(screen.getByText("BYOA")).toBeInTheDocument();
    expect(screen.getByText(/Custom credentials configured/)).toBeInTheDocument();
  });

  it("opens BYOA config dialog when Configure is clicked", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "manifest", has_credentials: false },
      ],
      [],
    );
    const user = userEvent.setup();

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Configure" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Configure" }));

    await waitFor(() => {
      expect(
        screen.getByText("Configure Salesforce OAuth App"),
      ).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Client ID")).toBeInTheDocument();
    expect(screen.getByLabelText("Client Secret")).toBeInTheDocument();
  });

  it("submits BYOA credentials", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "manifest", has_credentials: false },
      ],
      [],
    );
    mockPost.mockResolvedValue({
      data: {
        provider: "salesforce",
        created_at: "2026-03-05T12:00:00Z",
        updated_at: "2026-03-05T12:00:00Z",
      },
    });
    const user = userEvent.setup();

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Configure" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Configure" }));

    await waitFor(() => {
      expect(screen.getByLabelText("Client ID")).toBeInTheDocument();
    });

    await user.type(screen.getByLabelText("Client ID"), "my-client-id");
    await user.type(
      screen.getByLabelText("Client Secret"),
      "my-client-secret",
    );
    await user.click(screen.getByRole("button", { name: "Save Credentials" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/oauth/provider-configs",
        expect.objectContaining({
          body: {
            provider: "salesforce",
            client_id: "my-client-id",
            client_secret: "my-client-secret",
          },
        }),
      );
    });
  });

  it("shows delete confirmation for BYOA config", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "byoa", has_credentials: true },
      ],
      [
        {
          provider: "salesforce",
          created_at: "2026-03-05T12:00:00Z",
          updated_at: "2026-03-05T12:00:00Z",
        },
      ],
    );
    const user = userEvent.setup();

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", {
          name: "Remove Salesforce credentials",
        }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", {
        name: "Remove Salesforce credentials",
      }),
    );

    expect(screen.getByRole("button", { name: "Remove" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("calls DELETE endpoint after removal confirmation", async () => {
    mockApiFetch(
      [
        { id: "salesforce", scopes: ["api"], source: "byoa", has_credentials: true },
      ],
      [
        {
          provider: "salesforce",
          created_at: "2026-03-05T12:00:00Z",
          updated_at: "2026-03-05T12:00:00Z",
        },
      ],
    );
    mockDelete.mockResolvedValue({
      data: { provider: "salesforce", deleted_at: "2026-03-05T15:00:00Z" },
    });
    const user = userEvent.setup();

    render(<OAuthProviderSection />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", {
          name: "Remove Salesforce credentials",
        }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", {
        name: "Remove Salesforce credentials",
      }),
    );
    await user.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/oauth/provider-configs/{provider}",
        expect.objectContaining({
          params: { path: { provider: "salesforce" } },
        }),
      );
    });
  });
});
