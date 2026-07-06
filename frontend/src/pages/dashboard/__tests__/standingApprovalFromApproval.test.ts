import { describe, it, expect } from "vitest";
import {
  buildCreateStandingApprovalFromApproval,
  findMatchingActionConfigForApproval,
} from "../standingApprovalFromApproval";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";

function makeApproval(overrides?: Partial<ApprovalSummary>): ApprovalSummary {
  return {
    approval_id: "appr_test",
    agent_id: 1,
    action: {
      type: "protonmail.read_email",
      version: "1",
      parameters: { folder: "INBOX", message_id: 92 },
    },
    context: { description: "read", risk_level: "low" },
    status: "pending",
    expires_at: "2026-12-31T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    resource_details: {
      subject: "Receipt",
      from: ["invoice@anthropic.com"],
    },
    ...overrides,
  } as ApprovalSummary;
}

describe("buildCreateStandingApprovalFromApproval", () => {
  it("uses $meta.from instead of pinning message_id for email reads", () => {
    const request = buildCreateStandingApprovalFromApproval(makeApproval());
    expect(request.constraints).toEqual({
      folder: "*",
      message_id: "*",
      $meta: { from: "invoice@anthropic.com" },
    });
  });

  it("uses empty constraints when a backing action configuration is provided", () => {
    const request = buildCreateStandingApprovalFromApproval(makeApproval(), "ac_specific");
    expect(request.constraints).toEqual({});
    expect(request.source_action_configuration_id).toBe("ac_specific");
  });
});

describe("findMatchingActionConfigForApproval", () => {
  it("selects the most specific matching config", () => {
    const approval = makeApproval();
    const configs: ActionConfiguration[] = [
      {
        id: "ac_wild",
        agent_id: 1,
        connector_id: "protonmail",
        action_type: "protonmail.read_email",
        parameters: { folder: "*", message_id: "*" },
        status: "active",
        name: "Wildcard",
        description: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      } as ActionConfiguration,
      {
        id: "ac_specific",
        agent_id: 1,
        connector_id: "protonmail",
        action_type: "protonmail.read_email",
        parameters: {
          folder: "INBOX",
          message_id: "*",
          $meta: { from: "invoice@anthropic.com" },
        },
        status: "active",
        name: "Anthropic receipts",
        description: null,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      } as ActionConfiguration,
    ];

    expect(findMatchingActionConfigForApproval(configs, approval)?.id).toBe("ac_specific");
  });
});
