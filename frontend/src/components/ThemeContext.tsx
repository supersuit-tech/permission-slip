import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useState,
  type ReactNode,
} from "react";

type ResolvedTheme = "light" | "dark";
export type ThemePreference = "light" | "dark" | "system";

interface ThemeState {
  theme: ResolvedTheme;
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
}

const ThemeContext = createContext<ThemeState | undefined>(undefined);

const STORAGE_KEY = "permission-slip-theme";
const SYSTEM_QUERY = "(prefers-color-scheme: dark)";

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia?.(SYSTEM_QUERY)?.matches ? "dark" : "light";
}

function getInitialPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark" || stored === "system") {
      return stored;
    }
  } catch {
    // Storage may be disabled (private mode, quota exceeded, etc.)
  }
  return "system";
}

function resolveTheme(preference: ThemePreference): ResolvedTheme {
  return preference === "system" ? getSystemTheme() : preference;
}

function applyTheme(theme: ResolvedTheme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreferenceState] = useState<ThemePreference>(
    getInitialPreference,
  );
  const [theme, setTheme] = useState<ResolvedTheme>(() =>
    resolveTheme(getInitialPreference()),
  );

  useLayoutEffect(() => {
    const resolved = resolveTheme(preference);
    setTheme(resolved);
    applyTheme(resolved);
  }, [preference]);

  // When the user picked "system", follow OS changes in real time.
  useEffect(() => {
    if (preference !== "system") return;
    const mql = window.matchMedia?.(SYSTEM_QUERY);
    if (!mql) return;
    const handler = (event: MediaQueryListEvent) => {
      const next: ResolvedTheme = event.matches ? "dark" : "light";
      setTheme(next);
      applyTheme(next);
    };
    mql.addEventListener?.("change", handler);
    return () => mql.removeEventListener?.("change", handler);
  }, [preference]);

  const setPreference = useCallback((next: ThemePreference) => {
    setPreferenceState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Storage may be unavailable; preference still applies for the session.
    }
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, preference, setPreference }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
}
