import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPost,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { AddCredentialDialog } from "../AddCredentialDialog";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockApiFetch() {
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
    return Promise.resolve({ data: null });
  });
}

describe("AddCredentialDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;
  const onOpenChange = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
    onOpenChange.mockReset();
  });

  it("renders dialog with title when open", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(screen.getByText("Add Credential")).toBeInTheDocument();
  });

  it("does not render content when closed", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={false} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(screen.queryByText("Add Credential")).not.toBeInTheDocument();
  });

  it("renders service and label inputs", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(screen.getByLabelText("Service")).toBeInTheDocument();
    expect(screen.getByLabelText(/Label/)).toBeInTheDocument();
  });

  it("renders credential key-value fields", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(screen.getByLabelText("Credential key 1")).toBeInTheDocument();
    expect(screen.getByLabelText("Credential value 1")).toBeInTheDocument();
  });

  it("disables Store button when form is empty", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(
      screen.getByRole("button", { name: "Store Credential" }),
    ).toBeDisabled();
  });

  it("adds a new credential field when Add Field is clicked", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    await user.click(screen.getByRole("button", { name: "Add Field" }));

    expect(screen.getByLabelText("Credential key 2")).toBeInTheDocument();
    expect(screen.getByLabelText("Credential value 2")).toBeInTheDocument();
  });

  it("submits credentials via POST /v1/credentials", async () => {
    mockApiFetch();
    mockPost.mockResolvedValue({
      data: {
        id: "cred_new123",
        service: "github",
        label: "Personal Token",
        created_at: "2026-02-28T00:00:00Z",
      },
    });
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    await user.type(screen.getByLabelText("Service"), "github");
    await user.type(screen.getByLabelText(/Label/), "Personal Token");
    await user.type(screen.getByLabelText("Credential key 1"), "api_key");
    await user.type(
      screen.getByLabelText("Credential value 1"),
      "ghp_secret123",
    );
    await user.click(
      screen.getByRole("button", { name: "Store Credential" }),
    );

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/credentials",
        expect.objectContaining({
          body: {
            service: "github",
            credentials: { api_key: "ghp_secret123" },
            label: "Personal Token",
          },
        }),
      );
    });
  });

  it("closes dialog after successful submission", async () => {
    mockApiFetch();
    mockPost.mockResolvedValue({
      data: {
        id: "cred_new123",
        service: "github",
        created_at: "2026-02-28T00:00:00Z",
      },
    });
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    await user.type(screen.getByLabelText("Service"), "github");
    await user.type(screen.getByLabelText("Credential key 1"), "api_key");
    await user.type(
      screen.getByLabelText("Credential value 1"),
      "ghp_secret123",
    );
    await user.click(
      screen.getByRole("button", { name: "Store Credential" }),
    );

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("closes dialog when Cancel is clicked", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("hides credential values by default", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    const valueInput = screen.getByLabelText("Credential value 1");
    expect(valueInput).toHaveAttribute("type", "password");
  });

  it("toggles credential value visibility", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    const valueInput = screen.getByLabelText("Credential value 1");
    expect(valueInput).toHaveAttribute("type", "password");

    await user.click(screen.getByRole("button", { name: "Show values" }));
    expect(valueInput).toHaveAttribute("type", "text");

    await user.click(screen.getByRole("button", { name: "Hide values" }));
    expect(valueInput).toHaveAttribute("type", "password");
  });

  it("shows helper text for service field", () => {
    mockApiFetch();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    expect(
      screen.getByText(/Must match the service name expected/),
    ).toBeInTheDocument();
  });

  it("shows inline error for invalid service name", async () => {
    mockApiFetch();
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    const serviceInput = screen.getByLabelText("Service");
    await user.type(serviceInput, "1invalid");
    await user.type(screen.getByLabelText("Credential key 1"), "api_key");
    await user.type(
      screen.getByLabelText("Credential value 1"),
      "some_value",
    );
    await user.click(
      screen.getByRole("button", { name: "Store Credential" }),
    );

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/must start with a lowercase letter/),
    ).toBeInTheDocument();
  });

  it("shows inline error on API failure", async () => {
    mockApiFetch();
    mockPost.mockRejectedValue(new Error("Duplicate credentials"));
    const user = userEvent.setup();

    render(
      <AddCredentialDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper },
    );

    await user.type(screen.getByLabelText("Service"), "github");
    await user.type(screen.getByLabelText("Credential key 1"), "api_key");
    await user.type(
      screen.getByLabelText("Credential value 1"),
      "ghp_secret123",
    );
    await user.click(
      screen.getByRole("button", { name: "Store Credential" }),
    );

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
  });
});
