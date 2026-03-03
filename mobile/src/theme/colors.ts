/** Shared color palette for the Permission Slip mobile app. */
export const colors = {
  // Neutrals (gray scale)
  white: "#FFFFFF",
  gray50: "#F9FAFB",
  gray100: "#F3F4F6",
  gray200: "#E5E7EB",
  gray300: "#D1D5DB",
  gray400: "#9CA3AF",
  gray500: "#6B7280",
  gray700: "#374151",
  gray900: "#111827",

  // Semantic
  error: "#DC2626",
  success: "#059669",
  warning: "#D97706",

  // Risk level backgrounds and text
  riskLow: "#059669",
  riskLowBg: "#ECFDF5",
  riskMedium: "#D97706",
  riskMediumBg: "#FFFBEB",
  riskHigh: "#DC2626",
  riskHighBg: "#FEF2F2",

  // Primary / accent
  primary: "#2563EB",
  primaryBg: "#EFF6FF",
} as const;
