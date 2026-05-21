import { createElement } from "react";
import { Alert } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

const mockSetCustomHostConfig = jest.fn((..._args: unknown[]) => Promise.resolve());
const mockClearCustomHostConfig = jest.fn(() => Promise.resolve());
const mockClearStoredRefreshToken = jest.fn(() => Promise.resolve());

jest.mock("../../lib/customHostConfig", () => ({
  setCustomHostConfig: (host: string | null) =>
    mockSetCustomHostConfig(host),
  clearCustomHostConfig: () => mockClearCustomHostConfig(),
}));

jest.mock("../../lib/authStorage", () => ({
  clearStoredRefreshToken: () => mockClearStoredRefreshToken(),
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
}));

import ServerUrlSetupScreen from "../ServerUrlSetupScreen";

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findByProps({ testID });
}

describe("ServerUrlSetupScreen", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("saves the host URL and clears the stored refresh token", async () => {
    const onComplete = jest.fn();

    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(
        createElement(ServerUrlSetupScreen, { onComplete }),
      );
    });

    await act(async () => {
      findByTestId(renderer!, "server-url-setup-input").props.onChangeText(
        "https://my-pi.example.com:8080/api",
      );
    });
    await act(async () => {
      await findByTestId(renderer!, "server-url-setup-continue").props.onPress();
    });

    expect(mockSetCustomHostConfig).toHaveBeenCalledWith(
      "https://my-pi.example.com:8080/api",
    );
    expect(mockClearStoredRefreshToken).toHaveBeenCalled();
    expect(onComplete).toHaveBeenCalledTimes(1);
  });

  it("rejects URLs that do not parse", async () => {
    const alertSpy = jest.spyOn(Alert, "alert").mockImplementation(() => {});
    const onComplete = jest.fn();

    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(
        createElement(ServerUrlSetupScreen, { onComplete }),
      );
    });

    await act(async () => {
      findByTestId(renderer!, "server-url-setup-input").props.onChangeText(
        "not a url",
      );
    });
    await act(async () => {
      await findByTestId(renderer!, "server-url-setup-continue").props.onPress();
    });

    expect(alertSpy).toHaveBeenCalled();
    expect(mockSetCustomHostConfig).not.toHaveBeenCalled();
    expect(onComplete).not.toHaveBeenCalled();
    alertSpy.mockRestore();
  });

  it("shows a Cancel button only when onCancel is provided", async () => {
    const onComplete = jest.fn();
    const onCancel = jest.fn();

    let withCancel: ReactTestRenderer;
    await act(async () => {
      withCancel = create(
        createElement(ServerUrlSetupScreen, { onComplete, onCancel }),
      );
    });
    const cancelButton = withCancel!.root.findByProps({
      testID: "server-url-setup-cancel",
    });
    await act(async () => {
      cancelButton.props.onPress();
    });
    expect(onCancel).toHaveBeenCalledTimes(1);

    let withoutCancel: ReactTestRenderer;
    await act(async () => {
      withoutCancel = create(
        createElement(ServerUrlSetupScreen, { onComplete }),
      );
    });
    expect(() =>
      withoutCancel!.root.findByProps({ testID: "server-url-setup-cancel" }),
    ).toThrow();
  });

  it("clears all saved connection state when Reset is confirmed", async () => {
    const onComplete = jest.fn();
    const alertSpy = jest
      .spyOn(Alert, "alert")
      .mockImplementation((_t, _m, buttons) => {
        // Simulate the user tapping "Reset" in the confirmation dialog.
        const reset = buttons?.find((b) => b.text === "Reset");
        reset?.onPress?.();
      });

    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(
        createElement(ServerUrlSetupScreen, {
          onComplete,
          initialHostUrl: "https://old.example/api",
          onCancel: () => {},
        }),
      );
    });

    await act(async () => {
      await findByTestId(renderer!, "server-url-setup-reset").props.onPress();
    });

    expect(mockClearCustomHostConfig).toHaveBeenCalledTimes(1);
    expect(mockClearStoredRefreshToken).toHaveBeenCalledTimes(1);
    expect(onComplete).toHaveBeenCalledTimes(1);
    alertSpy.mockRestore();
  });
});
