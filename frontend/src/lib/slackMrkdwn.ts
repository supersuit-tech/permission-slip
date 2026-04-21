import DOMPurify from "dompurify";

const SANITIZE_CONFIG = {
  ALLOWED_TAGS: ["br", "strong", "em", "s", "code", "pre", "blockquote", "a"],
  ALLOWED_ATTR: ["href", "rel", "target"],
};

/**
 * Converts Slack mrkdwn to sanitized HTML.
 *
 * Assumes the backend has already resolved <@U123> → @alice and
 * <#C123|name> → #name. Handles: *bold*, _italic_, ~strike~, `code`,
 * ```code blocks```, >blockquotes, <url|label> links, and \n → <br>.
 *
 * Code spans are extracted first and protected from mrkdwn processing via
 * null-byte placeholders, then restored before sanitization.
 *
 * Output must be rendered via dangerouslySetInnerHTML — DOMPurify is applied
 * as the final step.
 */
export function slackMrkdwnToHtml(text: string): string {
  const protected_: string[] = [];

  let html = text;

  // Slack link format: <url> and <url|label>
  html = html.replace(
    /<(https?:\/\/[^|>\s]+)(?:\|([^>]*))?>/g,
    (_, url: string, label?: string) =>
      `<a href="${url}" target="_blank" rel="noopener noreferrer">${label ?? url}</a>`,
  );

  // Extract code blocks (triple backtick) — protect from further formatting
  html = html.replace(/```([\s\S]*?)```/g, (_, code: string) => {
    const idx = protected_.length;
    protected_.push(`<pre><code>${code}</code></pre>`);
    return `\x00P${idx}\x00`;
  });

  // Extract inline code — protect from further formatting
  html = html.replace(/`([^`\n]+)`/g, (_, code: string) => {
    const idx = protected_.length;
    protected_.push(`<code>${code}</code>`);
    return `\x00P${idx}\x00`;
  });

  // Bold *text*
  html = html.replace(
    /\*([^*\n]+)\*/g,
    (_, t: string) => `<strong>${t}</strong>`,
  );

  // Italic _text_
  html = html.replace(/_([^_\n]+)_/g, (_, t: string) => `<em>${t}</em>`);

  // Strikethrough ~text~
  html = html.replace(/~([^~\n]+)~/g, (_, t: string) => `<s>${t}</s>`);

  // Blockquotes — lines starting with >
  html = html.replace(
    /^>(.+)$/gm,
    (_, content: string) => `<blockquote>${content}</blockquote>`,
  );

  // Restore protected code spans
  html = html.replace(
    /\x00P(\d+)\x00/g,
    (_, idx: string) => protected_[Number(idx)] ?? "",
  );

  // Newlines → <br>
  html = html.replace(/\n/g, "<br>");

  return DOMPurify.sanitize(html, SANITIZE_CONFIG);
}
