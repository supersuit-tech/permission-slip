import {
  getStandingApprovalInstanceScopeLabel,
  hasFrozenStandingApprovalInstanceDisplay,
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

  describe("getStandingApprovalInstanceScopeLabel", () => {
    it("returns null while loading", () => {
      expect(getStandingApprovalInstanceScopeLabel(undefined, [], false)).toBeNull();
    });

    it("uses frozen display when present", () => {
      expect(
        getStandingApprovalInstanceScopeLabel("Personal", [{ display: "Work" }], true),
      ).toBe("Applies to Personal");
    });

    it("uses single instance display when only one exists", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(null, [{ display: "Personal" }], true),
      ).toBe("Applies to Personal");
    });

    it("shows all accounts for multiple instances without frozen display", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(
          undefined,
          [{ display: "Personal" }, { display: "Work" }],
          true,
        ),
      ).toBe("Applies to all accounts");
    });
  });
});
