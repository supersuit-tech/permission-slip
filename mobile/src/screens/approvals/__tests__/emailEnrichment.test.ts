import { emailDetailsUnavailable } from "../emailEnrichment";

describe("emailDetailsUnavailable", () => {
  it("returns true for a Proton Mail email action with no resource details", () => {
    expect(emailDetailsUnavailable("protonmail.archive_email", undefined)).toBe(
      true,
    );
    expect(emailDetailsUnavailable("protonmail.read_email", null)).toBe(true);
  });

  it("returns true when resource details lack email metadata", () => {
    expect(
      emailDetailsUnavailable("protonmail.delete", { unrelated: "value" }),
    ).toBe(true);
  });

  it("returns false when single-message enrichment is present", () => {
    expect(
      emailDetailsUnavailable("protonmail.read_email", {
        subject: "Hello",
        from: ["a@example.com"],
      }),
    ).toBe(false);
  });

  it("returns false when batch enrichment is present", () => {
    expect(
      emailDetailsUnavailable("protonmail.archive_email", {
        messages: { "231": { subject: "Hello" } },
      }),
    ).toBe(false);
  });

  it("returns false when reply enrichment is present", () => {
    expect(
      emailDetailsUnavailable("protonmail.reply_email", {
        in_reply_to: { subject: "Hello" },
      }),
    ).toBe(false);
  });

  it("returns false for actions that are not email-detail actions", () => {
    expect(emailDetailsUnavailable("protonmail.send_email", undefined)).toBe(
      false,
    );
    expect(emailDetailsUnavailable("github.create_issue", undefined)).toBe(
      false,
    );
  });
});
