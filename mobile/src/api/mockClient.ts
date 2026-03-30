/**
 * Mock API client for local development without a running backend.
 *
 * Returns realistic fixture data for all endpoints the mobile app uses.
 * Only active when EXPO_PUBLIC_MOCK_AUTH=true in __DEV__ mode.
 */
import type { components } from "./schema";

type ApprovalSummary = components["schemas"]["ApprovalSummary"];

const now = new Date();
const inOneHour = new Date(now.getTime() + 60 * 60 * 1000).toISOString();
const fiveMinAgo = new Date(now.getTime() - 5 * 60 * 1000).toISOString();
const tenMinAgo = new Date(now.getTime() - 10 * 60 * 1000).toISOString();
const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000).toISOString();

const MOCK_APPROVALS: ApprovalSummary[] = [
  {
    approval_id: "appr_mock_001",
    agent_id: 1,
    action: {
      type: "email.send",
      version: "1",
      parameters: {
        to: ["alice@example.com"],
        subject: "Weekly report summary",
      },
    },
    context: {
      description: "Send weekly analytics report to Alice",
      risk_level: "low",
    },
    status: "pending",
    expires_at: inOneHour,
    created_at: fiveMinAgo,
  },
  {
    approval_id: "appr_mock_002",
    agent_id: 2,
    action: {
      type: "slack.post",
      version: "1",
      parameters: {
        channel: "#engineering",
        message: "Deploy v2.4.1 to production",
      },
    },
    context: {
      description: "Post deployment notification to Slack",
      risk_level: "medium",
    },
    status: "pending",
    expires_at: inOneHour,
    created_at: tenMinAgo,
  },
  {
    approval_id: "appr_mock_003",
    agent_id: 1,
    action: {
      type: "db.delete",
      version: "1",
      parameters: {
        table: "inactive_users",
        filter: "last_login < 2025-01-01",
      },
    },
    context: {
      description: "Purge inactive user accounts older than 1 year",
      risk_level: "high",
    },
    status: "pending",
    expires_at: inOneHour,
    created_at: tenMinAgo,
  },
  {
    approval_id: "appr_mock_004",
    agent_id: 3,
    action: {
      type: "email.send",
      version: "1",
      parameters: {
        to: ["team@example.com"],
        subject: "Sprint retrospective notes",
      },
    },
    context: {
      description: "Send sprint retro summary to the team",
      risk_level: "low",
    },
    status: "approved",
    expires_at: oneHourAgo,
    created_at: oneHourAgo,
  },
  {
    approval_id: "appr_mock_005",
    agent_id: 2,
    action: {
      type: "github.merge",
      version: "1",
      parameters: {
        repo: "supersuit-tech/permission-slip",
        pr: 42,
      },
    },
    context: {
      description: "Auto-merge PR #42 after CI passes",
      risk_level: "medium",
    },
    status: "denied",
    expires_at: oneHourAgo,
    created_at: oneHourAgo,
  },
];

/** Simulate network latency for realistic feel. */
function delay(ms = 300): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

/**
 * Mock client that matches the openapi-fetch interface used by API hooks.
 * Only implements the endpoints the mobile app actually calls.
 */
const mockClient = {
  GET: async (url: string, options?: { params?: { query?: Record<string, string> } }) => {
    await delay();

    if (url === "/v1/approvals") {
      const status = options?.params?.query?.status ?? "pending";
      const filtered =
        status === "all"
          ? MOCK_APPROVALS
          : MOCK_APPROVALS.filter((a) => a.status === status);
      return { data: { data: filtered, has_more: false }, error: undefined };
    }

    return { data: undefined, error: { message: `Mock: unhandled GET ${url}` } };
  },

  POST: async (url: string) => {
    await delay();

    const match = url.match(/\/v1\/approvals\/([^/]+)\/(approve|deny)/);
    if (match) {
      const [, id, action] = match;
      const entry = MOCK_APPROVALS.find((a) => a.approval_id === id);
      if (entry) entry.status = action === "approve" ? "approved" : "denied";
      return action === "approve"
        ? { data: { confirmation_code: "ABC-123" }, error: undefined }
        : { data: {}, error: undefined };
    }

    return { data: undefined, error: { message: `Mock: unhandled POST ${url}` } };
  },
};

export default mockClient;
