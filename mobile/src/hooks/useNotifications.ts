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

export interface UseNotificationsOptions {
  /** Expo project ID. Optional — inferred in managed workflow builds. */
  projectId?: string;
  /** Callback invoked when a notification is received while the app is in the foreground. */
  onNotificationReceived?: (notification: Notifications.Notification) => void;
  /** Callback invoked when the user taps a notification (foreground, background, or cold start). */
  onNotificationTap?: (response: Notifications.NotificationResponse) => void;
}

export function useNotifications(options: UseNotificationsOptions = {}) {
  const { projectId, onNotificationReceived, onNotificationTap } = options;
  const [state, setState] = useState<NotificationState>({
    expoPushToken: null,
    permissionGranted: false,
    error: null,
  });

  const notificationListener = useRef<Notifications.EventSubscription | null>(null);
  const responseListener = useRef<Notifications.EventSubscription | null>(null);

  /** The last notification response (tap). Callers can read this for navigation. */
  const lastNotificationResponse = useRef<Notifications.NotificationResponse | null>(null);

  // Keep the callback refs fresh without re-creating the effect subscription
  const onNotificationReceivedRef = useRef(onNotificationReceived);
  onNotificationReceivedRef.current = onNotificationReceived;
  const onNotificationTapRef = useRef(onNotificationTap);
  onNotificationTapRef.current = onNotificationTap;

  const registerForPushNotifications = useCallback(async (): Promise<string | null> => {
    // Push notifications only work on physical devices
    if (!Device.isDevice) {
      if (__DEV__) {
        console.log(
          "[notifications] Running on simulator — push notifications are unavailable. " +
            "Use a physical device to test push notifications.",
        );
      }
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

    // Android requires a notification channel. We create a dedicated channel
    // for approval requests so users can configure notification preferences
    // (sound, vibration, etc.) independently from other future channels.
    if (Platform.OS === "android") {
      await Notifications.setNotificationChannelAsync("approval-requests", {
        name: "Approval Requests",
        description: "Notifications for new approval requests that need your attention",
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
      (notification) => {
        onNotificationReceivedRef.current?.(notification);
      },
    );

    // Listen for notification taps (user interaction — app in foreground or background)
    responseListener.current = Notifications.addNotificationResponseReceivedListener(
      (response) => {
        lastNotificationResponse.current = response;
        onNotificationTapRef.current?.(response);
      },
    );

    // Handle cold start: check if the app was launched by a notification tap.
    // getLastNotificationResponseAsync returns the response that opened the app
    // when it was fully killed, since the listener above only fires for taps
    // that happen while the JS runtime is already running.
    Notifications.getLastNotificationResponseAsync().then((response) => {
      if (response) {
        lastNotificationResponse.current = response;
        onNotificationTapRef.current?.(response);
      }
    });

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
