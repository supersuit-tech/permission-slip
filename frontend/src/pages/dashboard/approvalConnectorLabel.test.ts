import { describe, expect, it } from "vitest";
import { formatConnectorDisplayName } from "./approvalConnectorLabel";

describe("formatConnectorDisplayName", () => {
  it("appends instance display in parentheses when present", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: "Slack",
        actionType: "slack.send_message",
        instanceDisplay: "Engineering",
      }),
    ).toBe("Slack (Engineering)");
  });

  it("falls back to connector prefix when connectorName is missing", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: null,
        actionType: "slack.send_message",
        instanceDisplay: "Engineering",
      }),
    ).toBe("Slack (Engineering)");
  });

  it("falls back to legacy instanceLabel when display is absent", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: "Slack",
        actionType: "slack.send_message",
        instanceLabel: "Legacy",
      }),
    ).toBe("Slack (Legacy)");
  });

  it("prefers instanceDisplay over instanceLabel when both are present", () => {
    expect(
      formatConnectorDisplayName({
        connectorName: "Slack",
        actionType: "slack.send_message",
        instanceDisplay: "New",
        instanceLabel: "Old",
      }),
    ).toBe("Slack (New)");
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
