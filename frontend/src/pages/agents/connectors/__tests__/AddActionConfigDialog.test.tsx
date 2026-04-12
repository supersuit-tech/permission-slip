import { render, screen, waitFor, within } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../../api/__mocks__/client";
import { AddActionConfigDialog } from "../AddActionConfigDialog";
import type { ConnectorAction } from "../../../../hooks/useConnectorDetail";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const actions: ConnectorAction[] = [
  {
    action_type: "github.create_issue",
    name: "Create Issue",
    description: "",
    risk_level: "low",
    requires_payment_method: false,
    parameters_schema: {
      type: "object",
      required: ["repo", "title"],
      properties: {
        repo: { type: "string" },
        title: { type: "string" },
      },
    },
  },
];

const initialTemplate = {
  id: "tpl_1",
  connector_id: "github",
  action_type: "github.create_issue",
  name: "From template",
  description: "Template desc",
  parameters: { repo: "org/repo", title: "*" },
  created_at: "2026-01-01T00:00:00Z",
};

describe("AddActionConfigDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({ data: { data: [] } });
      }
      return Promise.resolve({ data: null });
    });
    wrapper = createAuthWrapper();
  });

  it("pre-fills from initialTemplate when dialog opens", async () => {
    const { rerender } = render(
      <AddActionConfigDialog
        open={false}
        onOpenChange={vi.fn()}
        agentId={42}
        connectorId="github"
        actions={actions}
        initialTemplate={initialTemplate}
      />,
      { wrapper },
    );

    rerender(
      <AddActionConfigDialog
        open
        onOpenChange={vi.fn()}
        agentId={42}
        connectorId="github"
        actions={actions}
        initialTemplate={initialTemplate}
      />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Action")).toHaveValue("github.create_issue");
    });

    expect(screen.getByLabelText("Name")).toHaveValue("From template");
    expect(screen.getByLabelText("Description (optional)")).toHaveValue(
      "Template desc",
    );
    expect(screen.getByLabelText("Repo")).toHaveValue("org/repo");

    const titleInput = screen.getByLabelText("Title");
    const titleGroup = titleInput.closest(".space-y-1\\.5")!;
    expect(
      within(titleGroup as HTMLElement).getByRole("checkbox"),
    ).toBeChecked();
  });
});
