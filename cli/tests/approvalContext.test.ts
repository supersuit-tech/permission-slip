import { buildApprovalContext } from "../src/approvals/approvalContext.js";

describe("buildApprovalContext", () => {
  it("returns undefined when no fields are set", () => {
    expect(buildApprovalContext({})).toBeUndefined();
    expect(buildApprovalContext({ sessionKey: "  " })).toBeUndefined();
  });

  it("includes session_key when provided", () => {
    expect(
      buildApprovalContext({ sessionKey: "agent:main:imessage" }),
    ).toEqual({ session_key: "agent:main:imessage" });
  });

  it("merges description, risk_level, and session_key", () => {
    expect(
      buildApprovalContext({
        description: "Send email",
        riskLevel: "low",
        sessionKey: "agent:main:telegram:direct:8935627010",
      }),
    ).toEqual({
      description: "Send email",
      risk_level: "low",
      session_key: "agent:main:telegram:direct:8935627010",
    });
  });

  it("trims session_key whitespace", () => {
    expect(
      buildApprovalContext({ sessionKey: "  agent:main:slack  " }),
    ).toEqual({ session_key: "agent:main:slack" });
  });
});
