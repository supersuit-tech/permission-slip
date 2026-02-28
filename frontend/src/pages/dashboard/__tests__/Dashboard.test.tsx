import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { Dashboard } from "../Dashboard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

describe("Dashboard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders all dashboard cards", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: { data: [], has_more: false } });

    render(<Dashboard />, { wrapper });

    expect(screen.getByText("Recent Activity")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("Registered Agents")).toBeInTheDocument();
    });
  });
});
