import { describe, expect, it } from "vitest";
import {
  connectorInstanceFromParameters,
  connectorInstanceFromStandingApprovalId,
  isAllAccountsConnectorInstance,
  mergeConnectorInstanceIntoParameters,
  parametersWithoutConnectorInstance,
  resolveConnectorInstanceAccountLabel,
  selectableConnectorInstancesForAccount,
  standingApprovalConnectorInstanceIdForUpdate,
} from "../connectorInstanceAccount";

const instances = [
  {
    connector_instance_id: "11111111-1111-1111-1111-111111111111",
    agent_id: 1,
    connector_id: "protonmail",
    display: "Personal",
    is_default: true,
    enabled_at: "2026-01-01T00:00:00Z",
  },
  {
    connector_instance_id: "22222222-2222-2222-2222-222222222222",
    agent_id: 1,
    connector_id: "protonmail",
    display: "Work",
    is_default: false,
    enabled_at: "2026-01-02T00:00:00Z",
  },
  {
    connector_instance_id: "33333333-3333-3333-3333-333333333333",
    agent_id: 1,
    connector_id: "protonmail",
    display: "",
    is_default: true,
    enabled_at: "2026-01-01T00:00:00Z",
  },
];

describe("connectorInstanceAccount", () => {
  it("treats absent, null, and * as all accounts", () => {
    expect(isAllAccountsConnectorInstance(undefined)).toBe(true);
    expect(isAllAccountsConnectorInstance(null)).toBe(true);
    expect(isAllAccountsConnectorInstance("*")).toBe(true);
  });

  it("resolves account labels from parameters", () => {
    expect(resolveConnectorInstanceAccountLabel("*", instances)).toBe(
      "All accounts",
    );
    expect(
      resolveConnectorInstanceAccountLabel(
        "11111111-1111-1111-1111-111111111111",
        instances,
      ),
    ).toBe("Personal");
    expect(resolveConnectorInstanceAccountLabel("Work", instances)).toBe(
      "Work",
    );
  });

  it("strips connector_instance from generic parameter lists", () => {
    expect(
      parametersWithoutConnectorInstance({
        repo: "org/repo",
        connector_instance: "*",
      }),
    ).toEqual({ repo: "org/repo" });
  });

  it("reads and writes connector_instance selectors", () => {
    expect(
      connectorInstanceFromParameters({ connector_instance: "Work" }),
    ).toBe("Work");
    expect(connectorInstanceFromParameters({})).toBe("*");
    expect(
      mergeConnectorInstanceIntoParameters({ repo: "x" }, "22222222-2222-2222-2222-222222222222"),
    ).toEqual({
      repo: "x",
      connector_instance: "22222222-2222-2222-2222-222222222222",
    });
  });

  it("maps standing approval connector_instance_id to selector values", () => {
    expect(connectorInstanceFromStandingApprovalId(null)).toBe("*");
    expect(connectorInstanceFromStandingApprovalId(undefined)).toBe("*");
    expect(
      connectorInstanceFromStandingApprovalId(
        "22222222-2222-2222-2222-222222222222",
      ),
    ).toBe("22222222-2222-2222-2222-222222222222");
    expect(
      standingApprovalConnectorInstanceIdForUpdate("*"),
    ).toBeNull();
    expect(
      standingApprovalConnectorInstanceIdForUpdate(
        "22222222-2222-2222-2222-222222222222",
      ),
    ).toBe("22222222-2222-2222-2222-222222222222");
  });

  it("excludes credential-less instances from account selectors", () => {
    expect(
      selectableConnectorInstancesForAccount(instances, "*").map(
        (i) => i.connector_instance_id,
      ),
    ).toEqual([
      "11111111-1111-1111-1111-111111111111",
      "22222222-2222-2222-2222-222222222222",
    ]);
  });

  it("keeps a selected credential-less instance in account selectors", () => {
    expect(
      selectableConnectorInstancesForAccount(
        instances,
        "33333333-3333-3333-3333-333333333333",
      ).map((i) => i.connector_instance_id),
    ).toEqual([
      "11111111-1111-1111-1111-111111111111",
      "22222222-2222-2222-2222-222222222222",
      "33333333-3333-3333-3333-333333333333",
    ]);
  });
});
