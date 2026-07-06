import { describe, expect, it } from "vitest";
import {
  buildMetaConstraintsFromForm,
  isBoilerplateStandingApprovalDescription,
  mergeStandingApprovalConstraints,
  metaValuesFromConstraints,
} from "../standingApprovalConstraints";

describe("standingApprovalConstraints", () => {
  it("reads $meta.from from stored constraints", () => {
    expect(
      metaValuesFromConstraints({
        message_id: "*",
        $meta: { from: "automated@airbnb.com" },
      }),
    ).toEqual({ from: "automated@airbnb.com" });
  });

  it("builds $meta constraints from form values", () => {
    expect(
      buildMetaConstraintsFromForm({ from: "automated@airbnb.com" }),
    ).toEqual({ from: "automated@airbnb.com" });
  });

  it("merges parameter and meta constraints", () => {
    expect(
      mergeStandingApprovalConstraints(
        { message_id: "*", folder: "*" },
        { from: "automated@airbnb.com" },
      ),
    ).toEqual({
      message_id: "*",
      folder: "*",
      $meta: { from: "automated@airbnb.com" },
    });
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
