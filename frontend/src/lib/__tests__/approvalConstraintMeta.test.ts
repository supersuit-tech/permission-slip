import { describe, it, expect } from "vitest";
import { resourceDetailsToConstraintMeta } from "../approvalConstraintMeta";

describe("resourceDetailsToConstraintMeta", () => {
  it("extracts sender metadata from resource details", () => {
    const meta = resourceDetailsToConstraintMeta({
      subject: "Receipt",
      from: ["invoice@anthropic.com"],
    });
    expect(meta).toEqual({
      sender: "invoice@anthropic.com",
      senders: ["invoice@anthropic.com"],
      messages: [
        {
          from: "invoice@anthropic.com",
          to: [],
          cc: [],
          bcc: [],
        },
      ],
    });
  });
});
