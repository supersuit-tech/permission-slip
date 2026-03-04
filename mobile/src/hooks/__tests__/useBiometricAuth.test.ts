/**
 * Tests for the useBiometricAuth hook.
 */
import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { Text } from "react-native";

// --- Mocks ---

const mockHasHardware = jest.fn();
const mockIsEnrolled = jest.fn();
const mockAuthenticate = jest.fn();

jest.mock("expo-local-authentication", () => ({
  hasHardwareAsync: () => mockHasHardware(),
  isEnrolledAsync: () => mockIsEnrolled(),
  authenticateAsync: (opts: unknown) => mockAuthenticate(opts),
}));

const mockGetItem = jest.fn();
const mockSetItem = jest.fn();

jest.mock("expo-secure-store", () => ({
  getItemAsync: (key: string) => mockGetItem(key),
  setItemAsync: (key: string, value: string) => mockSetItem(key, value),
}));

import { useBiometricAuth } from "../useBiometricAuth";

// --- Helpers ---

/** Captures hook return values so tests can inspect them. */
let captured: ReturnType<typeof useBiometricAuth>;

function TestHarness() {
  captured = useBiometricAuth({ userId: "test-user-123" });
  return createElement(
    Text,
    null,
    `status=${captured.status} enabled=${captured.isEnabled} authed=${captured.isAuthenticated}`,
  );
}

async function flush() {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 10));
  });
}

// --- Tests ---

describe("useBiometricAuth", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("reports unavailable when no biometric hardware", async () => {
    mockHasHardware.mockResolvedValue(false);

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    expect(captured.status).toBe("unavailable");
    expect(captured.isEnabled).toBe(false);
  });

  it("reports available when hardware exists but not enrolled", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(false);

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    expect(captured.status).toBe("available");
  });

  it("reports enrolled and loads saved preference", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(true);
    mockGetItem.mockResolvedValue("true");

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    expect(captured.status).toBe("enrolled");
    expect(captured.isEnabled).toBe(true);
  });

  it("toggleBiometric enables after successful verification", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(true);
    mockGetItem.mockResolvedValue(null);
    mockAuthenticate.mockResolvedValue({ success: true });

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    expect(captured.isEnabled).toBe(false);

    let result: boolean | undefined;
    await act(async () => {
      result = await captured.toggleBiometric(true);
    });
    expect(result).toBe(true);
    expect(captured.isEnabled).toBe(true);
    expect(mockSetItem).toHaveBeenCalledWith("biometric_auth_enabled_test-user-123", "true");
  });

  it("toggleBiometric does not enable after failed verification", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(true);
    mockGetItem.mockResolvedValue(null);
    mockAuthenticate.mockResolvedValue({ success: false });

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    let result: boolean | undefined;
    await act(async () => {
      result = await captured.toggleBiometric(true);
    });
    expect(result).toBe(false);
    expect(captured.isEnabled).toBe(false);
    expect(mockSetItem).not.toHaveBeenCalled();
  });

  it("authenticate returns true when biometric passes", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(true);
    mockGetItem.mockResolvedValue("true");
    mockAuthenticate.mockResolvedValue({ success: true });

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    let result: boolean | undefined;
    await act(async () => {
      result = await captured.authenticate();
    });
    expect(result).toBe(true);
    expect(captured.isAuthenticated).toBe(true);
  });

  it("skips biometric gate when not enabled", async () => {
    mockHasHardware.mockResolvedValue(true);
    mockIsEnrolled.mockResolvedValue(true);
    mockGetItem.mockResolvedValue(null);

    await act(async () => {
      renderer = create(createElement(TestHarness));
    });
    await flush();

    // When not enabled, isAuthenticated should be true (no gate)
    expect(captured.isEnabled).toBe(false);
    expect(captured.isAuthenticated).toBe(true);
  });
});
