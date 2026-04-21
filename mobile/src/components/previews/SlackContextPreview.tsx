import { useState } from "react";
import {
  Linking,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import type { components } from "../../api/schema";
import { slackMrkdwnToPlaintext } from "../../lib/slackMrkdwn";
import { colors } from "../../theme/colors";

type SlackContext = components["schemas"]["SlackContext"];
type SlackContextMessage = components["schemas"]["SlackContextMessage"];
type SlackContextChannel = components["schemas"]["SlackContextChannel"];
type SlackContextUserRef = components["schemas"]["SlackContextUserRef"];
type SlackContextFileMeta = components["schemas"]["SlackContextFileMeta"];
type SlackContextThread = components["schemas"]["SlackContextThread"];
type SlackContextWindow = components["schemas"]["SlackContextWindow"];

interface Props {
  slackContext: SlackContext;
}

export function SlackContextPreview({ slackContext }: Props) {
  const {
    context_scope,
    channel,
    recipient,
    target_message,
    thread,
    recent_messages,
    context_window,
  } = slackContext;

  return (
    <View testID="slack-context-preview">
      {channel && <ChannelHeader channel={channel} />}

      {context_scope === "self_dm" && (
        <ScopeBadge
          text="Note to self"
          testID="badge-self-dm"
        />
      )}
      {context_scope === "first_contact_dm" && (
        <ScopeBadge
          text="No prior messages with this user"
          testID="badge-first-contact"
        />
      )}
      {context_scope === "metadata_only" && (
        <ScopeBadge
          text="Context unavailable — open in Slack"
          testID="badge-metadata-only"
        />
      )}

      {recipient && context_scope !== "self_dm" && (
        <RecipientCard recipient={recipient} />
      )}

      {target_message && (
        <View style={styles.section} testID="target-message-section">
          <Text style={styles.sectionLabel}>Target message</Text>
          <MessageCard message={target_message} />
        </View>
      )}

      {context_scope === "thread" && thread && (
        <ThreadSection thread={thread} />
      )}

      {recent_messages && recent_messages.length > 0 && (
        <RecentMessagesSection
          messages={recent_messages}
          contextWindow={context_window}
        />
      )}
    </View>
  );
}

// ---------------------------------------------------------------------------
// Channel header
// ---------------------------------------------------------------------------

function ChannelHeader({ channel }: { channel: SlackContextChannel }) {
  const label = channel.is_dm
    ? "Direct message"
    : channel.name
      ? `#${channel.name}`
      : channel.id;

  return (
    <View style={styles.channelHeader} testID="channel-header">
      <View style={styles.channelTitleRow}>
        <Text style={styles.channelName} numberOfLines={1}>
          {label}
        </Text>
        {channel.is_private && !channel.is_dm && (
          <View style={styles.privateBadge} testID="private-badge">
            <Text style={styles.privateBadgeText}>Private</Text>
          </View>
        )}
        <TouchableOpacity
          onPress={() => openUrl(channel.permalink)}
          accessibilityLabel="Open in Slack"
          accessibilityRole="link"
          testID="open-in-slack"
        >
          <Text style={styles.openInSlack}>Open in Slack ↗</Text>
        </TouchableOpacity>
      </View>

      {channel.topic ? (
        <Text style={styles.channelMeta} numberOfLines={2} testID="channel-topic">
          {channel.topic}
        </Text>
      ) : channel.purpose ? (
        <Text style={styles.channelMeta} numberOfLines={2} testID="channel-purpose">
          {channel.purpose}
        </Text>
      ) : null}

      {channel.member_count != null && (
        <Text style={styles.memberCount} testID="member-count">
          {channel.member_count.toLocaleString()} members
        </Text>
      )}
    </View>
  );
}

// ---------------------------------------------------------------------------
// Recipient card (DMs)
// ---------------------------------------------------------------------------

function RecipientCard({ recipient }: { recipient: SlackContextUserRef }) {
  const displayName = recipient.real_name ?? recipient.name ?? recipient.id ?? "Unknown";
  const initials = getInitials(displayName);
  const avatarColors = getAvatarColors(displayName);

  return (
    <View style={styles.section} testID="recipient-card">
      <Text style={styles.sectionLabel}>Recipient</Text>
      <View style={styles.card}>
        <View style={[styles.avatar, { backgroundColor: avatarColors.bg }]}>
          <Text style={[styles.avatarText, { color: avatarColors.text }]}>
            {initials}
          </Text>
        </View>
        <View style={styles.recipientInfo}>
          <Text style={styles.recipientName}>{displayName}</Text>
          {recipient.title ? (
            <Text style={styles.recipientTitle}>{recipient.title}</Text>
          ) : null}
        </View>
      </View>
    </View>
  );
}

// ---------------------------------------------------------------------------
// Message card
// ---------------------------------------------------------------------------

function MessageCard({ message }: { message: SlackContextMessage }) {
  const authorName =
    message.user?.real_name ?? message.user?.name ?? (message.is_bot ? "Bot" : "Unknown");
  const plainText = slackMrkdwnToPlaintext(message.text);
  const timestamp = formatSlackTs(message.ts);
  const initials = getInitials(authorName);
  const avatarColors = getAvatarColors(authorName);

  return (
    <View style={styles.messageCard} testID="message-card">
      <View style={styles.messageHeader}>
        <View style={[styles.avatarSmall, { backgroundColor: avatarColors.bg }]}>
          <Text style={[styles.avatarTextSmall, { color: avatarColors.text }]}>
            {initials}
          </Text>
        </View>
        <View style={styles.messageAuthorBlock}>
          <View style={styles.messageAuthorRow}>
            <Text style={styles.messageAuthor}>{authorName}</Text>
            {message.is_bot && (
              <View style={styles.botBadge} testID="bot-badge">
                <Text style={styles.botBadgeText}>Bot</Text>
              </View>
            )}
          </View>
          <Text style={styles.messageTs}>{timestamp}</Text>
        </View>
        <TouchableOpacity
          onPress={() => openUrl(message.permalink)}
          accessibilityLabel="Open message in Slack"
          accessibilityRole="link"
          testID="message-permalink"
        >
          <Text style={styles.permalinkLink}>↗</Text>
        </TouchableOpacity>
      </View>

      <Text style={styles.messageText} testID="message-text">
        {plainText}
        {message.truncated && (
          <Text style={styles.truncatedNote}> [truncated]</Text>
        )}
      </Text>

      {message.files && message.files.length > 0 && (
        <View style={styles.fileList} testID="file-list">
          {message.files.map((f, i) => (
            <FileRow key={i} file={f} />
          ))}
        </View>
      )}
    </View>
  );
}

// ---------------------------------------------------------------------------
// Thread section
// ---------------------------------------------------------------------------

function ThreadSection({ thread }: { thread: SlackContextThread }) {
  return (
    <View style={styles.section} testID="thread-section">
      <Text style={styles.sectionLabel}>Thread</Text>
      {thread.parent && (
        <View style={styles.threadParent}>
          <MessageCard message={thread.parent} />
        </View>
      )}
      {thread.replies && thread.replies.length > 0 && (
        <View style={styles.replies} testID="thread-replies">
          {thread.replies.map((reply, i) => (
            <View key={i} style={styles.replyRow}>
              <View style={styles.replyConnector} />
              <View style={styles.replyContent}>
                <MessageCard message={reply} />
              </View>
            </View>
          ))}
        </View>
      )}
      {thread.truncated && (
        <Text style={styles.truncatedNote} testID="thread-truncated">
          Thread truncated — open in Slack to see all messages
        </Text>
      )}
    </View>
  );
}

// ---------------------------------------------------------------------------
// Recent messages accordion
// ---------------------------------------------------------------------------

function RecentMessagesSection({
  messages,
  contextWindow,
}: {
  messages: SlackContextMessage[];
  contextWindow?: SlackContextWindow;
}) {
  const [expanded, setExpanded] = useState(false);

  const label = buildRecentLabel(messages.length, contextWindow);

  return (
    <View style={styles.section} testID="recent-messages-section">
      <TouchableOpacity
        style={styles.accordionHeader}
        onPress={() => setExpanded((v) => !v)}
        accessibilityRole="button"
        accessibilityLabel={expanded ? "Collapse recent activity" : "Expand recent activity"}
        testID="recent-messages-toggle"
      >
        <Text style={styles.sectionLabel}>{label}</Text>
        <Text style={styles.accordionChevron}>{expanded ? "▲" : "▼"}</Text>
      </TouchableOpacity>

      {expanded && (
        <View testID="recent-messages-list">
          {messages.map((msg, i) => (
            <View key={i} style={i > 0 ? styles.messageSeparator : undefined}>
              <MessageCard message={msg} />
            </View>
          ))}
          {contextWindow?.truncated && (
            <Text style={styles.truncatedNote} testID="recent-truncated">
              More messages available — open in Slack
            </Text>
          )}
        </View>
      )}
    </View>
  );
}

// ---------------------------------------------------------------------------
// File row
// ---------------------------------------------------------------------------

function FileRow({ file }: { file: SlackContextFileMeta }) {
  return (
    <View style={styles.fileRow} testID="file-row">
      <Text style={styles.fileName} numberOfLines={1}>
        {file.filename}
      </Text>
      <Text style={styles.fileSize}>{formatBytes(file.size_bytes)}</Text>
    </View>
  );
}

// ---------------------------------------------------------------------------
// Scope badge (empty states)
// ---------------------------------------------------------------------------

function ScopeBadge({ text, testID }: { text: string; testID: string }) {
  return (
    <View style={styles.scopeBadge} testID={testID}>
      <Text style={styles.scopeBadgeText}>{text}</Text>
    </View>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function openUrl(url: string) {
  Linking.openURL(url).catch(() => undefined);
}

function formatSlackTs(ts: string): string {
  const ms = parseFloat(ts) * 1000;
  if (isNaN(ms)) return ts;
  return new Date(ms).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function getInitials(name: string): string {
  const words = name.trim().split(/\s+/);
  if (words.length === 1) return (words[0]?.charAt(0) ?? "?").toUpperCase();
  return (
    (words[0]?.charAt(0) ?? "") + (words[words.length - 1]?.charAt(0) ?? "")
  ).toUpperCase();
}

const AVATAR_COLORS = [
  { bg: "#DBEAFE", text: "#1E40AF" },
  { bg: colors.approvedBg, text: colors.approvedText },
  { bg: "#EDE9FE", text: "#5B21B6" },
  { bg: "#FCE7F3", text: "#9D174D" },
  { bg: colors.pendingBg, text: colors.pendingText },
  { bg: "#CFFAFE", text: "#155E75" },
  { bg: "#FFEDD5", text: "#9A3412" },
  { bg: "#FFE4E6", text: "#9F1239" },
] as const;

function getAvatarColors(name: string): { bg: string; text: string } {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  }
  const index = Math.abs(hash) % AVATAR_COLORS.length;
  const entry = AVATAR_COLORS[index]!;
  return { bg: entry.bg, text: entry.text };
}

function buildRecentLabel(
  count: number,
  contextWindow?: SlackContextWindow,
): string {
  const hours = contextWindow?.hours ?? 24;
  return `Channel activity in the last ${hours}h (${count} message${count !== 1 ? "s" : ""})`;
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const styles = StyleSheet.create({
  // Section wrapper
  section: {
    marginTop: 12,
  },
  sectionLabel: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray400,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 6,
  },

  // Channel header
  channelHeader: {
    backgroundColor: colors.white,
    borderRadius: 10,
    padding: 12,
    marginBottom: 4,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 1,
  },
  channelTitleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    flexWrap: "wrap",
  },
  channelName: {
    fontSize: 15,
    fontWeight: "700",
    color: colors.gray900,
    flex: 1,
  },
  privateBadge: {
    backgroundColor: colors.gray100,
    borderRadius: 4,
    paddingHorizontal: 6,
    paddingVertical: 2,
  },
  privateBadgeText: {
    fontSize: 11,
    color: colors.gray500,
    fontWeight: "600",
  },
  openInSlack: {
    fontSize: 12,
    color: colors.primary,
    fontWeight: "600",
  },
  channelMeta: {
    fontSize: 13,
    color: colors.gray500,
    marginTop: 4,
    lineHeight: 18,
  },
  memberCount: {
    fontSize: 12,
    color: colors.gray400,
    marginTop: 4,
  },

  // Recipient card
  card: {
    backgroundColor: colors.white,
    borderRadius: 10,
    padding: 12,
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 1,
  },
  avatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: "center",
    justifyContent: "center",
  },
  avatarText: {
    fontSize: 14,
    fontWeight: "700",
  },
  recipientInfo: {
    flex: 1,
  },
  recipientName: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.gray900,
  },
  recipientTitle: {
    fontSize: 12,
    color: colors.gray500,
    marginTop: 2,
  },

  // Message card
  messageCard: {
    backgroundColor: colors.white,
    borderRadius: 10,
    padding: 12,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 1,
  },
  messageHeader: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: 8,
    marginBottom: 8,
  },
  avatarSmall: {
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
  },
  avatarTextSmall: {
    fontSize: 11,
    fontWeight: "700",
  },
  messageAuthorBlock: {
    flex: 1,
  },
  messageAuthorRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    flexWrap: "wrap",
  },
  messageAuthor: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.gray900,
  },
  botBadge: {
    backgroundColor: colors.gray100,
    borderRadius: 4,
    paddingHorizontal: 5,
    paddingVertical: 1,
  },
  botBadgeText: {
    fontSize: 10,
    color: colors.gray500,
    fontWeight: "600",
  },
  messageTs: {
    fontSize: 11,
    color: colors.gray400,
    marginTop: 1,
  },
  permalinkLink: {
    fontSize: 14,
    color: colors.primary,
    paddingLeft: 4,
  },
  messageText: {
    fontSize: 13,
    color: colors.gray700,
    lineHeight: 19,
  },
  truncatedNote: {
    fontSize: 11,
    color: colors.gray400,
    fontStyle: "italic",
    marginTop: 4,
  },
  fileList: {
    marginTop: 8,
    gap: 4,
  },
  fileRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    backgroundColor: colors.gray50,
    borderRadius: 6,
    paddingHorizontal: 8,
    paddingVertical: 5,
  },
  fileName: {
    fontSize: 12,
    color: colors.gray700,
    flex: 1,
  },
  fileSize: {
    fontSize: 11,
    color: colors.gray400,
    marginLeft: 8,
  },

  // Thread
  threadParent: {
    marginBottom: 8,
  },
  replies: {
    gap: 6,
  },
  replyRow: {
    flexDirection: "row",
    gap: 0,
  },
  replyConnector: {
    width: 2,
    backgroundColor: colors.gray200,
    borderRadius: 1,
    marginRight: 10,
    marginLeft: 13,
  },
  replyContent: {
    flex: 1,
  },

  // Accordion
  accordionHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingVertical: 2,
  },
  accordionChevron: {
    fontSize: 10,
    color: colors.gray400,
  },
  messageSeparator: {
    marginTop: 8,
  },

  // Scope badges
  scopeBadge: {
    backgroundColor: colors.gray100,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 8,
    marginTop: 4,
  },
  scopeBadgeText: {
    fontSize: 13,
    color: colors.gray500,
    fontWeight: "500",
    textAlign: "center",
  },
});
