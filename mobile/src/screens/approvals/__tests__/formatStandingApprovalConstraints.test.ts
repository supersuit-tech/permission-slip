import { formatStandingApprovalConstraintsText } from "../formatStandingApprovalConstraints";

describe("formatStandingApprovalConstraintsText", () => {
  it("shows verified calendar names and hides wildcard event fields", () => {
    const text = formatStandingApprovalConstraintsText(
      {
        event_id: "*",
        calendar_id: "*",
        summary: "*",
        $meta: { calendar_id: "c_abc@group.calendar.google.com" },
      },
      {
        resources: {
          calendar_id: {
            "c_abc@group.calendar.google.com": { name: "Team Calendar" },
          },
        },
      },
    );
    expect(text).toBe("Verified calendar: Team Calendar");
  });
});
