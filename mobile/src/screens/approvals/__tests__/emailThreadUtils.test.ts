import {
  isEmailReplyAction,
  parseEmailThreadDetails,
} from "../emailThreadUtils";

describe("emailThreadUtils", () => {
  it("identifies google and microsoft send_email_reply actions", () => {
    expect(isEmailReplyAction("google.send_email_reply")).toBe(true);
    expect(isEmailReplyAction("microsoft.send_email_reply")).toBe(true);
    expect(isEmailReplyAction("email.send")).toBe(false);
  });

  it("parseEmailThreadDetails returns null when email_thread key is absent", () => {
    expect(parseEmailThreadDetails({ foo: 1 })).toBeNull();
    expect(parseEmailThreadDetails(undefined)).toBeNull();
  });

  it("parseEmailThreadDetails coerces partial payloads", () => {
    const parsed = parseEmailThreadDetails({
      email_thread: {
        subject: "Hello",
        messages: [
          {
            from: "a@b.com",
            attachments: [{ filename: "f.txt", size_bytes: 10 }],
          },
        ],
      },
    });
    expect(parsed).not.toBeNull();
    expect(parsed!.subject).toBe("Hello");
    expect(parsed!.messages).toHaveLength(1);
    const m = parsed!.messages[0]!;
    expect(m.from).toBe("a@b.com");
    expect(m.to).toEqual([]);
    expect(m.cc).toEqual([]);
    expect(m.truncated).toBe(false);
    expect(m.attachments?.[0]?.filename).toBe("f.txt");
  });
});
