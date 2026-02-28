import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ConnectorCredentialsSection } from "../ConnectorCredentialsSection";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const requiredCredentials = [
  { service: "github", auth_type: "api_key" as const },
];

const storedCredentials = {
  credentials: [
    {
      id: "cred_123",
      service: "github",
      label: "Personal Access Token",
      created_at: "2026-02-11T10:00:00Z",
    },
  ],
};

describe("ConnectorCredentialsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows no credentials required message when empty", () => {
    renderWithProviders(
      <ConnectorCredentialsSection requiredCredentials={[]} />,
    );
    expect(
      screen.getByText("This connector does not require any credentials."),
    ).toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockGet.mockReturnValue(new Promise(() => {}));
    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );
    expect(screen.getByText("Credentials")).toBeInTheDocument();
  });

  it("shows connected status with stored credentials", async () => {
    mockGet.mockResolvedValue({ data: storedCredentials });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });
    expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    expect(screen.getByText("Add Another")).toBeInTheDocument();
  });

  it("shows not configured status when no credentials stored", async () => {
    mockGet.mockResolvedValue({ data: { credentials: [] } });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Not configured")).toBeInTheDocument();
    });
    expect(screen.getByText("Connect")).toBeInTheDocument();
  });

  it("opens Add Credential dialog", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({ data: { credentials: [] } });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));

    expect(screen.getByText("Add Credential")).toBeInTheDocument();
    expect(screen.getByLabelText("API Key")).toBeInTheDocument();
  });

  it("stores credential through Add dialog", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({ data: { credentials: [] } });
    mockPost.mockResolvedValue({
      data: {
        id: "cred_new",
        service: "github",
        created_at: "2026-02-20T10:00:00Z",
      },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));
    await user.type(screen.getByLabelText("API Key"), "ghp_test_key");
    await user.click(screen.getByText("Store Credential"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith("/v1/credentials", {
        headers: { Authorization: "Bearer token" },
        body: {
          service: "github",
          credentials: { api_key: "ghp_test_key" },
        },
      });
    });
  });

  it("opens Remove Credential dialog", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({ data: storedCredentials });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Remove credential Personal Access Token"),
    );

    expect(screen.getByText("Remove Credential")).toBeInTheDocument();
    expect(
      screen.getByText(/This will permanently delete/),
    ).toBeInTheDocument();
  });

  it("deletes credential through Remove dialog", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({ data: storedCredentials });
    mockDelete.mockResolvedValue({
      data: { id: "cred_123", deleted_at: "2026-02-20T10:00:00Z" },
    });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={requiredCredentials}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Remove credential Personal Access Token"),
    );
    await user.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/credentials/{credential_id}",
        {
          headers: { Authorization: "Bearer token" },
          params: { path: { credential_id: "cred_123" } },
        },
      );
    });
  });

  it("renders basic auth fields for basic auth type", async () => {
    const user = userEvent.setup();
    mockGet.mockResolvedValue({ data: { credentials: [] } });

    renderWithProviders(
      <ConnectorCredentialsSection
        requiredCredentials={[
          { service: "jira", auth_type: "basic" as const },
        ]}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Connect")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Connect"));

    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByLabelText("Password / API Token")).toBeInTheDocument();
  });
});
