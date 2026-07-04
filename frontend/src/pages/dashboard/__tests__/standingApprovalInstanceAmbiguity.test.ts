import { describe, expect, it } from "vitest";
import {
  hasFrozenStandingApprovalInstanceDisplay,
  shouldShowStandingApprovalInstanceAmbiguityWarning,
  STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING,
} from "../standingApprovalInstanceAmbiguity";

describe("standingApprovalInstanceAmbiguity", () => {
  it("exports the reviewer-facing warning copy", () => {
    expect(STANDING_APPROVAL_INSTANCE_AMBIGUITY_WARNING).toBe(
      "Applies to an unspecified account",
    );
  });

  describe("hasFrozenStandingApprovalInstanceDisplay", () => {
    it("returns true for non-empty display strings", () => {
      expect(hasFrozenStandingApprovalInstanceDisplay("Personal")).toBe(true);
    });

    it("returns false for absent or blank display", () => {
      expect(hasFrozenStandingApprovalInstanceDisplay(null)).toBe(false);
      expect(hasFrozenStandingApprovalInstanceDisplay(undefined)).toBe(false);
      expect(hasFrozenStandingApprovalInstanceDisplay("   ")).toBe(false);
    });
  });

  describe("shouldShowStandingApprovalInstanceAmbiguityWarning", () => {
    it("hides when instance display is frozen", () => {
      expect(
        shouldShowStandingApprovalInstanceAmbiguityWarning("Personal", 3),
      ).toBe(false);
    });

    it("shows when display is absent and multiple instances exist", () => {
      expect(shouldShowStandingApprovalInstanceAmbiguityWarning(null, 2)).toBe(
        true,
      );
    });

    it("hides for a single instance", () => {
      expect(shouldShowStandingApprovalInstanceAmbiguityWarning(null, 1)).toBe(
        false,
      );
    });

    it("hides while instance count is still loading", () => {
      expect(
        shouldShowStandingApprovalInstanceAmbiguityWarning(null, undefined),
      ).toBe(false);
    });
  });
});
