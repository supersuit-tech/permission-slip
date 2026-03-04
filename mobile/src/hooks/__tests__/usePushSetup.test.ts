import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Module-level mocks ---

const mockRegisterForPushNotifications = jest.fn();
const mockRegisterToken = jest.fn();
const mockInvalidateQueries = jest.fn();
const mockUnregisterDirect = jest.fn();

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
    };
  },
}));

jest.mock("../useRegisterPushToken", () => ({
  useRegisterPushToken: () => ({
    registerToken: mockRegisterToken,
    isRegistering: false,
    registerError: null,
  }),
  unregisterPushTokenDirect: (...args: unknown[]) => mockUnregisterDirect(...args),
}));

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    authStatus: mockAuthStatus,
    session: mockSession,
  }),
}));

import { usePushSetup } from "../usePushSetup";

// --- Hook capture helper ---

interface HookCapture {
  isTokenRegistered: boolean;
}

function createHookCapture() {
  const capture: HookCapture = { isTokenRegistered: false };

  function Consumer() {
    const result = usePushSetup();
    capture.isTokenRegistered = result.isTokenRegistered;
    return null;
  }
  return { Consumer, capture };
}

describe("usePushSetup", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    mockAuthStatus = "unauthenticated";
    mockExpoPushToken = null;
    mockSession = null;
    capturedOnNotificationReceived = undefined;
    capturedOnNotificationTap = undefined;
    mockRegisterForPushNotifications.mockResolvedValue(null);
    mockRegisterToken.mockResolvedValue({});
    mockUnregisterDirect.mockResolvedValue(undefined);
  });

  afterEach(async () => {
    jest.useRealTimers();
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

    // Flush microtasks for the async registerToken call
    await act(async () => {});

    expect(mockRegisterToken).toHaveBeenCalledWith("ExponentPushToken[abc]");
  });

  it("sets isTokenRegistered to true after successful registration", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer, capture } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {});

    expect(capture.isTokenRegistered).toBe(true);
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

  it("retries registration with exponential backoff on failure", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";

    // Fail first two attempts, succeed on third
    mockRegisterToken
      .mockRejectedValueOnce(new Error("Network error"))
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValueOnce({});

    const { Consumer, capture } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // First attempt fails immediately
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);
    expect(capture.isTokenRegistered).toBe(false);

    // Advance past first retry delay (1000ms)
    await act(async () => {
      jest.advanceTimersByTime(1000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(2);
    expect(capture.isTokenRegistered).toBe(false);

    // Advance past second retry delay (2000ms)
    await act(async () => {
      jest.advanceTimersByTime(2000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(3);
    expect(capture.isTokenRegistered).toBe(true);
  });

  it("stops retrying after MAX_RETRIES failures", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockRegisterToken.mockRejectedValue(new Error("Network error"));

    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Attempt 1 (immediate)
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);

    // Attempt 2 (after 1000ms)
    await act(async () => {
      jest.advanceTimersByTime(1000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(2);

    // Attempt 3 (after 2000ms)
    await act(async () => {
      jest.advanceTimersByTime(2000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(3);

    // Attempt 4 (after 4000ms)
    await act(async () => {
      jest.advanceTimersByTime(4000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(4);

    // No more retries — advancing further should not trigger another call
    await act(async () => {
      jest.advanceTimersByTime(10000);
    });
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(4);
  });

  it("cancels retry timer on unmount", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockRegisterToken.mockRejectedValue(new Error("Network error"));

    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // First attempt fails, retry scheduled
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);

    // Unmount before retry fires
    await act(async () => {
      renderer!.unmount();
    });
    renderer = null;

    // Advance past retry delay — should NOT trigger another call
    await act(async () => {
      jest.advanceTimersByTime(5000);
    });
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);
  });

  it("unregisters token on sign-out using unregisterPushTokenDirect", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "captured-tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Let register complete
    await act(async () => {});

    expect(mockRegisterToken).toHaveBeenCalledWith("ExponentPushToken[abc]");

    // Simulate sign-out — session is now null
    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    await act(async () => {});

    // Should call the dedicated unregister function
    expect(mockUnregisterDirect).toHaveBeenCalledWith(
      "ExponentPushToken[abc]",
      "captured-tok",
    );
  });

  it("sets isTokenRegistered to false on sign-out", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    const { Consumer, capture } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {});
    expect(capture.isTokenRegistered).toBe(true);

    // Sign out
    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    expect(capture.isTokenRegistered).toBe(false);
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

  it("passes onNotificationTap callback to useNotifications", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(capturedOnNotificationTap).toBeDefined();
    expect(typeof capturedOnNotificationTap).toBe("function");
  });

  it("forwards notification tap to handleNotificationTap and invalidates cache", async () => {
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
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: ["approvals"],
    });
  });

  it("cancels in-flight retry when user signs out", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    mockRegisterToken.mockRejectedValue(new Error("Network error"));

    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // First attempt fails, retry scheduled for 1000ms
    await act(async () => {});
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);

    // Sign out before retry fires — should cancel the pending timer
    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    // Advance past when retry would have fired
    await act(async () => {
      jest.advanceTimersByTime(5000);
    });
    await act(async () => {});

    // Retry should NOT have fired after sign-out
    expect(mockRegisterToken).toHaveBeenCalledTimes(1);
  });

  it("calls unregisterPushTokenDirect even when it would fail", async () => {
    mockAuthStatus = "authenticated";
    mockSession = { access_token: "tok" };
    mockExpoPushToken = "ExponentPushToken[abc]";
    // unregisterPushTokenDirect is best-effort and never throws, but we
    // verify it's still called regardless of what happens internally
    mockUnregisterDirect.mockResolvedValue(undefined);
    const { Consumer } = createHookCapture();

    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {});

    mockAuthStatus = "unauthenticated";
    mockSession = null;
    await act(async () => {
      renderer!.update(createElement(Consumer));
    });

    await act(async () => {});

    expect(mockUnregisterDirect).toHaveBeenCalledWith(
      "ExponentPushToken[abc]",
      "tok",
    );
  });
});
