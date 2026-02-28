import { describe, it, expect, beforeEach } from "vitest";
import {
  MFA_PENDING_ENROLLMENT_KEY,
  savePendingEnrollment,
  hasPendingEnrollment,
  clearPendingEnrollment,
} from "../mfaPendingEnrollment";

describe("mfaPendingEnrollment", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  describe("savePendingEnrollment", () => {
    it("writes a JSON marker with the userId", () => {
      savePendingEnrollment("user-abc");
      const raw = sessionStorage.getItem(MFA_PENDING_ENROLLMENT_KEY);
      expect(raw).not.toBeNull();
      expect(JSON.parse(raw!)).toEqual({ userId: "user-abc" });
    });
  });

  describe("hasPendingEnrollment", () => {
    it("returns true when marker exists for the same user", () => {
      savePendingEnrollment("user-abc");
      expect(hasPendingEnrollment("user-abc")).toBe(true);
    });

    it("returns false when marker belongs to a different user", () => {
      savePendingEnrollment("user-abc");
      expect(hasPendingEnrollment("user-xyz")).toBe(false);
    });

    it("returns false when no marker exists", () => {
      expect(hasPendingEnrollment("user-abc")).toBe(false);
    });

    it("returns false for invalid JSON in sessionStorage", () => {
      sessionStorage.setItem(MFA_PENDING_ENROLLMENT_KEY, "not json");
      expect(hasPendingEnrollment("user-abc")).toBe(false);
    });

    it("returns false for non-object JSON value", () => {
      sessionStorage.setItem(MFA_PENDING_ENROLLMENT_KEY, '"just a string"');
      expect(hasPendingEnrollment("user-abc")).toBe(false);
    });

    it("returns false for object missing userId field", () => {
      sessionStorage.setItem(MFA_PENDING_ENROLLMENT_KEY, JSON.stringify({ foo: "bar" }));
      expect(hasPendingEnrollment("user-abc")).toBe(false);
    });

    it("returns false for legacy entry with empty userId", () => {
      sessionStorage.setItem(
        MFA_PENDING_ENROLLMENT_KEY,
        JSON.stringify({ userId: "" })
      );
      expect(hasPendingEnrollment("user-abc")).toBe(false);
    });
  });

  describe("clearPendingEnrollment", () => {
    it("removes the marker from sessionStorage", () => {
      savePendingEnrollment("user-abc");
      clearPendingEnrollment();
      expect(sessionStorage.getItem(MFA_PENDING_ENROLLMENT_KEY)).toBeNull();
    });

    it("does not throw when no marker exists", () => {
      expect(() => clearPendingEnrollment()).not.toThrow();
    });
  });
});
