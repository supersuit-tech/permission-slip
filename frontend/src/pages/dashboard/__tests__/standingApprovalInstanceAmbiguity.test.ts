import { describe, expect, it } from "vitest";
import {
  getStandingApprovalInstanceScopeLabel,
  hasFrozenStandingApprovalInstanceDisplay,
} from "../standingApprovalInstanceAmbiguity";

describe("standingApprovalInstanceAmbiguity", () => {
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

  describe("getStandingApprovalInstanceScopeLabel", () => {
    it("returns null while instance count is still loading", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(null, [], false),
      ).toBeNull();
    });

    it("uses frozen connector_instance_display when present", () => {
      expect(
        getStandingApprovalInstanceScopeLabel("Personal", [{ display: "Work" }], true),
      ).toBe("Applies to Personal");
    });

    it("uses the single instance display name when only one exists", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(null, [{ display: "Personal" }], true),
      ).toBe("Applies to Personal");
    });

    it("falls back for a single instance without display", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(null, [{}], true),
      ).toBe("Applies to this account");
    });

    it("shows all accounts when multiple instances exist and display is absent", () => {
      expect(
        getStandingApprovalInstanceScopeLabel(
          null,
          [{ display: "Personal" }, { display: "Work" }],
          true,
        ),
      ).toBe("Applies to all accounts");
    });
  });
});
