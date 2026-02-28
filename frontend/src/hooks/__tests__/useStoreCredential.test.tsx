import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPost, resetClientMocks } from "../../api/__mocks__/client";
import { useStoreCredential } from "../useStoreCredential";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useStoreCredential", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends store request with correct body", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: {
        id: "cred_abc123",
        service: "github",
        label: "Personal Access Token",
        created_at: "2026-02-11T10:00:00Z",
      },
    });

    const { result } = renderHook(() => useStoreCredential(), { wrapper });

    let response: unknown;
    await act(async () => {
      response = await result.current.storeCredential({
        service: "github",
        credentials: { api_key: "ghp_abc123" },
        label: "Personal Access Token",
      });
    });

    expect(mockPost).toHaveBeenCalledWith("/v1/credentials", {
      headers: { Authorization: "Bearer token" },
      body: {
        service: "github",
        credentials: { api_key: "ghp_abc123" },
        label: "Personal Access Token",
      },
    });
    expect(response).toEqual({
      id: "cred_abc123",
      service: "github",
      label: "Personal Access Token",
      created_at: "2026-02-11T10:00:00Z",
    });
    expect(result.current.isLoading).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useStoreCredential(), { wrapper });

    await expect(
      result.current.storeCredential({
        service: "github",
        credentials: { api_key: "ghp_abc123" },
      }),
    ).rejects.toThrow("Not authenticated");
  });

  it("surfaces server error message on structured error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: {
        error: {
          code: "duplicate_credential",
          message: "Credentials already stored",
        },
      },
    });

    const { result } = renderHook(() => useStoreCredential(), { wrapper });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.storeCredential({
          service: "github",
          credentials: { api_key: "ghp_abc123" },
        });
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Credentials already stored");
  });

  it("falls back to generic message on unstructured error", async () => {
    setupAuthMocks({ authenticated: true });
    mockPost.mockResolvedValue({
      data: undefined,
      error: "unexpected error shape",
    });

    const { result } = renderHook(() => useStoreCredential(), { wrapper });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.storeCredential({
          service: "github",
          credentials: { api_key: "ghp_abc123" },
        });
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to store credential");
  });
});
