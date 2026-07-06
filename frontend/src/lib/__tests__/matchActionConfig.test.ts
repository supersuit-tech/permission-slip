import { describe, it, expect } from "vitest";
import {
  execParamsSatisfyConfigConstraints,
  resourceDetailsToConstraintMeta,
} from "../matchActionConfig";

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

describe("execParamsSatisfyConfigConstraints", () => {
  const execParams = { folder: "INBOX", message_id: 92 };

  it("requires resolved metadata to match $meta constraints", () => {
    const constraints = {
      folder: "*",
      message_id: "*",
      $meta: { from: "invoice@anthropic.com" },
    };

    expect(
      execParamsSatisfyConfigConstraints(constraints, execParams, null),
    ).toBe(false);

    expect(
      execParamsSatisfyConfigConstraints(
        constraints,
        execParams,
        resourceDetailsToConstraintMeta({ from: ["invoice@anthropic.com"] }),
      ),
    ).toBe(true);
  });
});
