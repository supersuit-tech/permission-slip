import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import {
  type ConsentStatus,
  getStoredConsent,
  persistConsent,
  clearConsent,
} from "../lib/consent-cookie";

interface CookieConsentState {
  /** null = user hasn't decided yet; show the banner. */
  consent: ConsentStatus | null;
  accept: () => void;
  reject: () => void;
  /** Reset consent so the banner reappears (e.g. from a settings page). */
  reset: () => void;
}

const CookieConsentContext = createContext<CookieConsentState | undefined>(
  undefined,
);

export function CookieConsentProvider({ children }: { children: ReactNode }) {
  const [consent, setConsent] = useState<ConsentStatus | null>(getStoredConsent);

  const accept = useCallback(() => {
    setConsent("accepted");
    persistConsent("accepted");
  }, []);

  const reject = useCallback(() => {
    setConsent("rejected");
    persistConsent("rejected");
  }, []);

  const reset = useCallback(() => {
    setConsent(null);
    clearConsent();
  }, []);

  return (
    <CookieConsentContext.Provider value={{ consent, accept, reject, reset }}>
      {children}
    </CookieConsentContext.Provider>
  );
}

export function useCookieConsent() {
  const context = useContext(CookieConsentContext);
  if (context === undefined) {
    throw new Error(
      "useCookieConsent must be used within a CookieConsentProvider",
    );
  }
  return context;
}
