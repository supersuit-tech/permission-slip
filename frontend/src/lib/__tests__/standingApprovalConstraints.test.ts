import { describe, expect, it } from "vitest";
import {
  isBoilerplateStandingApprovalDescription,
  preservedNamespacesFromConstraints,
} from "../standingApprovalConstraints";

describe("standingApprovalConstraints", () => {
  it("preserves $data_window from stored constraints", () => {
    expect(
      preservedNamespacesFromConstraints({
        message_id: "*",
        $data_window: { last_days: 7 },
      }),
    ).toEqual({ data_window: { last_days: 7 } });
  });

  it("returns empty object when no data window is present", () => {
    expect(
      preservedNamespacesFromConstraints({ message_id: "*" }),
    ).toEqual({});
  });

  it("detects boilerplate auto-created descriptions", () => {
    expect(
      isBoilerplateStandingApprovalDescription(
        "Created automatically when approving a standing auto-approve rule proposal",
      ),
    ).toBe(true);
    expect(
      isBoilerplateStandingApprovalDescription("Standing auto-approve rule"),
    ).toBe(true);
    expect(
      isBoilerplateStandingApprovalDescription("Only Airbnb confirmations"),
    ).toBe(false);
  });
});
