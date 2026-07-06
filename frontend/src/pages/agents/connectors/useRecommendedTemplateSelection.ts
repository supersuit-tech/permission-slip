import { useCallback, useState } from "react";
import type { StandingApprovalTemplate } from "@/hooks/useStandingApprovalTemplates";
import type { OperationTypeUI } from "./recommendedTemplatesTypes";

export function useRecommendedTemplateSelection(
  liveTemplates: StandingApprovalTemplate[],
  getOperationType: (template: StandingApprovalTemplate) => OperationTypeUI,
) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const [quickRead, setQuickRead] = useState(true);
  const [quickWrite, setQuickWrite] = useState(false);
  const [quickEdit, setQuickEdit] = useState(false);
  const [quickDelete, setQuickDelete] = useState(false);

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
    const next = new Set<string>();
    for (const t of liveTemplates) {
      const op = getOperationType(t);
      const include =
        op === "read"
          ? quickRead
          : op === "write"
            ? quickWrite
            : op === "edit"
              ? quickEdit
              : quickDelete;
      if (include) {
        next.add(t.id);
      }
    }
    setSelectedIds(next);
  }, [liveTemplates, getOperationType, quickRead, quickWrite, quickEdit, quickDelete]);

  return {
    selectedIds,
    setSelectedIds,
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
    quickEdit,
    setQuickEdit,
    quickDelete,
    setQuickDelete,
  };
}
