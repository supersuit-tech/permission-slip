/**
 * Converts Slack mrkdwn to plain text for mobile display.
 *
 * The backend resolves `<@U123>` mentions and `<#C123|name>` channels
 * server-side before the payload reaches the client. This helper strips
 * remaining mrkdwn formatting syntax so the text reads naturally on mobile.
 */
export function slackMrkdwnToPlaintext(text: string): string {
  return text
    // Code blocks: ```text``` → text (before inline code)
    .replace(/```([\s\S]*?)```/g, (_, code: string) => code.trim())
    // Named links / channel refs: <URL|display> → display
    .replace(/<[^|>]+\|([^>]+)>/g, "$1")
    // Special broadcasts: <!here>, <!channel>, <!everyone>
    .replace(/<!here>/g, "@here")
    .replace(/<!channel>/g, "@channel")
    .replace(/<!everyone>/g, "@everyone")
    // Remaining angle-bracket refs (URLs, unresolved user IDs): <X> → X
    .replace(/<([^>]+)>/g, "$1")
    // Bold: *text* → text
    .replace(/\*([^*\n]+)\*/g, "$1")
    // Italic: _text_ → text
    .replace(/_([^_\n]+)_/g, "$1")
    // Strikethrough: ~text~ → text
    .replace(/~([^~\n]+)~/g, "$1")
    // Inline code: `text` → text
    .replace(/`([^`\n]+)`/g, "$1")
    // Blockquotes: strip leading ">" (with optional space)
    .replace(/^> ?/gm, "")
    .trim();
}
