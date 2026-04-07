import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { AuthError } from "@supabase/supabase-js";

import OtpStep from "../OtpStep";

// --- Helpers ---

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findByProps({ testID });
}

function findByTestIdOptional(renderer: ReactTestRenderer, testID: string) {
  const results = renderer.root.findAllByProps({ testID });
  return results.length > 0 ? results[0] : null;
}

function makeProps(overrides?: Partial<React.ComponentProps<typeof OtpStep>>) {
  return {
    email: "test@example.com",
    onVerify: jest.fn().mockResolvedValue({ error: null }),
    onResend: jest.fn().mockResolvedValue({ error: null }),
    onBack: jest.fn(),
    ...overrides,
  };
}

// --- Tests ---

describe("OtpStep", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("shows resend button", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const resendBtn = findByTestId(renderer!, "otp-resend");
    expect(resendBtn).toBeTruthy();
    expect(resendBtn.props.disabled).toBe(false);
  });

  it("does not show password link when onUsePassword is omitted", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const link = findByTestIdOptional(renderer!, "otp-use-password");
    expect(link).toBeNull();
  });

  it("shows password fallback when onUsePassword is provided", async () => {
    const onUsePassword = jest.fn();
    const props = makeProps({ onUsePassword });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const link = findByTestId(renderer!, "otp-use-password");
    expect(link).toBeTruthy();
    await act(async () => {
      await link.props.onPress();
    });
    expect(onUsePassword).toHaveBeenCalled();
  });

  it("shows success message after successful resend", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const resendBtn = findByTestId(renderer!, "otp-resend");
    await act(async () => {
      await resendBtn.props.onPress();
    });

    const msg = findByTestId(renderer!, "resend-message");
    expect(msg).toBeTruthy();
    expect(msg.props.children).toBe("Code sent!");
  });

  it("treats over_email_send_rate_limit as success on resend", async () => {
    const onResend = jest.fn().mockResolvedValue({
      error: {
        code: "over_email_send_rate_limit",
        message: "Rate limit",
      } as AuthError,
    });
    const props = makeProps({ onResend });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const resendBtn = findByTestId(renderer!, "otp-resend");
    await act(async () => {
      await resendBtn.props.onPress();
    });

    // Should show "Code sent!" — not an error
    const msg = findByTestId(renderer!, "resend-message");
    expect(msg).toBeTruthy();
    expect(msg.props.children).toBe("Code sent!");
  });

  it("shows error for non-rate-limit resend failures", async () => {
    const onResend = jest.fn().mockResolvedValue({
      error: {
        code: "unexpected_failure",
        message: "Server error",
      } as AuthError,
    });
    const props = makeProps({ onResend });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(OtpStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const resendBtn = findByTestId(renderer!, "otp-resend");
    await act(async () => {
      await resendBtn.props.onPress();
    });

    const msg = findByTestId(renderer!, "resend-message");
    expect(msg).toBeTruthy();
    expect(msg.props.children).toBe("Failed to resend. Please try again.");
  });
});
