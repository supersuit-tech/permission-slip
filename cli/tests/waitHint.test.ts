import {
  buildWaitCommand,
  pendingWaitFields,
  WAIT_HINT,
  isPendingApprovalStatus,
} from "../src/approvals/waitHint.js";

describe("waitHint", () => {
  it("buildWaitCommand returns permission-slip watch with approval id", () => {
    expect(buildWaitCommand("appr_abc123")).toBe("permission-slip watch appr_abc123");
  });

  it("pendingWaitFields uses identical hint and command", () => {
    const fields = pendingWaitFields("appr_x");
    expect(fields.wait_hint).toBe(WAIT_HINT);
    expect(fields.wait_command).toBe("permission-slip watch appr_x");
  });

  it("WAIT_HINT instructs agents to pass session key for session-routed wake channels", () => {
    expect(WAIT_HINT).toContain("--session-key");
    expect(WAIT_HINT).toContain("OpenClaw");
  });

  it("isPendingApprovalStatus is true only for pending", () => {
    expect(isPendingApprovalStatus("pending")).toBe(true);
    expect(isPendingApprovalStatus("approved")).toBe(false);
    expect(isPendingApprovalStatus("denied")).toBe(false);
  });
});
