import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { AuthProvider } from "@/auth/AuthContext";
import { ThemeProvider } from "@/components/ThemeContext";
import { Toaster } from "@/components/ui/sonner";

/**
 * Creates a test wrapper with a fresh QueryClient and MemoryRouter.
 *
 * The QueryClient is created once per call (stable across re-renders within a
 * single test) but fresh across tests when called in beforeEach or inline.
 *
 * Pass `initialEntries` to set the initial route (e.g. ["/agents/42"]).
 */
export function createAuthWrapper(initialEntries?: string[]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function AuthWrapper({ children }: { children: React.ReactNode }) {
    return (
      <MemoryRouter initialEntries={initialEntries}>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>{children}</AuthProvider>
        </QueryClientProvider>
      </MemoryRouter>
    );
  };
}

/**
 * Render helper that wraps the component in all app-level providers
 * (QueryClient, ThemeProvider, AuthProvider, MemoryRouter, Toaster).
 *
 * QueryClient is created per call (stable across re-renders, fresh per test).
 */
export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  function AllProviders({ children }: { children: React.ReactNode }) {
    return (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <AuthProvider>
              {children}
              <Toaster />
            </AuthProvider>
          </ThemeProvider>
        </QueryClientProvider>
      </MemoryRouter>
    );
  }
  return render(ui, { wrapper: AllProviders, ...options });
}
