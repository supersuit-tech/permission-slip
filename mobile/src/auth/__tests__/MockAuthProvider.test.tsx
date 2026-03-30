import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";

// Mock supabaseClient before importing anything that touches it
jest.mock("../../lib/supabaseClient", () => ({
  supabase: { auth: {} },
}));

import { MockAuthProvider } from "../MockAuthProvider";
import { useAuth } from "../AuthContext";

/** Tiny component that reads auth state and renders it as JSON. */
function AuthConsumer() {
  const { authStatus, user, session } = useAuth();
  return createElement("Text", {}, JSON.stringify({ authStatus, userId: user?.id, hasSession: !!session }));
}

describe("MockAuthProvider", () => {
  let renderer: ReactTestRenderer;

  afterEach(() => {
    act(() => renderer?.unmount());
  });

  it("immediately provides authenticated state", () => {
    act(() => {
      renderer = create(
        createElement(MockAuthProvider, null, createElement(AuthConsumer)),
      );
    });

    const text = renderer.root.findByType("Text" as any);
    const state = JSON.parse(text.children[0] as string);

    expect(state.authStatus).toBe("authenticated");
    expect(state.hasSession).toBe(true);
    expect(state.userId).toContain("mock-user");
  });

  it("provides no-op sendOtp/verifyOtp/signOut", async () => {
    let authState: ReturnType<typeof useAuth>;
    function Capture() {
      authState = useAuth();
      return null;
    }

    act(() => {
      renderer = create(
        createElement(MockAuthProvider, null, createElement(Capture)),
      );
    });

    const sendResult = await authState!.sendOtp("test@example.com");
    expect(sendResult.error).toBeNull();

    const verifyResult = await authState!.verifyOtp("test@example.com", "123456");
    expect(verifyResult.error).toBeNull();

    const signOutResult = await authState!.signOut();
    expect(signOutResult.error).toBeNull();
  });
});
