import { useCallback, useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useStandingApprovalTemplates } from "@/hooks/useStandingApprovalTemplates";
import type { StandingApprovalTemplate } from "@/hooks/useStandingApprovalTemplates";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { useApplyStandingApprovalTemplate } from "@/hooks/useApplyStandingApprovalTemplate";
import { useBulkApplyStandingApprovalTemplates } from "@/hooks/useBulkApplyStandingApprovalTemplates";
import { TemplateParamBadge } from "./TemplatePicker";
import { QuickSetupPanel } from "./QuickSetupPanel";
import { type OperationTypeUI } from "./recommendedTemplatesTypes";
import { useRecommendedTemplateSelection } from "./useRecommendedTemplateSelection";
import { templateIsApplied } from "./templateMatching";

export type { OperationTypeUI };

export interface RecommendedTemplatesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  existingRules?: StandingApproval[];
  onCustomize: (template: StandingApprovalTemplate) => void;
}

const operationSectionTitle: Record<OperationTypeUI, string> = {
  read: "Read actions",
  write: "Write actions",
  edit: "Edit actions",
  delete: "Delete actions",
};

export function RecommendedTemplatesDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  existingRules,
  onCustomize,
}: RecommendedTemplatesDialogProps) {
  const { templates, isLoading, error } =
    useStandingApprovalTemplates(connectorId);
  const { applyTemplate, isPending } = useApplyStandingApprovalTemplate();
  const { bulkApply, isBulkPending } = useBulkApplyStandingApprovalTemplates();
  const [pendingTemplateId, setPendingTemplateId] = useState<string | null>(
    null,
  );

  const actionTypeSet = useMemo(
    () => new Set(actions.map((a) => a.action_type)),
    [actions],
  );

  const actionNameByType = useMemo(() => {
    const m = new Map<string, string>();
    for (const a of actions) {
      m.set(a.action_type, a.name);
    }
    return m;
  }, [actions]);

  const operationTypeByActionType = useMemo(() => {
    const m = new Map<string, OperationTypeUI>();
    for (const a of actions) {
      m.set(a.action_type, a.operation_type);
    }
    return m;
  }, [actions]);

  const getOperationType = useCallback(
    (template: StandingApprovalTemplate): OperationTypeUI =>
      operationTypeByActionType.get(template.action_type) ?? "write",
    [operationTypeByActionType],
  );

  const liveTemplates = useMemo(() => {
    const rules = existingRules ?? [];
    return templates.filter(
      (t) =>
        actionTypeSet.has(t.action_type) && !templateIsApplied(t, rules),
    );
  }, [templates, actionTypeSet, existingRules]);
  const hasConnectorMatchingTemplates = useMemo(
    () => templates.some((t) => actionTypeSet.has(t.action_type)),
    [templates, actionTypeSet],
  );

  const {
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
  } = useRecommendedTemplateSelection(liveTemplates, getOperationType);

  const groupedByOperation = useMemo(() => {
    const opOrder: OperationTypeUI[] = ["read", "write", "edit", "delete"];
    const firstActionIndex = new Map<string, number>();
    actions.forEach((a, i) => {
      if (!firstActionIndex.has(a.action_type)) {
        firstActionIndex.set(a.action_type, i);
      }
    });

    const out: {
      operationType: OperationTypeUI;
      subgroups: {
        actionType: string;
        actionName: string;
        items: StandingApprovalTemplate[];
      }[];
    }[] = [];

    for (const op of opOrder) {
      const byAction = new Map<string, StandingApprovalTemplate[]>();
      for (const t of liveTemplates) {
        if (getOperationType(t) !== op) continue;
        const list = byAction.get(t.action_type) ?? [];
        list.push(t);
        byAction.set(t.action_type, list);
      }
      if (byAction.size === 0) continue;

      const subgroups = [...byAction.entries()]
        .sort(
          ([a], [b]) =>
            (firstActionIndex.get(a) ?? 999) - (firstActionIndex.get(b) ?? 999),
        )
        .map(([actionType, items]) => ({
          actionType,
          actionName: actionNameByType.get(actionType) ?? actionType,
          items,
        }));

      out.push({ operationType: op, subgroups });
    }
    return out;
  }, [liveTemplates, actions, actionNameByType, getOperationType]);

  async function handleUseTemplate(template: StandingApprovalTemplate) {
    setPendingTemplateId(template.id);
    try {
      const res = await applyTemplate({
        templateId: template.id,
        agentId,
      });
      const sa = res.standing_approval;
      toast.success(
        sa
          ? `Standing approval "${template.name}" created`
          : `Template "${template.name}" applied`,
      );
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create standing approval",
      );
    } finally {
      setPendingTemplateId(null);
    }
  }

  function handleCustomize(template: StandingApprovalTemplate) {
    onOpenChange(false);
    onCustomize(template);
  }

  async function handleBulkApply() {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    try {
      const res = await bulkApply({
        templateIds: ids,
        agentId,
      });
      const succeeded = res.results.filter((r) => r.success);
      const failed = res.results.filter((r) => !r.success);

      if (failed.length === 0) {
        toast.success(
          `${succeeded.length} standing approval${succeeded.length === 1 ? "" : "s"} created`,
        );
        setSelectedIds(new Set());
        onOpenChange(false);
      } else if (succeeded.length === 0) {
        toast.error("Failed to create standing approvals");
      } else {
        toast.warning(
          `${succeeded.length} of ${res.results.length} created. ${failed.length} failed.`,
        );
        const failedIds = new Set(failed.map((r) => r.template_id));
        setSelectedIds(failedIds);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to apply templates",
      );
    }
  }

  const anyPending = isPending || isBulkPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85dvh] flex-col sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Recommended Templates</DialogTitle>
          <DialogDescription>
            Start from a curated standing approval for this connector. Use a
            template as-is or customize it before saving.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-8">
            <Loader2
              className="text-muted-foreground size-5 animate-spin"
              aria-hidden="true"
            />
            <span className="text-muted-foreground text-sm">
              Loading templates...
            </span>
          </div>
        ) : error ? (
          <p className="text-destructive py-4 text-sm">{error}</p>
        ) : groupedByOperation.length === 0 ? (
          <p className="text-muted-foreground py-4 text-sm">
            {hasConnectorMatchingTemplates
              ? "You've already configured everything we recommend for this connector."
              : "No recommended templates are available for this connector."}
          </p>
        ) : (
          <>
            <QuickSetupPanel
              quickRead={quickRead}
              quickWrite={quickWrite}
              quickEdit={quickEdit}
              quickDelete={quickDelete}
              onQuickReadChange={setQuickRead}
              onQuickWriteChange={setQuickWrite}
              onQuickEditChange={setQuickEdit}
              onQuickDeleteChange={setQuickDelete}
              onApply={handleQuickApply}
              disabled={anyPending}
              applyDisabled={liveTemplates.length === 0}
            />

            <label className="flex items-center gap-2 py-1">
              <Checkbox
                checked={allSelected}
                onCheckedChange={toggleSelectAll}
                disabled={anyPending}
              />
              <span className="text-muted-foreground text-sm">
                Select all ({liveTemplates.length})
              </span>
            </label>

            <div className="min-h-0 flex-1 overflow-y-auto">
              <div className="space-y-6 py-2">
                {groupedByOperation.map((section) => {
                  const op = section.operationType;
                  const countInOp = templateIdsForOperation(op).length;
                  return (
                    <section key={op} className="space-y-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <h2 className="text-base font-semibold">
                          {operationSectionTitle[op]}
                        </h2>
                        <label className="flex items-center gap-2">
                          <Checkbox
                            checked={allSelectedInOperation(op)}
                            onCheckedChange={() => toggleSelectOperation(op)}
                            disabled={anyPending || countInOp === 0}
                          />
                          <span className="text-muted-foreground text-xs sm:text-sm">
                            Select all in section ({countInOp})
                          </span>
                        </label>
                      </div>
                      <div className="space-y-5 pl-0 sm:pl-1">
                        {section.subgroups.map((group) => (
                          <div key={group.actionType} className="space-y-3">
                            <h3 className="text-sm font-medium">
                              {group.actionName}
                            </h3>
                            <div className="space-y-3">
                              {group.items.map((template) => (
                                <RecommendedTemplateCard
                                  key={template.id}
                                  template={template}
                                  selected={selectedIds.has(template.id)}
                                  onToggleSelected={() =>
                                    toggleSelected(template.id)
                                  }
                                  onUseTemplate={() =>
                                    void handleUseTemplate(template)
                                  }
                                  onCustomize={() => handleCustomize(template)}
                                  disabled={anyPending}
                                  usePending={
                                    isPending &&
                                    pendingTemplateId === template.id
                                  }
                                />
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    </section>
                  );
                })}
              </div>
            </div>

            <div className="flex items-center justify-between border-t pt-3">
              <span className="text-muted-foreground text-sm">
                {selectedIds.size} of {liveTemplates.length} selected
              </span>
              <Button
                size="sm"
                onClick={() => void handleBulkApply()}
                disabled={selectedIds.size === 0 || anyPending}
              >
                {isBulkPending && (
                  <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                )}
                Enable Selected ({selectedIds.size})
              </Button>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function RecommendedTemplateCard({
  template,
  selected,
  onToggleSelected,
  onUseTemplate,
  onCustomize,
  disabled,
  usePending,
}: {
  template: StandingApprovalTemplate;
  selected: boolean;
  onToggleSelected: () => void;
  onUseTemplate: () => void;
  onCustomize: () => void;
  disabled: boolean;
  usePending: boolean;
}) {
  const constraintEntries = Object.entries(template.constraints);

  return (
    <div className="rounded-lg border border-input p-3">
      <div className="flex gap-3">
        <div className="pt-0.5">
          <Checkbox
            checked={selected}
            onCheckedChange={onToggleSelected}
            disabled={disabled}
          />
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <p className="text-sm font-medium">{template.name}</p>
          {template.description && (
            <p className="text-muted-foreground text-sm">{template.description}</p>
          )}
          {constraintEntries.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {constraintEntries.map(([key, value]) => (
                <TemplateParamBadge key={key} name={key} value={value} />
              ))}
            </div>
          )}
          <div className="flex flex-wrap gap-2 pt-1">
            <Button
              type="button"
              size="sm"
              onClick={onUseTemplate}
              disabled={disabled}
            >
              {usePending && (
                <Loader2 className="size-4 animate-spin" aria-hidden="true" />
              )}
              Use Template
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={onCustomize}
              disabled={disabled}
            >
              Customize
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
