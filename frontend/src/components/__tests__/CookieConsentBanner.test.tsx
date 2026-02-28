import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render } from "@testing-library/react";
import { CookieConsentProvider } from "../CookieConsentContext";
import { CookieConsentBanner } from "../CookieConsentBanner";
import { CONSENT_COOKIE_NAME } from "../../lib/consent-cookie";
import {
  setCookie,
  getCookie,
  clearConsentCookie,
} from "../../test-cookie-helpers";

function renderBanner() {
  return render(
    <CookieConsentProvider>
      <CookieConsentBanner />
    </CookieConsentProvider>,
  );
}

describe("CookieConsentBanner", () => {
  beforeEach(() => {
    clearConsentCookie();
  });

  afterEach(() => {
    clearConsentCookie();
  });

  it("renders the banner when no consent has been given", () => {
    renderBanner();
    expect(screen.getByRole("region", { name: /cookie consent/i })).toBeInTheDocument();
    expect(screen.getByText(/we use cookies/i)).toBeInTheDocument();
    expect(screen.getByText("Accept All")).toBeInTheDocument();
    expect(screen.getByText("Reject All")).toBeInTheDocument();
  });

  it("hides the banner after accepting", async () => {
    renderBanner();
    await userEvent.click(screen.getByText("Accept All"));
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
    expect(getCookie(CONSENT_COOKIE_NAME)).toBe("accepted");
  });

  it("hides the banner after rejecting", async () => {
    renderBanner();
    await userEvent.click(screen.getByText("Reject All"));
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
    expect(getCookie(CONSENT_COOKIE_NAME)).toBe("rejected");
  });

  it("does not render when consent was previously accepted", () => {
    setCookie(CONSENT_COOKIE_NAME, "accepted");
    renderBanner();
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("does not render when consent was previously rejected", () => {
    setCookie(CONSENT_COOKIE_NAME, "rejected");
    renderBanner();
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("contains a link to the privacy policy", () => {
    renderBanner();
    const link = screen.getByRole("link", { name: /privacy policy/i });
    expect(link).toHaveAttribute("href", "/policy/privacy");
  });
});
