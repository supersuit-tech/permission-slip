import { createElement } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { MockAuthProvider } from "../MockAuthProvider";
import { useAuth } from "../AuthContext";

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

    const text = renderer.root.findByType("Text" as never);
    const state = JSON.parse(text.children[0] as string);

    expect(state.authStatus).toBe("authenticated");
    expect(state.hasSession).toBe(true);
    expect(state.userId).toContain("mock-user");
  });

  it("provides no-op signInWithPassword/signUpWithPassword/signOut", async () => {
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

    const signInResult = await authState!.signInWithPassword("test@example.com", "pw");
    expect(signInResult.error).toBeNull();

    const signUpResult = await authState!.signUpWithPassword("test@example.com", "pw");
    expect(signUpResult.error).toBeNull();

    const signOutResult = await authState!.signOut();
    expect(signOutResult.error).toBeNull();
  });
});
