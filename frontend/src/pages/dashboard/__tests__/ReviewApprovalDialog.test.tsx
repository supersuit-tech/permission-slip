import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { ReviewApprovalDialog } from "../ReviewApprovalDialog";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

vi.mock("../../../api/client");

const futureDate = new Date(Date.now() + 600_000).toISOString();

function makeApproval(overrides?: Partial<ApprovalSummary>): ApprovalSummary {
  return {
    approval_id: "appr_test123",
    agent_id: 1,
    action: {
      type: "email.send",
      version: "1",
      parameters: { recipient: "user@example.com", subject: "Hello" },
    },
    context: {
      description: "Send an email",
      risk_level: "low",
    },
    status: "pending",
    expires_at: futureDate,
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as ApprovalSummary;
}

const mockAgents = [
  {
    agent_id: 1,
    status: "registered" as const,
    metadata: { name: "Test Bot" },
    confirmation_code: null,
    expires_at: null,
    created_at: "2026-01-01T00:00:00Z",
  },
];

function setupMocks({
  standingApprovals = [] as Array<{ agent_id: number; action_type: string }>,
} = {}) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/agents") {
      return Promise.resolve({ data: { data: mockAgents } });
    }
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
    }
    if (url.startsWith("/v1/connectors/")) {
      return Promise.resolve({ data: { id: "email", name: "Email", actions: [] } });
    }
    return Promise.resolve({ data: {} });
  });
}

function mockApproveSuccess() {
  mockPost.mockResolvedValueOnce({
    data: {
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
      confirmation_code: "ABC12-3DEFG",
      execution_status: "success",
      execution_result: null,
    },
  });
}

describe("ReviewApprovalDialog — auto-approve future requests", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("shows checkbox when no matching standing approval exists", async () => {
    setupMocks();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });
  });

  it("shows checkbox for parameterless actions", async () => {
    setupMocks();
    const approval = makeApproval({
      action: { type: "google.list_calendars", version: "1", parameters: {} },
    });

    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("Always allow this action")).not.toBeInTheDocument();
  });

  it("hides checkbox when a standing approval already exists for agent+action", async () => {
    setupMocks({
      standingApprovals: [{ agent_id: 1, action_type: "email.send" }],
    });

    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Approve")).toBeInTheDocument();
    });

    expect(
      screen.queryByLabelText("Auto-approve all future requests like this"),
    ).not.toBeInTheDocument();
  });

  it("ticking checkbox + Approve calls approve then createStandingApproval with pinned params", async () => {
    setupMocks();
    mockApproveSuccess();
    mockPost.mockResolvedValueOnce({
      data: {
        standing_approval_id: "sa_new",
        agent_id: 1,
        action_type: "email.send",
        status: "active",
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Auto-approve all future requests like this"),
    );
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(mockPost).toHaveBeenCalledTimes(2);
    const createCall = mockPost.mock.calls.find(
      (call) => call[0] === "/v1/standing-approvals/create",
    );
    expect(createCall).toBeDefined();
    expect(createCall?.[1]?.body).toMatchObject({
      agent_id: 1,
      action_type: "email.send",
      action_version: "1",
      constraints: {},
      expires_at: null,
    });

    expect(
      screen.getByText("Future matching requests will be auto-approved."),
    ).toBeInTheDocument();
  });

  it("creates standing approval directly from the approval action", async () => {
    setupMocks();
    mockApproveSuccess();
    mockPost.mockResolvedValueOnce({
      data: {
        standing_approval_id: "sa_new",
        agent_id: 1,
        action_type: "email.send",
        status: "active",
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Auto-approve all future requests like this"),
    );
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    const createCall = mockPost.mock.calls.find(
      (call) => call[0] === "/v1/standing-approvals/create",
    );
    expect(createCall).toBeDefined();
    expect(createCall?.[1]?.body).toMatchObject({
      agent_id: 1,
      action_type: "email.send",
      name: "Send an email",
      description: "Send an email",
      constraints: {
        recipient: "user@example.com",
        subject: "Hello",
      },
      expires_at: null,
    });

    expect(
      screen.getByText("Future matching requests will be auto-approved."),
    ).toBeInTheDocument();
  });

  it("standing approval failure does not block approve from succeeding", async () => {
    setupMocks();
    mockApproveSuccess();
    mockPost.mockRejectedValueOnce(new Error("Standing approval failed"));

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Auto-approve all future requests like this"),
    );
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(
      screen.queryByText("Future matching requests will be auto-approved."),
    ).not.toBeInTheDocument();
  });
});

describe("ReviewApprovalDialog — iMessage participants", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
    setupMocks();
  });

  it("shows participants row when resource_details includes handles", async () => {
    render(
      <ReviewApprovalDialog
        approval={makeApproval({
          action: {
            type: "imessage.send_message",
            version: "1",
            parameters: { chat_id: 1, text: "Hello" },
          },
          resource_details: {
            chat_name: "with Ben Kilmer",
            participants: ["+15551234567"],
          },
        })}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(screen.getByTestId("imessage-participants-row")).toBeInTheDocument();
    });
    expect(screen.getByText("Participants")).toBeInTheDocument();
    expect(screen.getByTestId("imessage-participants-row")).toHaveTextContent(
      "+15551234567",
    );
  });

  it("omits participants row when handles are absent", async () => {
    render(
      <ReviewApprovalDialog
        approval={makeApproval({
          action: {
            type: "imessage.send_message",
            version: "1",
            parameters: { chat_id: 1, text: "Hello" },
          },
          resource_details: {
            chat_name: "with Ben Kilmer",
          },
        })}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(screen.getByText("Parameters")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("imessage-participants-row")).not.toBeInTheDocument();
  });

  it("Shared Drive always-allow creates the Drive + Sheets workspace set", async () => {
    setupMocks();
    mockPost.mockImplementation((url: string) => {
      if (url === "/v1/approvals/{approval_id}/approve") {
        return Promise.resolve({
          data: {
            approval_id: "appr_test123",
            status: "approved",
            approved_at: new Date().toISOString(),
            confirmation_code: "ABC12-3DEFG",
            execution_status: "success",
            execution_result: null,
          },
        });
      }
      if (url === "/v1/standing-approvals/create") {
        return Promise.resolve({
          data: {
            standing_approval_id: "sa_new",
            agent_id: 1,
            action_type: "google.upload_drive_file",
            status: "active",
          },
        });
      }
      return Promise.resolve({ data: {} });
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval({
          action: {
            type: "google.upload_drive_file",
            version: "1",
            parameters: {
              name: "receipt.pdf",
              folder_id: "1nestedFolder",
            },
          },
          context: { description: "upload receipt", risk_level: "medium" },
          resource_details: {
            folder_name: "2026-639-receipts in Assistant Drive",
            drive_id: "0AKbIIKZ8knmBUk9PVA",
            drive_name: "Assistant Drive",
          },
        })}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    const checkboxLabel =
      "Auto-approve routine Drive and Sheets work inside Assistant Drive";
    await waitFor(() => {
      expect(screen.getByLabelText(checkboxLabel)).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText(checkboxLabel));
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(
        screen.getByText(
          "Future Drive and Sheets work inside this Shared Drive will be auto-approved.",
        ),
      ).toBeInTheDocument();
    });

    const createCalls = mockPost.mock.calls.filter(
      (call) => call[0] === "/v1/standing-approvals/create",
    );
    expect(createCalls).toHaveLength(9);
    expect(createCalls.map((call) => call[1]?.body?.action_type)).toEqual([
      "google.upload_drive_file",
      "google.create_drive_folder",
      "google.list_drive_files",
      "google.search_drive",
      "google.get_drive_file",
      "google.sheets_read_range",
      "google.sheets_write_range",
      "google.sheets_append_rows",
      "google.sheets_list_sheets",
    ]);
    expect(createCalls[5]?.[1]?.body?.constraints).toEqual({
      spreadsheet_id: "*",
      range: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
  });
});
