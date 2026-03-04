import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { NotificationResponse } from "expo-notifications";

// --- Module-level mocks ---

const mockClientGet = jest.fn();
let mockAuthStatus = "authenticated";
let mockSession: { access_token: string } | null = null;

const mockNavigate = jest.fn();
const mockIsReady = jest.fn(() => true);
const mockSetBadgeCount = jest.fn().mockResolvedValue(true);

jest.mock("expo-notifications", () => ({
  setBadgeCountAsync: (...args: unknown[]) => mockSetBadgeCount(...args),
}));

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockClientGet(...args) },
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    authStatus: mockAuthStatus,
    session: mockSession,
  }),
}));

jest.mock("../../navigation/RootNavigator", () => ({
  navigationRef: {
    isReady: () => mockIsReady(),
    navigate: (...args: unknown[]) => mockNavigate(...args),
  },
}));

import { useNotificationNavigation } from "../useNotificationNavigation";

// --- Helpers ---

function makeNotificationResponse(
  approvalId: string | undefined,
  identifier = "notif-1",
  actionIdentifier = "expo.modules.notifications.actions.DEFAULT",
): NotificationResponse {
  // Cast to NotificationResponse — tests only need the fields the hook reads
  return {
    actionIdentifier,
    notification: {
      date: Date.now(),
      request: {
        identifier,
        content: {
          data: approvalId ? { approval_id: approvalId } : {},
        },
      },
    },
  } as unknown as NotificationResponse;
}

const fakeApproval = {
  approval_id: "appr_abc123",
  agent_id: 1,
  action: { type: "test.action", parameters: {} },
  context: { risk_level: "low", details: {} },
  status: "pending",
  expires_at: "2026-12-31T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
};

function createHookCapture() {
  const capture = {} as ReturnType<typeof useNotificationNavigation>;
  function Consumer() {
    const result = useNotificationNavigation();
    capture.handleNotificationTap = result.handleNotificationTap;
    return null;
  }
  return { capture, Consumer };
}

describe("useNotificationNavigation", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "test-token" };
    mockIsReady.mockReturnValue(true);
    mockSetBadgeCount.mockResolvedValue(true);
    mockClientGet.mockResolvedValue({
      data: { data: [fakeApproval], has_more: false },
      error: undefined,
    });
  });

  afterEach(async () => {
    if (renderer) {
      await act(async () => {
        renderer!.unmount();
      });
      renderer = null;
    }
  });

  it("navigates to ApprovalDetail when a notification with approval_id is tapped", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockClientGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: { Authorization: "Bearer test-token" },
      params: { query: { status: "all", limit: 100 } },
    });
    expect(mockNavigate).toHaveBeenCalledWith("ApprovalDetail", {
      approvalId: "appr_abc123",
      approval: fakeApproval,
    });
  });

  it("clears the app badge count after navigating", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockSetBadgeCount).toHaveBeenCalledWith(0);
  });

  it("does not navigate when approval_id has invalid format", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Invalid format — doesn't match ^appr_[a-zA-Z0-9]{6,64}$
    const response = makeNotificationResponse("invalid-id");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockClientGet).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not navigate when approval_id is missing from notification data", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse(undefined);
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockClientGet).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not navigate when user is not authenticated", async () => {
    mockAuthStatus = "loading";
    mockSession = null;
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockClientGet).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not navigate when approval is not found in API response", async () => {
    mockClientGet.mockResolvedValue({
      data: { data: [], has_more: false },
      error: undefined,
    });

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_notfound");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockClientGet).toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not navigate when navigation is not ready", async () => {
    mockIsReady.mockReturnValue(false);

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not navigate when API returns an error", async () => {
    mockClientGet.mockResolvedValue({
      data: undefined,
      error: { message: "Unauthorized" },
    });

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("does not crash when API call throws", async () => {
    mockClientGet.mockRejectedValue(new Error("Network error"));

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("de-duplicates navigation for the same notification tap", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const response = makeNotificationResponse("appr_abc123", "same-notif");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    // Should only navigate once
    expect(mockNavigate).toHaveBeenCalledTimes(1);
  });

  it("queues notification on cold start and processes it once auth is ready", async () => {
    // Start with auth not ready (cold start scenario)
    mockAuthStatus = "loading";
    mockSession = null;

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Notification tap arrives before auth is ready
    const response = makeNotificationResponse("appr_abc123");
    await act(async () => {
      await capture.handleNotificationTap(response);
    });

    // Should not have navigated yet
    expect(mockClientGet).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();

    // Auth becomes ready
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "test-token" };
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    // Wait for the async navigateToApproval to complete
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    // Now it should have fetched and navigated
    expect(mockClientGet).toHaveBeenCalledWith("/v1/approvals", {
      headers: { Authorization: "Bearer test-token" },
      params: { query: { status: "all", limit: 100 } },
    });
    expect(mockNavigate).toHaveBeenCalledWith("ApprovalDetail", {
      approvalId: "appr_abc123",
      approval: fakeApproval,
    });
  });
});
