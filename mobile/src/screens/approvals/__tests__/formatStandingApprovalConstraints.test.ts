import {
  formatStandingApprovalConstraints,
  formatStandingApprovalConstraintsText,
} from "../formatStandingApprovalConstraints";

describe("formatStandingApprovalConstraints", () => {
  it("labels $meta from constraints as verified sender", () => {
    const lines = formatStandingApprovalConstraints({
      message_id: "*",
      $meta: { from: { $pattern: "auto-confirm@amazon.com" } },
    });
    expect(lines).toEqual([
      {
        label: "message_id",
        mode: "wildcard",
        value: "any",
        verified: false,
      },
      {
        label: "Verified sender",
        mode: "pattern",
        value: "auto-confirm@amazon.com",
        verified: true,
      },
    ]);
  });

  it("formats readable multiline text", () => {
    const text = formatStandingApprovalConstraintsText({
      $meta: { to: "team@mycorp.com" },
    });
    expect(text).toBe("Verified To: team@mycorp.com");
  });

  it("formats $data_window last_days", () => {
    const lines = formatStandingApprovalConstraints({
      chat_id: "42",
      $data_window: { last_days: 30 },
    });
    expect(lines).toContainEqual({
      label: "Data window",
      mode: "fixed",
      value: "last 30 days",
      verified: false,
    });
  });
});
