import { describe, it, expect } from "vitest";
import type { AuthError } from "../types";
import { safeErrorMessage } from "../errors";

function err(
  message: string,
  status: number,
  code?: string
): AuthError {
  return { message, status, code, name: "AuthApiError" };
}

describe("safeErrorMessage", () => {
  it("maps invalid_credentials code", () => {
    expect(safeErrorMessage(err("bad", 400, "invalid_credentials"))).toBe(
      "Invalid email or password. Please try again."
    );
  });

  it("maps invalid_token code", () => {
    expect(safeErrorMessage(err("bad", 401, "invalid_token"))).toBe(
      "Invalid email or password. Please try again."
    );
  });

  it("maps over_request_rate_limit code", () => {
    expect(safeErrorMessage(err("Rate limit", 429, "over_request_rate_limit"))).toBe(
      "Too many attempts. Please wait a moment and try again."
    );
  });

  it("returns generic fallback for unknown errors", () => {
    expect(safeErrorMessage(err("Something unexpected", 500, "unknown"))).toBe(
      "Something went wrong. Please try again."
    );
  });
});
