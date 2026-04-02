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

  // Status badges
  pendingBg: "#FEF3C7",
  pendingText: "#92400E",
  approvedBg: "#D1FAE5",
  approvedText: "#065F46",
  deniedBg: "#FEE2E2",
  deniedText: "#991B1B",

  // Primary / accent
  primary: "#6A2C91",
  primaryBg: "#F5F0FA",
  primaryBorder: "#DDD5E5",
  secondary: "#D4A843",

  // Cancelled / expired status
  cancelledBg: "#F3F4F6",
  cancelledText: "#374151",
} as const;
