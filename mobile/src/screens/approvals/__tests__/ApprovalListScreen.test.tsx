import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "../../../auth/AuthContext";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import ApprovalListScreen from "../ApprovalListScreen";
import {
  mockSession,
  mockApproval,
  createQueryClient,
  waitFor,
  hasTestId,
  hasText,
  findFirstByTestId,
} from "../../../__test-utils__";

// --- Mocks ---

const mockGet = jest.fn();

jest.mock("../../../api/client", () => ({
  __esModule: true,
  default: { GET: (...args: unknown[]) => mockGet(...args) },
}));

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 44, bottom: 34, left: 0, right: 0 }),
}));

const authMocks = {
  authChangeCallback: null as
    | ((event: AuthChangeEvent, session: Session | null) => void)
    | null,
};

jest.mock("../../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      onAuthStateChange: jest.fn(
        (cb: (event: AuthChangeEvent, session: Session | null) => void) => {
          authMocks.authChangeCallback = cb;
          Promise.resolve().then(() =>
            cb("INITIAL_SESSION" as AuthChangeEvent, null),
          );
          return { data: { subscription: { unsubscribe: jest.fn() } } };
        },
      ),
      signInWithOtp: jest.fn(),
      verifyOtp: jest.fn(),
      signOut: jest.fn().mockResolvedValue({ error: null }),
      mfa: {
        getAuthenticatorAssuranceLevel: jest.fn().mockResolvedValue({
          data: { currentLevel: "aal1", nextLevel: "aal1" },
          error: null,
        }),
      },
    },
  },
}));

// --- Helpers ---

function renderScreen() {
  const queryClient = createQueryClient();
  return create(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(AuthProvider, null, createElement(ApprovalListScreen)),
    ),
  );
}

async function authenticateAndRender(apiResponse: unknown) {
  mockGet.mockResolvedValue(apiResponse);

  let renderer: ReactTestRenderer;
  await act(async () => {
    renderer = renderScreen();
  });

  // Wait for initial auth to settle
  await waitFor(
    () => hasTestId(renderer!, "sign-out") || hasTestId(renderer!, "tab-pending"),
    { timeout: 1000 },
  ).catch(() => {
    // Screen might be in loading state still
  });

  const session = mockSession();
  await act(async () => {
    authMocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
  });

  // Wait for screen to render with data (loading indicator gone)
  await waitFor(() => hasTestId(renderer!, "tab-pending"));

  // Wait for API call to complete and data to render
  await waitFor(() => mockGet.mock.calls.length > 0);

  // Give React Query time to update the UI
  await act(async () => {
    await new Promise((r) => setTimeout(r, 50));
  });

  currentRenderer = renderer!;
  return renderer!;
}

// --- Tests ---

let currentRenderer: ReactTestRenderer | null = null;

describe("ApprovalListScreen", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    // Unmount to prevent React Query timers from firing after teardown.
    if (currentRenderer) {
      await act(async () => {
        currentRenderer!.unmount();
      });
      currentRenderer = null;
    }
  });

  it("renders the title and sign out button", async () => {
    const renderer = await authenticateAndRender({
      data: { data: [], has_more: false },
      error: undefined,
    });

    expect(hasText(renderer, "Approvals")).toBe(true);
    expect(hasTestId(renderer, "sign-out")).toBe(true);
  });

  it("renders three status filter tabs", async () => {
    const renderer = await authenticateAndRender({
      data: { data: [], has_more: false },
      error: undefined,
    });

    expect(hasTestId(renderer, "tab-pending")).toBe(true);
    expect(hasTestId(renderer, "tab-approved")).toBe(true);
    expect(hasTestId(renderer, "tab-denied")).toBe(true);
  });

  it("shows empty state when no pending approvals", async () => {
    const renderer = await authenticateAndRender({
      data: { data: [], has_more: false },
      error: undefined,
    });

    expect(hasText(renderer, "All clear")).toBe(true);
    expect(hasText(renderer, "You have no pending approval requests.")).toBe(true);
  });

  it("renders approval items when data is returned", async () => {
    const renderer = await authenticateAndRender({
      data: { data: [mockApproval], has_more: false },
      error: undefined,
    });

    expect(hasTestId(renderer, "approval-item-appr_abc123")).toBe(true);
    expect(hasText(renderer, "Send welcome email to new user")).toBe(true);
  });

  it("shows error state with retry button on API failure", async () => {
    const renderer = await authenticateAndRender({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    await waitFor(() => hasText(renderer, "Server error"));

    expect(hasText(renderer, "Server error")).toBe(true);
    expect(hasTestId(renderer, "retry-button")).toBe(true);
  });

  it("switches tabs and fetches with new status filter", async () => {
    const renderer = await authenticateAndRender({
      data: { data: [], has_more: false },
      error: undefined,
    });

    // Initially fetched with pending
    expect(mockGet).toHaveBeenCalledWith(
      "/v1/approvals",
      expect.objectContaining({
        params: { query: { status: "pending" } },
      }),
    );

    // Tap the "Approved" tab
    const approvedTab = findFirstByTestId(renderer, "tab-approved");
    await act(async () => {
      approvedTab?.props.onPress();
    });

    await waitFor(() =>
      mockGet.mock.calls.some(
        (call: unknown[]) =>
          (call[1] as { params: { query: { status: string } } })?.params?.query
            ?.status === "approved",
      ),
    );

    expect(mockGet).toHaveBeenCalledWith(
      "/v1/approvals",
      expect.objectContaining({
        params: { query: { status: "approved" } },
      }),
    );
  });
});
