import React, { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

// --- Mocks ---

jest.mock("../../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      getSession: jest.fn().mockResolvedValue({ data: { session: null }, error: null }),
      onAuthStateChange: jest.fn().mockReturnValue({
        data: { subscription: { unsubscribe: jest.fn() } },
      }),
      signInWithOtp: jest.fn(),
      verifyOtp: jest.fn(),
      signOut: jest.fn(),
    },
  },
}));

const mockAgents = [
  {
    agent_id: 42,
    status: "registered" as const,
    metadata: { name: "Deploy Bot" },
    created_at: "2026-01-01T00:00:00Z",
  },
];

jest.mock("../../../hooks/useAgents", () => ({
  useAgents: () => ({
    agents: mockAgents,
    isLoading: false,
    error: null,
  }),
  getAgentDisplayName: (agent: { agent_id: number; metadata?: unknown }) => {
    if (
      agent.metadata &&
      typeof agent.metadata === "object" &&
      "name" in agent.metadata
    ) {
      return (agent.metadata as { name: string }).name;
    }
    return `Agent ${agent.agent_id}`;
  },
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  SafeAreaProvider: ({ children }: { children: React.ReactNode }) => children,
}));

import ApprovalDetailScreen from "../ApprovalDetailScreen";

// --- Helpers ---

function makeApproval(overrides?: Partial<ApprovalSummary>): ApprovalSummary {
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

function renderDetail(approval: ApprovalSummary) {
  const route = {
    params: {
      approvalId: approval.approval_id,
      approval,
    },
    key: "test",
    name: "ApprovalDetail" as const,
  };
  const navigation = {} as any;

  return create(
    createElement(ApprovalDetailScreen, { route, navigation } as any),
  );
}

// --- Tests ---

describe("ApprovalDetailScreen", () => {
  let renderer: ReactTestRenderer;

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("renders without crashing", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(renderer.toJSON()).toBeTruthy();
  });

  it("displays agent name from agents list", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Deploy Bot");
  });

  it("displays action type", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("email.send");
  });

  it("displays context description", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Send welcome email to new user");
  });

  it("displays risk level badge", async () => {
    const approval = makeApproval({
      context: {
        description: "Delete database",
        risk_level: "high",
      },
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("High");
    expect(json).toContain("high-risk action");
  });

  it("displays parameter values", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("alice@example.com");
    expect(json).toContain("Welcome");
  });

  it("shows Approved banner for approved status", async () => {
    const approval = makeApproval({
      status: "approved",
      approved_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Approved");
  });

  it("shows Denied banner for denied status", async () => {
    const approval = makeApproval({
      status: "denied",
      denied_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Denied");
  });

  it("displays approval ID", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("appr_test123");
  });

  it("shows fallback agent name when not in agents list", async () => {
    const approval = makeApproval({ agent_id: 999 });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Agent 999");
  });
});
