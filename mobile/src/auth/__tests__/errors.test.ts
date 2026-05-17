import type { AuthError } from "../types";
import { safeErrorMessage } from "../errors";

function mockAuthError(code: string): AuthError {
  return {
    message: "raw internal message",
    name: "AuthApiError",
    status: 400,
    code,
  };
}

describe("safeErrorMessage", () => {
  it("maps invalid_credentials to user-friendly message", () => {
    expect(safeErrorMessage(mockAuthError("invalid_credentials"))).toBe(
      "Invalid email or password. Please try again.",
    );
  });

  it("returns a safe message for known error codes", () => {
    expect(safeErrorMessage(mockAuthError("over_request_rate_limit"))).toBe(
      "Too many attempts. Please wait a moment and try again.",
    );
  });

  it("returns generic message for unknown error codes", () => {
    expect(safeErrorMessage(mockAuthError("some_unknown_code"))).toBe(
      "Something went wrong. Please try again.",
    );
  });

  it("returns generic message when error has no code", () => {
    const error: AuthError = {
      message: "something broke",
      name: "AuthApiError",
      status: 500,
    };
    expect(safeErrorMessage(error)).toBe("Something went wrong. Please try again.");
  });
});
