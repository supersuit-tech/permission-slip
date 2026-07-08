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

  it("buildWaitCommand includes session key when provided", () => {
    expect(buildWaitCommand("appr_abc123", "agent:main:imessage")).toBe(
      "permission-slip watch appr_abc123 --session-key 'agent:main:imessage'",
    );
  });

  it("pendingWaitFields uses identical hint and command", () => {
    const fields = pendingWaitFields("appr_x");
    expect(fields.wait_hint).toBe(WAIT_HINT);
    expect(fields.wait_command).toBe("permission-slip watch appr_x");
  });

  it("pendingWaitFields includes session key in wait_command when provided", () => {
    const fields = pendingWaitFields("appr_x", false, "agent:main:slack");
    expect(fields.wait_command).toBe(
      "permission-slip watch appr_x --session-key 'agent:main:slack'",
    );
  });

  it("pendingWaitFields uses push wake hint when configured", () => {
    const fields = pendingWaitFields("appr_x", true);
    expect(fields.wait_hint).toContain("push wake webhook");
    expect(fields.wait_command).toBe("permission-slip watch appr_x");
  });

  it("WAIT_HINT instructs agents to pass session key on request and watch", () => {
    expect(WAIT_HINT).toContain("--session-key");
    expect(WAIT_HINT).toContain("on request");
    expect(WAIT_HINT).toContain("OpenClaw");
  });

  it("isPendingApprovalStatus is true only for pending", () => {
    expect(isPendingApprovalStatus("pending")).toBe(true);
    expect(isPendingApprovalStatus("approved")).toBe(false);
    expect(isPendingApprovalStatus("denied")).toBe(false);
  });
});
