import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useOAuthCallbackToast } from "../useOAuthCallbackToast";

const saasMode = vi.hoisted(() => ({ isSaas: true }));
const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));

vi.mock("@/lib/saas", () => ({
  get isSaas() {
    return saasMode.isSaas;
  },
}));

vi.mock("sonner", () => ({
  toast: {
    get success() {
      return toastMocks.success;
    },
    get error() {
      return toastMocks.error;
    },
  },
}));

function Harness() {
  useOAuthCallbackToast();
  return null;
}

function renderWithOauthUrl(search: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/${search}`]}>
        <Harness />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("useOAuthCallbackToast", () => {
  beforeEach(() => {
    saasMode.isSaas = true;
    toastMocks.success.mockClear();
    toastMocks.error.mockClear();
  });

  it("does not add hosted-beta hint for Google access_denied when not SaaS", async () => {
    saasMode.isSaas = false;
    renderWithOauthUrl(
      "?oauth_status=error&oauth_provider=google&oauth_error=access_denied",
    );

    await waitFor(() => {
      expect(toastMocks.error).toHaveBeenCalled();
    });
    expect(toastMocks.error).toHaveBeenCalledWith(
      expect.stringContaining("Failed to connect Google"),
      { description: undefined },
    );
  });

  it("adds hosted-beta hint for Google access_denied when SaaS", async () => {
    saasMode.isSaas = true;
    renderWithOauthUrl(
      "?oauth_status=error&oauth_provider=google&oauth_error=access_denied",
    );

    await waitFor(() => {
      expect(toastMocks.error).toHaveBeenCalled();
    });
    expect(toastMocks.error).toHaveBeenCalledWith(
      expect.stringContaining("Failed to connect Google"),
      {
        description: expect.stringContaining("support@supersuit.tech"),
      },
    );
  });
});
