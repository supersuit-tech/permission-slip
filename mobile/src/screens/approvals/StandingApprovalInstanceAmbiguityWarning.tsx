import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";
import { STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING } from "./standingApprovalInstanceAmbiguity";

interface StandingApprovalInstanceAmbiguityWarningProps {
  compact?: boolean;
}

export function StandingApprovalInstanceAmbiguityWarning({
  compact = false,
}: StandingApprovalInstanceAmbiguityWarningProps) {
  return (
    <View
      style={[styles.box, compact && styles.boxCompact]}
      accessibilityRole="alert"
    >
      <Text style={[styles.text, compact && styles.textCompact]}>
        {STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING}
      </Text>
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
    borderColor: colors.pendingBg,
    backgroundColor: colors.riskMediumBg,
  },
  boxCompact: {
    marginTop: 6,
    paddingVertical: 8,
  },
  text: {
    fontSize: 13,
    lineHeight: 18,
    color: colors.pendingText,
  },
  textCompact: {
    fontSize: 12,
    lineHeight: 16,
  },
});
