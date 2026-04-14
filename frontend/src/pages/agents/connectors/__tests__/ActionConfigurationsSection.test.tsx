import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockPut,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ActionConfigurationsSection } from "../ActionConfigurationsSection";
import type { ActionConfiguration } from "../../../../hooks/useActionConfigs";
import type { ConnectorAction } from "../../../../hooks/useConnectorDetail";
vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const mockActions: ConnectorAction[] = [
  {
    action_type: "github.create_issue",
    operation_type: "write",
    name: "Create Issue",
    description: "Create a new issue in a repository",
    risk_level: "low",
    requires_payment_method: false,
    parameters_schema: {
      type: "object",
      required: ["repo", "title"],
      properties: {
        repo: { type: "string", description: "Repository name" },
        title: { type: "string", description: "Issue title" },
        body: { type: "string", description: "Issue body" },
      },
    },
  },
  {
    action_type: "github.merge_pr",
    operation_type: "write",
    name: "Merge Pull Request",
    description: "Merge an open pull request",
    risk_level: "high",
    requires_payment_method: false,
    parameters_schema: {
      type: "object",
      required: ["repo", "pr"],
      properties: {
        repo: { type: "string" },
        pr: { type: "integer" },
      },
    },
  },
];

const mockConfigs: ActionConfiguration[] = [
  {
    id: "ac_001",
    agent_id: 42,
    connector_id: "github",
    action_type: "github.create_issue",
    parameters: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    status: "active",
    name: "Create bug issues",
    description: "Create issues in the main repo",
    created_at: "2026-02-20T10:00:00Z",
    updated_at: "2026-02-20T10:00:00Z",
  },
];

function renderSection({
  configs = [] as ActionConfiguration[],
  isLoading = false,
  error = null as string | null,
  connectorName = "GitHub",
} = {}) {
  return renderWithProviders(
    <ActionConfigurationsSection
      agentId={42}
      connectorId="github"
      connectorName={connectorName}
      actions={mockActions}
      configs={configs}
      isLoading={isLoading}
      error={error}
    />,
  );
}

describe("ActionConfigurationsSection", () => {
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
  });

  it("shows empty state with Add Configuration button", () => {
    renderSection();
    expect(
      screen.getByRole("button", { name: /Add Configuration/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Define which actions this agent can use and lock in parameter values/,
      ),
    ).toBeInTheDocument();
  });

  it("shows loading state", () => {
    renderSection({ isLoading: true });
    expect(screen.getByText("Action Configurations")).toBeInTheDocument();
  });

  it("shows error state", () => {
    renderSection({ error: "Failed to load" });
    expect(screen.getByText("Failed to load")).toBeInTheDocument();
  });

  it("renders existing configurations in a table", () => {
    renderSection({ configs: mockConfigs });

    expect(screen.getByText("Create bug issues")).toBeInTheDocument();
    expect(
      screen.getByText("Create issues in the main repo"),
    ).toBeInTheDocument();
    expect(screen.getByText("Create Issue")).toBeInTheDocument();
    expect(screen.getByText("github.create_issue")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows parameter pills with wildcard distinction", () => {
    renderSection({ configs: mockConfigs });

    // Fixed param
    expect(screen.getByText("repo:")).toBeInTheDocument();
    expect(
      screen.getByText("supersuit-tech/webapp"),
    ).toBeInTheDocument();

    // Wildcard params show *
    const wildcardBadges = screen.getAllByText("*");
    expect(wildcardBadges.length).toBe(2); // title and body
  });

  it("shows Add Configuration button when configs exist", () => {
    renderSection({ configs: mockConfigs });
    expect(screen.getByText("Add Configuration")).toBeInTheDocument();
  });

  it("opens Add dialog when clicking empty-state Add Configuration button", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(
      screen.getByRole("button", { name: /Add Configuration/ }),
    );
    expect(screen.getByText("Add Action Configuration")).toBeInTheDocument();
  });

  it("opens Add dialog when clicking Add Configuration with configs", async () => {
    const user = userEvent.setup();
    renderSection({ configs: mockConfigs });

    await user.click(screen.getByText("Add Configuration"));
    expect(
      screen.getByText("Add Action Configuration"),
    ).toBeInTheDocument();
    expect(screen.getByText("Select an action...")).toBeInTheDocument();
  });

  it("creates a new configuration through Add dialog", async () => {
    const user = userEvent.setup();
    mockPost.mockResolvedValue({
      data: {
        id: "ac_new",
        agent_id: 42,
        connector_id: "github",
        action_type: "github.create_issue",
        parameters: { repo: "my-repo", title: "*" },
        status: "active",
        name: "Test config",
        created_at: "2026-02-25T10:00:00Z",
        updated_at: "2026-02-25T10:00:00Z",
      },
    });

    renderSection({ configs: mockConfigs });
    await user.click(screen.getByText("Add Configuration"));

    // Select action
    await user.selectOptions(
      screen.getByLabelText("Action"),
      "github.create_issue",
    );

    // Fill in name
    await user.type(screen.getByLabelText("Name"), "Test config");

    // Fill in repo parameter with a fixed value
    await user.type(
      screen.getByLabelText("Repo"),
      "my-repo",
    );

    // Set title to wildcard via "Any value" checkbox.
    const titleInput = screen.getByLabelText("Title");
    const titleGroup = titleInput.closest(".space-y-1\\.5")!;
    await user.click(within(titleGroup as HTMLElement).getByRole("checkbox"));

    await user.click(screen.getByText("Create Configuration"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith("/v1/action-configurations", {
        headers: { Authorization: "Bearer token" },
        body: {
          agent_id: 42,
          connector_id: "github",
          action_type: "github.create_issue",
          name: "Test config",
          parameters: { repo: "my-repo", title: "*" },
        },
      });
    });
  });

  it("opens Edit dialog when clicking edit button", async () => {
    const user = userEvent.setup();
    renderSection({ configs: mockConfigs });

    await user.click(screen.getByLabelText("Edit Create bug issues"));
    expect(
      screen.getByText("Edit Action Configuration"),
    ).toBeInTheDocument();
    // Name should be pre-populated
    expect(screen.getByDisplayValue("Create bug issues")).toBeInTheDocument();
  });

  it("updates a configuration through Edit dialog", async () => {
    const user = userEvent.setup();
    mockPut.mockResolvedValue({
      data: {
        ...mockConfigs[0],
        name: "Updated name",
      },
    });

    renderSection({ configs: mockConfigs });
    await user.click(screen.getByLabelText("Edit Create bug issues"));

    const nameInput = screen.getByDisplayValue("Create bug issues");
    await user.clear(nameInput);
    await user.type(nameInput, "Updated name");
    await user.click(screen.getByText("Save Changes"));

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith(
        "/v1/action-configurations/{config_id}",
        expect.objectContaining({
          headers: { Authorization: "Bearer token" },
          params: { path: { config_id: "ac_001" } },
          body: expect.objectContaining({
            name: "Updated name",
          }),
        }),
      );
    });
  });

  it("opens Delete dialog when clicking delete button", async () => {
    const user = userEvent.setup();
    renderSection({ configs: mockConfigs });

    await user.click(screen.getByLabelText("Delete Create bug issues"));
    expect(
      screen.getByText("Delete Action Configuration"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/This will permanently delete/),
    ).toBeInTheDocument();
  });

  it("deletes a configuration through Delete dialog", async () => {
    const user = userEvent.setup();
    mockDelete.mockResolvedValue({
      data: { id: "ac_001", deleted_at: "2026-02-25T15:00:00Z" },
    });

    renderSection({ configs: mockConfigs });
    await user.click(screen.getByLabelText("Delete Create bug issues"));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith(
        "/v1/action-configurations/{config_id}",
        {
          headers: { Authorization: "Bearer token" },
          params: { path: { config_id: "ac_001" } },
        },
      );
    });
  });

  it("shows disabled status badge", () => {
    const disabledConfig: ActionConfiguration[] = [
      {
        ...mockConfigs[0]!,
        status: "disabled",
      },
    ];
    renderSection({ configs: disabledConfig });
    expect(screen.getByText("Disabled")).toBeInTheDocument();
  });

  it("hides Recommended Templates triggers while templates are loading", () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return new Promise(() => {});
      }
      return Promise.resolve({ data: null });
    });

    renderSection();
    expect(
      screen.queryByRole("button", { name: "Recommended Templates" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("Or start from a recommended template →"),
    ).not.toBeInTheDocument();
  });

  it("shows empty-state recommended template link when templates exist", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: {
            data: [
              {
                id: "tpl_1",
                connector_id: "github",
                action_type: "github.create_issue",
                name: "T",
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

    renderSection();

    await waitFor(() => {
      expect(
        screen.getByText("Or start from a recommended template →"),
      ).toBeInTheDocument();
    });
  });

  it("shows header Recommended Templates button when configs exist and templates exist", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: {
            data: [
              {
                id: "tpl_1",
                connector_id: "github",
                action_type: "github.create_issue",
                name: "T",
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

    renderSection({ configs: mockConfigs });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Recommended Templates" }),
      ).toBeInTheDocument();
    });
  });

  it("hides Recommended Templates button when every matching template is already configured", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: {
            data: [
              {
                id: "tpl_1",
                connector_id: "github",
                action_type: "github.create_issue",
                name: "Matches existing",
                description: null,
                parameters: {
                  repo: "supersuit-tech/webapp",
                  title: "*",
                  body: "*",
                },
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    // mockConfigs[0].parameters equals the template parameters above
    renderSection({ configs: mockConfigs });

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalled();
    });
    expect(
      screen.queryByRole("button", { name: "Recommended Templates" }),
    ).not.toBeInTheDocument();
  });

  it("keeps Recommended Templates button visible when an applied config diverges from the template", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return Promise.resolve({
          data: {
            data: [
              {
                id: "tpl_1",
                connector_id: "github",
                action_type: "github.create_issue",
                name: "Still recommended",
                description: null,
                // Different params than mockConfigs[0]
                parameters: { repo: "*", title: "*" },
                created_at: "2026-01-01T00:00:00Z",
              },
            ],
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    renderSection({ configs: mockConfigs });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Recommended Templates" }),
      ).toBeInTheDocument();
    });
  });

  it("hides recommended triggers when API returns no matching templates", async () => {
    let resolveTemplates: (v: unknown) => void;
    const templatesPromise = new Promise((resolve) => {
      resolveTemplates = resolve;
    });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/action-config-templates") {
        return templatesPromise;
      }
      return Promise.resolve({ data: null });
    });

    renderSection({ configs: mockConfigs });

    expect(
      screen.queryByRole("button", { name: "Recommended Templates" }),
    ).not.toBeInTheDocument();

    resolveTemplates!({
      data: {
        data: [
          {
            id: "tpl_stale",
            connector_id: "github",
            action_type: "removed.action",
            name: "Stale",
            description: null,
            parameters: {},
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
      },
    });

    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: "Recommended Templates" }),
      ).not.toBeInTheDocument();
    });
  });
});
