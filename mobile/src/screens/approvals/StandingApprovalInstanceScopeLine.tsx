import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";

interface StandingApprovalInstanceScopeLineProps {
  label: string;
  compact?: boolean;
}

export function StandingApprovalInstanceScopeLine({
  label,
  compact = false,
}: StandingApprovalInstanceScopeLineProps) {
  return (
    <View
      style={[styles.box, compact && styles.boxCompact]}
      accessibilityRole="text"
    >
      <Text style={[styles.text, compact && styles.textCompact]}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: {
    marginTop: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.gray200,
    backgroundColor: colors.gray50,
  },
  boxCompact: {
    marginTop: 6,
    paddingVertical: 8,
  },
  text: {
    fontSize: 13,
    lineHeight: 18,
    color: colors.gray500,
  },
  textCompact: {
    fontSize: 12,
    lineHeight: 16,
  },
});
