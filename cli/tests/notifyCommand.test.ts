import {
  expandNotifyCmd,
  expiredWakeMessage,
  notFoundWakeMessage,
  wakeMessage,
} from "../src/approvals/notifyCommand.js";

describe("notifyCommand", () => {
  it("expandNotifyCmd replaces id, status, and message placeholders", () => {
    const cmd = expandNotifyCmd(
      'echo "{id}" "{status}" "{message}"',
      "appr_1",
      "approved",
      "custom wake text",
    );
    expect(cmd).toBe('echo "appr_1" "approved" "custom wake text"');
  });

  it("expandNotifyCmd defaults message from id and status", () => {
    const cmd = expandNotifyCmd('"{message}"', "appr_1", "denied");
    expect(cmd).toBe('"Permission Slip appr_1 resolved: denied — continue the task"');
  });

  it("wake messages include approval_id and outcome", () => {
    expect(wakeMessage("appr_1", "approved")).toContain("appr_1");
    expect(wakeMessage("appr_1", "approved")).toContain("approved");
    expect(expiredWakeMessage("appr_1")).toContain("appr_1");
    expect(expiredWakeMessage("appr_1")).toContain("expired");
    expect(notFoundWakeMessage("appr_1")).toContain("appr_1");
  });
});
