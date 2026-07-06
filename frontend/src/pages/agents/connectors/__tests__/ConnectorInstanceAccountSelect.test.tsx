import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import { ConnectorInstanceAccountSelect } from "../ConnectorInstanceAccountSelect";

const namedInstance: AgentConnectorInstance = {
  connector_instance_id: "22222222-2222-2222-2222-222222222222",
  agent_id: 1,
  connector_id: "protonmail",
  display: "chiedo@chiedo.com",
  is_default: false,
  enabled_at: "2026-01-02T00:00:00Z",
};

const credentialLessInstance: AgentConnectorInstance = {
  connector_instance_id: "33333333-3333-3333-3333-333333333333",
  agent_id: 1,
  connector_id: "protonmail",
  display: "",
  is_default: true,
  enabled_at: "2026-01-01T00:00:00Z",
};

describe("ConnectorInstanceAccountSelect", () => {
  it("hides credential-less instances from the account dropdown", () => {
    render(
      <ConnectorInstanceAccountSelect
        id="account"
        value="*"
        onChange={vi.fn()}
        instances={[namedInstance, credentialLessInstance]}
      />,
    );

    const select = screen.getByLabelText("Account");
    const options = Array.from(select.querySelectorAll("option")).map(
      (option) => option.textContent,
    );

    expect(options).toEqual(["All accounts", "chiedo@chiedo.com"]);
  });

  it("keeps the currently selected credential-less instance visible", () => {
    render(
      <ConnectorInstanceAccountSelect
        id="account"
        value={credentialLessInstance.connector_instance_id}
        onChange={vi.fn()}
        instances={[namedInstance, credentialLessInstance]}
      />,
    );

    const select = screen.getByLabelText("Account") as HTMLSelectElement;
    const options = Array.from(select.querySelectorAll("option")).map(
      (option) => option.textContent,
    );

    expect(options).toEqual([
      "All accounts",
      "chiedo@chiedo.com",
      "Unnamed account",
    ]);
    expect(select.value).toBe(credentialLessInstance.connector_instance_id);
  });
});
