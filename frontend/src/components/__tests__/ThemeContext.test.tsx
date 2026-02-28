import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render } from "@testing-library/react";
import { ThemeProvider, useTheme } from "../ThemeContext";

function TestComponent() {
  const { theme, toggleTheme } = useTheme();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <button onClick={toggleTheme}>Toggle</button>
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

function mockMatchMedia(prefersDark: boolean) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: vi.fn().mockReturnValue({ matches: prefersDark }),
  });
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

  it("defaults to light when no stored preference and system prefers light", () => {
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("defaults to dark when system prefers dark", () => {
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("does not persist system-derived theme on initial mount", () => {
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(localStorage.getItem("permission-slip-theme")).toBeNull();
  });

  it("uses stored preference over system preference", () => {
    localStorage.setItem("permission-slip-theme", "dark");
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("toggles from light to dark", async () => {
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("light");

    await userEvent.click(screen.getByText("Toggle"));

    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("permission-slip-theme")).toBe("dark");
  });

  it("toggles from dark to light", async () => {
    localStorage.setItem("permission-slip-theme", "dark");
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");

    await userEvent.click(screen.getByText("Toggle"));

    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(localStorage.getItem("permission-slip-theme")).toBe("light");
  });

  it("falls back to system preference when localStorage throws", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("storage disabled");
    });
    mockMatchMedia(true);
    renderWithTheme();
    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    vi.restoreAllMocks();
    mockMatchMedia(false);
  });

  it("throws when useTheme is used outside ThemeProvider", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TestComponent />)).toThrow(
      "useTheme must be used within a ThemeProvider"
    );
    consoleSpy.mockRestore();
  });
});
