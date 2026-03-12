import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockPost,
  mockPut,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ActionConfigurationsSection } from "../ActionConfigurationsSection";
import type { ActionConfiguration } from "../../../../hooks/useActionConfigs";
import type { ConnectorAction } from "../../../../hooks/useConnectorDetail";
import type { CredentialSummary } from "../../../../hooks/useCredentials";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const mockActions: ConnectorAction[] = [
  {
    action_type: "github.create_issue",
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

const mockCredentials: CredentialSummary[] = [
  {
    id: "cred_123",
    service: "github",
    label: "Personal Access Token",
    created_at: "2026-02-11T10:00:00Z",
  },
];

const mockConfigs: ActionConfiguration[] = [
  {
    id: "ac_001",
    agent_id: 42,
    connector_id: "github",
    action_type: "github.create_issue",
    credential_id: "cred_123",
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
      credentials={mockCredentials}
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
  });

  it("shows empty state with Enable All Actions button", () => {
    renderSection();
    expect(
      screen.getByText("Enable All Actions"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Your agent can use any action from this connector/),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Advanced: configure individual actions"),
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
    expect(screen.getByText("Personal Access Token")).toBeInTheDocument();
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

  it("shows 'Not assigned' when no credential bound", () => {
    const configWithoutCred: ActionConfiguration[] = [
      {
        ...mockConfigs[0]!,
        credential_id: null,
      },
    ];
    renderSection({ configs: configWithoutCred });
    expect(screen.getByText("Not assigned")).toBeInTheDocument();
  });

  it("hides Add Configuration button in empty state", () => {
    renderSection();
    expect(screen.queryByText("Add Configuration")).not.toBeInTheDocument();
  });

  it("shows Add Configuration button when configs exist", () => {
    renderSection({ configs: mockConfigs });
    expect(screen.getByText("Add Configuration")).toBeInTheDocument();
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
      screen.getByLabelText("repo"),
      "my-repo",
    );

    // Set title to wildcard via dropdown.
    // Target the dropdown trigger within the "title" parameter group.
    const titleInput = screen.getByLabelText("title");
    const titleGroup = titleInput.closest(".space-y-1\\.5")!;
    await user.click(within(titleGroup as HTMLElement).getByRole("button", { name: /Fixed/ }));
    // Select "Wildcard" from the dropdown menu.
    await user.click(screen.getByRole("menuitemradio", { name: /Wildcard/ }));

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

  it("calls create with wildcard action_type when clicking Enable All Actions", async () => {
    const user = userEvent.setup();
    mockPost.mockResolvedValue({
      data: {
        id: "ac_wildcard",
        agent_id: 42,
        connector_id: "github",
        action_type: "*",
        parameters: {},
        status: "active",
        name: "All GitHub Actions",
        created_at: "2026-03-11T10:00:00Z",
        updated_at: "2026-03-11T10:00:00Z",
      },
    });

    renderSection();
    await user.click(screen.getByText("Enable All Actions"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith("/v1/action-configurations", {
        headers: { Authorization: "Bearer token" },
        body: {
          agent_id: 42,
          connector_id: "github",
          action_type: "*",
          name: "All GitHub Actions",
          parameters: {},
        },
      });
    });
  });

  it("renders wildcard config with All Actions badge", () => {
    const wildcardConfig: ActionConfiguration[] = [
      {
        id: "ac_wildcard",
        agent_id: 42,
        connector_id: "github",
        action_type: "*",
        credential_id: null,
        parameters: {},
        status: "active",
        name: "All GitHub Actions",
        description: null,
        created_at: "2026-03-11T10:00:00Z",
        updated_at: "2026-03-11T10:00:00Z",
      },
    ];
    renderSection({ configs: wildcardConfig });

    expect(screen.getByText("All GitHub Actions")).toBeInTheDocument();
    expect(screen.getByText("All Actions")).toBeInTheDocument();
    expect(
      screen.getByText(/All parameters — agent chooses freely/),
    ).toBeInTheDocument();
  });

  it("shows Enable All Actions footer link when configs exist and none are wildcard", () => {
    renderSection({ configs: mockConfigs });
    // The table has configs, so a footer link should appear
    const footerLink = screen.getByRole("button", { name: /Enable All Actions/ });
    expect(footerLink).toBeInTheDocument();
  });

  it("hides Enable All Actions footer link when a wildcard config exists", () => {
    const wildcardConfig: ActionConfiguration[] = [
      {
        id: "ac_wildcard",
        agent_id: 42,
        connector_id: "github",
        action_type: "*",
        credential_id: null,
        parameters: {},
        status: "active",
        name: "All GitHub Actions",
        description: null,
        created_at: "2026-03-11T10:00:00Z",
        updated_at: "2026-03-11T10:00:00Z",
      },
    ];
    renderSection({ configs: wildcardConfig });
    // The "Enable All Actions" text in the table row should exist but not a footer button
    const allButtons = screen.getAllByText("All Actions");
    // Only the badge in the table, not a footer link
    expect(allButtons).toHaveLength(1);
  });

  it("shows advanced option to add custom config in empty state", async () => {
    const user = userEvent.setup();
    renderSection();

    // Click the advanced toggle
    await user.click(
      screen.getByText("Advanced: configure individual actions"),
    );
    expect(
      screen.getByText("Add Custom Configuration"),
    ).toBeInTheDocument();
  });
});
