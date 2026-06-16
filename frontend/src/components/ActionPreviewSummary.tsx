/**
 * Action Preview Summary
 *
 * Converts action parameters into human-readable summaries for approvers.
 * Supports two rendering modes from a single formatter definition:
 *
 * - **Rich** (ActionPreviewSummary component): JSX with highlighted values
 * - **Plain** (buildSummary function): plain text for list rows and tooltips
 *
 * ## Adding a new action type formatter
 *
 * 1. Add an entry to ACTION_DESCRIBERS keyed by the action type string
 *    (e.g., "stripe.create_charge").
 * 2. The function receives the action parameters and returns SummaryPart[]
 *    or null (null falls through to the generic summary).
 * 3. Use text() for prose and val() for parameter values that should be
 *    highlighted in rich mode and quoted in plain mode.
 *
 * Example:
 * ```ts
 * "stripe.create_charge": (params) => {
 *   const amount = params.amount;
 *   if (amount == null) return null;
 *   return [text("Charge "), val(`$${amount}`)];
 * },
 * ```
 */
import type { ReactNode } from "react";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { renderTemplate, type SummaryPart } from "@/lib/displayTemplate";
import {
  formatDateTime,
  formatHighlightValue,
  humanizeKey,
  truncate,
} from "@/lib/formatValues";
import { emailDetailsUnavailable } from "@/lib/emailEnrichment";

interface ActionPreviewSummaryProps {
  /** Action type identifier, e.g. "github.create_issue". */
  actionType: string;
  /** Actual parameter values from the approval request. */
  parameters: Record<string, unknown>;
  /** Schema describing the parameters (from connector action). */
  schema: ParametersSchema | null;
  /** Human-readable action name from the connector, e.g. "Create Issue". */
  actionName: string | null;
  /** Display template from the connector manifest, e.g. "Send email to {{to}}". */
  displayTemplate?: string | null;
  /** Resource details resolved at approval creation time. */
  resourceDetails?: Record<string, unknown> | null;
}

// ---------------------------------------------------------------------------
// Structured summary parts — single definition, dual rendering
// ---------------------------------------------------------------------------

// Re-use the SummaryPart type from displayTemplate for consistency.
// Local helpers kept for the ACTION_DESCRIBERS registry.

function text(t: string): SummaryPart {
  return { kind: "text", text: t };
}

function val(t: string): SummaryPart {
  return { kind: "value", text: t };
}

/** Inline highlight for parameter values within the summary. */
function ValSpan({ children }: { children: ReactNode }) {
  return (
    <span className="bg-muted text-foreground inline rounded px-1 py-0.5 font-medium box-decoration-clone break-all">
      {children}
    </span>
  );
}

/** Render structured parts as rich JSX with highlighted values. */
function renderRich(parts: SummaryPart[]): ReactNode {
  return parts.map((part, i) =>
    part.kind === "value" ? (
      <ValSpan key={i}>{part.text}</ValSpan>
    ) : (
      part.text
    ),
  );
}

/** Render structured parts as a plain text string with quoted values. */
function renderPlain(parts: SummaryPart[]): string {
  return parts
    .map((part) =>
      part.kind === "value" ? `\u201C${part.text}\u201D` : part.text,
    )
    .join("");
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * Renders a concise, human-readable summary of an action for approvers.
 * Uses action-type-specific formatters when available, falling back to a
 * schema-driven or generic summary. Parameter values are visually highlighted
 * for quick scanning.
 */
export function ActionPreviewSummary({
  actionType,
  parameters,
  schema,
  actionName,
  displayTemplate,
  resourceDetails,
}: ActionPreviewSummaryProps) {
  const parts = buildParts(actionType, parameters, schema, actionName, displayTemplate, resourceDetails);

  return (
    <>
      <p className="text-sm leading-loose break-words" data-testid="action-preview-summary">
        {renderRich(parts)}
      </p>
      {emailDetailsUnavailable(actionType, resourceDetails) && (
        <p
          className="text-muted-foreground mt-1 text-xs italic"
          data-testid="email-details-unavailable"
        >
          Email details unavailable
        </p>
      )}
    </>
  );
}

/**
 * Builds a human-readable plain-text summary string.
 * Used by list rows and tooltips where rich rendering isn't needed.
 */
export function buildSummary(
  actionType: string,
  parameters: Record<string, unknown>,
  schema: ParametersSchema | null,
  actionName: string | null,
  displayTemplate?: string | null,
  resourceDetails?: Record<string, unknown> | null,
): string {
  return renderPlain(buildParts(actionType, parameters, schema, actionName, displayTemplate, resourceDetails));
}

/**
 * Core formatter: returns structured parts for an action.
 * Priority: ACTION_DESCRIBERS → template → generic fallback.
 */
function buildParts(
  actionType: string,
  parameters: Record<string, unknown>,
  schema: ParametersSchema | null,
  actionName: string | null,
  displayTemplate?: string | null,
  resourceDetails?: Record<string, unknown> | null,
): SummaryPart[] {
  // 1. Try action-specific describer (with resource details).
  const formatter = ACTION_DESCRIBERS[actionType];
  if (formatter) {
    const result = formatter(parameters, resourceDetails ?? undefined);
    if (result) return result;
  }

  // 2. Try display template from manifest.
  // Merge resourceDetails into the lookup so templates can resolve human-readable
  // names (e.g. {{channel_name}} → "#general") instead of raw IDs (#862).
  if (displayTemplate) {
    const lookup = resourceDetails
      ? { ...parameters, ...resourceDetails }
      : parameters;
    const templateParts = renderTemplate(displayTemplate, lookup);
    if (templateParts) return templateParts;
  }

  // 3. Fall back to generic.
  return buildGenericParts(actionType, parameters, schema, actionName, resourceDetails);
}

// ---------------------------------------------------------------------------
// Action-type-specific describers (single definition per action type)
// ---------------------------------------------------------------------------

/**
 * Returns structured summary parts for a specific action type,
 * or null to fall through to the generic schema-based summary.
 */
type ActionDescriber = (params: Record<string, unknown>, resourceDetails?: Record<string, unknown>) => SummaryPart[] | null;

/** Registry of action-type-specific formatters. Keyed by action_type string. */
const ACTION_DESCRIBERS: Record<string, ActionDescriber> = {
  "github.create_issue": (params) => {
    const owner = strVal(params.owner);
    const repo = strVal(params.repo);
    const title = strVal(params.title);
    if (!title) return null;
    const repoRef = owner && repo ? `${owner}/${repo}` : repo;
    const parts: SummaryPart[] = [text("Create issue "), val(title)];
    if (repoRef) parts.push(text(" in "), val(repoRef));
    return parts;
  },

  "github.merge_pr": (params) => {
    const owner = strVal(params.owner);
    const repo = strVal(params.repo);
    const prNumber = params.pull_number;
    if (prNumber == null) return null;
    const repoRef = owner && repo ? `${owner}/${repo}` : repo;
    const method = strVal(params.merge_method);
    const parts: SummaryPart[] = [text("Merge PR "), val(`#${String(prNumber)}`)];
    if (repoRef) parts.push(text(" in "), val(repoRef));
    if (method && method !== "merge") parts.push(text(` (${method})`));
    return parts;
  },

  "slack.send_message": (params, rd) => {
    const channel = strVal(rd?.channel_name) ?? strVal(params.channel);
    if (!channel) return null;
    const message = strVal(params.message);
    const parts: SummaryPart[] = [text("Send message to "), val(channel)];
    if (message) parts.push(text(` \u2014 ${truncate(message, 80)}`));
    return parts;
  },

  "slack.send_dm": (params, rd) => {
    const user = strVal(rd?.user_name) ?? strVal(params.user_id);
    if (!user) return null;
    const message = strVal(params.message);
    const parts: SummaryPart[] = [text("Send DM to "), val(user)];
    if (message) parts.push(text(` \u2014 ${truncate(message, 80)}`));
    return parts;
  },

  "slack.create_channel": (params) => {
    const name = strVal(params.name);
    if (!name) return null;
    const isPrivate = params.is_private === true;
    return [
      text(`Create ${isPrivate ? "private " : ""}channel `),
      val(`#${name}`),
    ];
  },

  "email.send": (params) => {
    const to = formatRecipients(params.to);
    const subject = strVal(params.subject);
    if (!to) return null;
    const parts: SummaryPart[] = [text("Send email to "), val(to)];
    if (subject) parts.push(text(" with subject "), val(subject));
    return parts;
  },

  "payment.charge": (params) => {
    const amount = params.amount;
    const currency = strVal(params.currency) ?? "USD";
    if (amount == null) return null;
    const formatted = formatCurrency(amount, currency);
    const desc = strVal(params.description);
    const parts: SummaryPart[] = [text("Charge "), val(formatted)];
    if (desc) parts.push(text(" for "), val(desc));
    return parts;
  },

  // ── Google Calendar ─────────────────────────────────────────────

  "google.delete_calendar_event": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    const parts: SummaryPart[] = [text("Delete event "), val(title)];
    const startTime = strVal(rd?.start_time);
    if (startTime) parts.push(text(" on "), val(formatDateTime(startTime)));
    return parts;
  },

  "google.update_calendar_event": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    const parts: SummaryPart[] = [text("Update event "), val(title)];
    const startTime = strVal(rd?.start_time);
    if (startTime) parts.push(text(" on "), val(formatDateTime(startTime)));
    return parts;
  },

  // ── Google Drive ────────────────────────────────────────────────

  "google.delete_drive_file": (_params, rd) => {
    const name = strVal(rd?.file_name);
    if (!name) return null;
    return [text("Delete file "), val(name)];
  },

  "google.get_drive_file": (_params, rd) => {
    const name = strVal(rd?.file_name);
    if (!name) return null;
    return [text("Get file "), val(name)];
  },

  // ── Google Docs ─────────────────────────────────────────────────

  "google.get_document": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    return [text("Get document "), val(title)];
  },

  "google.update_document": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    return [text("Update document "), val(title)];
  },

  // ── Google Sheets ───────────────────────────────────────────────

  "google.sheets_read_range": (params, rd) => {
    const title = strVal(rd?.title);
    const range = strVal(rd?.range) ?? strVal(params.range);
    if (!title) return null;
    const parts: SummaryPart[] = [text("Read "), val(title)];
    if (range) parts.push(text(" range "), val(range));
    return parts;
  },

  "google.sheets_write_range": (params, rd) => {
    const title = strVal(rd?.title);
    const range = strVal(rd?.range) ?? strVal(params.range);
    if (!title) return null;
    const parts: SummaryPart[] = [text("Write to "), val(title)];
    if (range) parts.push(text(" range "), val(range));
    return parts;
  },

  "google.sheets_append_rows": (params, rd) => {
    const title = strVal(rd?.title);
    const range = strVal(rd?.range) ?? strVal(params.range);
    if (!title) return null;
    const parts: SummaryPart[] = [text("Append rows to "), val(title)];
    if (range) parts.push(text(" range "), val(range));
    return parts;
  },

  "google.sheets_list_sheets": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    return [text("List sheets in "), val(title)];
  },

  // ── Google Slides ───────────────────────────────────────────────

  "google.get_presentation": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    return [text("Get presentation "), val(title)];
  },

  "google.add_slide": (_params, rd) => {
    const title = strVal(rd?.title);
    if (!title) return null;
    return [text("Add slide to "), val(title)];
  },

  // ── Gmail ───────────────────────────────────────────────────────

  "google.read_email": (_params, rd) => {
    const subject = strVal(rd?.subject);
    if (!subject) return null;
    const parts: SummaryPart[] = [text("Read email "), val(subject)];
    const from = strVal(rd?.from);
    if (from) parts.push(text(" from "), val(from));
    return parts;
  },

  "google.send_email_reply": (_params, rd) => {
    const subject = strVal(rd?.subject);
    if (!subject) return null;
    return [text("Reply to "), val(subject)];
  },

  "google.archive_email": (_params, rd) => {
    const subject = strVal(rd?.subject);
    if (!subject) return null;
    const parts: SummaryPart[] = [text("Archive email "), val(subject)];
    const from = strVal(rd?.from);
    if (from) parts.push(text(" from "), val(from));
    return parts;
  },

  // ── Proton Mail ─────────────────────────────────────────────────

  "protonmail.send_email": (params) => {
    const to = formatRecipients(params.to);
    const subject = strVal(params.subject);
    if (!to) return null;
    const parts: SummaryPart[] = [text("Send email to "), val(to)];
    if (subject) parts.push(text(" — "), val(subject));
    return parts;
  },

  "protonmail.read_inbox": (params) => {
    const folder = strVal(params.folder) ?? "INBOX";
    const limit =
      typeof params.limit === "number" && Number.isFinite(params.limit)
        ? params.limit
        : 10;
    const parts: SummaryPart[] = [
      text("Read "),
      val(String(limit)),
      text(" most recent in "),
      val(folder),
    ];
    if (params.unread_only === true) {
      parts.push(text(" (unread only)"));
    }
    return parts;
  },

  "protonmail.search_emails": (params) => {
    const folder = strVal(params.folder) ?? "INBOX";
    const parts: SummaryPart[] = [text("Search "), val(folder)];
    const filters: string[] = [];
    const subject = strVal(params.subject);
    if (subject) filters.push(`'${subject}'`);
    const from = strVal(params.from);
    if (from) filters.push(`from ${from}`);
    const since = strVal(params.since);
    if (since) filters.push(`since ${since}`);
    const before = strVal(params.before);
    if (before) filters.push(`before ${before}`);
    if (filters.length > 0) {
      parts.push(text(" for "), val(filters.join(", ")));
    }
    return parts;
  },

  "protonmail.read_email": (_params, rd) => {
    if (!rd) return null;
    const summary = protonmailEmailSummaryParts(rd);
    if (!summary) return null;
    return [text("Read email "), ...summary];
  },

  "protonmail.archive_email": (_params, rd) => {
    if (!rd) return null;
    const batch = protonmailBatchArchiveSummaryParts(rd);
    if (batch) return batch;
    const summary = protonmailEmailSummaryParts(rd);
    if (!summary) return null;
    return [text("Archive email "), ...summary];
  },

  "protonmail.reply_email": (_params, rd) => {
    if (!rd) return null;
    const meta = protonmailInReplyToRecord(rd);
    if (!meta) return null;
    const summary = protonmailEmailSummaryParts(meta);
    if (!summary) return null;
    return [text("Reply to "), ...summary];
  },

  "protonmail.list_folders": () => [text("List mailbox folders")],

  "protonmail.mark_read": (_params, rd) => protonmailUIDActionSummary("Mark as read", rd),
  "protonmail.mark_unread": (_params, rd) => protonmailUIDActionSummary("Mark as unread", rd),
  "protonmail.flag": (_params, rd) => protonmailUIDActionSummary("Flag email", rd),
  "protonmail.unflag": (_params, rd) => protonmailUIDActionSummary("Unflag email", rd),

  "protonmail.move_to_folder": (params, rd) => {
    const target = strVal(params.target_folder);
    const summary = protonmailUIDActionSummary("Move email", rd, target ? ` to ${target}` : null);
    return summary;
  },

  "protonmail.delete": (_params, rd) => protonmailUIDActionSummary("Delete email", rd),
};

// ---------------------------------------------------------------------------
// Generic / fallback summary builder
// ---------------------------------------------------------------------------

function buildGenericParts(
  actionType: string,
  parameters: Record<string, unknown>,
  schema: ParametersSchema | null,
  actionName: string | null,
  resourceDetails?: Record<string, unknown> | null,
): SummaryPart[] {
  const label = actionName ?? humanizeActionType(actionType);

  // Merge resourceDetails so resolved names (e.g. channel_name) appear in
  // generic highlights instead of raw IDs (#862).
  // Skip raw params whose resolved counterpart exists in resourceDetails
  // (e.g. skip "channel" when "channel_name" is present) to avoid showing both.
  const filtered = resourceDetails
    ? Object.fromEntries(
        Object.entries(parameters).filter(
          ([key]) => !(resourceDetails[`${key}_name`] != null),
        ),
      )
    : parameters;
  const merged = resourceDetails
    ? { ...filtered, ...resourceDetails }
    : parameters;
  const highlights = pickHighlights(merged, schema);
  if (highlights.length === 0) return [text(label)];

  const parts: SummaryPart[] = [text(`${label}: `)];
  for (let i = 0; i < highlights.length; i++) {
    const h = highlights[i];
    if (!h) continue;
    if (i > 0) parts.push(text(", "));
    parts.push(text(`${h.label} `), val(h.displayVal));
  }
  return parts;
}

interface Highlight {
  label: string;
  displayVal: string;
}

/**
 * Selects the most informative parameter key-value pairs for a summary.
 * Prefers required parameters and string/number values, limited to 3.
 */
function pickHighlights(
  parameters: Record<string, unknown>,
  schema: ParametersSchema | null,
): Highlight[] {
  const entries = Object.entries(parameters);
  if (entries.length === 0) return [];

  const requiredSet = new Set(schema?.required ?? []);
  const properties = schema?.properties;

  const schemaOrder =
    properties != null
      ? Object.keys(properties).reduce<Record<string, number>>(
          (acc, key, index) => {
            acc[key] = index;
            return acc;
          },
          {},
        )
      : null;

  // Sort: required first, then by schema property order, then alphabetical
  const sorted = [...entries].sort(([a], [b]) => {
    const aReq = requiredSet.has(a) ? 0 : 1;
    const bReq = requiredSet.has(b) ? 0 : 1;
    if (aReq !== bReq) return aReq - bReq;

    const aIndex = schemaOrder?.[a];
    const bIndex = schemaOrder?.[b];
    if (aIndex != null && bIndex != null && aIndex !== bIndex) {
      return aIndex - bIndex;
    }
    if (aIndex != null && bIndex == null) return -1;
    if (aIndex == null && bIndex != null) return 1;

    return a.localeCompare(b);
  });

  const highlights: Highlight[] = [];
  for (const [key, value] of sorted) {
    if (highlights.length >= 3) break;
    if (value == null) continue;
    const displayVal = formatHighlightValue(value);
    if (!displayVal) continue;
    const prop = properties?.[key];
    // Use x-ui label (short UI label) or humanized key — never the full schema
    // description, which is developer-facing help text and far too verbose for
    // inline summaries (see #862).
    const uiLabel = prop?.["x-ui"]?.label;
    const label = uiLabel ?? humanizeKey(key);
    highlights.push({ label, displayVal });
  }

  return highlights;
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

function strVal(v: unknown): string | null {
  if (typeof v === "string" && v.length > 0) return v;
  return null;
}

function formatRecipients(v: unknown): string | null {
  if (typeof v === "string") return v;
  if (Array.isArray(v)) {
    const strs = v.filter((x): x is string => typeof x === "string");
    if (strs.length === 0) return null;
    if (strs.length <= 3) return strs.join(", ");
    return `${strs[0] ?? ""}, ${strs[1] ?? ""}, and ${strs.length - 2} more`;
  }
  return null;
}

function protonmailInReplyToRecord(
  rd: Record<string, unknown>,
): Record<string, unknown> | null {
  const raw = rd.in_reply_to;
  if (raw != null && typeof raw === "object" && !Array.isArray(raw)) {
    return raw as Record<string, unknown>;
  }
  return null;
}

function protonmailUIDActionSummary(
  prefix: string,
  rd: Record<string, unknown> | null | undefined,
  suffix?: string | null,
): SummaryPart[] | null {
  if (!rd) {
    if (suffix) return [text(prefix), text(suffix)];
    return [text(prefix)];
  }
  const batch = protonmailBatchEmailSummaryParts(rd);
  if (batch) {
    const parts: SummaryPart[] = [text(prefix), ...batch];
    if (suffix) parts.push(text(suffix));
    return parts;
  }
  const summary = protonmailEmailSummaryParts(rd);
  if (!summary) {
    if (suffix) return [text(prefix), text(suffix)];
    return [text(prefix)];
  }
  const parts: SummaryPart[] = [text(`${prefix} `), ...summary];
  if (suffix) parts.push(text(suffix));
  return parts;
}

// Batch summaries show only the count — each email's subject/from/to/date is
// rendered individually by ProtonBatchEmailsPreview's collapsible rows.
function protonmailBatchEmailSummaryParts(
  rd: Record<string, unknown>,
): SummaryPart[] | null {
  const rawMessages = rd.messages;
  if (rawMessages == null || typeof rawMessages !== "object" || Array.isArray(rawMessages)) {
    return null;
  }
  const entries = Object.entries(rawMessages as Record<string, unknown>);
  if (entries.length <= 1) return null;

  return [text(" "), val(String(entries.length)), text(" emails")];
}

function protonmailEmailSummaryParts(
  meta: Record<string, unknown>,
): SummaryPart[] | null {
  const subject = strVal(meta.subject);
  if (!subject) return null;
  const parts: SummaryPart[] = [val(subject)];
  const from = formatRecipients(meta.from);
  if (from) parts.push(text(" from "), val(from));
  return parts;
}

function protonmailBatchArchiveSummaryParts(
  rd: Record<string, unknown>,
): SummaryPart[] | null {
  const batch = protonmailBatchEmailSummaryParts(rd);
  if (!batch) return null;
  return [text("Archive"), ...batch];
}

function formatCurrency(amount: unknown, currency: string): string {
  if (typeof amount !== "number" || !Number.isFinite(amount))
    return String(amount);
  const major = amount / 100;
  const symbol = CURRENCY_SYMBOLS[currency.toUpperCase()] ?? currency + " ";
  return `${symbol}${major.toFixed(2)}`;
}

const CURRENCY_SYMBOLS: Record<string, string> = {
  USD: "$",
  EUR: "\u20AC",
  GBP: "\u00A3",
  JPY: "\u00A5",
};

/**
 * Converts an action type string into a readable label.
 * Extracts the operation portion (last segment after the final dot),
 * replaces underscores with spaces, and capitalizes the first letter.
 *
 * This only runs as a fallback when no actionName is provided and
 * no specific ACTION_DESCRIBER matches. The connector prefix is
 * omitted because the UI already displays the full action type
 * separately, and naive capitalization of connector names (e.g.,
 * "Github" instead of "GitHub") would look wrong.
 *
 * Examples:
 * - "github.create_issue"  → "Create issue"
 * - "slack.send_message"   → "Send message"
 * - "com.acme.deploy.prod" → "Prod"
 */
function humanizeActionType(actionType: string): string {
  const parts = actionType.split(".");
  const operation = parts.length >= 2 ? parts[parts.length - 1] ?? actionType : actionType;
  const words = operation.replace(/_/g, " ");
  return words.charAt(0).toUpperCase() + words.slice(1);
}

