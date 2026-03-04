import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Module-level mocks ---

const mockRegisterForPushNotifications = jest.fn();
const mockRegisterToken = jest.fn();
const mockUnregisterToken = jest.fn();
const mockInvalidateQueries = jest.fn();

let mockAuthStatus = "unauthenticated";
let mockExpoPushToken: string | null = null;
let capturedOnNotificationReceived: ((n: unknown) => void) | undefined;

jest.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

jest.mock("../useNotifications", () => ({
  useNotifications: (options?: { onNotificationReceived?: (n: unknown) => void }) => {
    capturedOnNotificationReceived = options?.onNotificationReceived;
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
    unregisterToken: mockUnregisterToken,
    isRegistering: false,
    isUnregistering: false,
    registerError: null,
    unregisterError: null,
  }),
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    authStatus: mockAuthStatus,
    session: mockAuthStatus === "authenticated" ? { access_token: "tok" } : null,
  }),
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
    capturedOnNotificationReceived = undefined;
    mockRegisterForPushNotifications.mockResolvedValue(null);
    mockRegisterToken.mockResolvedValue({});
    mockUnregisterToken.mockResolvedValue({});
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

  it("unregisters token on sign-out", async () => {
    mockAuthStatus = "authenticated";
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

    // Simulate sign-out by re-creating with new auth status
    mockAuthStatus = "unauthenticated";
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(mockUnregisterToken).toHaveBeenCalledWith("ExponentPushToken[abc]");
  });

  it("invalidates approvals cache when a foreground notification is received", async () => {
    mockAuthStatus = "authenticated";
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

  it("does not crash when unregisterToken fails", async () => {
    mockAuthStatus = "authenticated";
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockUnregisterToken.mockRejectedValue(new Error("Network error"));
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    mockAuthStatus = "unauthenticated";
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    // Should not throw
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  });
});
