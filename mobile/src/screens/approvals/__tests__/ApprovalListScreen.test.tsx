import React, { createElement } from "react";
import { Text } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { ApprovalSummary } from "../../../hooks/useApprovals";
import { MOCK_AGENTS, mockGetAgentDisplayName } from "../testFixtures";

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

const mockApprovals: ApprovalSummary[] = [
  {
    approval_id: "appr_001",
    agent_id: 42,
    action: {
      type: "email.send",
      version: "1",
      parameters: {
        to: ["bob@example.com"],
        subject: "Test",
      },
    },
    context: {
      description: "Send test email",
      risk_level: "low",
    },
    status: "pending",
    expires_at: new Date(Date.now() + 300_000).toISOString(),
    created_at: new Date().toISOString(),
  },
];

let mockUseApprovalsReturn = {
  approvals: mockApprovals,
  hasMore: false,
  isLoading: false,
  isRefetching: false,
  error: null as string | null,
  refetch: jest.fn(),
  dataUpdatedAt: Date.now(),
};

jest.mock("../../../hooks/useApprovals", () => ({
  useApprovals: () => mockUseApprovalsReturn,
}));

jest.mock("../../../hooks/useAgents", () => ({
  useAgents: () => ({
    agents: MOCK_AGENTS.map((a) => ({ ...a, status: "registered" })),
    isLoading: false,
    error: null,
  }),
  getAgentDisplayName: mockGetAgentDisplayName,
}));

jest.mock("../../../auth/AuthContext", () => ({
  useAuth: () => ({
    signOut: jest.fn().mockResolvedValue({ error: null }),
    session: null,
    user: null,
    authStatus: "authenticated",
  }),
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  SafeAreaProvider: ({ children }: { children: React.ReactNode }) => children,
}));

jest.mock("@react-navigation/native", () => ({
  useIsFocused: () => true,
}));

import ApprovalListScreen from "../ApprovalListScreen";

// --- Helpers ---

function renderList() {
  const navigation = { navigate: jest.fn() } as any;
  const route = { key: "test", name: "ApprovalList" as const, params: undefined };
  return create(
    createElement(ApprovalListScreen, { navigation, route } as any),
  );
}

/** Extracts all text content from the rendered tree. */
function getAllText(renderer: ReactTestRenderer): string {
  const texts = renderer.root.findAllByType(Text);
  return texts.map((t) => {
    const children = t.props.children;
    if (typeof children === "string") return children;
    if (Array.isArray(children)) return children.filter((c) => typeof c === "string").join("");
    return "";
  }).join(" ");
}

// --- Tests ---

describe("ApprovalListScreen", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.useFakeTimers();
    mockUseApprovalsReturn = {
      approvals: mockApprovals,
      hasMore: false,
      isLoading: false,
      isRefetching: false,
      error: null,
      refetch: jest.fn(),
      dataUpdatedAt: Date.now(),
    };
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
    jest.useRealTimers();
  });

  it("renders without crashing", async () => {
    await act(async () => {
      renderer = renderList();
    });
    expect(renderer.toJSON()).toBeTruthy();
  });

  it("shows tab bar with pending, approved, denied", async () => {
    await act(async () => {
      renderer = renderList();
    });
    const allText = getAllText(renderer);
    expect(allText).toContain("Pending");
    expect(allText).toContain("Approved");
    expect(allText).toContain("Denied");
  });

  it("shows the title", async () => {
    await act(async () => {
      renderer = renderList();
    });
    const allText = getAllText(renderer);
    expect(allText).toContain("Permission Slip");
  });

  it("shows loading indicator when loading", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
      isLoading: true,
    };
    await act(async () => {
      renderer = renderList();
    });
    const root = renderer.root;
    const indicators = root.findAllByProps({ testID: "loading-indicator" });
    expect(indicators.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error state with retry button", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
      error: "Unable to load approvals. Please try again later.",
    };
    await act(async () => {
      renderer = renderList();
    });
    const allText = getAllText(renderer);
    expect(allText).toContain("Unable to load approvals");
    expect(allText).toContain("Retry");
  });

  it("shows empty state when no approvals", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
    };
    await act(async () => {
      renderer = renderList();
    });
    const allText = getAllText(renderer);
    expect(allText).toContain("No pending requests");
  });

  it("shows 'Updated just now' indicator when data was recently fetched", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      dataUpdatedAt: Date.now(),
    };
    await act(async () => {
      renderer = renderList();
    });
    const allText = getAllText(renderer);
    expect(allText).toContain("Updated just now");
  });

  it("hides last-updated indicator while loading", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
      isLoading: true,
      dataUpdatedAt: Date.now(),
    };
    await act(async () => {
      renderer = renderList();
    });
    const lastUpdated = renderer.root.findAll(
      (node) => node.props.testID === "last-updated",
    );
    expect(lastUpdated).toHaveLength(0);
  });

  it("hides last-updated indicator when data has never been fetched", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      dataUpdatedAt: 0,
    };
    await act(async () => {
      renderer = renderList();
    });
    const lastUpdated = renderer.root.findAll(
      (node) => node.props.testID === "last-updated",
    );
    expect(lastUpdated).toHaveLength(0);
  });
});
