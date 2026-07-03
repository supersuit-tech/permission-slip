import {
  applySessionKeyToDefaultTemplate,
  DEFAULT_NOTIFY_CMD_TEMPLATE,
  expandNotifyCmd,
  expiredWakeMessage,
  notFoundWakeMessage,
  validateNotifyCmdTemplate,
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

  it("expandNotifyCmd replaces session_key with shell-quoted value", () => {
    const cmd = expandNotifyCmd(
      "openclaw system event --session-key {session_key}",
      "appr_1",
      "approved",
      undefined,
      "agent:main:imessage",
    );
    expect(cmd).toBe("openclaw system event --session-key 'agent:main:imessage'");
  });

  it("expandNotifyCmd shell-quotes session keys with metacharacters", () => {
    const cmd = expandNotifyCmd(
      "notify --session-key {session_key}",
      "appr_1",
      "approved",
      undefined,
      "it's a test",
    );
    expect(cmd).toBe("notify --session-key 'it'\\''s a test'");
  });

  it("validateNotifyCmdTemplate throws when template has session_key but no key provided", () => {
    expect(() =>
      validateNotifyCmdTemplate("openclaw system event --session-key {session_key}"),
    ).toThrow("contains {session_key} but no --session-key was provided");
  });

  it("expandNotifyCmd throws when template has session_key but no key provided", () => {
    expect(() =>
      expandNotifyCmd("openclaw system event --session-key {session_key}", "appr_1", "approved"),
    ).toThrow("contains {session_key} but no --session-key was provided");
  });

  it("applySessionKeyToDefaultTemplate appends quoted session key to default template", () => {
    const cmd = applySessionKeyToDefaultTemplate(
      DEFAULT_NOTIFY_CMD_TEMPLATE,
      "agent:main:imessage",
    );
    expect(cmd).toBe(
      'openclaw system event --text "{message}" --mode now --session-key \'agent:main:imessage\'',
    );
  });

  it("applySessionKeyToDefaultTemplate leaves custom templates unchanged", () => {
    const custom = 'echo "{message}" --session-key {session_key}';
    expect(applySessionKeyToDefaultTemplate(custom, "agent:main")).toBe(custom);
  });

  it("applySessionKeyToDefaultTemplate is a no-op without session key", () => {
    expect(applySessionKeyToDefaultTemplate(DEFAULT_NOTIFY_CMD_TEMPLATE)).toBe(
      DEFAULT_NOTIFY_CMD_TEMPLATE,
    );
  });

  it("wake messages include approval_id and outcome", () => {
    expect(wakeMessage("appr_1", "approved")).toContain("appr_1");
    expect(wakeMessage("appr_1", "approved")).toContain("approved");
    expect(expiredWakeMessage("appr_1")).toContain("appr_1");
    expect(expiredWakeMessage("appr_1")).toContain("expired");
    expect(notFoundWakeMessage("appr_1")).toContain("appr_1");
  });
});
