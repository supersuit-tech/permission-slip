jest.mock("expo-linking", () => ({
  createURL: jest.fn(() => "permissionslip://"),
}));

import { linking } from "../linking";

describe("linking config", () => {
  it("includes custom scheme and universal link prefixes", () => {
    expect(linking.prefixes).toContain("permissionslip://");
    expect(linking.prefixes).toContain("https://app.permissionslip.dev");
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
