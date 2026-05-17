import { render, screen } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { resetClientMocks } from "../../../api/__mocks__/client";
import { SecuritySection } from "../SecuritySection";

vi.mock("../../../api/client");

describe("SecuritySection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    wrapper = createAuthWrapper(["/settings"]);
  });

  it("renders static security copy", () => {
    render(<SecuritySection />, { wrapper });

    expect(screen.getByText("Security")).toBeInTheDocument();
    expect(
      screen.getByText(/Two-factor authentication and authenticator enrollment/i),
    ).toBeInTheDocument();
  });
});
