import { ExpoConfig } from "expo/config";

const projectId =
  process.env.EXPO_PROJECT_ID || "6bbabfc7-f70d-45f7-bdc2-4f8387d14006";

const config: ExpoConfig = {
  name: "Permission Slip",
  slug: "permission-slip",
  owner: process.env.EXPO_OWNER || "supersuit-tech",
  version: "1.0.0",
  runtimeVersion: { policy: "appVersion" as const },
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
    bundleIdentifier: process.env.APP_BUNDLE_ID || "dev.permissionslip.app",
    associatedDomains: ["applinks:app.permissionslip.dev"],
    infoPlist: {
      UIBackgroundModes: ["remote-notification"],
      NSFaceIDUsageDescription:
        "Authenticate with Face ID to access Permission Slip",
      ITSAppUsesNonExemptEncryption: false,
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
    package: process.env.APP_BUNDLE_ID || "dev.permissionslip.app",
    adaptiveIcon: {
      backgroundColor: "#6A2C91",
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
    url: `https://u.expo.dev/${projectId}`,
    enabled: true,
    fallbackToCacheTimeout: 0,
  },
  extra: {
    eas: {
      projectId,
    },
  },
};

export default config;
