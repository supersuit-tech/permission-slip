import { describe, expect, it } from "vitest";

describe("useStandingApprovalRequests", () => {
  it("exports a hook module", async () => {
    const mod = await import("./useStandingApprovalRequests");
    expect(typeof mod.useStandingApprovalRequests).toBe("function");
  });
});
