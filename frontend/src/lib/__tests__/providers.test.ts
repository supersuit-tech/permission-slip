import { describe, expect, it } from "vitest";
import { providerLabel, formatServiceName } from "../providers";

describe("providerLabel", () => {
  it("returns known provider labels", () => {
    expect(providerLabel("google")).toBe("Google");
    expect(providerLabel("netlify")).toBe("Netlify");
    expect(providerLabel("microsoft")).toBe("Microsoft");
    expect(providerLabel("intercom")).toBe("Intercom");
  });

  it("capitalises unknown provider IDs", () => {
    expect(providerLabel("stripe")).toBe("Stripe");
    expect(providerLabel("zoom")).toBe("Zoom");
  });

  it("handles empty string", () => {
    expect(providerLabel("")).toBe("");
  });
});

describe("formatServiceName", () => {
  it("formats hyphenated service names", () => {
    expect(formatServiceName("netlify-api-key")).toBe("API Key");
  });

  it("formats underscore service names", () => {
    expect(formatServiceName("intercom_oauth")).toBe("Intercom OAuth");
  });

  it("handles simple service names", () => {
    // Single-segment names (no prefix to strip) get title-cased
    expect(formatServiceName("netlify")).toBe("Netlify");
  });

  it("preserves API and OAuth casing", () => {
    expect(formatServiceName("my-api-token")).toBe("API Token");
    expect(formatServiceName("my-oauth-key")).toBe("OAuth Key");
  });
});
