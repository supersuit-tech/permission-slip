import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ImessageParticipantsRow } from "../ImessageParticipantsRow";
import {
  formatImessageParticipants,
  parseImessageParticipants,
} from "@/lib/imessageParticipantsUtils";

describe("parseImessageParticipants", () => {
  it("returns handles when participants is a non-empty string array", () => {
    expect(
      parseImessageParticipants({
        participants: ["+15551234567"],
      }),
    ).toEqual(["+15551234567"]);
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

describe("ImessageParticipantsRow", () => {
  it("renders participants when present", () => {
    render(
      <ImessageParticipantsRow
        resourceDetails={{
          chat_name: "with Jane Appleseed",
          participants: ["+15551234567"],
        }}
      />,
    );
    expect(screen.getByTestId("imessage-participants-row")).toBeInTheDocument();
    expect(screen.getByText("Participants")).toBeInTheDocument();
    expect(screen.getByText("+15551234567")).toBeInTheDocument();
  });

  it("renders nothing when participants are absent", () => {
    const { container } = render(
      <ImessageParticipantsRow resourceDetails={{ chat_name: "with Jane" }} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});
