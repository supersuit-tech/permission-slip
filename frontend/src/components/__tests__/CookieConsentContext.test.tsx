import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render } from "@testing-library/react";
import { CookieConsentProvider, useCookieConsent } from "../CookieConsentContext";
import { CONSENT_COOKIE_NAME } from "../../lib/consent-cookie";
import {
  setCookie,
  getCookie,
  clearConsentCookie,
} from "../../test-cookie-helpers";

function TestComponent() {
  const { consent, accept, reject, reset } = useCookieConsent();
  return (
    <div>
      <span data-testid="consent">{consent ?? "null"}</span>
      <button onClick={accept}>Accept</button>
      <button onClick={reject}>Reject</button>
      <button onClick={reset}>Reset</button>
    </div>
  );
}

function renderWithProvider() {
  return render(
    <CookieConsentProvider>
      <TestComponent />
    </CookieConsentProvider>,
  );
}

describe("CookieConsentContext", () => {
  beforeEach(() => {
    clearConsentCookie();
  });

  afterEach(() => {
    clearConsentCookie();
  });

  it("defaults to null when no stored consent", () => {
    renderWithProvider();
    expect(screen.getByTestId("consent")).toHaveTextContent("null");
  });

  it("reads stored 'accepted' consent on mount", () => {
    setCookie(CONSENT_COOKIE_NAME, "accepted");
    renderWithProvider();
    expect(screen.getByTestId("consent")).toHaveTextContent("accepted");
  });

  it("reads stored 'rejected' consent on mount", () => {
    setCookie(CONSENT_COOKIE_NAME, "rejected");
    renderWithProvider();
    expect(screen.getByTestId("consent")).toHaveTextContent("rejected");
  });

  it("ignores invalid stored values", () => {
    setCookie(CONSENT_COOKIE_NAME, "maybe");
    renderWithProvider();
    expect(screen.getByTestId("consent")).toHaveTextContent("null");
  });

  it("sets consent to accepted and persists it", async () => {
    renderWithProvider();
    await userEvent.click(screen.getByText("Accept"));
    expect(screen.getByTestId("consent")).toHaveTextContent("accepted");
    expect(getCookie(CONSENT_COOKIE_NAME)).toBe("accepted");
  });

  it("sets consent to rejected and persists it", async () => {
    renderWithProvider();
    await userEvent.click(screen.getByText("Reject"));
    expect(screen.getByTestId("consent")).toHaveTextContent("rejected");
    expect(getCookie(CONSENT_COOKIE_NAME)).toBe("rejected");
  });

  it("resets consent and clears cookie", async () => {
    setCookie(CONSENT_COOKIE_NAME, "accepted");
    renderWithProvider();
    expect(screen.getByTestId("consent")).toHaveTextContent("accepted");

    await userEvent.click(screen.getByText("Reset"));
    expect(screen.getByTestId("consent")).toHaveTextContent("null");
    expect(getCookie(CONSENT_COOKIE_NAME)).toBeNull();
  });

  it("falls back to null when document.cookie throws", () => {
    const original = Object.getOwnPropertyDescriptor(Document.prototype, "cookie");
    Object.defineProperty(document, "cookie", {
      get() {
        throw new Error("cookies disabled");
      },
      configurable: true,
    });
    try {
      renderWithProvider();
      expect(screen.getByTestId("consent")).toHaveTextContent("null");
    } finally {
      if (original) {
        Object.defineProperty(document, "cookie", original);
      } else {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete (document as unknown as Record<string, unknown>)["cookie"];
      }
    }
  });

  it("throws when useCookieConsent is used outside provider", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TestComponent />)).toThrow(
      "useCookieConsent must be used within a CookieConsentProvider",
    );
    consoleSpy.mockRestore();
  });
});
