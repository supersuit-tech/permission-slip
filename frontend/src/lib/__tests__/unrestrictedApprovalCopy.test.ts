import { describe, it, expect } from "vitest";
import {
  effectiveRiskLevel,
  unrestrictedApprovalWarning,
} from "@/lib/unrestrictedApprovalCopy";

describe("unrestrictedApprovalCopy", () => {
  it("treats missing risk_level as high", () => {
    expect(effectiveRiskLevel(undefined)).toBe("high");
    expect(effectiveRiskLevel(null)).toBe("high");
    expect(effectiveRiskLevel("unknown")).toBe("high");
  });

  it("keeps known risk levels", () => {
    expect(effectiveRiskLevel("low")).toBe("low");
    expect(effectiveRiskLevel("medium")).toBe("medium");
    expect(effectiveRiskLevel("high")).toBe("high");
  });

  it("scales warning copy by risk", () => {
    expect(unrestrictedApprovalWarning("low")).toMatch(/low-risk/i);
    expect(unrestrictedApprovalWarning("medium")).toMatch(/medium-risk/i);
    expect(unrestrictedApprovalWarning("high")).toMatch(/high-risk/i);
    expect(unrestrictedApprovalWarning(undefined)).toMatch(/high-risk/i);
    expect(unrestrictedApprovalWarning("low")).toMatch(
      /No approval prompts will be sent/,
    );
  });
});
