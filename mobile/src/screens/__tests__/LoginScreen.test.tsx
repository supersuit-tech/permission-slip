import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

const mockSignInWithPassword = jest.fn();
const mockSignUpWithPassword = jest.fn();
const mockOpenServerSetup = jest.fn();

jest.mock("../../auth/AuthContext", () => ({
  useAuth: () => ({
    signInWithPassword: mockSignInWithPassword,
    signUpWithPassword: mockSignUpWithPassword,
    session: null,
    user: null,
    authStatus: "unauthenticated" as const,
  }),
}));

jest.mock("../../lib/serverSetupContext", () => ({
  useServerSetup: () => ({ openServerSetup: mockOpenServerSetup }),
}));

import LoginScreen from "../LoginScreen";

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findByProps({ testID });
}

describe("LoginScreen", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockSignInWithPassword.mockResolvedValue({ error: null });
    mockSignUpWithPassword.mockResolvedValue({ error: null });
  });

  it("shows email and password fields", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });
    expect(findByTestId(renderer!, "email-input")).toBeTruthy();
    expect(findByTestId(renderer!, "password-input")).toBeTruthy();
  });

  it("calls signInWithPassword on submit in sign-in mode", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });

    await act(async () => {
      findByTestId(renderer!, "email-input").props.onChangeText("a@b.co");
    });
    await act(async () => {
      findByTestId(renderer!, "password-input").props.onChangeText("password12345");
    });

    await act(async () => {
      await findByTestId(renderer!, "login-submit").props.onPress();
    });

    expect(mockSignInWithPassword).toHaveBeenCalledWith("a@b.co", "password12345");
    expect(mockSignUpWithPassword).not.toHaveBeenCalled();
  });

  it("calls signUpWithPassword after switching to create account", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });

    await act(async () => {
      findByTestId(renderer!, "mode-signup").props.onPress();
    });

    await act(async () => {
      findByTestId(renderer!, "email-input").props.onChangeText("new@b.co");
    });
    await act(async () => {
      findByTestId(renderer!, "password-input").props.onChangeText("password12345");
    });

    await act(async () => {
      await findByTestId(renderer!, "login-submit").props.onPress();
    });

    expect(mockSignUpWithPassword).toHaveBeenCalledWith("new@b.co", "password12345");
  });

  it("opens the server setup overlay when the Change server URL link is tapped", async () => {
    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(createElement(LoginScreen));
    });

    await act(async () => {
      findByTestId(renderer!, "login-change-server").props.onPress();
    });

    expect(mockOpenServerSetup).toHaveBeenCalledTimes(1);
  });
});
