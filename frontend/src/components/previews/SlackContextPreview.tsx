import { useState } from "react";
import {
  Hash,
  Lock,
  AtSign,
  ExternalLink,
  Paperclip,
  Bot,
  ChevronRight,
  Users,
} from "lucide-react";
import type { components } from "@/api/schema";
import { slackMrkdwnToHtml } from "@/lib/slackMrkdwn";
import { getInitials } from "@/components/ui/avatar";

type SlackContext = components["schemas"]["SlackContext"];
type SlackContextMessage = components["schemas"]["SlackContextMessage"];
type SlackContextChannel = components["schemas"]["SlackContextChannel"];
type SlackContextUserRef = components["schemas"]["SlackContextUserRef"];
type SlackContextFileMeta = components["schemas"]["SlackContextFileMeta"];

interface SlackContextPreviewProps {
  slackContext: SlackContext;
}

function formatSlackTs(ts: string): string {
  const seconds = parseFloat(ts);
  if (isNaN(seconds)) return ts;
  return new Date(seconds * 1000).toLocaleString(undefined, {
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

function UserAvatar({ user }: { user: SlackContextUserRef }) {
  const displayName = user.real_name ?? user.name ?? "?";
  const initials = getInitials(displayName);
  if (user.avatar_url) {
    return (
      <img
        src={user.avatar_url}
        alt={displayName}
        className="size-6 rounded-full object-cover"
      />
    );
  }
  return (
    <span
      className="flex size-6 shrink-0 items-center justify-center rounded-full bg-violet-100 text-[10px] font-semibold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
      aria-hidden="true"
    >
      {initials}
    </span>
  );
}

function FileList({ files }: { files: SlackContextFileMeta[] }) {
  if (files.length === 0) return null;
  return (
    <div className="mt-1.5 flex flex-wrap gap-1.5">
      {files.map((f) => (
        <span
          key={`${f.filename}-${f.size_bytes}`}
          className="inline-flex items-center gap-1 rounded border bg-muted/50 px-2 py-0.5 text-xs text-muted-foreground"
        >
          <Paperclip className="size-3 shrink-0" aria-hidden="true" />
          {f.filename}
          <span className="opacity-60">({formatBytes(f.size_bytes)})</span>
        </span>
      ))}
    </div>
  );
}

function MessageCard({
  message,
  compact = false,
}: {
  message: SlackContextMessage;
  compact?: boolean;
}) {
  const html = slackMrkdwnToHtml(message.text);
  return (
    <div className={`flex gap-2 ${compact ? "py-1.5" : "py-2"}`}>
      {message.user ? (
        <UserAvatar user={message.user} />
      ) : (
        <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-[10px] text-muted-foreground">
          <Bot className="size-3.5" aria-hidden="true" />
        </span>
      )}
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-1.5 flex-wrap">
          <span className="text-xs font-semibold">
            {message.user?.real_name ?? message.user?.name ?? "Bot"}
          </span>
          {message.is_bot && (
            <span className="rounded bg-muted px-1 py-0 text-[10px] text-muted-foreground">
              APP
            </span>
          )}
          <span className="text-[11px] text-muted-foreground">
            {formatSlackTs(message.ts)}
          </span>
          <a
            href={message.permalink}
            target="_blank"
            rel="noopener noreferrer"
            className="ml-auto text-[11px] text-muted-foreground hover:text-foreground inline-flex items-center gap-0.5"
            aria-label="View message in Slack"
          >
            <ExternalLink className="size-2.5" aria-hidden="true" />
          </a>
        </div>
        {message.truncated && (
          <p className="mb-0.5 text-[11px] text-amber-600 dark:text-amber-400">
            Message truncated
          </p>
        )}
        <div
          className="slack-message-body text-sm leading-relaxed [&_blockquote]:border-l-2 [&_blockquote]:border-muted-foreground/40 [&_blockquote]:pl-2 [&_blockquote]:text-muted-foreground [&_code]:rounded [&_code]:bg-muted/70 [&_code]:px-1 [&_code]:py-0 [&_code]:text-xs [&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:bg-muted/70 [&_pre]:p-2 [&_pre]:text-xs"
          dangerouslySetInnerHTML={{ __html: html }}
        />
        {message.files && message.files.length > 0 && (
          <FileList files={message.files} />
        )}
      </div>
    </div>
  );
}

function ChannelHeader({ channel }: { channel: SlackContextChannel }) {
  const Icon = channel.is_dm
    ? AtSign
    : channel.is_private
      ? Lock
      : Hash;

  return (
    <div className="mb-3 flex items-start justify-between gap-2">
      <div className="flex items-start gap-2 min-w-0">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-violet-50 ring-1 ring-violet-200 dark:bg-violet-950 dark:ring-violet-800">
          <Icon
            className="size-4 text-violet-600 dark:text-violet-400"
            aria-hidden="true"
          />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-sm font-semibold truncate">
              {channel.is_dm ? "" : channel.is_private ? "" : "#"}
              {channel.name ?? channel.id}
            </span>
            {channel.is_private && !channel.is_dm && (
              <span className="rounded-full bg-muted px-1.5 py-0 text-[10px] text-muted-foreground">
                Private
              </span>
            )}
          </div>
          {(channel.topic || channel.purpose) && (
            <p className="text-xs text-muted-foreground truncate mt-0.5">
              {channel.topic || channel.purpose}
            </p>
          )}
          {channel.member_count !== undefined && !channel.is_dm && (
            <p className="mt-0.5 flex items-center gap-1 text-[11px] text-muted-foreground">
              <Users className="size-3" aria-hidden="true" />
              {channel.member_count.toLocaleString()} members
            </p>
          )}
        </div>
      </div>
      <a
        href={channel.permalink}
        target="_blank"
        rel="noopener noreferrer"
        className="shrink-0 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
        aria-label="Open channel in Slack"
      >
        <ExternalLink className="size-3.5" aria-hidden="true" />
        Open in Slack
      </a>
    </div>
  );
}

function RecipientCard({ user }: { user: SlackContextUserRef }) {
  return (
    <div className="flex items-center gap-2 rounded-lg bg-muted/40 px-3 py-2 mb-3">
      <UserAvatar user={user} />
      <div className="min-w-0">
        <p className="text-xs font-semibold truncate">
          {user.real_name ?? user.name}
        </p>
        {user.title && (
          <p className="text-[11px] text-muted-foreground truncate">
            {user.title}
          </p>
        )}
      </div>
    </div>
  );
}

function ThreadSection({
  thread,
}: {
  thread: NonNullable<SlackContext["thread"]>;
}) {
  const replyCount = thread.replies?.length ?? 0;
  return (
    <div>
      {thread.parent && (
        <div className="border-b pb-2">
          <p className="mb-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wide">
            Original message
          </p>
          <MessageCard message={thread.parent} />
        </div>
      )}
      {replyCount > 0 && (
        <div className="pt-2">
          <p className="mb-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wide">
            {replyCount} {replyCount === 1 ? "reply" : "replies"}
            {thread.truncated && " (showing most recent)"}
          </p>
          <div className="divide-y">
            {(thread.replies ?? []).map((msg, i) => (
              <MessageCard key={i} message={msg} compact />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function RecentActivitySection({
  messages,
  contextWindow,
  label,
}: {
  messages: SlackContextMessage[];
  contextWindow?: SlackContext["context_window"];
  label: string;
}) {
  const [open, setOpen] = useState(false);
  const count = contextWindow?.message_count ?? messages.length;
  const hours = contextWindow?.hours ?? 24;

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-1.5 text-left text-xs font-medium text-muted-foreground hover:text-foreground transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 rounded"
        aria-expanded={open}
      >
        <ChevronRight
          className="size-3.5 shrink-0 transition-transform duration-150"
          style={{ transform: open ? "rotate(90deg)" : "rotate(0deg)" }}
          aria-hidden="true"
        />
        {label} — last {hours}h ({count} {count === 1 ? "message" : "messages"})
        {contextWindow?.truncated && " · truncated"}
      </button>
      {open && messages.length > 0 && (
        <div className="mt-2 divide-y border rounded-lg px-3 bg-muted/20">
          {messages.map((msg, i) => (
            <MessageCard key={i} message={msg} compact />
          ))}
        </div>
      )}
    </div>
  );
}

export function SlackContextPreview({ slackContext }: SlackContextPreviewProps) {
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
    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
      {channel && <ChannelHeader channel={channel} />}

      {context_scope === "self_dm" && (
        <div className="flex items-center justify-center rounded-lg bg-muted/50 px-3 py-3 text-sm text-muted-foreground">
          Note to self
        </div>
      )}

      {context_scope === "first_contact_dm" && (
        <div className="flex items-center justify-center rounded-lg bg-muted/50 px-3 py-3 text-sm text-muted-foreground">
          No prior messages with this user
        </div>
      )}

      {context_scope === "metadata_only" && (
        <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-3">
          <span className="text-sm text-muted-foreground">
            Context unavailable
          </span>
          {channel && (
            <a
              href={channel.permalink}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            >
              <ExternalLink className="size-3" aria-hidden="true" />
              Open in Slack
            </a>
          )}
        </div>
      )}

      {recipient &&
        (context_scope === "recent_dm" ||
          context_scope === "first_contact_dm") && (
          <RecipientCard user={recipient} />
        )}

      {target_message && (
        <div className="mb-3">
          <p className="mb-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wide">
            Target message
          </p>
          <div className="rounded-lg border bg-muted/20 px-3">
            <MessageCard message={target_message} />
          </div>
        </div>
      )}

      {context_scope === "thread" && thread && (
        <ThreadSection thread={thread} />
      )}

      {(context_scope === "recent_channel" || context_scope === "recent_dm") &&
        recent_messages &&
        recent_messages.length > 0 && (
          <RecentActivitySection
            messages={recent_messages}
            contextWindow={context_window}
            label={
              context_scope === "recent_dm" ? "DM history" : "Channel activity"
            }
          />
        )}
    </div>
  );
}
