import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Mocks ---

const mockVerifyMfa = jest.fn();
const mockSignOut = jest.fn();

jest.mock("../AuthContext", () => ({
  useAuth: () => ({
    verifyMfa: mockVerifyMfa,
    signOut: mockSignOut,
    session: null,
    user: null,
    authStatus: "mfa_required" as const,
  }),
}));

jest.mock("../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      onAuthStateChange: jest.fn().mockReturnValue({
        data: { subscription: { unsubscribe: jest.fn() } },
      }),
    },
  },
}));

import MfaChallengeScreen from "../MfaChallengeScreen";

// --- Helpers ---

function findByTestId(
  renderer: ReactTestRenderer,
  testID: string
) {
  return renderer.root.findByProps({ testID });
}

function findByTestIdOptional(
  renderer: ReactTestRenderer,
  testID: string
) {
  const matches = renderer.root.findAllByProps({ testID });
  return matches.length > 0 ? matches[0] : null;
}

// --- Tests ---

describe("MfaChallengeScreen", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    mockVerifyMfa.mockResolvedValue({ error: null });
    mockSignOut.mockResolvedValue({ error: null });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("renders the title and subtitle", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const texts = renderer!.root
      .findAllByType("Text" as any)
      .map((t) => {
        const children = t.props.children;
        return typeof children === "string" ? children : null;
      })
      .filter(Boolean);

    expect(texts).toContain("Two-factor authentication");
    expect(texts).toContain(
      "Enter the 6-digit code from your authenticator app to continue."
    );
  });

  it("strips non-digit characters from input", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    await act(async () => {
      jest.runAllTimers();
    });

    const input = findByTestId(renderer!, "mfa-code-input");
    await act(async () => {
      input.props.onChangeText("12ab34");
    });

    expect(input.props.value).toBe("1234");
  });

  it("disables verify button when code is less than 6 digits", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const verifyButton = findByTestId(renderer!, "mfa-verify");
    expect(verifyButton.props.disabled).toBe(true);

    // Enter partial code
    const input = findByTestId(renderer!, "mfa-code-input");
    await act(async () => {
      input.props.onChangeText("123");
    });

    expect(findByTestId(renderer!, "mfa-verify").props.disabled).toBe(true);
  });

  it("calls verifyMfa on submit with 6-digit code", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const input = findByTestId(renderer!, "mfa-code-input");
    await act(async () => {
      input.props.onChangeText("123456");
    });

    const verifyButton = findByTestId(renderer!, "mfa-verify");
    expect(verifyButton.props.disabled).toBe(false);

    await act(async () => {
      verifyButton.props.onPress();
    });

    expect(mockVerifyMfa).toHaveBeenCalledWith("123456");
  });

  it("shows error message when verification fails", async () => {
    mockVerifyMfa.mockResolvedValue({
      error: {
        message: "Invalid code",
        name: "AuthApiError",
        status: 400,
        code: "mfa_verification_failed",
      },
    });

    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const input = findByTestId(renderer!, "mfa-code-input");
    await act(async () => {
      input.props.onChangeText("999999");
    });

    await act(async () => {
      findByTestId(renderer!, "mfa-verify").props.onPress();
    });

    const errorElement = findByTestId(renderer!, "mfa-error");
    expect(errorElement).toBeTruthy();
    expect(errorElement.props.children).toBe(
      "Invalid code. Please check your authenticator app and try again."
    );
  });

  it("calls signOut when sign out button is pressed", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const signOutButton = findByTestId(renderer!, "mfa-sign-out");
    await act(async () => {
      signOutButton.props.onPress();
    });

    expect(mockSignOut).toHaveBeenCalled();
  });

  it("does not show error message initially", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(MfaChallengeScreen));
    });

    const errorElement = findByTestIdOptional(renderer!, "mfa-error");
    expect(errorElement).toBeNull();
  });
});
