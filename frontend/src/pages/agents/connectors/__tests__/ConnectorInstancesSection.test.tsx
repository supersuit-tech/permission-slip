import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockPatch,
  mockPut,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ConnectorInstancesSection } from "../ConnectorInstancesSection";
import type { InstanceCredentialBinding } from "@/hooks/useConnectorInstanceCredentialBindings";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const mockUseConnectorInstanceCredentialBindings = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/useConnectorInstanceCredentialBindings", () => ({
  useConnectorInstanceCredentialBindings: mockUseConnectorInstanceCredentialBindings,
}));

const defaultInstance = {
  connector_instance_id: "11111111-1111-1111-1111-111111111111",
  agent_id: 42,
  connector_id: "slack",
  display: "Acme",
  is_default: true,
  enabled_at: "2026-02-18T10:00:00Z",
};

const secondInstance = {
  connector_instance_id: "22222222-2222-2222-2222-222222222222",
  agent_id: 42,
  connector_id: "slack",
  display: "Sales",
  is_default: false,
  enabled_at: "2026-02-19T10:00:00Z",
};

function oauthBinding(oauthId: string): InstanceCredentialBinding {
  return {
    agent_id: 42,
    connector_id: "slack",
    credential_id: null,
    oauth_connection_id: oauthId,
  };
}

function bindingsQueryResult(twoInstances: boolean) {
  const m = new Map<string, InstanceCredentialBinding | null>();
  m.set(defaultInstance.connector_instance_id, oauthBinding("oconn_1"));
  if (twoInstances) {
    m.set(secondInstance.connector_instance_id, oauthBinding("oconn_2"));
  }
  return {
    data: m,
    isLoading: false,
    isPending: false,
    isFetching: false,
    status: "success" as const,
  };
}

function normalizeMockPath(path: unknown): string {
  if (typeof path === "string") {
    try {
      if (path.startsWith("http")) {
        return new URL(path).pathname;
      }
    } catch {
      /* ignore */
    }
    return path;
  }
  if (path instanceof Request) {
    try {
      return new URL(path.url).pathname;
    } catch {
      return "";
    }
  }
  return "";
}

const LIST_INSTANCES_PATH =
  "/v1/agents/{agent_id}/connectors/{connector_id}/instances";

function mockStandardGets(
  twoInstances = false,
  options?: { secondNeedsReauth?: boolean },
) {
  mockGet.mockImplementation((path: unknown) => {
    const p = normalizeMockPath(path);
    if (path === LIST_INSTANCES_PATH || (p.includes("/connectors/") && p.endsWith("/instances") && !p.includes("/credential"))) {
      return Promise.resolve({
        data: {
          data: twoInstances ? [defaultInstance, secondInstance] : [defaultInstance],
        },
      });
    }
    if (p.endsWith("/credential") && p.includes("/connectors/") && !p.includes("/instances/")) {
      return Promise.resolve({
        data: {
          agent_id: 42,
          connector_id: "slack",
          credential_id: null,
          oauth_connection_id: "oconn_1",
        },
      });
    }
    if (p.endsWith("/credentials")) {
      return Promise.resolve({ data: { credentials: [] } });
    }
    if (p.includes("/oauth/connections")) {
      return Promise.resolve({
        data: {
          connections: [
            {
              id: "oconn_1",
              provider: "slack",
              status: "active",
              display_name: "Acme",
              connected_at: "2026-02-11T10:00:00Z",
            },
            {
              id: "oconn_2",
              provider: "slack",
              status: options?.secondNeedsReauth ? "needs_reauth" : "active",
              display_name: "Sales",
              connected_at: "2026-02-12T10:00:00Z",
            },
          ],
        },
      });
    }
    if (p.includes("/oauth/providers")) {
      return Promise.resolve({ data: { providers: [] } });
    }
    return Promise.resolve({ data: {} });
  });
}

describe("ConnectorInstancesSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    mockPost.mockResolvedValue({ data: {} });
    mockPut.mockResolvedValue({ data: {} });
    mockPatch.mockResolvedValue({ data: {} });
    mockDelete.mockResolvedValue({ data: {} });
    setupAuthMocks({ authenticated: true });
    mockUseConnectorInstanceCredentialBindings.mockImplementation(() =>
      bindingsQueryResult(false),
    );
  });

  it("lists credentials with default marked", async () => {
    mockStandardGets(false);
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(false),
    );
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Acme/)).toBeInTheDocument();
    });
    expect(screen.getByText("Default")).toBeInTheDocument();
  });

  it("enables a credential by creating an instance and assigning it", async () => {
    mockStandardGets(false);
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(false),
    );
    mockPost.mockResolvedValue({
      data: {
        ...secondInstance,
        is_default: false,
      },
    });
    const user = userEvent.setup();
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Acme/)).toBeInTheDocument();
    });
    const salesCheckbox = screen.getByRole("checkbox", {
      name: /Slack OAuth — Sales/i,
    });
    await user.click(salesCheckbox);
    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
    const createCall = mockPost.mock.calls.find(
      (c) => c[0] === "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
    );
    expect(createCall?.[1]?.body).toEqual({});
    await waitFor(() => {
      expect(mockPut).toHaveBeenCalled();
    });
    const putCall = mockPut.mock.calls.find(
      (c) =>
        c[0] ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential",
    );
    expect(putCall?.[1]?.body).toEqual({
      credential_id: undefined,
      oauth_connection_id: "oconn_2",
    });
  });

  it("sets default instance via PATCH", async () => {
    mockStandardGets(true);
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(true),
    );
    const user = userEvent.setup();
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Sales/)).toBeInTheDocument();
    });
    const makeDefaultButtons = screen.getAllByRole("button", {
      name: /Make default/i,
    });
    const salesMakeDefault = makeDefaultButtons.find((b) => !b.hasAttribute("disabled"));
    expect(salesMakeDefault).toBeTruthy();
    await user.click(salesMakeDefault!);
    await waitFor(() => {
      expect(mockPatch).toHaveBeenCalled();
    });
    const call = mockPatch.mock.calls.find(
      (c) =>
        c[0] ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
    );
    expect(call?.[1]?.body).toEqual({ is_default: true });
    expect(call?.[1]?.params?.path?.instance_id).toBe(secondInstance.connector_instance_id);
  });

  it("disables a non-default credential by removing binding and deleting instance", async () => {
    mockStandardGets(true);
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(true),
    );
    const user = userEvent.setup();
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Sales/)).toBeInTheDocument();
    });
    const salesCheckbox = screen.getByRole("checkbox", {
      name: /Slack OAuth — Sales/i,
    });
    await user.click(salesCheckbox);
    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalled();
    });
    const credDelete = mockDelete.mock.calls.find(
      (c) =>
        c[0] ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential",
    );
    expect(credDelete?.[1]?.params?.path?.instance_id).toBe(
      secondInstance.connector_instance_id,
    );
    const instDelete = mockDelete.mock.calls.find(
      (c) =>
        c[0] ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
    );
    expect(instDelete?.[1]?.params?.path?.instance_id).toBe(
      secondInstance.connector_instance_id,
    );
  });

  it("shows needs_reauth credentials in the checklist with a warning and re-authorize button", async () => {
    mockStandardGets(true, { secondNeedsReauth: true });
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(true),
    );
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    // Row still appears, checkbox reflects the binding (enabled).
    const salesCheckbox = await screen.findByRole("checkbox", {
      name: /Slack OAuth — Sales/i,
    });
    expect(salesCheckbox).toBeChecked();
    // Badge next to the row conveys the re-auth state.
    expect(screen.getAllByText(/Needs re-authorization/i).length).toBeGreaterThan(0);
    // Re-authorize button is rendered.
    expect(
      screen.getByRole("button", { name: /Re-authorize/i }),
    ).toBeInTheDocument();
    // Summary banner at the top shows the count.
    expect(
      screen.getByText(/1 credential needs re-authorization/i),
    ).toBeInTheDocument();
  });

  it("does not auto-delete the default instance when its credential binding is empty", async () => {
    mockStandardGets(false);
    const emptyBinding = new Map<string, InstanceCredentialBinding | null>();
    emptyBinding.set(defaultInstance.connector_instance_id, {
      agent_id: 42,
      connector_id: "slack",
      credential_id: null,
      oauth_connection_id: null,
    });
    mockUseConnectorInstanceCredentialBindings.mockReturnValue({
      data: emptyBinding,
      isLoading: false,
      isPending: false,
      isFetching: false,
      status: "success" as const,
    });
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Acme/)).toBeInTheDocument();
    });
    await waitFor(
      () => {
        const instDeletes = mockDelete.mock.calls.filter(
          (c) =>
            c[0] ===
            "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
        );
        expect(instDeletes.length).toBe(0);
      },
      { timeout: 1500 },
    );
  });

  it("auto-deletes a non-default instance with no credential binding (orphan cleanup)", async () => {
    mockStandardGets(true);
    const m = new Map<string, InstanceCredentialBinding | null>();
    m.set(defaultInstance.connector_instance_id, oauthBinding("oconn_1"));
    m.set(secondInstance.connector_instance_id, {
      agent_id: 42,
      connector_id: "slack",
      credential_id: null,
      oauth_connection_id: null,
    });
    mockUseConnectorInstanceCredentialBindings.mockReturnValue({
      data: m,
      isLoading: false,
      isPending: false,
      isFetching: false,
      status: "success" as const,
    });
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
          },
        ]}
      />,
    );
    await waitFor(() => {
      expect(screen.getByText(/Slack OAuth — Sales/)).toBeInTheDocument();
    });
    await waitFor(() => {
      const instDeletes = mockDelete.mock.calls.filter(
        (c) =>
          c[0] ===
          "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
      );
      expect(instDeletes.length).toBeGreaterThan(0);
      expect(instDeletes[0]?.[1]?.params?.path?.instance_id).toBe(
        secondInstance.connector_instance_id,
      );
    });
  });

  it("re-authorize button navigates to the OAuth authorize URL with replace param", async () => {
    mockStandardGets(true, { secondNeedsReauth: true });
    mockUseConnectorInstanceCredentialBindings.mockReturnValue(
      bindingsQueryResult(true),
    );
    const assignHref = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        ...window.location,
        set href(url: string) {
          assignHref(url);
        },
        get href() {
          return "http://localhost/";
        },
      },
    });
    const user = userEvent.setup();
    renderWithProviders(
      <ConnectorInstancesSection
        agentId={42}
        connectorId="slack"
        requiredCredentials={[
          {
            service: "slack",
            auth_type: "oauth2",
            oauth_provider: "slack",
            oauth_scopes: ["chat:write"],
          },
        ]}
      />,
    );
    const button = await screen.findByRole("button", { name: /Re-authorize/i });
    await user.click(button);
    expect(assignHref).toHaveBeenCalled();
    const url = assignHref.mock.calls[0]?.[0] as string;
    expect(url).toContain("/v1/oauth/slack/authorize");
    expect(url).toContain("replace=oconn_2");
    expect(url).toContain("scope=chat%3Awrite");
  });
});
