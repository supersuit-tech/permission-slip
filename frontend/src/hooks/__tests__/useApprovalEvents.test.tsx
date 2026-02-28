import { renderHook, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, beforeEach, vi, afterEach, afterAll } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { AuthProvider } from "../../auth/AuthContext";
import { createAuthWrapper } from "../../test-helpers";
import { useApprovalEvents } from "../useApprovalEvents";

vi.mock("../../lib/supabaseClient");

// Mock EventSource since jsdom doesn't have it
class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  listeners: Map<string, Set<EventListener>> = new Map();
  closed = false;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: EventListener) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }
    this.listeners.get(type)!.add(listener);
  }

  removeEventListener(type: string, listener: EventListener) {
    this.listeners.get(type)?.delete(listener);
  }

  close() {
    this.closed = true;
  }

  // Test helper: simulate an event
  emit(type: string, data?: string) {
    const event = new MessageEvent(type, { data: data ?? "{}" });
    this.listeners.get(type)?.forEach((listener) => listener(event));
  }

  static reset() {
    MockEventSource.instances = [];
  }
}

// Save and install the mock so we can restore after tests.
const originalEventSource = globalThis.EventSource;
Object.defineProperty(globalThis, "EventSource", {
  value: MockEventSource,
  writable: true,
  configurable: true,
});

afterAll(() => {
  if (originalEventSource) {
    Object.defineProperty(globalThis, "EventSource", {
      value: originalEventSource,
      writable: true,
      configurable: true,
    });
  } else {
    // @ts-expect-error EventSource didn't exist before (e.g. jsdom)
    delete globalThis.EventSource;
  }
});

describe("useApprovalEvents", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    MockEventSource.reset();
    wrapper = createAuthWrapper();
  });

  afterEach(() => {
    MockEventSource.reset();
  });

  it("creates an EventSource when authenticated", () => {
    setupAuthMocks({ authenticated: true });

    renderHook(() => useApprovalEvents(), { wrapper });

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0]!.url).toContain("/api/v1/approvals/events");
    expect(MockEventSource.instances[0]!.url).toContain("token=");
  });

  it("does not create EventSource when unauthenticated", () => {
    setupAuthMocks({ authenticated: false });

    renderHook(() => useApprovalEvents(), { wrapper });

    expect(MockEventSource.instances).toHaveLength(0);
  });

  it("registers event listeners for approval events", () => {
    setupAuthMocks({ authenticated: true });

    renderHook(() => useApprovalEvents(), { wrapper });

    const es = MockEventSource.instances[0]!;
    expect(es.listeners.has("approval_created")).toBe(true);
    expect(es.listeners.has("approval_resolved")).toBe(true);
    expect(es.listeners.has("approval_cancelled")).toBe(true);
  });

  it("closes EventSource on unmount", () => {
    setupAuthMocks({ authenticated: true });

    const { unmount } = renderHook(() => useApprovalEvents(), { wrapper });

    const es = MockEventSource.instances[0]!;
    expect(es.closed).toBe(false);

    unmount();

    expect(es.closed).toBe(true);
  });

  it("cleans up event listeners on unmount", () => {
    setupAuthMocks({ authenticated: true });

    const { unmount } = renderHook(() => useApprovalEvents(), { wrapper });

    const es = MockEventSource.instances[0]!;
    expect(es.listeners.get("approval_created")?.size).toBe(1);

    unmount();

    expect(es.listeners.get("approval_created")?.size).toBe(0);
    expect(es.listeners.get("approval_resolved")?.size).toBe(0);
    expect(es.listeners.get("approval_cancelled")?.size).toBe(0);
  });

  it.each([
    ["approval_created", "appr_123"],
    ["approval_resolved", "appr_456"],
    ["approval_cancelled", "appr_789"],
  ] as const)("invalidates approvals query on %s event", (eventType, approvalId) => {
    setupAuthMocks({ authenticated: true });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const spy = vi.spyOn(queryClient, "invalidateQueries");
    function SpyWrapper({ children }: { children: React.ReactNode }) {
      return (
        <MemoryRouter>
          <QueryClientProvider client={queryClient}>
            <AuthProvider>{children}</AuthProvider>
          </QueryClientProvider>
        </MemoryRouter>
      );
    }

    renderHook(() => useApprovalEvents(), { wrapper: SpyWrapper });

    const es = MockEventSource.instances[0]!;
    act(() => {
      es.emit(eventType, `{"approval_id":"${approvalId}"}`);
    });

    expect(spy).toHaveBeenCalledWith({ queryKey: ["approvals"] });
  });
});
