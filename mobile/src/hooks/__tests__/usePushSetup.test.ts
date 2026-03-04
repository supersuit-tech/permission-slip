import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Module-level mocks ---

const mockRegisterForPushNotifications = jest.fn();
const mockRegisterToken = jest.fn();
const mockInvalidateQueries = jest.fn();
const mockClientPost = jest.fn();

let mockAuthStatus = "unauthenticated";
let mockExpoPushToken: string | null = null;
let mockSession: { access_token: string } | null = null;
let capturedOnNotificationReceived: ((n: unknown) => void) | undefined;
let capturedOnNotificationTap: ((r: unknown) => void) | undefined;
const mockHandleNotificationTap = jest.fn();

jest.mock("../useNotificationNavigation", () => ({
  useNotificationNavigation: () => ({
    handleNotificationTap: mockHandleNotificationTap,
  }),
}));

jest.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

jest.mock("../useNotifications", () => ({
  useNotifications: (options?: {
    onNotificationReceived?: (n: unknown) => void;
    onNotificationTap?: (r: unknown) => void;
  }) => {
    capturedOnNotificationReceived = options?.onNotificationReceived;
    capturedOnNotificationTap = options?.onNotificationTap;
    return {
      expoPushToken: mockExpoPushToken,
      permissionGranted: mockExpoPushToken !== null,
      error: null,
      registerForPushNotifications: mockRegisterForPushNotifications,
      lastNotificationResponse: { current: null },
    };
  },
}));

jest.mock("../useRegisterPushToken", () => ({
  useRegisterPushToken: () => ({
    registerToken: mockRegisterToken,
    isRegistering: false,
    registerError: null,
  }),
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    authStatus: mockAuthStatus,
    session: mockSession,
  }),
}));

jest.mock("../../api/client", () => ({
  __esModule: true,
  default: { POST: (...args: unknown[]) => mockClientPost(...args) },
}));

import { usePushSetup } from "../usePushSetup";

// --- Hook capture helper ---

function createHookCapture() {
  function Consumer() {
    usePushSetup();
    return null;
  }
  return { Consumer };
}

describe("usePushSetup", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthStatus = "unauthenticated";
    mockExpoPushToken = null;
    mockSession = null;
    capturedOnNotificationReceived = undefined;
    capturedOnNotificationTap = undefined;
    mockRegisterForPushNotifications.mockResolvedValue(null);
    mockRegisterToken.mockResolvedValue({});
    mockClientPost.mockResolvedValue({ data: {}, error: undefined });
  });

  afterEach(async () => {
    if (renderer) {
      await act(async () => {
        renderer!.unmount();
      });
      renderer = null;
    }
  });

  it("calls registerForPushNotifications when authenticated", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(mockRegisterForPushNotifications).toHaveBeenCalled();
  });

  it("does not call registerForPushNotifications when unauthenticated", async () => {
    mockAuthStatus = "unauthenticated";
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(mockRegisterForPushNotifications).not.toHaveBeenCalled();
  });

  it("registers token with backend when token is available and authenticated", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Wait for the async registerToken call
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(mockRegisterToken).toHaveBeenCalledWith("ExponentPushToken[abc]");
  });

  it("does not register token when unauthenticated even if token exists", async () => {
    mockAuthStatus = "unauthenticated";
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(mockRegisterToken).not.toHaveBeenCalled();
  });

  it("does not crash when registerToken fails", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockRegisterToken.mockRejectedValue(new Error("Network error"));
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Should not throw
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  });

  it("unregisters token on sign-out using captured access token", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "captured-tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Let register complete
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(mockRegisterToken).toHaveBeenCalledWith("ExponentPushToken[abc]");

    // Simulate sign-out — session is now null
    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    // Should use the captured token to call the API directly
    expect(mockClientPost).toHaveBeenCalledWith(
      "/v1/push-subscriptions/unregister",
      {
        headers: { Authorization: "Bearer captured-tok" },
        body: { expo_token: "ExponentPushToken[abc]" },
      },
    );
  });

  it("invalidates approvals cache when a foreground notification is received", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(capturedOnNotificationReceived).toBeDefined();
    capturedOnNotificationReceived?.({ request: { content: { title: "Test" } } });

    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: ["approvals"],
    });
  });

  it("passes handleNotificationTap as onNotificationTap to useNotifications", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(capturedOnNotificationTap).toBe(mockHandleNotificationTap);
  });

  it("forwards notification tap to handleNotificationTap", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const fakeResponse = {
      actionIdentifier: "default",
      notification: { request: { content: { data: { approval_id: "appr_123" } } } },
    };
    capturedOnNotificationTap?.(fakeResponse);

    expect(mockHandleNotificationTap).toHaveBeenCalledWith(fakeResponse);
  });

  it("does not crash when unregister API call fails", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockClientPost.mockRejectedValue(new Error("Network error"));
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    // Should not throw
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  });
});
