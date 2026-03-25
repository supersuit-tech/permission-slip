import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "@/auth/AuthContext";
import { CookieConsentProvider } from "@/components/CookieConsentContext";
import { setupAuthMocks } from "@/auth/__tests__/fixtures";
import { mockGet, resetClientMocks } from "@/api/__mocks__/client";
import { ApproveRedirectPage } from "../ApproveRedirectPage";

vi.mock("@/lib/supabaseClient");
vi.mock("@/api/client");

const futureDate = new Date(Date.now() + 600_000).toISOString();

const mockApproval = {
  approval_id: "appr_abc123",
  agent_id: 1,
  action: {
    type: "email.send",
    version: "1",
    parameters: { recipient: "user@example.com", subject: "Hello" },
  },
  context: {
    description: "Send an email",
    risk_level: "low",
  },
  status: "pending",
  expires_at: futureDate,
  created_at: "2026-01-01T00:00:00Z",
};

const mockAgents = [
  {
    agent_id: 1,
    status: "registered",
    metadata: { name: "Test Bot" },
    confirmation_code: null,
    expires_at: null,
    created_at: "2026-01-01T00:00:00Z",
  },
];

function renderPage(approvalId = "appr_abc123") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={[`/approve/${approvalId}`]}>
      <QueryClientProvider client={queryClient}>
        <CookieConsentProvider>
          <AuthProvider>
            <Routes>
              <Route path="/approve/:approvalId" element={<ApproveRedirectPage />} />
              <Route path="/" element={<div data-testid="dashboard">Dashboard</div>} />
            </Routes>
          </AuthProvider>
        </CookieConsentProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("ApproveRedirectPage", () => {
  beforeEach(() => {
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state while fetching approval", () => {
    mockGet.mockImplementation(() => new Promise(() => {})); // never resolves
    renderPage();
    expect(screen.getByText("Loading approval…")).toBeInTheDocument();
  });

  it("shows error state when approval fetch fails", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/approvals/{approval_id}") {
        return Promise.resolve({
          error: { message: "Not found" },
          response: { status: 404 },
        });
      }
      if (url === "/v1/agents") {
        return Promise.resolve({ data: { data: mockAgents } });
      }
      return Promise.resolve({ data: { data: [] } });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Approval not found")).toBeInTheDocument();
    });
  });

  it("navigates to dashboard when clicking Go to Dashboard on error", async () => {
    const user = userEvent.setup();
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/approvals/{approval_id}") {
        return Promise.resolve({
          error: { message: "Not found" },
          response: { status: 404 },
        });
      }
      if (url === "/v1/agents") {
        return Promise.resolve({ data: { data: mockAgents } });
      }
      return Promise.resolve({ data: { data: [] } });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Go to Dashboard")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Go to Dashboard"));
    expect(screen.getByTestId("dashboard")).toBeInTheDocument();
  });

  it("opens ReviewApprovalDialog when approval loads", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/approvals/{approval_id}") {
        return Promise.resolve({ data: mockApproval });
      }
      if (url === "/v1/agents") {
        return Promise.resolve({ data: { data: mockAgents } });
      }
      if (url === "/v1/standing-approvals") {
        return Promise.resolve({ data: { data: [] } });
      }
      if (url === "/v1/action-configurations") {
        return Promise.resolve({ data: { data: [] } });
      }
      return Promise.resolve({ data: { data: [] } });
    });

    renderPage();

    // The ReviewApprovalDialog should open and show the Approve button
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Approve/i })).toBeInTheDocument();
    });
  });

  it("resolves agent display name from agents list", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/approvals/{approval_id}") {
        return Promise.resolve({ data: mockApproval });
      }
      if (url === "/v1/agents") {
        return Promise.resolve({ data: { data: mockAgents } });
      }
      if (url === "/v1/standing-approvals") {
        return Promise.resolve({ data: { data: [] } });
      }
      if (url === "/v1/action-configurations") {
        return Promise.resolve({ data: { data: [] } });
      }
      return Promise.resolve({ data: { data: [] } });
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Test Bot")).toBeInTheDocument();
    });
  });
});
