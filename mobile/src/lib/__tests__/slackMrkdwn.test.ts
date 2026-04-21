import { slackMrkdwnToPlaintext } from "../slackMrkdwn";

describe("slackMrkdwnToPlaintext", () => {
  it("returns plain text unchanged", () => {
    expect(slackMrkdwnToPlaintext("Hello world")).toBe("Hello world");
  });

  it("strips bold syntax", () => {
    expect(slackMrkdwnToPlaintext("This is *bold* text")).toBe("This is bold text");
  });

  it("strips italic syntax", () => {
    expect(slackMrkdwnToPlaintext("This is _italic_ text")).toBe("This is italic text");
  });

  it("strips strikethrough syntax", () => {
    expect(slackMrkdwnToPlaintext("This is ~struck~ text")).toBe("This is struck text");
  });

  it("strips inline code backticks", () => {
    expect(slackMrkdwnToPlaintext("Use `npm install`")).toBe("Use npm install");
  });

  it("strips code block backticks", () => {
    expect(slackMrkdwnToPlaintext("```const x = 1;```")).toBe("const x = 1;");
  });

  it("strips blockquote prefix", () => {
    expect(slackMrkdwnToPlaintext("> quoted text")).toBe("quoted text");
  });

  it("converts named link to display text", () => {
    expect(slackMrkdwnToPlaintext("<https://example.com|click here>")).toBe("click here");
  });

  it("unwraps plain URL angle brackets", () => {
    expect(slackMrkdwnToPlaintext("<https://example.com>")).toBe("https://example.com");
  });

  it("converts <!here> to @here", () => {
    expect(slackMrkdwnToPlaintext("<!here> please review")).toBe("@here please review");
  });

  it("converts <!channel> to @channel", () => {
    expect(slackMrkdwnToPlaintext("Heads up <!channel>")).toBe("Heads up @channel");
  });

  it("converts <!everyone> to @everyone", () => {
    expect(slackMrkdwnToPlaintext("<!everyone> announcement")).toBe("@everyone announcement");
  });

  it("handles already-resolved mentions as plain text", () => {
    // Backend resolves <@U123> → @alice before the payload reaches the client
    expect(slackMrkdwnToPlaintext("Hey @alice, check this")).toBe(
      "Hey @alice, check this",
    );
  });

  it("handles multiline text", () => {
    const input = "*First* line\n> Quoted\n_Second_ line";
    expect(slackMrkdwnToPlaintext(input)).toBe("First line\nQuoted\nSecond line");
  });

  it("handles empty string", () => {
    expect(slackMrkdwnToPlaintext("")).toBe("");
  });
});
