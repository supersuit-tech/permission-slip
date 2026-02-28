import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { AuthError } from "@supabase/supabase-js";
import {
  aalResponse,
  mockAuth,
  mockMfa,
  mockSession,
  setupAuthMocks,
  unverifiedFactor,
  verifiedFactor,
} from "./fixtures";
import { AuthProvider, useAuth } from "../AuthContext";
import type { AuthState } from "../types";

vi.mock("../../lib/supabaseClient");

/**
 * Waits for the auth provider to finish its async initialization (AAL check).
 * MFA-related hooks can't be called while authStatus is still "loading".
 */
async function waitForAuthReady(hook: {
  result: { current: Pick<AuthState, "authStatus"> };
}) {
  await waitFor(() => {
    expect(hook.result.current.authStatus).not.toBe("loading");
  });
}

describe("AuthContext", () => {
  beforeEach(() => {
    setupAuthMocks();
  });

  describe("AuthProvider", () => {
    it("starts in loading state before onAuthStateChange fires", () => {
      // Override to NOT fire the callback immediately
      mockAuth.onAuthStateChange.mockReturnValue({
        data: {
          subscription: {
            id: "test",
            callback: vi.fn(),
            unsubscribe: vi.fn(),
          },
        },
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      expect(result.current.authStatus).toBe("loading");
      expect(result.current.session).toBeNull();
      expect(result.current.user).toBeNull();
    });

    it("sets unauthenticated when session is null", () => {
      setupAuthMocks({ authenticated: false });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      expect(result.current.authStatus).toBe("unauthenticated");
      expect(result.current.session).toBeNull();
      expect(result.current.user).toBeNull();
    });

    it("sets authenticated with session and user", async () => {
      setupAuthMocks({ authenticated: true });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      // onAuthStateChange callback is async (AAL check), so wait for state.
      await waitFor(() => {
        expect(result.current.authStatus).toBe("authenticated");
      });
      expect(result.current.session).toBe(mockSession);
      expect(result.current.user).toBe(mockSession.user);
    });

    it("unsubscribes on unmount", () => {
      const unsubscribe = vi.fn();
      mockAuth.onAuthStateChange.mockReturnValue({
        data: {
          subscription: { id: "test", callback: vi.fn(), unsubscribe },
        },
      });

      const { unmount } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      expect(unsubscribe).not.toHaveBeenCalled();
      unmount();
      expect(unsubscribe).toHaveBeenCalledOnce();
    });
  });

  describe("useAuth", () => {
    it("throws when used outside AuthProvider", () => {
      // Suppress React error boundary console noise
      const spy = vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() => renderHook(() => useAuth())).toThrow(
        "useAuth must be used within an AuthProvider"
      );
      spy.mockRestore();
    });
  });

  describe("sendOtp", () => {
    it("calls supabase signInWithOtp and returns result", async () => {
      mockAuth.signInWithOtp.mockResolvedValue({
        data: { user: null, session: null },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      const response = await result.current.sendOtp("test@example.com");

      expect(mockAuth.signInWithOtp).toHaveBeenCalledWith({
        email: "test@example.com",
      });
      expect(response).toEqual({ error: null });
    });

    it("returns error from supabase", async () => {
      const authError = new AuthError("Rate limit", 429);
      mockAuth.signInWithOtp.mockResolvedValue({
        data: { user: null, session: null },
        error: authError,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      const response = await result.current.sendOtp("test@example.com");

      expect(response).toEqual({ error: authError });
    });
  });

  describe("verifyOtp", () => {
    it("calls supabase verifyOtp with email type", async () => {
      mockAuth.verifyOtp.mockResolvedValue({
        data: { session: mockSession, user: mockSession.user },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      const response = await result.current.verifyOtp(
        "test@example.com",
        "123456"
      );

      expect(mockAuth.verifyOtp).toHaveBeenCalledWith({
        email: "test@example.com",
        token: "123456",
        type: "email",
      });
      expect(response).toEqual({ error: null });
    });
  });

  describe("signOut", () => {
    it("calls supabase signOut", async () => {
      mockAuth.signOut.mockResolvedValue({ error: null });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      const response = await result.current.signOut();

      expect(mockAuth.signOut).toHaveBeenCalled();
      expect(response).toEqual({ error: null });
    });
  });

  describe("mfa_required", () => {
    it("sets authStatus to mfa_required when AAL indicates MFA needed", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
        aalResponse("aal1", "aal2")
      );

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitFor(() => {
        expect(result.current.authStatus).toBe("mfa_required");
      });
    });

    it("sets authenticated when AAL levels match (no MFA enrolled)", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.getAuthenticatorAssuranceLevel.mockResolvedValue(
        aalResponse("aal1", "aal1")
      );

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitFor(() => {
        expect(result.current.authStatus).toBe("authenticated");
      });
    });
  });

  describe("verifyMfa", () => {
    it("calls challengeAndVerify with the verified TOTP factor", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.listFactors.mockResolvedValue({
        data: { all: [verifiedFactor], totp: [verifiedFactor] },
        error: null,
      });
      // challengeAndVerify response shape only matters for the error field;
      // AuthContext doesn't inspect the data payload.
      mockMfa.challengeAndVerify.mockResolvedValue({
        data: mockSession,
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.verifyMfa("123456");

      expect(mockMfa.challengeAndVerify).toHaveBeenCalledWith({
        factorId: "factor-1",
        code: "123456",
      });
      expect(response).toEqual({ error: null });
    });

    it("returns error when no verified TOTP factor exists", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.listFactors.mockResolvedValue({
        data: { all: [], totp: [] },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.verifyMfa("123456");

      expect(response.error).toBeDefined();
      expect(response.error?.code).toBe("mfa_factor_not_found");
    });
  });

  describe("enrollMfa", () => {
    it("returns factorId, qrCode, and secret on success", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.enroll.mockResolvedValue({
        data: {
          id: "factor-new",
          type: "totp",
          totp: {
            qr_code: "data:image/svg+xml;...",
            secret: "JBSWY3DPEHPK3PXP",
            uri: "otpauth://totp/test",
          },
        },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.enrollMfa();

      expect(response.data).toEqual({
        factorId: "factor-new",
        qrCode: "data:image/svg+xml;...",
        secret: "JBSWY3DPEHPK3PXP",
      });
      expect(response.error).toBeNull();
    });

    it("returns error when enroll fails", async () => {
      setupAuthMocks({ authenticated: true });
      const authError = new AuthError("Enroll failed", 500);
      mockMfa.enroll.mockResolvedValue({
        data: null,
        error: authError,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.enrollMfa();

      expect(response.data).toBeNull();
      expect(response.error).toBe(authError);
    });
  });

  describe("confirmMfaEnrollment", () => {
    it("calls challengeAndVerify and sets authenticated on success", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.challengeAndVerify.mockResolvedValue({
        data: mockSession,
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.confirmMfaEnrollment(
        "factor-1",
        "123456"
      );

      expect(mockMfa.challengeAndVerify).toHaveBeenCalledWith({
        factorId: "factor-1",
        code: "123456",
      });
      expect(response).toEqual({ error: null });
    });
  });

  describe("unenrollMfa", () => {
    it("calls mfa.unenroll with the factor ID", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.unenroll.mockResolvedValue({ data: { id: "factor-1" }, error: null });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.unenrollMfa("factor-1");

      expect(mockMfa.unenroll).toHaveBeenCalledWith({
        factorId: "factor-1",
      });
      expect(response).toEqual({ error: null });
    });
  });

  describe("listMfaFactors", () => {
    it("returns TOTP factors from supabase", async () => {
      setupAuthMocks({ authenticated: true });
      const factors = [verifiedFactor];
      mockMfa.listFactors.mockResolvedValue({
        data: { all: factors, totp: factors },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.listMfaFactors();

      expect(response.factors).toEqual(factors);
      expect(response.error).toBeNull();
    });

    it("returns empty array when no factors exist", async () => {
      setupAuthMocks({ authenticated: true });
      mockMfa.listFactors.mockResolvedValue({
        data: { all: [], totp: [] },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.listMfaFactors();

      expect(response.factors).toEqual([]);
      expect(response.error).toBeNull();
    });

    it("includes unverified TOTP factors (for stale enrollment cleanup)", async () => {
      setupAuthMocks({ authenticated: true });
      // Supabase only puts verified factors in data.totp; unverified are
      // only in data.all. Our wrapper must pull from data.all so the
      // enrollment flow can find and clean up stale unverified factors.
      mockMfa.listFactors.mockResolvedValue({
        data: { all: [unverifiedFactor], totp: [] },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.listMfaFactors();

      expect(response.factors).toEqual([unverifiedFactor]);
      expect(response.error).toBeNull();
    });

    it("excludes non-TOTP factors from results", async () => {
      setupAuthMocks({ authenticated: true });
      const phoneFactor = {
        id: "factor-phone",
        status: "verified" as const,
        factor_type: "phone" as const,
        created_at: "2024-01-01",
        updated_at: "2024-01-01",
      };
      mockMfa.listFactors.mockResolvedValue({
        data: {
          all: [verifiedFactor, phoneFactor],
          totp: [verifiedFactor],
          phone: [phoneFactor],
        },
        error: null,
      });

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      });

      await waitForAuthReady({ result });

      const response = await result.current.listMfaFactors();

      expect(response.factors).toEqual([verifiedFactor]);
      expect(response.error).toBeNull();
    });
  });
});
