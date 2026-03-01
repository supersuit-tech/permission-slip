import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "../errors";

function mockAuthError(code: string): AuthError {
  return {
    message: "raw internal message",
    name: "AuthApiError",
    status: 400,
    code,
  } as AuthError;
}

describe("safeErrorMessage", () => {
  it("returns a safe message for known error codes", () => {
    expect(safeErrorMessage(mockAuthError("otp_expired"))).toBe(
      "Your code has expired. Please request a new one."
    );
    expect(safeErrorMessage(mockAuthError("over_request_rate_limit"))).toBe(
      "Too many attempts. Please wait a moment and try again."
    );
  });

  it("returns generic message for unknown error codes", () => {
    expect(safeErrorMessage(mockAuthError("some_unknown_code"))).toBe(
      "Something went wrong. Please try again."
    );
  });

  it("returns generic message when error has no code", () => {
    const error = {
      message: "something broke",
      name: "AuthApiError",
      status: 500,
    } as AuthError;
    expect(safeErrorMessage(error)).toBe(
      "Something went wrong. Please try again."
    );
  });
});
