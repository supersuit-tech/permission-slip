import { runWatchLoop } from "../src/approvals/watchLoop.js";

describe("runWatchLoop", () => {
  const approvalId = "appr_test";
  const notifyTemplate = 'notify "{id}" "{status}"';

  it("exits on terminal status and fires notify", async () => {
    const notified: string[] = [];
    let polls = 0;

    const result = await runWatchLoop({
      approvalId,
      expiresAt: new Date(Date.now() + 60_000),
      intervalMs: 1,
      notifyCmdTemplate: notifyTemplate,
      poll: async () => {
        polls += 1;
        return polls === 1
          ? { kind: "status", status: "pending", expiresAt: new Date(Date.now() + 60_000).toISOString() }
          : { kind: "status", status: "approved" };
      },
      sleep: async () => {},
      runNotify: async (cmd) => {
        notified.push(cmd);
      },
    });

    expect(result.status).toBe("approved");
    expect(result.notified).toBe(true);
    expect(notified).toEqual(['notify "appr_test" "approved"']);
    expect(result.wake_message).toContain("appr_test");
    expect(result.wake_message).toContain("approved");
  });

  it("exits on not_found and fires notify", async () => {
    const notified: string[] = [];

    const result = await runWatchLoop({
      approvalId,
      expiresAt: new Date(Date.now() + 60_000),
      intervalMs: 1,
      notifyCmdTemplate: notifyTemplate,
      poll: async () => ({ kind: "not_found" }),
      sleep: async () => {},
      runNotify: async (cmd) => {
        notified.push(cmd);
      },
    });

    expect(result.status).toBe("not_found");
    expect(result.notified).toBe(true);
    expect(notified).toEqual(['notify "appr_test" "not_found"']);
  });

  it("exits on expiry and fires notify", async () => {
    const notified: string[] = [];
    const start = Date.now();
    let nowMs = start;

    const result = await runWatchLoop({
      approvalId,
      expiresAt: new Date(start + 5),
      intervalMs: 1,
      notifyCmdTemplate: notifyTemplate,
      poll: async () => ({
        kind: "status",
        status: "pending",
        expiresAt: new Date(start + 5).toISOString(),
      }),
      sleep: async () => {
        nowMs += 10;
      },
      now: () => new Date(nowMs),
      runNotify: async (cmd) => {
        notified.push(cmd);
      },
    });

    expect(result.status).toBe("expired");
    expect(result.notified).toBe(true);
    expect(notified).toEqual(['notify "appr_test" "expired"']);
    expect(result.wake_message).toContain("expired");
  });

  it("tolerates transient errors until expiry", async () => {
    const notified: string[] = [];
    const start = Date.now();
    let nowMs = start;
    let polls = 0;

    const result = await runWatchLoop({
      approvalId,
      expiresAt: new Date(start + 20),
      intervalMs: 5,
      notifyCmdTemplate: notifyTemplate,
      poll: async () => {
        polls += 1;
        if (polls < 3) {
          return { kind: "error", message: "network timeout" };
        }
        return { kind: "status", status: "denied" };
      },
      sleep: async () => {
        nowMs += 5;
      },
      now: () => new Date(nowMs),
      runNotify: async (cmd) => {
        notified.push(cmd);
      },
    });

    expect(result.status).toBe("denied");
    expect(polls).toBe(3);
    expect(notified).toEqual(['notify "appr_test" "denied"']);
  });

  it("reports notified false when notify command fails", async () => {
    const result = await runWatchLoop({
      approvalId,
      expiresAt: new Date(Date.now() + 60_000),
      intervalMs: 1,
      notifyCmdTemplate: notifyTemplate,
      poll: async () => ({ kind: "status", status: "approved" }),
      sleep: async () => {},
      runNotify: async () => {
        throw new Error("notify failed");
      },
    });

    expect(result.status).toBe("approved");
    expect(result.notified).toBe(false);
  });
});
