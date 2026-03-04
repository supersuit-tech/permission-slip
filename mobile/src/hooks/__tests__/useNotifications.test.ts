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
  AndroidImportance: { HIGH: 4 },
}));

// Mock expo-device
jest.mock("expo-device", () => ({
  isDevice: true,
}));

import { useNotifications, type NotificationState } from "../useNotifications";

const mockGetPermissions = Notifications.getPermissionsAsync as jest.Mock;
const mockRequestPermissions = Notifications.requestPermissionsAsync as jest.Mock;
const mockGetToken = Notifications.getExpoPushTokenAsync as jest.Mock;
const mockSetChannel = Notifications.setNotificationChannelAsync as jest.Mock;

// --- Hook capture helper ---

interface Capture extends NotificationState {
  registerForPushNotifications: () => Promise<string | null>;
}

function createHookCapture(projectId?: string) {
  const capture = {} as Capture;
  function Consumer() {
    const result = useNotifications(projectId);
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

    Object.defineProperty(Device, "isDevice", { value: true, writable: true });
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

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockSetChannel).toHaveBeenCalledWith(
      "default",
      expect.objectContaining({
        name: "Default",
        importance: Notifications.AndroidImportance.HIGH,
      }),
    );

    Object.defineProperty(Platform, "OS", { value: originalOS, writable: true });
  });

  it("does not create Android channel on iOS", async () => {
    const originalOS = Platform.OS;
    Object.defineProperty(Platform, "OS", { value: "ios", writable: true });

    const { capture, Consumer } = createHookCapture();
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockSetChannel).not.toHaveBeenCalled();

    Object.defineProperty(Platform, "OS", { value: originalOS, writable: true });
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

  it("passes projectId to getExpoPushTokenAsync when provided", async () => {
    const { capture, Consumer } = createHookCapture("my-project-id");
    await act(async () => {
      renderer = create(createElement(Consumer));
    });

    await act(async () => {
      await capture.registerForPushNotifications();
    });

    expect(mockGetToken).toHaveBeenCalledWith({ projectId: "my-project-id" });
  });
});
