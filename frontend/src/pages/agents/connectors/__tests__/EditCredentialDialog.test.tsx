import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "@/auth/__tests__/fixtures";
import { createAuthWrapper } from "@/test-helpers";
import { mockPatch, resetClientMocks } from "@/api/__mocks__/client";
import { EditCredentialDialog } from "../EditCredentialDialog";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";

vi.mock("@/api/client");

const staticCredential: RequiredCredential = {
  service: "proton",
  auth_type: "custom",
  fields: [
    {
      key: "username",
      label: "Username",
      secret: false,
      required: true,
      placeholder: "Bridge username",
    },
    {
      key: "password",
      label: "Password",
      secret: true,
      required: true,
      placeholder: "Bridge password",
    },
  ],
};

const storedCredential: CredentialSummary = {
  id: "cred_abc123",
  service: "proton",
  label: "Work Proton",
  created_at: "2026-02-11T10:00:00Z",
};

describe("EditCredentialDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;
  const onOpenChange = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    wrapper = createAuthWrapper();
    onOpenChange.mockReset();
    mockPatch.mockResolvedValue({
      data: {
        id: storedCredential.id,
        service: storedCredential.service,
        label: "Work Proton",
        created_at: storedCredential.created_at,
      },
    });
  });

  it("prefills the label and shows blank-field hints", () => {
    render(
      <EditCredentialDialog
        open
        onOpenChange={onOpenChange}
        credential={staticCredential}
        storedCredential={storedCredential}
      />,
      { wrapper },
    );

    expect(screen.getByLabelText(/Label/)).toHaveValue("Work Proton");
    expect(
      screen.getAllByText("Leave blank to keep current value").length,
    ).toBeGreaterThan(0);
  });

  it("submits label-only edit", async () => {
    const user = userEvent.setup();

    render(
      <EditCredentialDialog
        open
        onOpenChange={onOpenChange}
        credential={staticCredential}
        storedCredential={storedCredential}
      />,
      { wrapper },
    );

    const labelInput = screen.getByLabelText(/Label/);
    await user.clear(labelInput);
    await user.type(labelInput, "Personal Proton");
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
        headers: { Authorization: expect.stringMatching(/^Bearer /) },
        params: { path: { credential_id: "cred_abc123" } },
        body: { label: "Personal Proton" },
      });
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("submits clear label when label is emptied", async () => {
    const user = userEvent.setup();

    render(
      <EditCredentialDialog
        open
        onOpenChange={onOpenChange}
        credential={staticCredential}
        storedCredential={storedCredential}
      />,
      { wrapper },
    );

    await user.clear(screen.getByLabelText(/Label/));
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
        headers: { Authorization: expect.stringMatching(/^Bearer /) },
        params: { path: { credential_id: "cred_abc123" } },
        body: { label: null },
      });
    });
  });

  it("submits only non-blank credential fields", async () => {
    const user = userEvent.setup();

    render(
      <EditCredentialDialog
        open
        onOpenChange={onOpenChange}
        credential={staticCredential}
        storedCredential={storedCredential}
      />,
      { wrapper },
    );

    await user.type(screen.getByLabelText("Password"), "new-bridge-password");
    await user.click(screen.getByRole("button", { name: "Save Changes" }));

    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
        headers: { Authorization: expect.stringMatching(/^Bearer /) },
        params: { path: { credential_id: "cred_abc123" } },
        body: { credentials: { password: "new-bridge-password" } },
      });
    });
  });
});
