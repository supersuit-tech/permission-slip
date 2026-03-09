import { describe, it, expect } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "../errors";

describe("safeErrorMessage", () => {
  it("maps otp_expired code", () => {
    const error = new AuthError("Token expired", 401, "otp_expired");
    expect(safeErrorMessage(error)).toBe(
      "Your code has expired. Please request a new one."
    );
  });

  it("maps over_email_send_rate_limit code", () => {
    const error = new AuthError(
      "Rate limit",
      429,
      "over_email_send_rate_limit"
    );
    expect(safeErrorMessage(error)).toBe(
      "Too many login emails sent. If you already received a code, you can still use it — otherwise wait a few minutes and try again."
    );
  });

  it("maps over_request_rate_limit code", () => {
    const error = new AuthError("Rate limit", 429, "over_request_rate_limit");
    expect(safeErrorMessage(error)).toBe(
      "Too many attempts. Please wait a moment and try again."
    );
  });

  it("maps mfa_factor_not_found code", () => {
    const error = new AuthError("Factor not found", 400, "mfa_factor_not_found");
    expect(safeErrorMessage(error)).toBe(
      "No authenticator found. Please re-enroll."
    );
  });

  it("maps mfa_verification_failed code", () => {
    const error = new AuthError("Verification failed", 400, "mfa_verification_failed");
    expect(safeErrorMessage(error)).toBe(
      "Invalid code. Please check your authenticator app and try again."
    );
  });

  it("maps mfa_challenge_expired code", () => {
    const error = new AuthError("Challenge expired", 400, "mfa_challenge_expired");
    expect(safeErrorMessage(error)).toBe(
      "Verification timed out. Please try again."
    );
  });

  it("maps mfa_enroll_failed code", () => {
    const error = new AuthError("Enroll failed", 500, "mfa_enroll_failed");
    expect(safeErrorMessage(error)).toBe(
      "Failed to start authenticator setup. Please try again."
    );
  });

  it("returns generic fallback for unknown errors", () => {
    const error = new AuthError("Something unexpected", 500);
    expect(safeErrorMessage(error)).toBe(
      "Something went wrong. Please try again."
    );
  });
});
