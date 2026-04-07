import { ExpoConfig, ConfigContext } from "expo/config";

export default (_config: ConfigContext): ExpoConfig => ({
  name: "Permission Slip",
  slug: "permission-slip",
  owner: "supersuit-tech",
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
    bundleIdentifier: "dev.permissionslip.app",
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
    package: "dev.permissionslip.app",
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
    url: `https://u.expo.dev/${process.env.EXPO_PROJECT_ID}`,
    enabled: true,
    fallbackToCacheTimeout: 0,
  },
  extra: {
    eas: {
      projectId: "6bbabfc7-f70d-45f7-bdc2-4f8387d14006",
    },
  },
});
