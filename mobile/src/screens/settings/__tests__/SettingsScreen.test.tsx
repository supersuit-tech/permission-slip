import React, { createElement } from "react";
import { Alert, Linking } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { NotificationPreference } from "../../../hooks/useNotificationPreferences";

// --- Mocks ---

jest.mock("../../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      getSession: jest
        .fn()
        .mockResolvedValue({ data: { session: null }, error: null }),
      onAuthStateChange: jest.fn().mockReturnValue({
        data: { subscription: { unsubscribe: jest.fn() } },
      }),
      signInWithOtp: jest.fn(),
      verifyOtp: jest.fn(),
      signOut: jest.fn(),
    },
  },
}));

const mockSignOut = jest.fn();
jest.mock("../../../auth/AuthContext", () => ({
  useAuth: () => ({
    signOut: mockSignOut,
    session: null,
    user: null,
    authStatus: "authenticated",
  }),
}));

jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn().mockResolvedValue(null),
  setItemAsync: jest.fn().mockResolvedValue(undefined),
  deleteItemAsync: jest.fn().mockResolvedValue(undefined),
}));

jest.mock("expo-constants", () => ({
  __esModule: true,
  default: {
    expoConfig: {
      extra: {
        gitCommitHash: "abc1234def5678",
        gitCommitTimestamp: "2026-04-16T10:30:00+00:00",
      },
    },
  },
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  SafeAreaProvider: ({ children }: { children: React.ReactNode }) => children,
}));

const mockPreferences: NotificationPreference[] = [
  { channel: "email", enabled: true, available: true },
  { channel: "mobile-push", enabled: true, available: true },
];

let mockPrefsReturn = {
  preferences: mockPreferences,
  isLoading: false,
  error: null as string | null,
  refetch: jest.fn(),
};

jest.mock("../../../hooks/useNotificationPreferences", () => ({
  useNotificationPreferences: () => mockPrefsReturn,
}));

const mockUpdatePreferences = jest.fn().mockResolvedValue({});

jest.mock("../../../hooks/useUpdateNotificationPreferences", () => ({
  useUpdateNotificationPreferences: () => ({
    updatePreferences: mockUpdatePreferences,
    isUpdating: false,
    error: null,
  }),
}));

const mockTypePreferences = [
  { notification_type: "standing_execution" as const, enabled: true },
];

let mockTypePrefsReturn = {
  preferences: mockTypePreferences,
  isLoading: false,
  error: null as string | null,
  refetch: jest.fn(),
};

jest.mock("../../../hooks/useNotificationTypePreferences", () => ({
  NOTIFICATION_TYPE_STANDING_EXECUTION: "standing_execution",
  useNotificationTypePreferences: () => mockTypePrefsReturn,
}));

const mockUpdateTypePreferences = jest.fn().mockResolvedValue({});

jest.mock("../../../hooks/useUpdateNotificationTypePreferences", () => ({
  useUpdateNotificationTypePreferences: () => ({
    updatePreferences: mockUpdateTypePreferences,
    isUpdating: false,
    error: null,
  }),
}));

const mockDeleteAccount = jest.fn().mockResolvedValue({});

jest.mock("../../../hooks/useDeleteAccount", () => ({
  useDeleteAccount: () => ({
    deleteAccount: mockDeleteAccount,
    isDeleting: false,
    error: null,
  }),
}));

// Import after mocks
import SettingsScreen from "../SettingsScreen";

// --- Helpers ---

function renderScreen() {
  const navigation = { navigate: jest.fn(), goBack: jest.fn() } as never;
  const route = {
    key: "Settings",
    name: "Settings" as const,
    params: undefined,
  };
  return create(createElement(SettingsScreen, { navigation, route }));
}

/** Find the first node matching a testID (host components only). */
function findByTestId(renderer: ReactTestRenderer, testID: string) {
  const nodes = renderer.root.findAll(
    (node) =>
      typeof node.type === "string" && node.props.testID === testID,
  );
  return nodes;
}

function hasText(renderer: ReactTestRenderer, text: string): boolean {
  const nodes = renderer.root.findAll(
    (node) =>
      typeof node.children?.[0] === "string" && node.children[0] === text,
  );
  return nodes.length > 0;
}

// --- Tests ---

describe("SettingsScreen", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.clearAllMocks();
    mockPrefsReturn = {
      preferences: mockPreferences,
      isLoading: false,
      error: null,
      refetch: jest.fn(),
    };
    mockTypePrefsReturn = {
      preferences: mockTypePreferences,
      isLoading: false,
      error: null,
      refetch: jest.fn(),
    };
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("renders the push notifications toggle", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(hasText(renderer, "Push Notifications")).toBe(true);
    expect(findByTestId(renderer, "mobile-push-toggle").length).toBeGreaterThanOrEqual(1);
  });

  it("shows toggle in enabled state when mobile-push is enabled", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    const toggle = findByTestId(renderer, "mobile-push-toggle")[0];
    expect(toggle?.props.value).toBe(true);
  });

  it("shows toggle in disabled state when mobile-push is disabled", async () => {
    mockPrefsReturn = {
      ...mockPrefsReturn,
      preferences: [
        { channel: "mobile-push", enabled: false, available: true },
      ],
    };
    await act(async () => {
      renderer = renderScreen();
    });
    const toggle = findByTestId(renderer, "mobile-push-toggle")[0];
    expect(toggle?.props.value).toBe(false);
  });

  it("calls updatePreferences when toggle is pressed", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    // Find the Switch node that has the onValueChange handler
    const toggleNodes = renderer.root.findAll(
      (node) =>
        node.props.testID === "mobile-push-toggle" &&
        typeof node.props.onValueChange === "function",
    );
    const toggle = toggleNodes[0];

    await act(async () => {
      toggle?.props.onValueChange(false);
    });

    expect(mockUpdatePreferences).toHaveBeenCalledWith([
      { channel: "mobile-push", enabled: false },
    ]);
  });

  it("shows loading indicator when preferences are loading", async () => {
    mockPrefsReturn = {
      ...mockPrefsReturn,
      isLoading: true,
      preferences: [],
    };
    mockTypePrefsReturn = {
      ...mockTypePrefsReturn,
      isLoading: false,
    };
    await act(async () => {
      renderer = renderScreen();
    });
    expect(findByTestId(renderer, "prefs-loading").length).toBeGreaterThanOrEqual(1);
    expect(findByTestId(renderer, "mobile-push-toggle")).toHaveLength(0);
  });

  it("shows error state when preferences fail to load", async () => {
    mockPrefsReturn = {
      ...mockPrefsReturn,
      error: "Failed to load",
      preferences: [],
    };
    mockTypePrefsReturn = {
      ...mockTypePrefsReturn,
      error: null,
    };
    await act(async () => {
      renderer = renderScreen();
    });
    expect(hasText(renderer, "Failed to load")).toBe(true);
    expect(findByTestId(renderer, "mobile-push-toggle")).toHaveLength(0);
  });

  it("renders auto-approval execution toggle", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(hasText(renderer, "Notify me about")).toBe(true);
    expect(hasText(renderer, "Auto-approval executions")).toBe(true);
    expect(findByTestId(renderer, "standing-execution-toggle").length).toBeGreaterThanOrEqual(1);
  });

  it("calls updateTypePreferences when auto-approval toggle is pressed", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    const toggle = renderer.root.findAll(
      (node) =>
        node.props.testID === "standing-execution-toggle" &&
        typeof node.props.onValueChange === "function",
    )[0];

    await act(async () => {
      toggle?.props.onValueChange(false);
    });

    expect(mockUpdateTypePreferences).toHaveBeenCalledWith([
      { notification_type: "standing_execution", enabled: false },
    ]);
  });

  it("renders the sign out button", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(findByTestId(renderer, "sign-out-button").length).toBeGreaterThanOrEqual(1);
    expect(hasText(renderer, "Sign Out")).toBe(true);
  });

  it("defaults to enabled when mobile-push preference is missing", async () => {
    mockPrefsReturn = {
      ...mockPrefsReturn,
      preferences: [{ channel: "email", enabled: true, available: true }],
    };
    await act(async () => {
      renderer = renderScreen();
    });
    const toggle = findByTestId(renderer, "mobile-push-toggle")[0];
    expect(toggle?.props.value).toBe(true);
  });

  it("renders the delete account button", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(findByTestId(renderer, "delete-account-button").length).toBeGreaterThanOrEqual(1);
    expect(hasText(renderer, "Delete Account")).toBe(true);
  });

  it("renders the privacy policy link", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(findByTestId(renderer, "privacy-policy-link").length).toBeGreaterThanOrEqual(1);
    expect(hasText(renderer, "Privacy Policy")).toBe(true);
  });

  it("renders the terms of service link", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    expect(findByTestId(renderer, "terms-link").length).toBeGreaterThanOrEqual(1);
    expect(hasText(renderer, "Terms of Service")).toBe(true);
  });

  it("calls deleteAccount after confirming the delete dialog", async () => {
    const alertSpy = jest
      .spyOn(Alert, "alert")
      .mockImplementation((_title, _msg, buttons) => {
        const confirm = buttons?.find(
          (b) => typeof b === "object" && b.style === "destructive",
        );
        if (confirm && typeof confirm === "object" && confirm.onPress) {
          confirm.onPress();
        }
      });
    await act(async () => {
      renderer = renderScreen();
    });
    const btn = renderer.root.findAll(
      (node) =>
        node.props.testID === "delete-account-button" &&
        typeof node.props.onPress === "function",
    )[0];
    await act(async () => {
      btn?.props.onPress();
    });
    expect(mockDeleteAccount).toHaveBeenCalledTimes(1);
    alertSpy.mockRestore();
  });

  it("opens privacy policy URL when tapped", async () => {
    const openURLSpy = jest
      .spyOn(Linking, "openURL")
      .mockResolvedValue(undefined as never);
    await act(async () => {
      renderer = renderScreen();
    });
    const link = renderer.root.findAll(
      (node) =>
        node.props.testID === "privacy-policy-link" &&
        typeof node.props.onPress === "function",
    )[0];
    await act(async () => {
      link?.props.onPress();
    });
    expect(openURLSpy).toHaveBeenCalledWith("https://app.permissionslip.dev/policy/privacy");
    openURLSpy.mockRestore();
  });

  it("renders the git commit hash at the bottom", async () => {
    await act(async () => {
      renderer = renderScreen();
    });
    const hashNodes = findByTestId(renderer, "git-commit-hash");
    expect(hashNodes.length).toBeGreaterThanOrEqual(1);
    const textContent = hashNodes[0]?.children?.join("") ?? "";
    expect(textContent).toContain("abc1234");
    expect(textContent).toContain("Apr 16, 2026");
  });

  it("opens terms of service URL when tapped", async () => {
    const openURLSpy = jest
      .spyOn(Linking, "openURL")
      .mockResolvedValue(undefined as never);
    await act(async () => {
      renderer = renderScreen();
    });
    const link = renderer.root.findAll(
      (node) =>
        node.props.testID === "terms-link" &&
        typeof node.props.onPress === "function",
    )[0];
    await act(async () => {
      link?.props.onPress();
    });
    expect(openURLSpy).toHaveBeenCalledWith("https://app.permissionslip.dev/policy/terms");
    openURLSpy.mockRestore();
  });
});
