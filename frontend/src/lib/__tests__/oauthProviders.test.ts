import { describe, it, expect } from "vitest";
import { providerLabel } from "../oauthProviders";

describe("providerLabel", () => {
  it("returns canonical label for known providers", () => {
    expect(providerLabel("google")).toBe("Google");
    expect(providerLabel("linkedin")).toBe("LinkedIn");
    expect(providerLabel("linear")).toBe("Linear");
    expect(providerLabel("microsoft")).toBe("Microsoft");
    expect(providerLabel("meta")).toBe("Meta");
    expect(providerLabel("salesforce")).toBe("Salesforce");
  });

  it("title-cases unknown provider IDs", () => {
    expect(providerLabel("github")).toBe("Github");
    expect(providerLabel("slack")).toBe("Slack");
  });

  it("handles empty string gracefully", () => {
    expect(providerLabel("")).toBe("");
  });
});
