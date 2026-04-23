import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import {
  BETA_SUPPORT_EMAIL,
  GoogleBetaInlineNote,
  GoogleBetaNoticeDialog,
} from "../GoogleBetaNoticeDialog";

describe("GoogleBetaNoticeDialog", () => {
  it("renders the three key warnings when open", () => {
    render(
      <GoogleBetaNoticeDialog
        open
        onOpenChange={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(
      screen.getByText("Your Google account must be added to the beta"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Google will say "Google hasn't verified this app"/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/You'll need to reconnect every ~7 days/),
    ).toBeInTheDocument();
  });

  it("includes a mailto: link to support", () => {
    render(
      <GoogleBetaNoticeDialog
        open
        onOpenChange={() => {}}
        onContinue={() => {}}
      />,
    );

    const link = screen.getByRole("link", { name: BETA_SUPPORT_EMAIL });
    expect(link.getAttribute("href")).toMatch(
      new RegExp(`^mailto:${BETA_SUPPORT_EMAIL}`),
    );
  });

  it("invokes onContinue when the continue button is clicked", async () => {
    const user = userEvent.setup();
    const onContinue = vi.fn();
    render(
      <GoogleBetaNoticeDialog
        open
        onOpenChange={() => {}}
        onContinue={onContinue}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: /continue to google/i }),
    );
    expect(onContinue).toHaveBeenCalledTimes(1);
  });

  it("labels the button differently in reconnect mode", () => {
    render(
      <GoogleBetaNoticeDialog
        open
        mode="reconnect"
        onOpenChange={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(
      screen.getByRole("button", { name: /continue to reconnect/i }),
    ).toBeInTheDocument();
  });

  it("invokes onOpenChange(false) when Cancel is clicked", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    render(
      <GoogleBetaNoticeDialog
        open
        onOpenChange={onOpenChange}
        onContinue={() => {}}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

describe("GoogleBetaInlineNote", () => {
  it("renders the beta support email", () => {
    render(<GoogleBetaInlineNote />);
    const link = screen.getByRole("link", { name: BETA_SUPPORT_EMAIL });
    expect(link.getAttribute("href")).toBe(`mailto:${BETA_SUPPORT_EMAIL}`);
  });
});
