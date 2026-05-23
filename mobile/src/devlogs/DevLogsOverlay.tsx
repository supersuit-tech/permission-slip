import { useMemo, useState, useSyncExternalStore } from "react";
import {
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import {
  isDevModeEnabled,
  subscribeDevMode,
} from "../lib/devModeConfig";
import {
  clearDevLogs,
  useDevLogs,
  type DevLogEntry,
} from "./devLogsStore";

/**
 * Fixed bottom overlay that lists every API request/response captured by
 * {@link loggingMiddleware}. Visible only while the Developer Mode toggle
 * in Settings is on; mounted unconditionally in App.tsx so flipping the
 * toggle takes effect without a remount.
 */
export default function DevLogsOverlay() {
  const enabled = useSyncExternalStore(
    subscribeDevMode,
    isDevModeEnabled,
    isDevModeEnabled,
  );
  const entries = useDevLogs();
  const [expanded, setExpanded] = useState(true);
  const [focusedId, setFocusedId] = useState<string | null>(null);

  const focused = useMemo(
    () => entries.find((e) => e.id === focusedId) ?? null,
    [entries, focusedId],
  );

  if (!enabled) return null;

  return (
    <View
      style={[styles.container, expanded ? styles.expanded : styles.collapsed]}
      pointerEvents="box-none"
    >
      <View style={styles.header}>
        <Pressable
          onPress={() => setExpanded((v) => !v)}
          accessibilityRole="button"
          accessibilityLabel={expanded ? "Collapse dev logs" : "Expand dev logs"}
          style={styles.headerButton}
        >
          <Text style={styles.headerTitle}>
            Dev logs ({entries.length}) {expanded ? "▾" : "▴"}
          </Text>
        </Pressable>
        {expanded && entries.length > 0 ? (
          <Pressable
            onPress={() => {
              clearDevLogs();
              setFocusedId(null);
            }}
            accessibilityRole="button"
            accessibilityLabel="Clear dev logs"
            style={styles.headerButton}
          >
            <Text style={styles.clearText}>Clear</Text>
          </Pressable>
        ) : null}
      </View>

      {expanded ? (
        focused ? (
          <DetailView
            entry={focused}
            onClose={() => setFocusedId(null)}
          />
        ) : (
          <EntryList entries={entries} onSelect={setFocusedId} />
        )
      ) : null}
    </View>
  );
}

function EntryList({
  entries,
  onSelect,
}: {
  entries: DevLogEntry[];
  onSelect: (id: string) => void;
}) {
  if (entries.length === 0) {
    return (
      <View style={styles.emptyState}>
        <Text style={styles.emptyText}>
          No requests yet. Trigger an action to see logs here.
        </Text>
      </View>
    );
  }
  // Newest first.
  const ordered = [...entries].reverse();
  return (
    <ScrollView style={styles.list} contentContainerStyle={styles.listContent}>
      {ordered.map((entry) => (
        <TouchableOpacity
          key={entry.id}
          onPress={() => onSelect(entry.id)}
          style={styles.row}
          accessibilityRole="button"
          accessibilityLabel={`Show details for ${entry.method} ${entry.url}`}
        >
          <Text style={styles.rowText} numberOfLines={1}>
            <Text style={[styles.status, statusStyle(entry)]}>
              {entry.status ?? "ERR"}
            </Text>{" "}
            <Text style={styles.method}>{entry.method}</Text>{" "}
            {shortPath(entry.url)}{" "}
            <Text style={styles.duration}>{entry.durationMs}ms</Text>
          </Text>
        </TouchableOpacity>
      ))}
    </ScrollView>
  );
}

function DetailView({
  entry,
  onClose,
}: {
  entry: DevLogEntry;
  onClose: () => void;
}) {
  return (
    <View style={styles.detail}>
      <View style={styles.detailHeader}>
        <Text style={styles.detailTitle} numberOfLines={2}>
          <Text style={[styles.status, statusStyle(entry)]}>
            {entry.status ?? "ERR"}
          </Text>{" "}
          {entry.method} {entry.url}
        </Text>
        <Pressable
          onPress={onClose}
          accessibilityRole="button"
          accessibilityLabel="Close detail"
          style={styles.headerButton}
        >
          <Text style={styles.clearText}>Back</Text>
        </Pressable>
      </View>
      <Text style={styles.detailMeta}>
        {entry.durationMs}ms · {new Date(entry.startedAt).toLocaleTimeString()}
      </Text>
      <ScrollView style={styles.detailBody}>
        <Text style={styles.bodyText} selectable>
          {entry.body || "(empty body)"}
        </Text>
      </ScrollView>
    </View>
  );
}

function shortPath(url: string): string {
  // Strip protocol+host for readability; keep query string.
  const match = url.match(/^[a-z]+:\/\/[^/]+(\/.*)$/i);
  return match?.[1] ?? url;
}

function statusStyle(entry: DevLogEntry) {
  if (entry.isError) return styles.statusError;
  if (entry.status != null && entry.status >= 300) return styles.statusWarn;
  return styles.statusOk;
}

const styles = StyleSheet.create({
  container: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: "rgba(17, 17, 17, 0.94)",
    borderTopLeftRadius: 12,
    borderTopRightRadius: 12,
    borderTopWidth: 1,
    borderColor: "rgba(255,255,255,0.15)",
  },
  collapsed: {
    paddingBottom: 8,
  },
  expanded: {
    maxHeight: "45%",
    paddingBottom: 12,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  headerButton: {
    paddingVertical: 4,
    paddingHorizontal: 8,
  },
  headerTitle: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "600",
    letterSpacing: 0.3,
  },
  clearText: {
    color: "#9cc4ff",
    fontSize: 12,
    fontWeight: "600",
  },
  list: {
    flexGrow: 0,
  },
  listContent: {
    paddingHorizontal: 12,
    paddingBottom: 12,
  },
  emptyState: {
    paddingHorizontal: 16,
    paddingBottom: 16,
  },
  emptyText: {
    color: "rgba(255,255,255,0.6)",
    fontSize: 12,
  },
  row: {
    paddingVertical: 4,
  },
  rowText: {
    color: "#fff",
    fontFamily: "Menlo",
    fontSize: 11,
  },
  status: {
    fontWeight: "700",
  },
  statusOk: {
    color: "#7ee787",
  },
  statusWarn: {
    color: "#f0b000",
  },
  statusError: {
    color: "#ff7b72",
  },
  method: {
    color: "#9cc4ff",
    fontWeight: "600",
  },
  duration: {
    color: "rgba(255,255,255,0.5)",
  },
  detail: {
    paddingHorizontal: 12,
    paddingBottom: 12,
  },
  detailHeader: {
    flexDirection: "row",
    alignItems: "flex-start",
    justifyContent: "space-between",
    gap: 8,
  },
  detailTitle: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "600",
    flexShrink: 1,
  },
  detailMeta: {
    color: "rgba(255,255,255,0.5)",
    fontSize: 11,
    marginTop: 4,
    marginBottom: 6,
  },
  detailBody: {
    maxHeight: 200,
    backgroundColor: "rgba(0,0,0,0.35)",
    borderRadius: 6,
    padding: 8,
  },
  bodyText: {
    color: "#e6edf3",
    fontFamily: "Menlo",
    fontSize: 11,
    lineHeight: 15,
  },
});
