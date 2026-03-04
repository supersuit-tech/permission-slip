/**
 * Renders a list of key-value pairs in a card layout. Used for displaying
 * approval parameters, context details, and timeline entries. Handles both
 * short (inline) and long (stacked) values automatically.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";
import { formatParamValue } from "./approvalUtils";

export interface KeyValueEntry {
  label: string;
  value: unknown;
}

interface KeyValueListProps {
  entries: KeyValueEntry[];
}

export function KeyValueList({ entries }: KeyValueListProps) {
  return (
    <>
      {entries.map(({ label, value }, index) => {
        const formatted =
          typeof value === "string" ? value : formatParamValue(value);
        const isLong = formatted.length > 40 || formatted.includes("\n");
        const isLast = index === entries.length - 1;
        return (
          <View
            key={label}
            style={[
              isLong ? styles.rowVertical : styles.row,
              isLast && styles.rowLast,
            ]}
          >
            <Text style={styles.key}>{label}</Text>
            <Text
              style={isLong ? styles.valueFull : styles.value}
              selectable
            >
              {formatted}
            </Text>
          </View>
        );
      })}
    </>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  rowVertical: {
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  rowLast: {
    borderBottomWidth: 0,
  },
  key: {
    fontSize: 13,
    fontWeight: "500",
    color: colors.gray500,
    marginRight: 12,
    flexShrink: 0,
  },
  value: {
    fontSize: 13,
    color: colors.gray900,
    flex: 1,
    textAlign: "right",
  },
  valueFull: {
    fontSize: 13,
    color: colors.gray900,
    marginTop: 4,
    lineHeight: 19,
  },
});
