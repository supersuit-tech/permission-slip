/**
 * Hook that manages Expo push notification setup: requesting permissions,
 * retrieving the Expo push token, and configuring notification handlers.
 *
 * This hook does NOT register/unregister the token with the backend — that
 * responsibility belongs to the caller (see useRegisterPushToken).
 */
import { useCallback, useEffect, useRef, useState } from "react";
import { Platform } from "react-native";
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";

/** Configure how notifications are presented when the app is in the foreground. */
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

export interface NotificationState {
  /** The Expo push token string, e.g. "ExponentPushToken[abc123]". null until retrieved. */
  expoPushToken: string | null;
  /** Whether push notification permission has been granted. */
  permissionGranted: boolean;
  /** Error message if permission was denied or token retrieval failed. */
  error: string | null;
}

/**
 * Requests push notification permissions and retrieves the Expo push token.
 *
 * Returns the token, permission status, error state, and a ref for the most
 * recent notification response (for deep-link handling by callers).
 *
 * @param projectId - The Expo project ID (from app.json extra or Constants).
 *   Optional — Expo infers it in managed workflow builds.
 */
export function useNotifications(projectId?: string) {
  const [state, setState] = useState<NotificationState>({
    expoPushToken: null,
    permissionGranted: false,
    error: null,
  });

  const notificationListener = useRef<Notifications.EventSubscription | null>(null);
  const responseListener = useRef<Notifications.EventSubscription | null>(null);

  /** The last notification response (tap). Callers can read this for navigation. */
  const lastNotificationResponse = useRef<Notifications.NotificationResponse | null>(null);

  const registerForPushNotifications = useCallback(async (): Promise<string | null> => {
    // Push notifications only work on physical devices
    if (!Device.isDevice) {
      setState((s) => ({
        ...s,
        error: "Push notifications require a physical device",
      }));
      return null;
    }

    // Check existing permission status
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    let finalStatus = existingStatus;

    // Request permission if not already granted
    if (existingStatus !== "granted") {
      const { status } = await Notifications.requestPermissionsAsync();
      finalStatus = status;
    }

    if (finalStatus !== "granted") {
      setState((s) => ({
        ...s,
        permissionGranted: false,
        error: "Push notification permission was denied",
      }));
      return null;
    }

    // Android requires a notification channel
    if (Platform.OS === "android") {
      await Notifications.setNotificationChannelAsync("default", {
        name: "Default",
        importance: Notifications.AndroidImportance.HIGH,
        vibrationPattern: [0, 250, 250, 250],
        lightColor: "#1A1A2E",
      });
    }

    // Get the Expo push token
    try {
      const tokenData = await Notifications.getExpoPushTokenAsync(
        projectId ? { projectId } : undefined,
      );
      const token = tokenData.data;

      setState({
        expoPushToken: token,
        permissionGranted: true,
        error: null,
      });

      return token;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to get push token";
      setState((s) => ({
        ...s,
        permissionGranted: true,
        error: message,
      }));
      return null;
    }
  }, [projectId]);

  // Set up notification listeners on mount
  useEffect(() => {
    // Listen for incoming notifications while app is foregrounded
    notificationListener.current = Notifications.addNotificationReceivedListener(
      (_notification) => {
        // No-op: foreground notification display is handled by setNotificationHandler above.
        // Callers can extend this if needed.
      },
    );

    // Listen for notification taps (user interaction)
    responseListener.current = Notifications.addNotificationResponseReceivedListener(
      (response) => {
        lastNotificationResponse.current = response;
      },
    );

    return () => {
      if (notificationListener.current) {
        notificationListener.current.remove();
      }
      if (responseListener.current) {
        responseListener.current.remove();
      }
    };
  }, []);

  return {
    ...state,
    lastNotificationResponse,
    registerForPushNotifications,
  };
}
