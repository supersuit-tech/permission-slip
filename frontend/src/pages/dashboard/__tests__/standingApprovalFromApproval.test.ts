import { describe, it, expect } from "vitest";
import {
  buildCreateStandingApprovalFromApproval,
  buildCreateStandingApprovalsFromApproval,
  standingApprovalsToCreateFromApproval,
} from "../standingApprovalFromApproval";
import { GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES } from "../googleSharedDriveWorkspace";
import type { ApprovalSummary } from "@/hooks/useApprovals";

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
    expect(request.name).toBe("read");
    expect(request.description).toBe("read");
    expect(request.expires_at).toBeNull();
    expect(request.confirm_unrestricted).toBeUndefined();
  });

  it("confirms unrestricted when the derived constraints are all wildcards", () => {
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
        context: { description: "upload receipt", risk_level: "medium" },
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

  it("pins exact folder_id when the upload is not on a Shared Drive", () => {
    const request = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.upload_drive_file",
          version: "1",
          parameters: { name: "notes.md", folder_id: "1myDriveFolder" },
        },
        resource_details: { folder_name: "Receipts" },
      }),
    );
    expect(request.constraints).toEqual({
      name: "notes.md",
      folder_id: "1myDriveFolder",
    });
  });

  it("uses $meta.drive_id for Shared Drive list, get, and Sheets reads", () => {
    const list = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.list_drive_files",
          version: "1",
          parameters: { folder_id: "1nested", query: "receipt" },
        },
        resource_details: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
      }),
    );
    expect(list.constraints).toEqual({
      folder_id: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });

    const getFile = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.get_drive_file",
          version: "1",
          parameters: { file_id: "1file" },
        },
        resource_details: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
      }),
    );
    expect(getFile.constraints).toEqual({
      file_id: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });

    const sheets = buildCreateStandingApprovalFromApproval(
      makeApproval({
        action: {
          type: "google.sheets_read_range",
          version: "1",
          parameters: { spreadsheet_id: "1sheet", range: "Sheet1!A1:B2" },
        },
        resource_details: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
      }),
    );
    expect(sheets.constraints).toEqual({
      spreadsheet_id: "*",
      range: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
  });

  it("expands Shared Drive always-allow into the Drive + Sheets workspace set", () => {
    const requests = buildCreateStandingApprovalsFromApproval(
      makeApproval({
        action: {
          type: "google.upload_drive_file",
          version: "1",
          parameters: { name: "receipt.pdf", folder_id: "1nested" },
        },
        resource_details: {
          drive_id: "0AKbIIKZ8knmBUk9PVA",
          drive_name: "Assistant Drive",
        },
      }),
    );
    expect(requests.map((req) => req.action_type)).toEqual([
      ...GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES,
    ]);
    expect(requests[0]?.name).toBe("Upload drive file — inside Assistant Drive");
    expect(requests.find((req) => req.action_type === "google.sheets_read_range")?.constraints).toEqual({
      spreadsheet_id: "*",
      range: "*",
      $meta: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
  });

  it("skips workspace actions that already have a standing approval", () => {
    const approval = makeApproval({
      action: {
        type: "google.list_drive_files",
        version: "1",
        parameters: { folder_id: "1nested" },
      },
      resource_details: { drive_id: "0AKbIIKZ8knmBUk9PVA" },
    });
    const requests = standingApprovalsToCreateFromApproval(approval, [
      { agent_id: 1, action_type: "google.upload_drive_file" },
    ]);
    expect(requests.map((req) => req.action_type)).not.toContain(
      "google.upload_drive_file",
    );
    expect(requests.map((req) => req.action_type)).toContain(
      "google.list_drive_files",
    );
  });
});
