import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockPatch, resetClientMocks } from "../../api/__mocks__/client";
import { useUpdateCredential } from "../useUpdateCredential";

vi.mock("../../api/client");

describe("useUpdateCredential", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("sends patch request with label-only update", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: {
        id: "cred_abc123",
        service: "github",
        label: "Work GitHub",
        created_at: "2026-02-11T10:00:00Z",
      },
    });

    const { result } = renderHook(() => useUpdateCredential(), { wrapper });

    await settleAuthHydration();
    let response: unknown;
    await act(async () => {
      response = await result.current.updateCredential("cred_abc123", {
        label: "Work GitHub",
      });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      params: { path: { credential_id: "cred_abc123" } },
      body: { label: "Work GitHub" },
    });
    expect(response).toEqual({
      id: "cred_abc123",
      service: "github",
      label: "Work GitHub",
      created_at: "2026-02-11T10:00:00Z",
    });
  });

  it("sends patch request to clear label", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: {
        id: "cred_abc123",
        service: "github",
        created_at: "2026-02-11T10:00:00Z",
      },
    });

    const { result } = renderHook(() => useUpdateCredential(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current.updateCredential("cred_abc123", { label: null });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      params: { path: { credential_id: "cred_abc123" } },
      body: { label: null },
    });
  });

  it("sends patch request with partial credential fields", async () => {
    setupAuthMocks({ authenticated: true });
    mockPatch.mockResolvedValue({
      data: {
        id: "cred_abc123",
        service: "proton",
        created_at: "2026-02-11T10:00:00Z",
      },
    });

    const { result } = renderHook(() => useUpdateCredential(), { wrapper });

    await settleAuthHydration();
    await act(async () => {
      await result.current.updateCredential("cred_abc123", {
        credentials: { password: "new-secret" },
      });
    });

    expect(mockPatch).toHaveBeenCalledWith("/v1/credentials/{credential_id}", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      params: { path: { credential_id: "cred_abc123" } },
      body: { credentials: { password: "new-secret" } },
    });
  });

  it("throws when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useUpdateCredential(), { wrapper });

    await settleAuthHydration();
    await expect(
      result.current.updateCredential("cred_abc123", { label: "Work" }),
    ).rejects.toThrow("Not authenticated");
  });
});
