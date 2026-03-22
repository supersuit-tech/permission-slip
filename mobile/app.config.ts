import { ExpoConfig, ConfigContext } from "expo/config";

export default ({ config }: ConfigContext): ExpoConfig => ({
  name: "Permission Slip",
  slug: "permission-slip",
  version: "1.0.0",
  // Only include runtimeVersion during EAS builds — Expo Go doesn't support it
  ...(process.env.EAS_BUILD
    ? { runtimeVersion: { policy: "appVersion" as const } }
    : {}),
  scheme: "permissionslip",
  orientation: "portrait",
  icon: "./assets/icon.png",
  userInterfaceStyle: "light",
  splash: {
    image: "./assets/splash-icon.png",
    resizeMode: "contain",
    backgroundColor: "#ffffff",
  },
  ios: {
    supportsTablet: true,
    bundleIdentifier: "dev.permissionslip.app",
    associatedDomains: ["applinks:app.permissionslip.dev"],
    infoPlist: {
      UIBackgroundModes: ["remote-notification"],
      NSFaceIDUsageDescription:
        "Authenticate with Face ID to access Permission Slip",
    },
  },
  plugins: [
    [
      "expo-notifications",
      {
        icon: "./assets/icon.png",
        color: "#1A1A2E",
      },
    ],
    "expo-local-authentication",
    "expo-updates",
  ],
  android: {
    package: "dev.permissionslip.app",
    adaptiveIcon: {
      backgroundColor: "#E6F4FE",
      foregroundImage: "./assets/android-icon-foreground.png",
      backgroundImage: "./assets/android-icon-background.png",
      monochromeImage: "./assets/android-icon-monochrome.png",
    },
    intentFilters: [
      {
        action: "VIEW",
        autoVerify: true,
        data: [
          {
            scheme: "https",
            host: "app.permissionslip.dev",
            pathPrefix: "/permission-slip/approve/",
          },
        ],
        category: ["DEFAULT", "BROWSABLE"],
      },
    ],
    predictiveBackGestureEnabled: false,
  },
  web: {
    favicon: "./assets/favicon.png",
  },
  updates: {
    url: "https://u.expo.dev/${EXPO_PROJECT_ID}",
    enabled: true,
    fallbackToCacheTimeout: 0,
  },
});
