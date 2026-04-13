import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../../test-helpers";
import {
  mockGet,
  mockPost,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { RecommendedTemplatesDialog } from "../RecommendedTemplatesDialog";
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
    parameters_schema: {},
  },
  {
    action_type: "github.merge_pr",
    name: "Merge Pull Request",
    description: "",
    risk_level: "high",
    requires_payment_method: false,
    parameters_schema: {},
  },
];

const baseTemplates = [
  {
    id: "tpl_a",
    connector_id: "github",
    action_type: "github.create_issue",
    name: "All open",
    description: "Desc A",
    parameters: { repo: "*", title: "*" },
    standing_approval: { duration_days: 30 },
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "tpl_b",
    connector_id: "github",
    action_type: "github.merge_pr",
    name: "Merge main",
    description: "Desc B",
    parameters: { repo: "supersuit-tech/webapp", pr: 1 },
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "tpl_dead",
    connector_id: "github",
    action_type: "removed.action",
    name: "Stale",
    description: null,
    parameters: {},
    created_at: "2026-01-01T00:00:00Z",
  },
];

describe("RecommendedTemplatesDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;
  const onCustomize = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    wrapper = createAuthWrapper();
    onCustomize.mockReset();
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({ data: { data: baseTemplates } });
      }
      return Promise.resolve({ data: null });
    });
  });

  function renderDialog(
    props: Partial<{
      open: boolean;
      onOpenChange: (open: boolean) => void;
    }> = {},
  ) {
    return render(
      <RecommendedTemplatesDialog
        open
        onOpenChange={props.onOpenChange ?? vi.fn()}
        agentId={42}
        connectorId="github"
        actions={actions}
        onCustomize={onCustomize}
        {...props}
      />,
      { wrapper },
    );
  }

  it("groups templates by action type in connector action order", async () => {
    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("Create Issue")).toBeInTheDocument();
    });

    const headings = screen.getAllByRole("heading", { level: 3 });
    expect(headings.map((h) => h.textContent)).toEqual([
      "Create Issue",
      "Merge Pull Request",
    ]);

    expect(screen.getByText("All open")).toBeInTheDocument();
    expect(screen.getByText("Merge main")).toBeInTheDocument();
    expect(screen.queryByText("Stale")).not.toBeInTheDocument();
  });

  it("filters out dead templates whose action_type is not on the connector", async () => {
    renderDialog();

    await waitFor(() => {
      expect(screen.queryByText("Stale")).not.toBeInTheDocument();
    });
  });

  it("shows loading state while templates load", () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return new Promise(() => {});
      }
      return Promise.resolve({ data: null });
    });

    renderDialog();

    expect(screen.getByText("Loading templates...")).toBeInTheDocument();
  });

  it("shows error state when template fetch fails", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: undefined,
          error: { message: "fail" },
        });
      }
      return Promise.resolve({ data: null });
    });

    renderDialog();

    await waitFor(() => {
      expect(
        screen.getByText(
          "Unable to load configuration templates. Please try again later.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows empty state when no live templates", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: {
            data: [
              {
                id: "only_dead",
                connector_id: "github",
                action_type: "gone.action",
                name: "Nope",
                description: null,
                parameters: {},
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    renderDialog();

    await waitFor(() => {
      expect(
        screen.getByText(
          "No recommended templates are available for this connector.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("creates config on Use Template with default approval_mode and closes dialog", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockPost.mockResolvedValue({
      data: {
        action_configuration: {
          id: "ac_new",
          agent_id: 42,
          connector_id: "github",
          action_type: "github.create_issue",
          parameters: { repo: "*", title: "*" },
          status: "active",
          name: "All open",
          created_at: "2026-02-25T10:00:00Z",
          updated_at: "2026-02-25T10:00:00Z",
        },
        standing_approval: {
          standing_approval_id: "sa_new",
          agent_id: 42,
          user_id: "user",
          action_type: "github.create_issue",
          action_version: "1",
          constraints: { repo: "*", title: "*" },
          status: "active",
          execution_count: 0,
          starts_at: "2026-02-25T10:00:00Z",
          expires_at: "2026-03-25T10:00:00Z",
          created_at: "2026-02-25T10:00:00Z",
          source_action_configuration_id: "ac_new",
        },
      },
    });

    renderDialog({ onOpenChange });

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    const tplCard = screen.getByText("All open").closest(".rounded-lg")!;
    await user.click(within(tplCard as HTMLElement).getByRole("button", { name: "Use Template" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
    const [url, opts] = mockPost.mock.calls[0] as [
      string,
      { body: { agent_id: number; approval_mode: string }; params: { path: { id: string } } },
    ];
    expect(url).toContain("/v1/action-config-templates/{id}/apply");
    expect(opts.params.path.id).toBe("tpl_a");
    expect(opts.body).toEqual({ agent_id: 42, approval_mode: "auto_approve" });

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("shows error toast and keeps dialog open on failed create", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockPost.mockRejectedValue(new Error("network down"));

    renderDialog({ onOpenChange });

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    const tplCard = screen.getByText("All open").closest(".rounded-lg")!;
    await user.click(within(tplCard as HTMLElement).getByRole("button", { name: "Use Template" }));

    await waitFor(() => {
      expect(screen.getByText("Recommended Templates")).toBeInTheDocument();
    });
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });

  it("Customize closes dialog and invokes onCustomize with template and approval mode", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    renderDialog({ onOpenChange });

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    const tplCard = screen.getByText("All open").closest(".rounded-lg")!;
    await user.click(within(tplCard as HTMLElement).getByRole("button", { name: "Customize" }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(onCustomize).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "tpl_a",
        action_type: "github.create_issue",
        name: "All open",
      }),
      "auto_approve",
    );
  });

  it("disables both buttons on all cards while create is pending", async () => {
    const user = userEvent.setup();
    let resolvePost: (v: unknown) => void;
    const pending = new Promise((resolve) => {
      resolvePost = resolve;
    });
    mockPost.mockReturnValue(pending);

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    const firstCard = screen.getByText("All open").closest(".rounded-lg")!;
    await user.click(
      within(firstCard as HTMLElement).getByRole("button", { name: "Use Template" }),
    );

    const useButtons = screen.getAllByRole("button", { name: "Use Template" });
    const customizeButtons = screen.getAllByRole("button", { name: "Customize" });
    for (const b of [...useButtons, ...customizeButtons]) {
      expect(b).toBeDisabled();
    }

    resolvePost!({
      data: {
        action_configuration: {
          id: "ac_new",
          agent_id: 42,
          connector_id: "github",
          action_type: "github.create_issue",
          parameters: {},
          status: "active",
          name: "All open",
          created_at: "2026-02-25T10:00:00Z",
          updated_at: "2026-02-25T10:00:00Z",
        },
      },
    });

    await waitFor(() => {
      expect(useButtons[0]).not.toBeDisabled();
    });
  });

  it("second click on Use Template does not fire while first is pending", async () => {
    const user = userEvent.setup();
    let resolvePost: (v: unknown) => void;
    const pending = new Promise((resolve) => {
      resolvePost = resolve;
    });
    mockPost.mockReturnValue(pending);

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    const tplCard = screen.getByText("All open").closest(".rounded-lg")!;
    const useBtn = within(tplCard as HTMLElement).getByRole("button", {
      name: "Use Template",
    });
    await user.click(useBtn);
    await user.click(useBtn);

    expect(mockPost).toHaveBeenCalledTimes(1);

    resolvePost!({
      data: {
        action_configuration: {
          id: "ac_new",
          agent_id: 42,
          connector_id: "github",
          action_type: "github.create_issue",
          parameters: {},
          status: "active",
          name: "All open",
          created_at: "2026-02-25T10:00:00Z",
          updated_at: "2026-02-25T10:00:00Z",
        },
      },
    });
  });

  it("defaults approval mode based on template standing_approval", async () => {
    renderDialog();
    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    // tpl_a has standing_approval → defaults to Auto-approve
    const tplACard = screen.getByText("All open").closest(".rounded-lg")!;
    const autoRadioA = within(tplACard as HTMLElement).getByRole("radio", { name: "Auto-approve" });
    expect(autoRadioA).toHaveAttribute("aria-checked", "true");

    // tpl_b has no standing_approval → defaults to Requires approval
    const tplBCard = screen.getByText("Merge main").closest(".rounded-lg")!;
    const reqRadioB = within(tplBCard as HTMLElement).getByRole("radio", { name: "Requires approval" });
    expect(reqRadioB).toHaveAttribute("aria-checked", "true");
  });

  it("sends toggled approval_mode when user switches before Use Template", async () => {
    const user = userEvent.setup();
    mockPost.mockResolvedValue({
      data: {
        action_configuration: {
          id: "ac_new",
          agent_id: 42,
          connector_id: "github",
          action_type: "github.merge_pr",
          parameters: { repo: "supersuit-tech/webapp", pr: 1 },
          status: "active",
          name: "Merge main",
          created_at: "2026-02-25T10:00:00Z",
          updated_at: "2026-02-25T10:00:00Z",
        },
      },
    });

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("Merge main")).toBeInTheDocument();
    });

    // tpl_b defaults to requires_approval — switch to auto_approve
    const tplBCard = screen.getByText("Merge main").closest(".rounded-lg")!;
    await user.click(within(tplBCard as HTMLElement).getByRole("radio", { name: "Auto-approve" }));
    await user.click(within(tplBCard as HTMLElement).getByRole("button", { name: "Use Template" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
    const [, opts] = mockPost.mock.calls[0] as [
      string,
      { body: { agent_id: number; approval_mode: string } },
    ];
    expect(opts.body.approval_mode).toBe("auto_approve");
  });

  it("sends requires_approval when user switches off auto-approve", async () => {
    const user = userEvent.setup();
    mockPost.mockResolvedValue({
      data: {
        action_configuration: {
          id: "ac_new",
          agent_id: 42,
          connector_id: "github",
          action_type: "github.create_issue",
          parameters: { repo: "*", title: "*" },
          status: "active",
          name: "All open",
          created_at: "2026-02-25T10:00:00Z",
          updated_at: "2026-02-25T10:00:00Z",
        },
      },
    });

    renderDialog();

    await waitFor(() => {
      expect(screen.getByText("All open")).toBeInTheDocument();
    });

    // tpl_a defaults to auto_approve — switch to requires_approval
    const tplACard = screen.getByText("All open").closest(".rounded-lg")!;
    await user.click(within(tplACard as HTMLElement).getByRole("radio", { name: "Requires approval" }));
    await user.click(within(tplACard as HTMLElement).getByRole("button", { name: "Use Template" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
    const [, opts] = mockPost.mock.calls[0] as [
      string,
      { body: { agent_id: number; approval_mode: string } },
    ];
    expect(opts.body.approval_mode).toBe("requires_approval");
  });
});
