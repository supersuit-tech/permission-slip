import { execSync } from "child_process";
import { ExpoConfig } from "expo/config";

const projectId =
  process.env.EXPO_PROJECT_ID || "6bbabfc7-f70d-45f7-bdc2-4f8387d14006";

let gitCommitHash = "unknown";
try {
  gitCommitHash = execSync("git rev-parse HEAD", { encoding: "utf-8" }).trim();
} catch {
  // Git not available at build time — leave as "unknown"
}

let gitCommitTimestamp = "unknown";
try {
  gitCommitTimestamp = execSync("git log -1 --format=%cI HEAD", {
    encoding: "utf-8",
  }).trim();
} catch {
  // Git not available at build time — leave as "unknown"
}

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
    supportsTablet: false,
    bundleIdentifier: process.env.APP_BUNDLE_ID || "dev.permissionslip.app",
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
    gitCommitHash,
    gitCommitTimestamp,
    eas: {
      projectId,
    },
  },
};

export default config;
