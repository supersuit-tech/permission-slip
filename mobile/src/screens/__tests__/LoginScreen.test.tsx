import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Mocks ---

const mockSendOtp = jest.fn();
const mockVerifyOtp = jest.fn();

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    sendOtp: mockSendOtp,
    verifyOtp: mockVerifyOtp,
    session: null,
    user: null,
    authStatus: "unauthenticated" as const,
  }),
}));

import LoginScreen from "../LoginScreen";

// --- Helpers ---

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findByProps({ testID });
}

function findByTestIdOptional(renderer: ReactTestRenderer, testID: string) {
  const results = renderer.root.findAllByProps({ testID });
  return results.length > 0 ? results[0] : null;
}

// --- Tests ---

describe("LoginScreen", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("shows email step initially", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });
    // Flush auto-focus timer
    await act(async () => {
      jest.runAllTimers();
    });
    expect(findByTestId(renderer!, "email-input")).toBeTruthy();
    expect(findByTestIdOptional(renderer!, "otp-input")).toBeNull();
  });

  it("transitions to OTP step after successful send", async () => {
    mockSendOtp.mockResolvedValue({ error: null });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "email-input");
    await act(async () => {
      input.props.onChangeText("test@example.com");
    });

    const submit = findByTestId(renderer!, "email-submit");
    await act(async () => {
      await submit.props.onPress();
    });
    await act(async () => {
      jest.runAllTimers();
    });

    expect(findByTestIdOptional(renderer!, "otp-input")).toBeTruthy();
  });

  it("transitions to OTP step on over_email_send_rate_limit", async () => {
    mockSendOtp.mockResolvedValue({
      error: { code: "over_email_send_rate_limit", message: "Rate limit" },
    });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "email-input");
    await act(async () => {
      input.props.onChangeText("test@example.com");
    });

    const submit = findByTestId(renderer!, "email-submit");
    await act(async () => {
      await submit.props.onPress();
    });
    await act(async () => {
      jest.runAllTimers();
    });

    // Should advance to OTP step even on rate limit
    expect(findByTestIdOptional(renderer!, "otp-input")).toBeTruthy();
    // Should not show error on the email step
    expect(findByTestIdOptional(renderer!, "email-error")).toBeNull();
  });

  it("stays on email step on non-rate-limit error", async () => {
    mockSendOtp.mockResolvedValue({
      error: {
        code: "unexpected_failure",
        message: "Server error",
        name: "AuthApiError",
        status: 500,
      },
    });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "email-input");
    await act(async () => {
      input.props.onChangeText("test@example.com");
    });

    const submit = findByTestId(renderer!, "email-submit");
    await act(async () => {
      await submit.props.onPress();
    });
    await act(async () => {
      jest.runAllTimers();
    });

    // Should stay on email step
    expect(findByTestId(renderer!, "email-input")).toBeTruthy();
    expect(findByTestIdOptional(renderer!, "otp-input")).toBeNull();
  });
});
