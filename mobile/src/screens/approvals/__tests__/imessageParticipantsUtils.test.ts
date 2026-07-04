import {
  formatImessageParticipants,
  parseImessageParticipants,
} from "../imessageParticipantsUtils";

describe("parseImessageParticipants", () => {
  it("returns handles when participants is a non-empty string array", () => {
    expect(
      parseImessageParticipants({
        participants: ["+15551234567", "ben@example.com"],
      }),
    ).toEqual(["+15551234567", "ben@example.com"]);
  });

  it("returns null when participants is absent", () => {
    expect(parseImessageParticipants({ chat_name: "with Jane" })).toBeNull();
  });

  it("returns null when participants is empty", () => {
    expect(parseImessageParticipants({ participants: [] })).toBeNull();
  });
});

describe("formatImessageParticipants", () => {
  it("joins multiple handles with commas", () => {
    expect(
      formatImessageParticipants(["+15551111111", "+15552222222"]),
    ).toBe("+15551111111, +15552222222");
  });
});
