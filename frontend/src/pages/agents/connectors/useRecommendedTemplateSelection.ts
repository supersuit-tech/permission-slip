import { useCallback, useState } from "react";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ApprovalMode, OperationTypeUI } from "./recommendedTemplatesTypes";

export function useRecommendedTemplateSelection(
  liveTemplates: ActionConfigTemplate[],
  getOperationType: (template: ActionConfigTemplate) => OperationTypeUI,
) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [approvalModes, setApprovalModes] = useState<Record<string, ApprovalMode>>({});

  const [quickRead, setQuickRead] = useState<ApprovalMode>("auto_approve");
  const [quickWrite, setQuickWrite] = useState<ApprovalMode>("requires_approval");
  const [quickDelete, setQuickDelete] = useState<ApprovalMode>("requires_approval");

  const getApprovalMode = useCallback(
    (template: ActionConfigTemplate): ApprovalMode =>
      approvalModes[template.id] ??
      (template.standing_approval != null ? "auto_approve" : "requires_approval"),
    [approvalModes],
  );

  const handleApprovalModeChange = useCallback(
    (templateId: string, mode: ApprovalMode) => {
      setApprovalModes((prev) => ({ ...prev, [templateId]: mode }));
    },
    [],
  );

  const toggleSelected = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const allSelected =
    liveTemplates.length > 0 && selectedIds.size === liveTemplates.length;

  const toggleSelectAll = useCallback(() => {
    if (allSelected) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(liveTemplates.map((t) => t.id)));
    }
  }, [allSelected, liveTemplates]);

  const templateIdsForOperation = useCallback(
    (op: OperationTypeUI) =>
      liveTemplates.filter((t) => getOperationType(t) === op).map((t) => t.id),
    [liveTemplates, getOperationType],
  );

  const allSelectedInOperation = useCallback(
    (op: OperationTypeUI) => {
      const ids = templateIdsForOperation(op);
      return ids.length > 0 && ids.every((id) => selectedIds.has(id));
    },
    [templateIdsForOperation, selectedIds],
  );

  const toggleSelectOperation = useCallback(
    (op: OperationTypeUI) => {
      const ids = templateIdsForOperation(op);
      const allOn =
        ids.length > 0 && ids.every((id) => selectedIds.has(id));
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (allOn) {
          for (const id of ids) {
            next.delete(id);
          }
        } else {
          for (const id of ids) {
            next.add(id);
          }
        }
        return next;
      });
    },
    [templateIdsForOperation, selectedIds],
  );

  const handleQuickApply = useCallback(() => {
    setApprovalModes((prev) => {
      const next = { ...prev };
      for (const t of liveTemplates) {
        const op = getOperationType(t);
        next[t.id] =
          op === "read"
            ? quickRead
            : op === "write"
              ? quickWrite
              : quickDelete;
      }
      return next;
    });
    setSelectedIds(new Set(liveTemplates.map((t) => t.id)));
  }, [liveTemplates, getOperationType, quickRead, quickWrite, quickDelete]);

  return {
    selectedIds,
    setSelectedIds,
    approvalModes,
    getApprovalMode,
    handleApprovalModeChange,
    allSelected,
    toggleSelectAll,
    toggleSelected,
    templateIdsForOperation,
    allSelectedInOperation,
    toggleSelectOperation,
    handleQuickApply,
    quickRead,
    setQuickRead,
    quickWrite,
    setQuickWrite,
    quickDelete,
    setQuickDelete,
  };
}
