import React, { createElement } from "react";
import { Alert } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// --- Mocks ---

const mockDenyApproval = jest.fn();
let mockIsDenying = false;

jest.mock("../../../hooks/useDenyApproval", () => ({
  useDenyApproval: () => ({
    denyApproval: mockDenyApproval,
    isPending: mockIsDenying,
  }),
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  SafeAreaProvider: ({ children }: { children: React.ReactNode }) => children,
}));

import { DenyAction } from "../DenyAction";

// --- Helpers ---

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findAll(
    (node) => node.props.testID === testID && typeof node.props.onPress === "function",
  );
}

function hasTestId(renderer: ReactTestRenderer, testID: string): boolean {
  const json = JSON.stringify(renderer.toJSON());
  return json.includes(`"testID":"${testID}"`);
}

/** Press the deny button, then confirm in the Alert dialog. */
async function pressAndConfirmDeny(renderer: ReactTestRenderer, alertSpy: jest.SpyInstance) {
  const denyButton = findByTestId(renderer, "deny-button")[0];
  await act(async () => {
    denyButton!.props.onPress();
  });

  const alertButtons = alertSpy.mock.calls[0]![2] as Array<{
    text: string;
    onPress?: () => void;
  }>;
  const confirmButton = alertButtons.find((b) => b.text === "Deny");
  await act(async () => {
    await confirmButton!.onPress!();
  });
}

// --- Tests ---

describe("DenyAction", () => {
  let renderer: ReactTestRenderer;
  const mockOnDenied = jest.fn();

  beforeEach(() => {
    jest.useFakeTimers();
    mockDenyApproval.mockReset();
    mockOnDenied.mockReset();
    mockIsDenying = false;
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
    jest.useRealTimers();
  });

  it("renders deny button", async () => {
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });
    expect(hasTestId(renderer, "deny-button")).toBe(true);
  });

  it("shows loading spinner and disables button when isDenying is true", async () => {
    mockIsDenying = true;
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    // Should show the loading indicator
    expect(hasTestId(renderer, "deny-loading")).toBe(true);

    // The deny button should be disabled
    const denyNodes = renderer.root.findAll(
      (node) => node.props.testID === "deny-button",
    );
    const pressableNode = denyNodes.find(
      (node) => node.props.disabled !== undefined,
    );
    expect(pressableNode?.props.disabled).toBe(true);
  });

  it("shows confirmation alert when deny button is pressed", async () => {
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    const denyButton = findByTestId(renderer, "deny-button")[0];
    await act(async () => {
      denyButton!.props.onPress();
    });

    expect(alertSpy).toHaveBeenCalledWith(
      "Deny Request",
      "Are you sure you want to deny this request?",
      expect.arrayContaining([
        expect.objectContaining({ text: "Cancel", style: "cancel" }),
        expect.objectContaining({ text: "Deny", style: "destructive" }),
      ]),
    );
    alertSpy.mockRestore();
  });

  it("calls denyApproval with correct ID when confirmed", async () => {
    mockDenyApproval.mockResolvedValue(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_xyz", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    expect(mockDenyApproval).toHaveBeenCalledWith("appr_xyz");
    alertSpy.mockRestore();
  });

  it("shows denied confirmation after successful deny", async () => {
    mockDenyApproval.mockResolvedValue(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    expect(hasTestId(renderer, "denied-confirmation")).toBe(true);
    expect(hasTestId(renderer, "deny-button")).toBe(false);
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Request Denied");
    expect(json).toContain("Back to List");
    alertSpy.mockRestore();
  });

  it("auto-navigates after 1.5s delay on successful deny", async () => {
    mockDenyApproval.mockResolvedValue(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    // Should not have called onDenied yet
    expect(mockOnDenied).not.toHaveBeenCalled();

    // Advance timers by 1.5s
    await act(async () => {
      jest.advanceTimersByTime(1500);
    });

    expect(mockOnDenied).toHaveBeenCalledTimes(1);
    alertSpy.mockRestore();
  });

  it("back-to-list button calls onDenied immediately", async () => {
    mockDenyApproval.mockResolvedValue(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    const backButton = findByTestId(renderer, "back-to-list-button")[0];
    await act(async () => {
      backButton!.props.onPress();
    });

    expect(mockOnDenied).toHaveBeenCalledTimes(1);
    alertSpy.mockRestore();
  });

  it("shows error alert on deny failure and keeps button visible", async () => {
    mockDenyApproval.mockRejectedValue(new Error("Network error"));
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    expect(alertSpy).toHaveBeenCalledWith(
      "Error",
      "Failed to deny request. Please try again.",
    );

    // Button should still be visible
    expect(hasTestId(renderer, "deny-button")).toBe(true);
    expect(hasTestId(renderer, "denied-confirmation")).toBe(false);
    alertSpy.mockRestore();
  });

  it("cleans up auto-navigate timer on unmount", async () => {
    mockDenyApproval.mockResolvedValue(undefined);
    const alertSpy = jest.spyOn(Alert, "alert");
    await act(async () => {
      renderer = create(
        createElement(DenyAction, { approvalId: "appr_1", onDenied: mockOnDenied }),
      );
    });

    await pressAndConfirmDeny(renderer, alertSpy);

    // Unmount before timer fires
    await act(async () => {
      renderer.unmount();
    });

    // Advance timers — onDenied should NOT be called since component unmounted
    jest.advanceTimersByTime(2000);
    expect(mockOnDenied).not.toHaveBeenCalled();
    alertSpy.mockRestore();
  });
});
