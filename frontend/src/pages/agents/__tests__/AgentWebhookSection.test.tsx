import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockDelete,
  mockGet,
  mockPut,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { AgentWebhookSection } from "../AgentWebhookSection";

vi.mock("../../../api/client");

const sharedWarning =
  "Another of your agents is registered with this same webhook URL.";

function mockWebhookGet(
  data: {
    configured?: boolean;
    webhook_url?: string;
    warning?: string;
    test?: {
      configured?: boolean;
      success?: boolean;
      message?: string;
      latency_ms?: number;
    };
  } | null = { configured: false },
) {
  mockGet.mockImplementation((url: string, opts?: { params?: { query?: { test?: boolean } } }) => {
    if (url !== "/v1/agents/{agent_id}/webhook") {
      return Promise.resolve({ data: null });
    }
    if (opts?.params?.query?.test) {
      return Promise.resolve({
        data: {
          configured: true,
          webhook_url: data?.webhook_url ?? "http://100.64.0.5:18789/hooks",
          warning: data?.warning,
          test: data?.test ?? {
            configured: true,
            success: true,
            message: "Test wake delivered successfully",
            latency_ms: 42,
          },
        },
      });
    }
    return Promise.resolve({ data });
  });
}

describe("AgentWebhookSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    wrapper = createAuthWrapper();
  });

  it("renders empty state when webhook is not configured", async () => {
    mockWebhookGet({ configured: false });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByText(/No push wake webhook configured/i),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /Configure webhook/i }),
    ).toBeInTheDocument();
  });

  it("renders configured webhook URL and status", async () => {
    mockWebhookGet({
      configured: true,
      webhook_url: "http://100.64.0.5:18789/hooks",
    });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(screen.getByText("http://100.64.0.5:18789/hooks")).toBeInTheDocument();
    });
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Test wake/i }),
    ).toBeInTheDocument();
  });

  it("saves webhook configuration from the dialog", async () => {
    const user = userEvent.setup();
    mockWebhookGet({ configured: false });
    mockPut.mockResolvedValue({
      data: {
        configured: true,
        webhook_url: "http://100.64.0.5:18789/hooks",
        test: {
          configured: true,
          success: true,
          message: "Test wake delivered successfully",
          latency_ms: 12,
        },
      },
    });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Configure webhook/i }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /Configure webhook/i }));
    await user.type(
      screen.getByLabelText(/Hooks URL/i),
      "http://100.64.0.5:18789/hooks",
    );
    await user.type(screen.getByLabelText(/Hooks token/i), "secret-token");
    await user.click(screen.getByRole("button", { name: /Save webhook/i }));

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith("/v1/agents/{agent_id}/webhook", {
        headers: { Authorization: expect.stringMatching(/^Bearer /) },
        params: { path: { agent_id: 1 } },
        body: {
          url: "http://100.64.0.5:18789/hooks",
          token: "secret-token",
        },
      });
    });
  });

  it("shows test wake success result", async () => {
    const user = userEvent.setup();
    mockWebhookGet({
      configured: true,
      webhook_url: "http://100.64.0.5:18789/hooks",
      test: {
        configured: true,
        success: true,
        message: "Test wake delivered successfully",
        latency_ms: 42,
      },
    });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Test wake/i }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /Test wake/i }));

    await waitFor(() => {
      expect(screen.getByText("Test wake succeeded")).toBeInTheDocument();
    });
    expect(screen.getByText("Latency: 42 ms")).toBeInTheDocument();
    expect(mockGet).toHaveBeenCalledWith("/v1/agents/{agent_id}/webhook", {
      headers: { Authorization: expect.stringMatching(/^Bearer /) },
      params: {
        path: { agent_id: 1 },
        query: { test: true },
      },
    });
  });

  it("shows test wake failure result", async () => {
    const user = userEvent.setup();
    mockWebhookGet({
      configured: true,
      webhook_url: "http://100.64.0.5:18789/hooks",
      test: {
        configured: true,
        success: false,
        message: "webhook returned HTTP 500",
      },
    });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Test wake/i }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /Test wake/i }));

    await waitFor(() => {
      expect(screen.getByText("Test wake failed")).toBeInTheDocument();
    });
    expect(screen.getByText("webhook returned HTTP 500")).toBeInTheDocument();
  });

  it("renders shared URL warning when present", async () => {
    mockWebhookGet({
      configured: true,
      webhook_url: "http://100.64.0.5:18789/hooks",
      warning: sharedWarning,
    });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(screen.getByText(sharedWarning)).toBeInTheDocument();
    });
  });

  it("clears webhook after confirmation", async () => {
    const user = userEvent.setup();
    mockWebhookGet({
      configured: true,
      webhook_url: "http://100.64.0.5:18789/hooks",
    });
    mockDelete.mockResolvedValue({ data: { cleared: true }, error: null });

    render(<AgentWebhookSection agentId={1} />, { wrapper });
    await settleAuthHydration();

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Clear/i })).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /^Clear$/i }));
    await user.click(screen.getByRole("button", { name: /Clear webhook/i }));

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith("/v1/agents/{agent_id}/webhook", {
        headers: { Authorization: expect.stringMatching(/^Bearer /) },
        params: { path: { agent_id: 1 } },
      });
    });
  });
});
