import {
  expandNotifyCmd,
  expiredWakeMessage,
  notFoundWakeMessage,
  wakeMessage,
} from "../src/approvals/notifyCommand.js";

describe("notifyCommand", () => {
  it("expandNotifyCmd replaces id and status placeholders", () => {
    const cmd = expandNotifyCmd(
      'echo "{id}" "{status}"',
      "appr_1",
      "approved",
    );
    expect(cmd).toBe('echo "appr_1" "approved"');
  });

  it("wake messages include approval_id and outcome", () => {
    expect(wakeMessage("appr_1", "approved")).toContain("appr_1");
    expect(wakeMessage("appr_1", "approved")).toContain("approved");
    expect(expiredWakeMessage("appr_1")).toContain("appr_1");
    expect(expiredWakeMessage("appr_1")).toContain("expired");
    expect(notFoundWakeMessage("appr_1")).toContain("appr_1");
  });
});
