import React, { createElement } from "react";
import { Text } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { ApprovalSummary } from "../../../hooks/useApprovals";
import { MOCK_AGENTS, mockGetAgentDisplayName } from "../testFixtures";

// --- Mocks ---

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

const mockDenyApproval = jest.fn().mockResolvedValue(undefined);
const mockDenyAllApprovals = jest.fn().mockResolvedValue({ denied_count: 1, skipped_count: 0 });
const mockDenyRuleRequest = jest.fn().mockResolvedValue({});

jest.mock("../../../hooks/useDenyApproval", () => ({
  useDenyApproval: () => ({
    denyApproval: mockDenyApproval,
    isPending: false,
    error: null,
    reset: jest.fn(),
  }),
}));

jest.mock("../../../hooks/useDenyAllApprovals", () => ({
  useDenyAllApprovals: () => ({
    denyAllApprovals: mockDenyAllApprovals,
    isPending: false,
    error: null,
    reset: jest.fn(),
  }),
}));

jest.mock("../../../hooks/useDenyStandingApprovalRequest", () => ({
  useDenyStandingApprovalRequest: () => ({
    mutateAsync: mockDenyRuleRequest,
    isPending: false,
  }),
}));

let mockUseStandingApprovalRequestsReturn = {
  requests: [] as Array<Record<string, unknown>>,
  isLoading: false,
  isRefetching: false,
  error: null as string | null,
  refetch: jest.fn(),
  dataUpdatedAt: Date.now(),
};

jest.mock("../../../hooks/useStandingApprovalRequests", () => ({
  useStandingApprovalRequests: () => mockUseStandingApprovalRequestsReturn,
}));

const mockUseStandingApprovalInstanceScope = jest.fn(() => ({
  scopeLabel: "Applies to all accounts",
}));

jest.mock("../../../hooks/useStandingApprovalInstanceScope", () => ({
  useStandingApprovalInstanceScope: () => mockUseStandingApprovalInstanceScope(),
}));

jest.mock("../../../hooks/useStandingApprovalConnectorLabel", () => ({
  useStandingApprovalConnectorLabel: () => ({
    connectorLabel: "Proton Mail",
  }),
}));

jest.mock("../../../hooks/useAgents", () => ({
  useAgents: () => ({
    agents: MOCK_AGENTS.map((a) => ({ ...a, status: "registered" })),
    isLoading: false,
    error: null,
  }),
  getAgentDisplayName: mockGetAgentDisplayName,
}));

jest.mock("../../../hooks/useActionSchema", () => ({
  useActionSchema: () => ({
    displayTemplate: null,
    actionName: null,
    isLoading: false,
  }),
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

/** Finds a pressable node by testID. */
function findPressableByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findAll(
    (node) => node.props.testID === testID && typeof node.props.onPress === "function",
  )[0];
}

// --- Tests ---

describe("ApprovalListScreen", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.useFakeTimers();
    mockDenyApproval.mockClear();
    mockDenyAllApprovals.mockClear();
    mockDenyRuleRequest.mockClear();
    mockUseStandingApprovalInstanceScope.mockReturnValue({
      scopeLabel: "Applies to all accounts",
    });
    mockUseStandingApprovalRequestsReturn = {
      requests: [],
      isLoading: false,
      isRefetching: false,
      error: null,
      refetch: jest.fn(),
      dataUpdatedAt: Date.now(),
    };
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

  it("shows account scope line on rule proposal rows", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
    };
    mockUseStandingApprovalRequestsReturn = {
      ...mockUseStandingApprovalRequestsReturn,
      requests: [
        {
          request_id: "sar_ambiguous",
          agent_id: 42,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    };
    mockUseStandingApprovalInstanceScope.mockReturnValue({
      scopeLabel: "Applies to all accounts",
    });

    await act(async () => {
      renderer = renderList();
    });

    const allText = getAllText(renderer);
    expect(allText).toContain("Applies to all accounts");
  });

  it("shows pinned account scope line when hook returns a specific account", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
    };
    mockUseStandingApprovalRequestsReturn = {
      ...mockUseStandingApprovalRequestsReturn,
      requests: [
        {
          request_id: "sar_clear",
          agent_id: 42,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          connector_instance_display: "Personal",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    };

    mockUseStandingApprovalInstanceScope.mockReturnValue({
      scopeLabel: "Applies to Personal",
    });

    await act(async () => {
      renderer = renderList();
    });

    const allText = getAllText(renderer);
    expect(allText).toContain("Applies to Personal");
  });

  it("shows decline all button for standalone pending approvals", async () => {
    await act(async () => {
      renderer = renderList();
    });

    const declineAllButton = findPressableByTestId(renderer, "decline-all-button");
    expect(declineAllButton).toBeTruthy();
    expect(declineAllButton!.props.accessibilityLabel).toBe(
      "Decline all 1 pending requests",
    );
  });

  it("declines a single approval when inline X is pressed", async () => {
    await act(async () => {
      renderer = renderList();
    });

    const declineButton = findPressableByTestId(renderer, "decline-approval-appr_001");

    await act(async () => {
      declineButton!.props.onPress();
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(mockDenyApproval).toHaveBeenCalledWith("appr_001");
  });

  it("declines a rule proposal when inline X is pressed", async () => {
    mockUseApprovalsReturn = {
      ...mockUseApprovalsReturn,
      approvals: [],
    };
    mockUseStandingApprovalRequestsReturn = {
      ...mockUseStandingApprovalRequestsReturn,
      requests: [
        {
          request_id: "sar_001",
          agent_id: 42,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    };

    await act(async () => {
      renderer = renderList();
    });

    const declineButton = findPressableByTestId(renderer, "decline-rule-sar_001");

    await act(async () => {
      declineButton!.props.onPress();
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(mockDenyRuleRequest).toHaveBeenCalledWith("sar_001");
  });
});
