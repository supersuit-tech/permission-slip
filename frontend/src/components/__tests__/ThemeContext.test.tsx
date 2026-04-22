import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render } from "@testing-library/react";
import { ThemeProvider, useTheme } from "../ThemeContext";

function TestComponent() {
  const { theme, preference, setPreference } = useTheme();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <span data-testid="preference">{preference}</span>
      <button onClick={() => setPreference("light")}>Pick Light</button>
      <button onClick={() => setPreference("dark")}>Pick Dark</button>
      <button onClick={() => setPreference("system")}>Pick System</button>
    </div>
  );
}

function renderWithTheme() {
  return render(
    <ThemeProvider>
      <TestComponent />
    </ThemeProvider>
  );
}

type Listener = (event: MediaQueryListEvent) => void;

interface FakeMediaQueryList {
  matches: boolean;
  addEventListener: (type: "change", listener: Listener) => void;
  removeEventListener: (type: "change", listener: Listener) => void;
  emit: (matches: boolean) => void;
}

function mockMatchMedia(prefersDark: boolean): FakeMediaQueryList {
  const listeners = new Set<Listener>();
  const mql: FakeMediaQueryList = {
    matches: prefersDark,
    addEventListener: (_type, listener) => {
      listeners.add(listener);
    },
    removeEventListener: (_type, listener) => {
      listeners.delete(listener);
    },
    emit: (matches: boolean) => {
      mql.matches = matches;
      const event = { matches } as MediaQueryListEvent;
      listeners.forEach((l) => l(event));
    },
  };
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: vi.fn().mockReturnValue(mql),
  });
  return mql;
}

describe("ThemeContext", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    mockMatchMedia(false);
  });

  afterEach(() => {
    document.documentElement.classList.remove("dark");
  });

  it("defaults preference to system when nothing is stored", () => {
    renderWithTheme();
    expect(screen.getByTestId("preference")).toHaveTextContent("system");
    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("resolves to dark when system prefers dark and preference is system", () => {
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("preference")).toHaveTextContent("system");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("does not persist the system-derived theme on initial mount", () => {
    mockMatchMedia(true);
    renderWithTheme();
    expect(localStorage.getItem("permission-slip-theme")).toBeNull();
  });

  it("uses stored light preference over system preference", () => {
    mockMatchMedia(true);
    localStorage.setItem("permission-slip-theme", "light");
    renderWithTheme();
    expect(screen.getByTestId("preference")).toHaveTextContent("light");
    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("uses stored dark preference over system preference", () => {
    localStorage.setItem("permission-slip-theme", "dark");
    renderWithTheme();
    expect(screen.getByTestId("preference")).toHaveTextContent("dark");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("stores and applies an explicit light preference", async () => {
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");

    await userEvent.click(screen.getByText("Pick Light"));

    expect(screen.getByTestId("preference")).toHaveTextContent("light");
    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(localStorage.getItem("permission-slip-theme")).toBe("light");
  });

  it("stores and applies an explicit dark preference", async () => {
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("light");

    await userEvent.click(screen.getByText("Pick Dark"));

    expect(screen.getByTestId("preference")).toHaveTextContent("dark");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("permission-slip-theme")).toBe("dark");
  });

  it("follows OS changes in real time when preference is system", async () => {
    const mql = mockMatchMedia(false);
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("light");

    act(() => mql.emit(true));
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);

    act(() => mql.emit(false));
    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("ignores OS changes when an explicit preference is set", async () => {
    const mql = mockMatchMedia(false);
    renderWithTheme();

    await userEvent.click(screen.getByText("Pick Light"));
    act(() => mql.emit(true));

    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("switching back to system re-resolves from the OS", async () => {
    mockMatchMedia(true);
    localStorage.setItem("permission-slip-theme", "light");
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("light");

    await userEvent.click(screen.getByText("Pick System"));

    expect(screen.getByTestId("preference")).toHaveTextContent("system");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("permission-slip-theme")).toBe("system");
  });

  it("falls back to system preference when localStorage throws on read", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("storage disabled");
    });
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("preference")).toHaveTextContent("system");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    vi.restoreAllMocks();
    mockMatchMedia(false);
  });

  it("still applies the preference when localStorage throws on write", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("storage disabled");
    });
    renderWithTheme();

    await userEvent.click(screen.getByText("Pick Dark"));

    expect(screen.getByTestId("preference")).toHaveTextContent("dark");
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    vi.restoreAllMocks();
  });

  it("throws when useTheme is used outside ThemeProvider", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TestComponent />)).toThrow(
      "useTheme must be used within a ThemeProvider"
    );
    consoleSpy.mockRestore();
  });
});
