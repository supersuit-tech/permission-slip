import { describe, expect, it } from "vitest";
import { providerLabel, serviceLabel, authTypeLabel } from "../labels";

describe("providerLabel", () => {
  it("returns known provider labels", () => {
    expect(providerLabel("airtable")).toBe("Airtable");
    expect(providerLabel("google")).toBe("Google");
    expect(providerLabel("netlify")).toBe("Netlify");
    expect(providerLabel("microsoft")).toBe("Microsoft");
    expect(providerLabel("intercom")).toBe("Intercom");
    expect(providerLabel("github")).toBe("GitHub");
    expect(providerLabel("hubspot")).toBe("HubSpot");
  });

  it("capitalises unknown provider IDs", () => {
    expect(providerLabel("unknown")).toBe("Unknown");
  });

  it("handles empty string", () => {
    expect(providerLabel("")).toBe("");
  });
});

describe("serviceLabel", () => {
  it("returns known service labels", () => {
    expect(serviceLabel("netlify-api-key")).toBe("Netlify API Key");
    expect(serviceLabel("github_pat")).toBe("GitHub Personal Access Token");
  });

  it("falls back to provider label for bare provider IDs", () => {
    expect(serviceLabel("github")).toBe("GitHub");
    expect(serviceLabel("slack")).toBe("Slack");
  });

  it("falls back to raw service ID for unknown services", () => {
    expect(serviceLabel("some-service")).toBe("some-service");
  });
});

describe("authTypeLabel", () => {
  it("returns known auth type labels", () => {
    expect(authTypeLabel("api_key")).toBe("API Key");
    expect(authTypeLabel("oauth2")).toBe("OAuth");
    expect(authTypeLabel("basic")).toBe("Username & Password");
    expect(authTypeLabel("custom")).toBe("Custom");
  });

  it("falls back to raw auth type for unknown types", () => {
    expect(authTypeLabel("unknown")).toBe("unknown");
  });
});
