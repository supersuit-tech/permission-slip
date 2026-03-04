import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { Platform } from "react-native";
import * as Notifications from "expo-notifications";
import * as Device from "expo-device";

// Mock expo-notifications
jest.mock("expo-notifications", () => ({
  setNotificationHandler: jest.fn(),
  getPermissionsAsync: jest.fn(),
  requestPermissionsAsync: jest.fn(),
  getExpoPushTokenAsync: jest.fn(),
  setNotificationChannelAsync: jest.fn(),
  addNotificationReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
  addNotificationResponseReceivedListener: jest.fn(() => ({ remove: jest.fn() })),
  getLastNotificationResponseAsync: jest.fn().mockResolvedValue(null),
  AndroidImportance: { HIGH: 4 },
}));

// Mock expo-device
jest.mock("expo-device", () => ({
  isDevice: true,
}));

import {
  useNotifications,
  type NotificationState,
  type UseNotificationsOptions,
} from "../useNotifications";

const mockGetPermissions = Notifications.getPermissionsAsync as jest.Mock;
const mockRequestPermissions = Notifications.requestPermissionsAsync as jest.Mock;
const mockGetToken = Notifications.getExpoPushTokenAsync as jest.Mock;
const mockSetChannel = Notifications.setNotificationChannelAsync as jest.Mock;

// --- Hook capture helper ---

interface Capture extends NotificationState {
  registerForPushNotifications: () => Promise<string | null>;
}

function createHookCapture(options: UseNotificationsOptions = {}) {
  const capture = {} as Capture;
  function Consumer() {
    const result = useNotifications(options);
    capture.expoPushToken = result.expoPushToken;
    capture.permissionGranted = result.permissionGranted;
    capture.error = result.error;
    capture.registerForPushNotifications = result.registerForPushNotifications;
    return null;
  }
  return { capture, Consumer };
}

describe("useNotifications", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    jest.clearAllMocks();
    mockGetPermissions.mockResolvedValue({ status: "granted" });
    mockRequestPermissions.mockResolvedValue({ status: "granted" });
    mockGetToken.mockResolvedValue({ data: "ExponentPushToken[test123]" });
  });

  afterEach(async () => {
    if (renderer) {
      await act(async () => {
        renderer!.unmount();
      });
      renderer = null;
    }
  });

  it("starts with null token and no error", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });
    expect(capture.expoPushToken).toBeNull();
    expect(capture.permissionGranted).toBe(false);
    expect(capture.error).toBeNull();
  });

  it("requests permissions and retrieves token", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    let token: string | null = null;
    await act(async () => {
      token = await capture.registerForPushNotifications();
    });

    expect(token).toBe("ExponentPushToken[test123]");
    expect(capture.expoPushToken).toBe("ExponentPushToken[test123]");
    expect(capture.permissionGranted).toBe(true);
    expect(capture.error).toBeNull();
  });

  it("skips requesting permissions if already granted", async () => {
    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockRequestPermissions).not.toHaveBeenCalled();
  });

  it("requests permissions if not yet granted", async () => {
    mockGetPermissions.mockResolvedValue({ status: "undetermined" });
    mockRequestPermissions.mockResolvedValue({ status: "granted" });

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockRequestPermissions).toHaveBeenCalled();
    expect(capture.expoPushToken).toBe("ExponentPushToken[test123]");
  });

  it("sets error when permission is denied", async () => {
    mockGetPermissions.mockResolvedValue({ status: "denied" });
    mockRequestPermissions.mockResolvedValue({ status: "denied" });

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    let token: string | null = "placeholder";
    await act(async () => {
      token = await capture.registerForPushNotifications();
    });

    expect(token).toBeNull();
    expect(capture.expoPushToken).toBeNull();
    expect(capture.permissionGranted).toBe(false);
    expect(capture.error).toBe("Push notification permission was denied");
  });

  it("sets error when not on a physical device", async () => {
    Object.defineProperty(Device, "isDevice", { value: false, writable: true });

    try {
      const { capture, Consumer } = createHookCapture();
      await act(async () => {
        renderer = create(createElement(Consumer));
      });

      let token: string | null = "placeholder";
      await act(async () => {
        token = await capture.registerForPushNotifications();
      });

      expect(token).toBeNull();
      expect(capture.error).toBe("Push notifications require a physical device");
    } finally {
      Object.defineProperty(Device, "isDevice", { value: true, writable: true });
    }
  });

  it("sets error when token retrieval fails", async () => {
    mockGetToken.mockRejectedValue(new Error("Network error"));

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(capture.expoPushToken).toBeNull();
    expect(capture.permissionGranted).toBe(true);
    expect(capture.error).toBe("Network error");
  });

  it("creates Android notification channel on Android", async () => {
    const originalOS = Platform.OS;
    Object.defineProperty(Platform, "OS", { value: "android", writable: true });

    try {
      const { capture, Consumer } = createHookCapture();
      await act(async () => {
        renderer = create(createElement(Consumer));
      });

      await act(async () => {
        await capture.registerForPushNotifications();
      });

      expect(mockSetChannel).toHaveBeenCalledWith(
        "approval-requests",
        expect.objectContaining({
          name: "Approval Requests",
          importance: Notifications.AndroidImportance.HIGH,
        }),
      );
    } finally {
      Object.defineProperty(Platform, "OS", { value: originalOS, writable: true });
    }
  });

  it("does not create Android channel on iOS", async () => {
    const originalOS = Platform.OS;
    Object.defineProperty(Platform, "OS", { value: "ios", writable: true });

    try {
      const { capture, Consumer } = createHookCapture();
      await act(async () => {
        renderer = create(createElement(Consumer));
      });

      await act(async () => {
        await capture.registerForPushNotifications();
      });

      expect(mockSetChannel).not.toHaveBeenCalled();
    } finally {
      Object.defineProperty(Platform, "OS", { value: originalOS, writable: true });
    }
  });

  it("sets up and cleans up notification listeners", async () => {
    const sub1 = { remove: jest.fn() };
    const sub2 = { remove: jest.fn() };
    (Notifications.addNotificationReceivedListener as jest.Mock).mockReturnValue(sub1);
    (Notifications.addNotificationResponseReceivedListener as jest.Mock).mockReturnValue(sub2);

    const { Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    expect(Notifications.addNotificationReceivedListener).toHaveBeenCalled();
    expect(Notifications.addNotificationResponseReceivedListener).toHaveBeenCalled();

    await act(async () => {
      renderer!.unmount();
    });
    renderer = null;

    expect(sub1.remove).toHaveBeenCalled();
    expect(sub2.remove).toHaveBeenCalled();
  });

  it("calls onNotificationReceived when a foreground notification arrives", async () => {
    const onNotificationReceived = jest.fn();
    let receivedCallback: ((n: unknown) => void) | null = null;

    (Notifications.addNotificationReceivedListener as jest.Mock).mockImplementation(
      (cb: (n: unknown) => void) => {
        receivedCallback = cb;
        return { remove: jest.fn() };
      },
    );

    const { Consumer } = createHookCapture({ onNotificationReceived });
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const fakeNotification = { request: { content: { title: "New approval" } } };
    await act(async () => {
      receivedCallback?.(fakeNotification);
    });

    expect(onNotificationReceived).toHaveBeenCalledWith(fakeNotification);
  });

  it("passes projectId to getExpoPushTokenAsync when provided", async () => {
    const { capture, Consumer } = createHookCapture({ projectId: "my-project-id" });
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockGetToken).toHaveBeenCalledWith({ projectId: "my-project-id" });
  });

  it("calls onNotificationTap when a notification is tapped", async () => {
    const onNotificationTap = jest.fn();
    let responseCallback: ((r: unknown) => void) | null = null;

    (Notifications.addNotificationResponseReceivedListener as jest.Mock).mockImplementation(
      (cb: (r: unknown) => void) => {
        responseCallback = cb;
        return { remove: jest.fn() };
      },
    );

    const { Consumer } = createHookCapture({ onNotificationTap });
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    const fakeResponse = {
      actionIdentifier: "default",
      notification: { request: { content: { data: { approval_id: "appr_123" } } } },
    };
    await act(async () => {
      responseCallback?.(fakeResponse);
    });

    expect(onNotificationTap).toHaveBeenCalledWith(fakeResponse);
  });

  it("calls onNotificationTap on cold start when getLastNotificationResponseAsync returns a response", async () => {
    const onNotificationTap = jest.fn();
    const coldStartResponse = {
      actionIdentifier: "default",
      notification: { request: { content: { data: { approval_id: "appr_cold" } } } },
    };

    (Notifications.getLastNotificationResponseAsync as jest.Mock).mockResolvedValue(
      coldStartResponse,
    );

    const { Consumer } = createHookCapture({ onNotificationTap });
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    // Wait for the async getLastNotificationResponseAsync to resolve
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(onNotificationTap).toHaveBeenCalledWith(coldStartResponse);
  });

  it("does not call onNotificationTap on cold start when no launch notification exists", async () => {
    const onNotificationTap = jest.fn();
    (Notifications.getLastNotificationResponseAsync as jest.Mock).mockResolvedValue(null);

    const { Consumer } = createHookCapture({ onNotificationTap });
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(onNotificationTap).not.toHaveBeenCalled();
  });
});
