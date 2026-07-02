import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { ReviewBulkApprovalDialog } from "../ReviewBulkApprovalDialog";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

vi.mock("../../../api/client");

const futureDate = new Date(Date.now() + 600_000).toISOString();

const calendarDisplayTemplate =
  "Create event {{summary}} on {{start_time:datetime}} with {{attendees:count}} attendees";

const mockGoogleConnector = {
  id: "google",
  name: "Google",
  actions: [
    {
      action_type: "google.create_calendar_event",
      name: "Create Calendar Event",
      display_template: calendarDisplayTemplate,
      parameters_schema: {
        type: "object",
        required: ["summary", "start_time", "end_time"],
        properties: {
          summary: { type: "string", description: "Event title" },
          start_time: { type: "string", description: "Start time in RFC 3339 format" },
          end_time: { type: "string", description: "End time in RFC 3339 format" },
          calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
          attendees: { type: "array", description: "Attendee emails" },
        },
      },
    },
  ],
};

function makeCalendarItem(
  approvalId: string,
  summary: string,
  startTime: string,
): ApprovalSummary {
  return {
    approval_id: approvalId,
    agent_id: 1,
    action: {
      type: "google.create_calendar_event",
      version: "1",
      parameters: {
        calendar_id: "primary",
        summary,
        start_time: startTime,
        end_time: "2026-10-31T00:00:00-04:00",
        attendees: [],
      },
    },
    context: { description: "Create calendar event", risk_level: "medium" },
    status: "pending",
    expires_at: futureDate,
    created_at: "2026-01-01T00:00:00Z",
  } as ApprovalSummary;
}

const mockBulkGroup = {
  bulk_group_id: "bulk_test123",
  action_type: "google.create_calendar_event",
  item_count: 2,
  expires_at: futureDate,
  items: [
    makeCalendarItem("appr_1", "Team offsite", "2026-10-03T00:00:00-04:00"),
    makeCalendarItem("appr_2", "Sprint planning", "2026-10-10T00:00:00-04:00"),
  ],
};

function setupMocks({ connectorLoaded = true } = {}) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith("/v1/approval-groups/")) {
      return Promise.resolve({ data: mockBulkGroup });
    }
    if (url.startsWith("/v1/connectors/")) {
      if (connectorLoaded) {
        return Promise.resolve({ data: mockGoogleConnector });
      }
      return new Promise(() => {});
    }
    return Promise.resolve({ data: {} });
  });
}

describe("ReviewBulkApprovalDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders template-based summaries with event titles", async () => {
    setupMocks();
    render(
      <ReviewBulkApprovalDialog
        bulkGroupId="bulk_test123"
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText(/Team offsite/)).toBeInTheDocument();
    });
    expect(screen.getByText(/Sprint planning/)).toBeInTheDocument();
    expect(screen.queryByText(/2026-10-03T00:00:00/)).not.toBeInTheDocument();
  });

  it("falls back gracefully while connector detail is loading", async () => {
    setupMocks({ connectorLoaded: false });
    render(
      <ReviewBulkApprovalDialog
        bulkGroupId="bulk_test123"
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText(/Team offsite/)).toBeInTheDocument();
    });
  });
});
