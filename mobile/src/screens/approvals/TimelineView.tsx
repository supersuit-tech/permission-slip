/**
 * Vertical timeline component — displays timestamped events as a
 * connected line with dots. Terminal events (approved/denied) use
 * semantic colors.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";

export interface TimelineEntry {
  label: string;
  value: string;
  /** Semantic color for the dot — "success", "error", or default gray. */
  dotColor?: "success" | "error";
}

interface TimelineViewProps {
  entries: TimelineEntry[];
}

export function TimelineView({ entries }: TimelineViewProps) {
  return (
    <View style={styles.container}>
      {entries.map((entry, index) => {
        const isLast = index === entries.length - 1;
        const dotBg =
          entry.dotColor === "success"
            ? colors.success
            : entry.dotColor === "error"
              ? colors.error
              : colors.gray300;

        return (
          <View key={entry.label} style={styles.row}>
            {/* Left rail: dot + connecting line */}
            <View style={styles.rail}>
              <View style={[styles.dot, { backgroundColor: dotBg }]} />
              {!isLast && <View style={styles.line} />}
            </View>
            {/* Right content: label + timestamp */}
            <View style={styles.content}>
              <Text style={styles.label}>{entry.label}</Text>
              <Text style={styles.value}>{entry.value}</Text>
            </View>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingLeft: 4,
  },
  row: {
    flexDirection: "row",
    minHeight: 32,
  },
  rail: {
    width: 20,
    alignItems: "center",
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginTop: 5,
  },
  line: {
    width: 2,
    flex: 1,
    backgroundColor: colors.gray200,
    marginVertical: 2,
  },
  content: {
    flex: 1,
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
    paddingBottom: 12,
    marginLeft: 8,
  },
  label: {
    fontSize: 13,
    fontWeight: "500",
    color: colors.gray500,
  },
  value: {
    fontSize: 13,
    color: colors.gray900,
  },
});
