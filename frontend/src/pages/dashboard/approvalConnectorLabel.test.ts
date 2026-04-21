import { describe, expect, it } from "vitest";
import { formatConnectorDisplayName } from "./approvalConnectorLabel";

describe("formatConnectorDisplayName", () => {
  it("appends instance label in parentheses when present", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: "Slack",
        actionType: "slack.send_message",
        instanceLabel: "Engineering",
      }),
    ).toBe("Slack (Engineering)");
  });

  it("falls back to connector prefix when connectorName is missing", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: null,
        actionType: "slack.send_message",
        instanceLabel: "Engineering",
      }),
    ).toBe("Slack (Engineering)");
  });

  it("uses connector name only when no instance label", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: "Slack",
        actionType: "slack.send_message",
        instanceLabel: undefined,
      }),
    ).toBe("Slack");
  });
});
