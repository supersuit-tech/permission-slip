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

function readMobileVersion(): string {
  try {
    const tag = execSync("git describe --tags --match 'mobile/v*' --abbrev=0", {
      encoding: "utf-8",
    }).trim();
    return tag.replace(/^mobile\/v/, "");
  } catch {
    return "1.0.0";
  }
}

const appVersion = readMobileVersion();

const config: ExpoConfig = {
  name: "Permission Slip",
  slug: "permission-slip",
  owner: process.env.EXPO_OWNER || "supersuit-tech",
  version: appVersion,
  // runtimeVersion is intentionally pinned and decoupled from `version`. EAS
  // only delivers an OTA to a native binary whose runtimeVersion matches the
  // published update's. With the previous `policy: "appVersion"`, every
  // auto-incremented mobile/v* tag bumped runtimeVersion and orphaned all
  // existing installs from OTA updates — defeating the point of OTA. Keep this
  // pinned to the value shipped in the installed native builds ("1.0.1") so
  // JS-only OTA updates keep reaching them. Only bump it when the native layer
  // actually changes (new native module, Expo SDK upgrade), which genuinely
  // requires a fresh native build anyway.
  runtimeVersion: "1.0.1",
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
