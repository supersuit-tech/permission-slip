/**
 * Displays the confirmation code after an approval in a large, prominent card.
 * The code is shown in XXX-XXX format with a copy-to-clipboard button.
 */
import { useCallback, useState } from "react";
import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import * as Clipboard from "expo-clipboard";
import { colors } from "../../theme/colors";

interface ConfirmationCodeCardProps {
  code: string;
}

export function ConfirmationCodeCard({ code }: ConfirmationCodeCardProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    await Clipboard.setStringAsync(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [code]);

  return (
    <View style={styles.container} accessibilityRole="alert">
      <Text style={styles.label}>Confirmation Code</Text>
      <Text style={styles.code} selectable testID="confirmation-code">
        {code}
      </Text>
      <TouchableOpacity
        style={styles.copyButton}
        onPress={handleCopy}
        accessibilityLabel={copied ? "Code copied" : "Copy confirmation code"}
        accessibilityRole="button"
        testID="copy-code-button"
      >
        <Text style={styles.copyText}>
          {copied ? "Copied!" : "Copy Code"}
        </Text>
      </TouchableOpacity>
      <Text style={styles.hint}>
        Share this code with the agent to authorize the action
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: colors.primaryBg,
    borderRadius: 12,
    padding: 20,
    alignItems: "center",
    borderWidth: 1,
    borderColor: "#BFDBFE",
  },
  label: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray500,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  code: {
    fontSize: 32,
    fontWeight: "700",
    fontFamily: "monospace",
    letterSpacing: 4,
    color: colors.gray900,
    marginBottom: 12,
  },
  copyButton: {
    backgroundColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 10,
    paddingHorizontal: 24,
    marginBottom: 12,
  },
  copyText: {
    color: colors.white,
    fontSize: 14,
    fontWeight: "600",
  },
  hint: {
    fontSize: 12,
    color: colors.gray400,
    textAlign: "center",
  },
});
