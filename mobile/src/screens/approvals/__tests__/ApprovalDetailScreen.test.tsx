import React, { createElement } from "react";
import { Alert } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { ApprovalSummary } from "../../../hooks/useApprovals";
import { makeApproval, MOCK_AGENTS, mockGetAgentDisplayName } from "../testFixtures";

// --- Mocks ---

const mockApproveApproval = jest.fn();
const mockDenyApproval = jest.fn();

jest.mock("../../../hooks/useApproveApproval", () => ({
  useApproveApproval: () => ({
    approveApproval: mockApproveApproval,
    isPending: false,
    error: null,
    reset: jest.fn(),
  }),
}));

jest.mock("../../../hooks/useDenyApproval", () => ({
  useDenyApproval: () => ({
    denyApproval: mockDenyApproval,
    isPending: false,
    error: null,
    reset: jest.fn(),
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

jest.mock("expo-clipboard", () => ({
  setStringAsync: jest.fn(),
}));

import ApprovalDetailScreen from "../ApprovalDetailScreen";

// --- Helpers ---

const mockGoBack = jest.fn();

function renderDetail(approval: ApprovalSummary) {
  const route = {
    params: {
      approvalId: approval.approval_id,
      approval,
    },
    key: "test",
    name: "ApprovalDetail" as const,
  };
  const navigation = { goBack: mockGoBack } as any;

  return create(
    createElement(ApprovalDetailScreen, { route, navigation } as any),
  );
}

function hasTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findAll((node) => node.props.testID === testID).length > 0;
}

function findFirstByTestId(renderer: ReactTestRenderer, testID: string) {
  const matches = renderer.root.findAll((node) => node.props.testID === testID);
  return matches[0];
}

// --- Tests ---

describe("ApprovalDetailScreen", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
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

  it("shows email thread card for google.send_email_reply with thread in context", async () => {
    const approval = makeApproval({
      action: {
        type: "google.send_email_reply",
        version: "1",
        parameters: { thread_id: "t1", body: "Reply text" },
      },
      context: {
        description: "Reply to thread",
        risk_level: "low",
        details: {
          email_thread: {
            subject: "Re: Topic",
            messages: [
              {
                from: "x@y.com",
                to: ["z@y.com"],
                cc: [],
                date: "2026-04-20T12:00:00Z",
                body_html: "",
                body_text: "Thread body",
                snippet: "",
                message_id: "mid",
                truncated: false,
              },
            ],
          },
        },
      },
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Email thread");
    expect(json).toContain("Re: Topic");
    expect(json).toContain("Thread body");
    expect(json).not.toContain('"label":"email_thread"');
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

  // --- Approve/Deny flow tests ---

  it("shows approve and deny buttons for pending approvals", async () => {
    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasTestId(renderer, "approve-button")).toBe(true);
    expect(hasTestId(renderer, "deny-button")).toBe(true);
  });

  it("does not show approve/deny buttons for approved approvals", async () => {
    const approval = makeApproval({
      status: "approved",
      approved_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasTestId(renderer, "approve-button")).toBe(false);
    expect(hasTestId(renderer, "deny-button")).toBe(false);
  });

  it("does not show approve/deny buttons for denied approvals", async () => {
    const approval = makeApproval({
      status: "denied",
      denied_at: new Date().toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasTestId(renderer, "approve-button")).toBe(false);
  });

  it("does not show approve/deny buttons for expired approvals", async () => {
    const approval = makeApproval({
      expires_at: new Date(Date.now() - 60_000).toISOString(),
    });
    await act(async () => {
      renderer = renderDetail(approval);
    });
    expect(hasTestId(renderer, "approve-button")).toBe(false);
  });

  it("shows success banner after successful approval", async () => {
    mockApproveApproval.mockResolvedValueOnce({
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
    });

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    // Press approve button
    const approveButton = findFirstByTestId(renderer, "approve-button");
    await act(async () => {
      approveButton?.props.onPress();
    });

    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Request Approved");

    // Approve/deny buttons should be gone
    expect(hasTestId(renderer, "approve-button")).toBe(false);
  });

  it("shows alert on approve failure", async () => {
    mockApproveApproval.mockRejectedValueOnce(new Error("Approval has expired"));
    const alertSpy = jest.spyOn(Alert, "alert");

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const approveButton = findFirstByTestId(renderer, "approve-button");
    await act(async () => {
      approveButton?.props.onPress();
    });

    expect(alertSpy).toHaveBeenCalledWith(
      "Approval Failed",
      "Approval has expired",
    );
  });

  it("shows deny confirmation dialog and processes denial", async () => {
    mockDenyApproval.mockResolvedValueOnce(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const denyButton = findFirstByTestId(renderer, "deny-button");
    await act(async () => {
      denyButton?.props.onPress();
    });

    // Alert.alert should be called with deny confirmation
    expect(alertSpy).toHaveBeenCalledWith(
      "Deny Request",
      "Are you sure you want to deny this request?",
      expect.arrayContaining([
        expect.objectContaining({ text: "Cancel" }),
        expect.objectContaining({ text: "Deny", style: "destructive" }),
      ]),
    );

    // Simulate pressing "Deny" in the alert
    const denyAction = alertSpy.mock.calls[0]?.[2]?.find(
      (btn) => btn.text === "Deny",
    );
    await act(async () => {
      await denyAction?.onPress?.();
    });

    expect(mockDenyApproval).toHaveBeenCalledWith("appr_test123");

    // Should show denied banner
    expect(hasTestId(renderer, "denied-banner")).toBe(true);
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Request Denied");
  });

  it("shows alert on deny failure", async () => {
    mockDenyApproval.mockRejectedValueOnce(new Error("Already resolved"));
    const alertSpy = jest.spyOn(Alert, "alert");

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const denyButton = findFirstByTestId(renderer, "deny-button");
    await act(async () => {
      denyButton?.props.onPress();
    });

    // Simulate pressing "Deny" in the alert
    const denyAction = alertSpy.mock.calls[0]?.[2]?.find(
      (btn) => btn.text === "Deny",
    );
    await act(async () => {
      await denyAction?.onPress?.();
    });

    // Second alert call should be the error alert with server message
    expect(alertSpy).toHaveBeenCalledWith(
      "Denial Failed",
      "Already resolved",
    );
  });

  it("shows Done button after approval", async () => {
    mockApproveApproval.mockResolvedValueOnce({
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
    });

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const approveButton = findFirstByTestId(renderer, "approve-button");
    await act(async () => {
      approveButton?.props.onPress();
    });

    expect(hasTestId(renderer, "done-button")).toBe(true);
  });

  it("shows Done button after denial", async () => {
    mockDenyApproval.mockResolvedValueOnce(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const denyButton = findFirstByTestId(renderer, "deny-button");
    await act(async () => {
      denyButton?.props.onPress();
    });

    const denyAction = alertSpy.mock.calls[0]?.[2]?.find(
      (btn) => btn.text === "Deny",
    );
    await act(async () => {
      await denyAction?.onPress?.();
    });

    expect(hasTestId(renderer, "done-button")).toBe(true);
  });

  it("Done button navigates back to list", async () => {
    mockApproveApproval.mockResolvedValueOnce({
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
    });

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    const approveButton = findFirstByTestId(renderer, "approve-button");
    await act(async () => {
      approveButton?.props.onPress();
    });

    const doneButton = findFirstByTestId(renderer, "done-button");
    await act(async () => {
      doneButton?.props.onPress();
    });

    expect(mockGoBack).toHaveBeenCalled();
  });

  it("hides action buttons after successful approval", async () => {
    mockApproveApproval.mockResolvedValueOnce({
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
    });

    const approval = makeApproval();
    await act(async () => {
      renderer = renderDetail(approval);
    });

    // Buttons should exist before approval
    expect(hasTestId(renderer, "approve-button")).toBe(true);

    const approveButton = findFirstByTestId(renderer, "approve-button");
    await act(async () => {
      approveButton?.props.onPress();
    });

    // Buttons should be gone after approval
    expect(hasTestId(renderer, "approve-button")).toBe(false);
    expect(hasTestId(renderer, "deny-button")).toBe(false);
  });
});
