import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { tryAppReviewLogin } from "../appReviewAuth";
import { supabase } from "../../lib/supabaseClient";

vi.mock("../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      setSession: vi.fn(),
    },
  },
}));

const mockSetSession = vi.mocked(supabase.auth.setSession);

describe("tryAppReviewLogin", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.stubEnv("VITE_API_BASE_URL", "/api");
    vi.clearAllMocks();
    globalThis.fetch = vi.fn() as typeof fetch;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.unstubAllEnvs();
  });

  it("POSTs to app-review-login and sets Supabase session on success", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        access_token: "at",
        refresh_token: "rt",
      }),
    } as Response);
    mockSetSession.mockResolvedValue({
      data: { session: null, user: null },
      error: null,
    });

    const result = await tryAppReviewLogin("review@example.com", "static-code");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "http://localhost/api/v1/auth/app-review-login",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: "review@example.com",
          otp: "static-code",
        }),
      })
    );
    expect(mockSetSession).toHaveBeenCalledWith({
      access_token: "at",
      refresh_token: "rt",
    });
    expect(result.error).toBeNull();
  });

  it("returns invalid_credentials-shaped error when response is not ok", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: false,
      status: 401,
    } as Response);

    const result = await tryAppReviewLogin("a@b.com", "wrong");

    expect(mockSetSession).not.toHaveBeenCalled();
    expect(result.error?.code).toBe("invalid_credentials");
    expect(result.error?.status).toBe(401);
  });

  it("returns network_error when fetch throws", async () => {
    vi.mocked(globalThis.fetch).mockRejectedValue(new Error("offline"));

    const result = await tryAppReviewLogin("a@b.com", "x");

    expect(result.error?.code).toBe("network_error");
  });
});
