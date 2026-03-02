/**
 * Risk level badge component — renders a colored pill (Low / Medium / High)
 * based on the approval's risk_level. Returns null when no level is provided.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";

interface RiskBadgeProps {
  level?: "low" | "medium" | "high" | null;
}

export function RiskBadge({ level }: RiskBadgeProps) {
  if (!level) return null;

  const config = RISK_STYLES[level];
  return (
    <View
      style={[styles.badge, { backgroundColor: config.bg }]}
      accessibilityLabel={`Risk level: ${level}`}
    >
      <Text style={[styles.text, { color: config.text }]}>
        {level.charAt(0).toUpperCase() + level.slice(1)}
      </Text>
    </View>
  );
}

const RISK_STYLES = {
  low: { bg: colors.riskLowBg, text: colors.riskLow },
  medium: { bg: colors.riskMediumBg, text: colors.riskMedium },
  high: { bg: colors.riskHighBg, text: colors.riskHigh },
} as const;

const styles = StyleSheet.create({
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 12,
  },
  text: {
    fontSize: 11,
    fontWeight: "600",
  },
});
