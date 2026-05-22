import React, { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { ApprovalActions } from "../ApprovalActions";

function renderActions(
  overrides: Partial<React.ComponentProps<typeof ApprovalActions>> = {},
) {
  const props = {
    onApprove: jest.fn(),
    onDeny: jest.fn(),
    isApproving: false,
    isDenying: false,
    disabled: false,
    ...overrides,
  };
  return create(createElement(ApprovalActions, props));
}

function hasTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.findAll((node) => node.props.testID === testID).length > 0;
}

describe("ApprovalActions — auto-approve checkbox", () => {
  let renderer: ReactTestRenderer;

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("shows checkbox when showAutoApproveCheckbox is true", async () => {
    await act(async () => {
      renderer = renderActions({ showAutoApproveCheckbox: true });
    });
    expect(hasTestId(renderer, "auto-approve-checkbox")).toBe(true);
  });

  it("hides checkbox when showAutoApproveCheckbox is false", async () => {
    await act(async () => {
      renderer = renderActions({ showAutoApproveCheckbox: false });
    });
    expect(hasTestId(renderer, "auto-approve-checkbox")).toBe(false);
  });

  it("calls onAutoApproveFutureChange when checkbox is pressed", async () => {
    const onAutoApproveFutureChange = jest.fn();
    await act(async () => {
      renderer = renderActions({
        showAutoApproveCheckbox: true,
        autoApproveFuture: false,
        onAutoApproveFutureChange,
      });
    });

    const checkbox = renderer.root.findByProps({ testID: "auto-approve-checkbox" });
    await act(async () => {
      checkbox.props.onPress();
    });

    expect(onAutoApproveFutureChange).toHaveBeenCalledWith(true);
  });
});
