import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../../test-helpers";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import {
  mockGet,
  mockPost,
  mockPatch,
  mockDelete,
  resetClientMocks,
} from "../../../../api/__mocks__/client";
import { ConnectorInstancesSection } from "../ConnectorInstancesSection";

vi.mock("../../../../lib/supabaseClient");
vi.mock("../../../../api/client");

const defaultInstance = {
  connector_instance_id: "11111111-1111-1111-1111-111111111111",
  agent_id: 42,
  connector_id: "slack",
  label: "Engineering",
  is_default: true,
  enabled_at: "2026-02-18T10:00:00Z",
};

const secondInstance = {
  connector_instance_id: "22222222-2222-2222-2222-222222222222",
  agent_id: 42,
  connector_id: "slack",
  label: "Sales",
  is_default: false,
  enabled_at: "2026-02-19T10:00:00Z",
};

function mockStandardGets(twoInstances = false) {
  mockGet.mockImplementation((path: string) => {
    if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/instances") {
      return Promise.resolve({
        data: {
          data: twoInstances ? [defaultInstance, secondInstance] : [defaultInstance],
        },
      });
    }
    if (
      path ===
      "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential"
    ) {
      return Promise.resolve({
        data: {
          agent_id: 42,
          connector_id: "slack",
          credential_id: null,
          oauth_connection_id: "oconn_1",
        },
      });
    }
    if (path === "/v1/agents/{agent_id}/connectors/{connector_id}/credential") {
      return Promise.resolve({
        data: {
          agent_id: 42,
          connector_id: "slack",
          credential_id: null,
          oauth_connection_id: "oconn_1",
        },
      });
    }
    if (path === "/v1/credentials") {
      return Promise.resolve({ data: { credentials: [] } });
    }
    if (path === "/v1/oauth/connections") {
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
          ],
        },
      });
    }
    if (path === "/v1/oauth/providers") {
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
    mockPatch.mockResolvedValue({ data: {} });
    mockDelete.mockResolvedValue({ data: {} });
    setupAuthMocks({ authenticated: true });
  });

  it("renders instances with default badge", async () => {
    mockStandardGets(false);
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
      expect(screen.getByText("Engineering")).toBeInTheDocument();
    });
    expect(screen.getByText("Default")).toBeInTheDocument();
  });

  it("creates a new instance when Add another is submitted", async () => {
    mockStandardGets(false);
    mockPost.mockResolvedValue({
      data: {
        ...secondInstance,
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
      expect(screen.getByText("Engineering")).toBeInTheDocument();
    });
    await user.click(screen.getByRole("button", { name: /Add another/i }));
    await user.type(screen.getByLabelText(/Label/i), "Sales");
    await user.click(screen.getByRole("button", { name: /^Add$/i }));
    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
    const call = mockPost.mock.calls.find(
      (c) => c[0] === "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
    );
    expect(call?.[1]?.body).toEqual({ label: "Sales" });
  });

  it("sets default instance via PATCH", async () => {
    mockStandardGets(true);
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
      expect(screen.getByText("Sales")).toBeInTheDocument();
    });
    await user.click(screen.getByRole("button", { name: /Make default/i }));
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

  it("removes non-default instance via DELETE", async () => {
    mockStandardGets(true);
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
      expect(screen.getByText("Sales")).toBeInTheDocument();
    });
    await user.click(screen.getByRole("button", { name: /Remove instance/i }));
    await user.click(screen.getByRole("button", { name: /^Remove$/i }));
    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalled();
    });
    const call = mockDelete.mock.calls.find(
      (c) =>
        c[0] ===
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
    );
    expect(call?.[1]?.params?.path?.instance_id).toBe(secondInstance.connector_instance_id);
  });
});
