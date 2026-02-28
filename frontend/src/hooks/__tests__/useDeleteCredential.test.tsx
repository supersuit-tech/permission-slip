import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockDelete, resetClientMocks } from "../../api/__mocks__/client";
import { useDeleteCredential } from "../useDeleteCredential";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

describe("useDeleteCredential", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends delete request with correct path params", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: {
        id: "cred_abc123",
        deleted_at: "2026-02-11T15:00:00Z",
      },
    });

    const { result } = renderHook(() => useDeleteCredential(), { wrapper });

    let response: unknown;
    await act(async () => {
      response = await result.current.deleteCredential("cred_abc123");
    });

    expect(mockDelete).toHaveBeenCalledWith(
      "/v1/credentials/{credential_id}",
      {
        headers: { Authorization: "Bearer token" },
        params: {
          path: { credential_id: "cred_abc123" },
        },
      },
    );
    expect(response).toEqual({
      id: "cred_abc123",
      deleted_at: "2026-02-11T15:00:00Z",
    });
    expect(result.current.isLoading).toBe(false);
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useDeleteCredential(), { wrapper });

    await expect(
      result.current.deleteCredential("cred_abc123"),
    ).rejects.toThrow("Not authenticated");
  });

  it("surfaces server error message on structured error", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: undefined,
      error: {
        error: {
          code: "credential_not_found",
          message: "Credential ID not found",
        },
      },
    });

    const { result } = renderHook(() => useDeleteCredential(), { wrapper });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.deleteCredential("cred_abc123");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Credential ID not found");
  });

  it("falls back to generic message on unstructured error", async () => {
    setupAuthMocks({ authenticated: true });
    mockDelete.mockResolvedValue({
      data: undefined,
      error: "unexpected error shape",
    });

    const { result } = renderHook(() => useDeleteCredential(), { wrapper });

    let error: Error | undefined;
    await act(async () => {
      try {
        await result.current.deleteCredential("cred_abc123");
      } catch (e) {
        error = e as Error;
      }
    });

    expect(error?.message).toBe("Failed to delete credential");
  });
});
