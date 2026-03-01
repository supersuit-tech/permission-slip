import "@testing-library/jest-dom/vitest";

// jsdom doesn't implement ResizeObserver — stub it for Radix UI primitives
// (e.g. Checkbox) that rely on it.
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// jsdom doesn't implement matchMedia — stub it for ThemeProvider and any
// component that checks prefers-color-scheme.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// Node.js v22+ exposes a stub `localStorage` global that overwrites jsdom's
// real Storage implementation when vitest copies Node globals into the jsdom
// window.  Replace it with a proper in-memory Storage so tests can call
// setItem / getItem / removeItem / clear as expected.
(function installLocalStorage() {
  const store: Record<string, string> = {};
  const storage: Storage = {
    get length() {
      return Object.keys(store).length;
    },
    key(index: number): string | null {
      return Object.keys(store)[index] ?? null;
    },
    getItem(key: string): string | null {
      return Object.prototype.hasOwnProperty.call(store, key)
        ? (store[key] ?? null)
        : null;
    },
    setItem(key: string, value: string): void {
      store[key] = String(value);
    },
    removeItem(key: string): void {
      delete store[key];
    },
    clear(): void {
      Object.keys(store).forEach((k) => delete store[k]);
    },
  };

  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    enumerable: true,
    get: () => storage,
  });
})();
