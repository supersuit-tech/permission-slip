import { describe, it, expect } from "vitest";
import validation from "@/lib/validation";
import {
  EMAIL_OTP_DIGIT_LENGTH,
  TOTP_DIGIT_LENGTH,
} from "@/lib/otpDigitLengths";

describe("otpDigitLengths", () => {
  it("matches shared validation.json (email OTP and TOTP)", () => {
    expect(EMAIL_OTP_DIGIT_LENGTH).toBe(validation.emailOtpCode.length);
    expect(TOTP_DIGIT_LENGTH).toBe(validation.totpCode.length);
  });
});
