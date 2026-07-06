import { describe, it, expect } from "vitest";
import {
  execParamsSatisfyConfigConstraints,
  findBestMatchingActionConfig,
  resourceDetailsToConstraintMeta,
} from "../matchActionConfig";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";

function makeConfig(
  overrides: Partial<ActionConfiguration> = {},
): ActionConfiguration {
  return {
    id: "ac_test",
    agent_id: 1,
    connector_id: "protonmail",
    action_type: "protonmail.read_email",
    parameters: { folder: "*", message_id: "*" },
    status: "active",
    name: "Wildcard read",
    description: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as ActionConfiguration;
}

describe("findBestMatchingActionConfig", () => {
  const execParams = { folder: "INBOX", message_id: 92 };

  it("prefers the specific config over the wildcard when both match", () => {
    const wildcard = makeConfig({ id: "ac_wild", parameters: { folder: "*", message_id: "*" } });
    const specific = makeConfig({
      id: "ac_specific",
      parameters: {
        folder: "INBOX",
        message_id: "*",
        $meta: { from: "invoice@anthropic.com" },
      },
    });
    const meta = resourceDetailsToConstraintMeta({
      subject: "Receipt",
      from: ["invoice@anthropic.com"],
    });

    const match = findBestMatchingActionConfig(
      [wildcard, specific],
      "protonmail.read_email",
      execParams,
      meta,
    );
    expect(match?.id).toBe("ac_specific");
  });

  it("requires resolved metadata to match $meta constraints", () => {
    const specific = makeConfig({
      id: "ac_specific",
      parameters: {
        folder: "*",
        message_id: "*",
        $meta: { from: "invoice@anthropic.com" },
      },
    });

    expect(
      findBestMatchingActionConfig(
        [specific],
        "protonmail.read_email",
        execParams,
        null,
      ),
    ).toBeNull();

    expect(
      execParamsSatisfyConfigConstraints(
        specific.parameters as Record<string, unknown>,
        execParams,
        resourceDetailsToConstraintMeta({ from: ["invoice@anthropic.com"] }),
      ),
    ).toBe(true);
  });
});
