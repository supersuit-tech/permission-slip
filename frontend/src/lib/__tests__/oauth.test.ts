import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getOAuthAuthorizeUrl } from "../oauth";

describe("getOAuthAuthorizeUrl", () => {
  const originalLocation = window.location;

  beforeEach(() => {
    // Pin window.location so tests don't depend on jsdom's default URL.
    Object.defineProperty(window, "location", {
      value: new URL("http://localhost/settings/connectors"),
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
  });
  it("builds authorize URL without scopes", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_123");
    expect(url).toContain("/v1/oauth/google/authorize?");
    expect(url).toContain("access_token=tok_123");
    expect(url).not.toContain("scope=");
  });

  it("appends scopes as repeated query params", () => {
    const url = getOAuthAuthorizeUrl("slack", "tok_abc", [
      "chat:write",
      "channels:read",
    ]);
    expect(url).toContain("scope=chat%3Awrite");
    expect(url).toContain("scope=channels%3Aread");
  });

  it("URL-encodes scopes with special characters", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_x", [
      "https://www.googleapis.com/auth/gmail.send",
    ]);
    expect(url).toContain(
      "scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.send",
    );
  });

  it("skips scopes when array is empty", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_y", []);
    expect(url).not.toContain("scope=");
  });

  it("skips scopes when undefined", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_z", undefined);
    expect(url).not.toContain("scope=");
  });

  it("appends replace param when replaceId is provided", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_r", {
      scopes: ["openid"],
      replaceId: "oconn_abc123",
    });
    expect(url).toContain("scope=openid");
    expect(url).toContain("replace=oconn_abc123");
  });

  it("omits replace param when replaceId is not set", () => {
    const url = getOAuthAuthorizeUrl("google", "tok_n", {
      scopes: ["openid"],
    });
    expect(url).not.toContain("replace=");
  });
});
