import {
  isProtonReplyAction,
  parseProtonInReplyTo,
} from "../protonInReplyToUtils";

describe("protonInReplyToUtils", () => {
  it("identifies protonmail.reply_email", () => {
    expect(isProtonReplyAction("protonmail.reply_email")).toBe(true);
    expect(isProtonReplyAction("protonmail.send_email")).toBe(false);
  });

  it("parses in_reply_to metadata", () => {
    const parsed = parseProtonInReplyTo({
      in_reply_to: {
        subject: "Weekly Update",
        from: ["alice@example.com"],
        to: ["me@proton.me"],
        date: "2026-01-15T10:00:00Z",
      },
    });
    expect(parsed?.subject).toBe("Weekly Update");
    expect(parsed?.from).toEqual(["alice@example.com"]);
  });

  it("returns null for invalid payloads", () => {
    expect(parseProtonInReplyTo(undefined)).toBeNull();
    expect(parseProtonInReplyTo({ in_reply_to: "bad" })).toBeNull();
  });
});
