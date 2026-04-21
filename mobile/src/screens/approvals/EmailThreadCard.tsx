/**
 * Renders normalized `email_thread` for email reply approvals (plain text bodies).
 * Latest message is expanded; earlier messages live under a collapsible section.
 */
import { useMemo, useState } from "react";
import {
  Alert,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { colors } from "../../theme/colors";
import { formatTimestamp } from "./approvalUtils";
import type { EmailThread, EmailThreadMessage } from "./emailThreadUtils";

interface EmailThreadCardProps {
  thread: EmailThread | null;
  testID?: string;
}

function formatSizeBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function messageBodyText(m: EmailThreadMessage): string {
  const trimmed = m.body_text?.trim() ?? "";
  if (trimmed.length > 0) return m.body_text;
  const sn = m.snippet?.trim() ?? "";
  return sn.length > 0 ? m.snippet : "(No message body)";
}

function TruncationNote({ visible }: { visible: boolean }) {
  if (!visible) return null;
  return (
    <TouchableOpacity
      accessibilityRole="button"
      accessibilityLabel="Show more about truncated message"
      onPress={() =>
        Alert.alert(
          "Message truncated",
          "This message body was shortened on the server (about 20 KB maximum per field). The full text is not available in this preview.",
        )
      }
      style={styles.truncationLink}
      testID="email-thread-truncation-note"
    >
      <Text style={styles.truncationLinkText}>Show more</Text>
    </TouchableOpacity>
  );
}

function AddressLine({ label, values }: { label: string; values: string[] }) {
  if (values.length === 0) return null;
  return (
    <Text style={styles.metaLine} selectable>
      <Text style={styles.metaLabel}>{label}: </Text>
      {values.join(", ")}
    </Text>
  );
}

function AttachmentChips({
  attachments,
}: {
  attachments: NonNullable<EmailThreadMessage["attachments"]>;
}) {
  return (
    <View style={styles.attachRow} accessibilityRole="list">
      {attachments.map((a) => (
        <View
          key={`${a.filename}-${a.size_bytes}`}
          style={styles.attachChip}
          accessibilityRole="text"
          accessibilityLabel={`Attachment ${a.filename}`}
        >
          <Text style={styles.attachChipText} numberOfLines={1}>
            {a.filename}
            {a.size_bytes > 0 ? ` · ${formatSizeBytes(a.size_bytes)}` : ""}
          </Text>
        </View>
      ))}
    </View>
  );
}

function MessageBlock({
  message,
  testID,
}: {
  message: EmailThreadMessage;
  testID?: string;
}) {
  const body = messageBodyText(message);
  const dateLabel = message.date ? formatTimestamp(message.date) : "—";

  return (
    <View style={styles.messageBlock} testID={testID}>
      <Text style={styles.fromLine} selectable numberOfLines={3}>
        {message.from || "(Unknown sender)"}
      </Text>
      <AddressLine label="To" values={message.to} />
      <AddressLine label="Cc" values={message.cc} />
      <Text style={styles.dateLine} selectable>
        {dateLabel}
      </Text>
      <Text style={styles.bodyText} selectable>
        {body}
      </Text>
      <TruncationNote visible={message.truncated} />
      {message.attachments && message.attachments.length > 0 ? (
        <AttachmentChips attachments={message.attachments} />
      ) : null}
    </View>
  );
}

export function EmailThreadCard({ thread, testID }: EmailThreadCardProps) {
  const [earlierOpen, setEarlierOpen] = useState(false);

  const { latest, earlier } = useMemo(() => {
    const msgs = thread?.messages ?? [];
    if (msgs.length === 0) {
      return { latest: null as EmailThreadMessage | null, earlier: [] as EmailThreadMessage[] };
    }
    return {
      latest: msgs[msgs.length - 1] ?? null,
      earlier: msgs.slice(0, -1),
    };
  }, [thread?.messages]);

  const hasThread =
    (thread?.subject?.trim().length ?? 0) > 0 ||
    (thread?.messages?.length ?? 0) > 0;

  return (
    <View style={styles.card} testID={testID}>
      <Text style={styles.sectionLabel}>Email thread</Text>
      {!hasThread ? (
        <Text style={styles.emptyText} testID="email-thread-empty">
          No conversation loaded for this request.
        </Text>
      ) : (
        <>
          {thread?.subject ? (
            <Text style={styles.subject} selectable numberOfLines={4}>
              {thread.subject}
            </Text>
          ) : null}

          {latest ? (
            <MessageBlock message={latest} testID="email-thread-latest" />
          ) : null}

          {earlier.length > 0 ? (
            <View style={styles.earlierSection}>
              <TouchableOpacity
                accessibilityRole="button"
                accessibilityState={{ expanded: earlierOpen }}
                onPress={() => setEarlierOpen((o) => !o)}
                style={styles.earlierToggle}
                testID="email-thread-earlier-toggle"
              >
                <Text style={styles.earlierToggleText}>
                  {earlierOpen ? "▼" : "▶"} Earlier in this thread ({earlier.length})
                </Text>
              </TouchableOpacity>
              {earlierOpen ? (
                <View style={styles.earlierList} testID="email-thread-earlier-list">
                  {earlier.map((m, i) => (
                    <View
                      key={m.message_id || `earlier-${i}`}
                      style={styles.earlierItem}
                    >
                      <MessageBlock
                        message={m}
                        testID={i === 0 ? "email-thread-earlier-first" : undefined}
                      />
                    </View>
                  ))}
                </View>
              ) : null}
            </View>
          ) : null}
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
  },
  sectionLabel: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray400,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 12,
  },
  subject: {
    fontSize: 16,
    fontWeight: "700",
    color: colors.gray900,
    marginBottom: 14,
  },
  emptyText: {
    fontSize: 14,
    color: colors.gray500,
    lineHeight: 20,
  },
  messageBlock: {
    marginBottom: 0,
  },
  fromLine: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
    marginBottom: 6,
  },
  metaLine: {
    fontSize: 13,
    color: colors.gray700,
    marginBottom: 4,
    lineHeight: 18,
  },
  metaLabel: {
    fontWeight: "600",
    color: colors.gray500,
  },
  dateLine: {
    fontSize: 12,
    color: colors.gray400,
    marginBottom: 10,
  },
  bodyText: {
    fontSize: 14,
    color: colors.gray900,
    lineHeight: 22,
  },
  truncationLink: {
    alignSelf: "flex-start",
    marginTop: 8,
    paddingVertical: 4,
  },
  truncationLinkText: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.primary,
  },
  attachRow: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
    marginTop: 10,
  },
  attachChip: {
    maxWidth: "100%",
    backgroundColor: colors.gray100,
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderWidth: 1,
    borderColor: colors.gray200,
  },
  attachChipText: {
    fontSize: 12,
    color: colors.gray700,
  },
  earlierSection: {
    marginTop: 16,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: colors.gray200,
  },
  earlierToggle: {
    paddingVertical: 8,
  },
  earlierToggleText: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.primary,
  },
  earlierList: {
    marginTop: 4,
  },
  earlierItem: {
    marginBottom: 16,
    paddingBottom: 16,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray200,
  },
});
