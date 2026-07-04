import {
  hasFrozenStandingApprovalInstanceDisplay,
  shouldShowStandingApprovalInstanceAmbiguityWarning,
} from "../standingApprovalInstanceAmbiguity";

describe("standingApprovalInstanceAmbiguity", () => {
  describe("hasFrozenStandingApprovalInstanceDisplay", () => {
    it("returns true for non-empty display strings", () => {
      expect(hasFrozenStandingApprovalInstanceDisplay("Sales")).toBe(true);
    });

    it("returns false for absent or blank display", () => {
      expect(hasFrozenStandingApprovalInstanceDisplay(null)).toBe(false);
      expect(hasFrozenStandingApprovalInstanceDisplay("")).toBe(false);
    });
  });

  describe("shouldShowStandingApprovalInstanceAmbiguityWarning", () => {
    it("shows when display is absent and multiple instances exist", () => {
      expect(shouldShowStandingApprovalInstanceAmbiguityWarning(undefined, 2)).toBe(
        true,
      );
    });

    it("hides when display is present", () => {
      expect(
        shouldShowStandingApprovalInstanceAmbiguityWarning("Personal", 2),
      ).toBe(false);
    });

    it("hides for one or zero instances", () => {
      expect(shouldShowStandingApprovalInstanceAmbiguityWarning(null, 1)).toBe(
        false,
      );
      expect(shouldShowStandingApprovalInstanceAmbiguityWarning(null, 0)).toBe(
        false,
      );
    });
  });
});
