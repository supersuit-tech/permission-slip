import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  CONSENT_COOKIE_NAME,
  getStoredConsent,
  persistConsent,
  clearConsent,
} from "../consent-cookie";
import {
  setCookie,
  getCookie,
  clearConsentCookie,
} from "../../test-cookie-helpers";

const OLD_STORAGE_KEY = "permission-slip-cookie-consent";

describe("consent-cookie", () => {
  beforeEach(() => {
    clearConsentCookie();
    localStorage.removeItem(OLD_STORAGE_KEY);
  });
  afterEach(() => {
    clearConsentCookie();
    localStorage.removeItem(OLD_STORAGE_KEY);
  });

  describe("getStoredConsent", () => {
    it("returns null when no cookie is set", () => {
      expect(getStoredConsent()).toBeNull();
    });

    it("returns 'accepted' when cookie is 'accepted'", () => {
      setCookie(CONSENT_COOKIE_NAME, "accepted");
      expect(getStoredConsent()).toBe("accepted");
    });

    it("returns 'rejected' when cookie is 'rejected'", () => {
      setCookie(CONSENT_COOKIE_NAME, "rejected");
      expect(getStoredConsent()).toBe("rejected");
    });

    it("returns null for invalid cookie values", () => {
      setCookie(CONSENT_COOKIE_NAME, "maybe");
      expect(getStoredConsent()).toBeNull();
    });
  });

  describe("persistConsent", () => {
    it("sets the accepted cookie", () => {
      persistConsent("accepted");
      expect(getCookie(CONSENT_COOKIE_NAME)).toBe("accepted");
    });

    it("sets the rejected cookie", () => {
      persistConsent("rejected");
      expect(getCookie(CONSENT_COOKIE_NAME)).toBe("rejected");
    });
  });

  describe("clearConsent", () => {
    it("removes the consent cookie", () => {
      persistConsent("accepted");
      expect(getCookie(CONSENT_COOKIE_NAME)).toBe("accepted");

      clearConsent();
      expect(getCookie(CONSENT_COOKIE_NAME)).toBeNull();
    });
  });

  describe("localStorage migration", () => {
    it("migrates 'accepted' from localStorage to cookie", () => {
      localStorage.setItem(OLD_STORAGE_KEY, "accepted");
      expect(getStoredConsent()).toBe("accepted");
      expect(getCookie(CONSENT_COOKIE_NAME)).toBe("accepted");
      expect(localStorage.getItem(OLD_STORAGE_KEY)).toBeNull();
    });

    it("migrates 'rejected' from localStorage to cookie", () => {
      localStorage.setItem(OLD_STORAGE_KEY, "rejected");
      expect(getStoredConsent()).toBe("rejected");
      expect(getCookie(CONSENT_COOKIE_NAME)).toBe("rejected");
      expect(localStorage.getItem(OLD_STORAGE_KEY)).toBeNull();
    });

    it("does not migrate invalid localStorage values", () => {
      localStorage.setItem(OLD_STORAGE_KEY, "maybe");
      expect(getStoredConsent()).toBeNull();
      expect(getCookie(CONSENT_COOKIE_NAME)).toBeNull();
    });

    it("prefers existing cookie over localStorage", () => {
      setCookie(CONSENT_COOKIE_NAME, "rejected");
      localStorage.setItem(OLD_STORAGE_KEY, "accepted");
      expect(getStoredConsent()).toBe("rejected");
      expect(localStorage.getItem(OLD_STORAGE_KEY)).toBe("accepted");
    });
  });
});
