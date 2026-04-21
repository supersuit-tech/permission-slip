import { describe, it, expect } from "vitest";
import { slackMrkdwnToHtml } from "../slackMrkdwn";

describe("slackMrkdwnToHtml", () => {
  it("converts bold *text*", () => {
    expect(slackMrkdwnToHtml("*hello*")).toContain("<strong>hello</strong>");
  });

  it("converts italic _text_", () => {
    expect(slackMrkdwnToHtml("_hello_")).toContain("<em>hello</em>");
  });

  it("converts strikethrough ~text~", () => {
    expect(slackMrkdwnToHtml("~hello~")).toContain("<s>hello</s>");
  });

  it("converts inline code `text`", () => {
    expect(slackMrkdwnToHtml("`code`")).toContain("<code>code</code>");
  });

  it("converts triple-backtick code blocks", () => {
    const result = slackMrkdwnToHtml("```const x = 1;```");
    expect(result).toContain("<pre>");
    expect(result).toContain("<code>");
    expect(result).toContain("const x = 1;");
  });

  it("converts Slack link <url>", () => {
    const result = slackMrkdwnToHtml("<https://example.com>");
    expect(result).toContain('href="https://example.com"');
    expect(result).toContain('rel="noopener noreferrer"');
    expect(result).toContain('target="_blank"');
  });

  it("converts Slack link <url|label>", () => {
    const result = slackMrkdwnToHtml("<https://example.com|Click here>");
    expect(result).toContain('href="https://example.com"');
    expect(result).toContain("Click here");
  });

  it("links have rel=noopener noreferrer", () => {
    const result = slackMrkdwnToHtml("<https://example.com|link>");
    expect(result).toContain('rel="noopener noreferrer"');
  });

  it("converts blockquote lines starting with >", () => {
    const result = slackMrkdwnToHtml(">quoted text");
    expect(result).toContain("<blockquote>");
    expect(result).toContain("quoted text");
  });

  it("converts newlines to <br>", () => {
    const result = slackMrkdwnToHtml("line1\nline2");
    expect(result).toContain("<br>");
  });

  it("sanitizes script tags", () => {
    const result = slackMrkdwnToHtml("<script>alert(1)</script>hello");
    expect(result).not.toContain("<script>");
    expect(result).toContain("hello");
  });

  it("sanitizes event handler attributes", () => {
    const result = slackMrkdwnToHtml(
      "<a href='https://x.com' onclick='alert(1)'>x</a>",
    );
    expect(result).not.toContain("onclick");
  });

  it("does not double-process content inside code blocks", () => {
    const result = slackMrkdwnToHtml("```*not bold*```");
    expect(result).toContain("*not bold*");
    expect(result).not.toContain("<strong>");
  });

  it("does not convert Slack link syntax inside fenced code blocks", () => {
    const result = slackMrkdwnToHtml("```<https://example.com|link>```");
    expect(result).not.toContain("<a ");
    expect(result).toContain("https://example.com");
  });

  it("does not convert Slack link syntax inside inline code", () => {
    const result = slackMrkdwnToHtml("`<https://example.com>`");
    expect(result).not.toContain("<a ");
  });

  it("handles already-resolved mentions as plain text", () => {
    const result = slackMrkdwnToHtml("@alice and #general");
    expect(result).toContain("@alice");
    expect(result).toContain("#general");
  });

  it("returns empty string for empty input", () => {
    expect(slackMrkdwnToHtml("")).toBe("");
  });
});
