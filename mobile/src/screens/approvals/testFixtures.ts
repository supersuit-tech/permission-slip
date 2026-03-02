import type { ApprovalSummary } from "../../hooks/useApprovals";

export const MOCK_AGENTS = [
  {
    agent_id: 42,
    status: "registered" as const,
    metadata: { name: "Deploy Bot" },
    created_at: "2026-01-01T00:00:00Z",
  },
];

export function mockGetAgentDisplayName(agent: {
  agent_id: number;
  metadata?: unknown;
}): string {
  if (
    agent.metadata &&
    typeof agent.metadata === "object" &&
    "name" in agent.metadata
  ) {
    return (agent.metadata as { name: string }).name;
  }
  return `Agent ${agent.agent_id}`;
}

/** Creates a test ApprovalSummary with sensible defaults. */
export function makeApproval(
  overrides?: Partial<ApprovalSummary>,
): ApprovalSummary {
  return {
    approval_id: "appr_test123",
    agent_id: 42,
    action: {
      type: "email.send",
      version: "1",
      parameters: {
        to: ["alice@example.com"],
        subject: "Welcome",
        body: "Hello!",
      },
    },
    context: {
      description: "Send welcome email to new user",
      risk_level: "low",
    },
    status: "pending",
    expires_at: new Date(Date.now() + 300_000).toISOString(),
    created_at: new Date(Date.now() - 60_000).toISOString(),
    ...overrides,
  };
}
