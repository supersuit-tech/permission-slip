import type { ConnectorAction } from "@/hooks/useConnectorDetail";

export type ApprovalMode = "auto_approve" | "requires_approval";

export type OperationTypeUI = ConnectorAction["operation_type"];

export const approvalModeOptions: { label: string; value: ApprovalMode }[] = [
  { label: "Auto-approve", value: "auto_approve" },
  { label: "Requires approval", value: "requires_approval" },
];
