jest.mock("expo-linking", () => ({
  createURL: jest.fn(() => "permissionslip://"),
}));

import { linking } from "../linking";

describe("linking config", () => {
  it("includes custom scheme prefix", () => {
    expect(linking.prefixes).toContain("permissionslip://");
  });

  it("maps approval deep link to DeepLinkDetail screen", () => {
    expect(linking.config?.screens?.DeepLinkDetail).toBe(
      "permission-slip/approve/:approvalId",
    );
  });

  it("maps root to ApprovalList screen", () => {
    expect(linking.config?.screens?.ApprovalList).toBe("");
  });
});
