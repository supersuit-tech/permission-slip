import { buildCreateStandingApprovalFromApproval } from "../standingApprovalFromApproval";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

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

  it("creates a standing approval directly from the approval action", () => {
    const request = buildCreateStandingApprovalFromApproval(makeApproval());
    expect(request.agent_id).toBe(1);
    expect(request.action_type).toBe("protonmail.read_email");
    expect(request.action_version).toBe("1");
    expect(request.expires_at).toBeNull();
    expect(request).not.toHaveProperty("source_action_configuration_id");
    expect(request.confirm_unrestricted).toBeUndefined();
  });

  it("pins exact parameter values for non-email actions", () => {
    const request = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "email.send",
          version: "1",
          parameters: { to: "team@example.com", subject: "Hello" },
        },
      }),
    );
    expect(request.constraints).toEqual({
      to: "team@example.com",
      subject: "Hello",
    });
    expect(request.confirm_unrestricted).toBeUndefined();
  });

  it("confirms unrestricted when derived constraints are all wildcards", () => {
    const request = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.list_calendars",
          version: "1",
          parameters: {},
        },
        resource_details: undefined,
      }),
    );
    expect(request.constraints).toEqual({});
    expect(request.confirm_unrestricted).toBe(true);
  });

  it("uses $meta.drive_id instead of pinning folder_id for Shared Drive uploads", () => {
    const request = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.upload_drive_file",
          version: "1",
          parameters: {
            name: "receipt.pdf",
            folder_id: "1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d",
          },
        },
        resource_details: {
          folder_name: "2026-639-receipts in Assistant Drive",
          drive_id: "0AKbIIKZ8knmBUk9PVA",
        },
      }),
    );
    expect(request.constraints).toEqual({
      folder_id: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
    expect(request.confirm_unrestricted).toBeUndefined();
  });

  it("uses $meta.drive_id for Shared Drive folder creates", () => {
    const request = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.create_drive_folder",
          version: "1",
          parameters: {
            name: "2026-639-receipts",
            parent_id: "0AKbIIKZ8knmBUk9PVA",
          },
        },
        resource_details: {
          folder_name: "Assistant Drive in the / directory",
          drive_id: "0AKbIIKZ8knmBUk9PVA",
        },
      }),
    );
    expect(request.constraints).toEqual({
      parent_id: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
  });
});
