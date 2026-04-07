import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { AuthError } from "@supabase/supabase-js";

import PasswordStep from "../PasswordStep";

// --- Helpers ---

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findByProps({ testID });
}

function findByTestIdOptional(renderer: ReactTestRenderer, testID: string) {
  const results = renderer.root.findAllByProps({ testID });
  return results.length > 0 ? results[0] : null;
}

function makeProps(
  overrides?: Partial<React.ComponentProps<typeof PasswordStep>>
) {
  return {
    email: "test@example.com",
    onSubmit: jest.fn().mockResolvedValue({ error: null }),
    onBack: jest.fn(),
    ...overrides,
  };
}

// --- Tests ---

describe("PasswordStep", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("shows password input and submit button", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(PasswordStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "password-input");
    expect(input).toBeTruthy();

    const submitBtn = findByTestId(renderer!, "password-submit");
    expect(submitBtn).toBeTruthy();
  });

  it("calls onSubmit with entered password when submit is pressed", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(PasswordStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "password-input");
    await act(async () => {
      input.props.onChangeText("my-secret-password");
    });

    const submitBtn = findByTestId(renderer!, "password-submit");
    await act(async () => {
      await submitBtn.props.onPress();
    });

    expect(props.onSubmit).toHaveBeenCalledWith("my-secret-password");
  });

  it("shows error on authentication failure", async () => {
    const onSubmit = jest.fn().mockResolvedValue({
      error: {
        code: "invalid_credentials",
        message: "Invalid login credentials",
      } as AuthError,
    });
    const props = makeProps({ onSubmit });
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(PasswordStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    // No error initially
    expect(findByTestIdOptional(renderer!, "password-error")).toBeNull();

    // Type a password and submit
    const input = findByTestId(renderer!, "password-input");
    await act(async () => {
      input.props.onChangeText("wrong-password");
    });

    const submitBtn = findByTestId(renderer!, "password-submit");
    await act(async () => {
      await submitBtn.props.onPress();
    });

    const errorEl = findByTestId(renderer!, "password-error");
    expect(errorEl).toBeTruthy();
    expect(errorEl.props.children).toBeTruthy();
  });

  it("calls onBack when back button is pressed", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(PasswordStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const backBtn = findByTestId(renderer!, "password-back");
    await act(async () => {
      backBtn.props.onPress();
    });

    expect(props.onBack).toHaveBeenCalledTimes(1);
  });

  it("disables submit when password is empty", async () => {
    const props = makeProps();
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(PasswordStep, props));
    });
    await act(async () => {
      jest.runAllTimers();
    });

    const submitBtn = findByTestId(renderer!, "password-submit");
    expect(submitBtn.props.disabled).toBe(true);
  });
});
