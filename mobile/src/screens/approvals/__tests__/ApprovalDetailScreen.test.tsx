import React, { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { ApprovalSummary } from "../../../hooks/useApprovals";
import { makeApproval, MOCK_AGENTS, mockGetAgentDisplayName } from "../testFixtures";

// --- Mocks ---

const mockDenyApproval = jest.fn();

jest.mock("../../../hooks/useDenyApproval", () => ({
  useDenyApproval: () => ({
    denyApproval: mockDenyApproval,
    isPending: false,
  }),
}));

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

jest.mock("../../../hooks/useAgents", () => ({
  useAgents: () => ({
    agents: MOCK_AGENTS,
    isLoading: false,
  }),
  getAgentDisplayName: mockGetAgentDisplayName,
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  SafeAreaProvider: ({ children }: { children: React.ReactNode }) => children,
}));

import ApprovalDetailScreen from "../ApprovalDetailScreen";

// --- Helpers ---

function renderDetail(approval: ApprovalSummary) {
  const route = {
    params: {
      approvalId: approval.approval_id,
      approval,
    },
    key: "test",
    name: "ApprovalDetail" as const,
  };
  const navigation = {
    canGoBack: jest.fn().mockReturnValue(true),
    goBack: jest.fn(),
  } as any;

  return create(
    createElement(ApprovalDetailScreen, { route, navigation } as any),
  );
}

function hasDenyButton(renderer: ReactTestRenderer): boolean {
  const json = JSON.stringify(renderer.toJSON());
  return json.includes('"testID":"deny-button"');
}

// --- Tests ---

describe("ApprovalDetailScreen", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.useFakeTimers();
    mockDenyApproval.mockReset();
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
    jest.useRealTimers();
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

  // --- Deny action visibility ---

  it("shows deny button for pending approval", async () => {
    const approval = makeApproval({ status: "pending" });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasDenyButton(renderer)).toBe(true);
  });

  it("does not show deny button for denied approval", async () => {
    const approval = makeApproval({
      status: "denied",
      denied_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasDenyButton(renderer)).toBe(false);
  });

  it("does not show deny button for approved approval", async () => {
    const approval = makeApproval({
      status: "approved",
      approved_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasDenyButton(renderer)).toBe(false);
  });

  it("does not show deny button for expired approval", async () => {
    const approval = makeApproval({
      status: "pending",
      expires_at: new Date(Date.now() - 60_000).toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasDenyButton(renderer)).toBe(false);
  });
});
