import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { DataRetentionSection } from "../DataRetentionSection";

vi.mock("../../../api/client");

describe("DataRetentionSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
    setupAuthMocks({ authenticated: true });
  });

  it("shows audit retention from API", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/profile/data-retention") {
        return Promise.resolve({
          data: {
            audit_retention_days: 90,
            effective_retention_days: 90,
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    render(<DataRetentionSection />, { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(screen.getByText("90 days")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves

    render(<DataRetentionSection />, { wrapper });

    await settleAuthHydration();
    expect(
      screen.getByRole("status", { name: "Loading data retention policy" }),
    ).toBeInTheDocument();
  });
});
