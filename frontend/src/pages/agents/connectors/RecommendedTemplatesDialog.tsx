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
import { Label } from "@/components/ui/label";
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { useApplyActionConfigTemplate } from "@/hooks/useApplyActionConfigTemplate";
import { useBulkApplyActionConfigTemplates } from "@/hooks/useBulkApplyActionConfigTemplates";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { TemplateParamBadge } from "./TemplatePicker";
import { cn } from "@/lib/utils";

export type ApprovalMode = "auto_approve" | "requires_approval";

export type OperationTypeUI = ConnectorAction["operation_type"];

export interface RecommendedTemplatesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  onCustomize: (template: ActionConfigTemplate, approvalMode: ApprovalMode) => void;
}

const approvalModeOptions: { label: string; value: ApprovalMode }[] = [
  { label: "Auto-approve", value: "auto_approve" },
  { label: "Requires approval", value: "requires_approval" },
];

const operationSectionTitle: Record<OperationTypeUI, string> = {
  read: "Read actions",
  write: "Write actions",
  delete: "Delete actions",
};

function defaultApprovalMode(template: ActionConfigTemplate): ApprovalMode {
  return template.standing_approval != null ? "auto_approve" : "requires_approval";
}

export function RecommendedTemplatesDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  onCustomize,
}: RecommendedTemplatesDialogProps) {
  const { templates, isLoading, error } =
    useActionConfigTemplates(connectorId);
  const { applyTemplate, isPending } = useApplyActionConfigTemplate();
  const { bulkApply, isBulkPending } = useBulkApplyActionConfigTemplates();
  const [pendingTemplateId, setPendingTemplateId] = useState<string | null>(
    null,
  );
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [approvalModes, setApprovalModes] = useState<Record<string, ApprovalMode>>({});

  const [quickRead, setQuickRead] = useState<ApprovalMode>("auto_approve");
  const [quickWrite, setQuickWrite] = useState<ApprovalMode>("requires_approval");
  const [quickDelete, setQuickDelete] = useState<ApprovalMode>("requires_approval");

  const getApprovalMode = useCallback(
    (template: ActionConfigTemplate): ApprovalMode =>
      approvalModes[template.id] ?? defaultApprovalMode(template),
    [approvalModes],
  );

  const handleApprovalModeChange = useCallback(
    (templateId: string, mode: ApprovalMode) => {
      setApprovalModes((prev) => ({ ...prev, [templateId]: mode }));
    },
    [],
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
    (template: ActionConfigTemplate): OperationTypeUI =>
      operationTypeByActionType.get(template.action_type) ?? "write",
    [operationTypeByActionType],
  );

  const liveTemplates = useMemo(
    () => templates.filter((t) => actionTypeSet.has(t.action_type)),
    [templates, actionTypeSet],
  );

  const groupedByOperation = useMemo(() => {
    const opOrder: OperationTypeUI[] = ["read", "write", "delete"];
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
        items: ActionConfigTemplate[];
      }[];
    }[] = [];

    for (const op of opOrder) {
      const byAction = new Map<string, ActionConfigTemplate[]>();
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
      return (
        ids.length > 0 && ids.every((id) => selectedIds.has(id))
      );
    },
    [templateIdsForOperation, selectedIds],
  );

  const toggleSelectOperation = useCallback(
    (op: OperationTypeUI) => {
      const ids = templateIdsForOperation(op);
      const allOn = ids.length > 0 && ids.every((id) => selectedIds.has(id));
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
    [templateIdsForOperation],
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

  async function handleUseTemplate(template: ActionConfigTemplate) {
    const approvalMode = getApprovalMode(template);
    setPendingTemplateId(template.id);
    try {
      const res = await applyTemplate({
        templateId: template.id,
        agentId,
        approvalMode,
      });
      const sa = res.standing_approval;
      toast.success(
        sa
          ? `Configuration "${template.name}" created with auto-approval`
          : `Configuration "${template.name}" created`,
      );
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create action configuration",
      );
    } finally {
      setPendingTemplateId(null);
    }
  }

  function handleCustomize(template: ActionConfigTemplate) {
    onOpenChange(false);
    onCustomize(template, getApprovalMode(template));
  }

  async function handleBulkApply() {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    const modes: Record<string, ApprovalMode> = {};
    for (const id of ids) {
      const tpl = liveTemplates.find((t) => t.id === id);
      if (tpl) {
        modes[id] = getApprovalMode(tpl);
      }
    }

    try {
      const res = await bulkApply({
        templateIds: ids,
        agentId,
        approvalModes: modes,
      });
      const succeeded = res.results.filter((r) => r.success);
      const failed = res.results.filter((r) => !r.success);

      if (failed.length === 0) {
        toast.success(
          `${succeeded.length} configuration${succeeded.length === 1 ? "" : "s"} created`,
        );
        setSelectedIds(new Set());
        onOpenChange(false);
      } else if (succeeded.length === 0) {
        toast.error("Failed to create configurations");
      } else {
        toast.warning(
          `${succeeded.length} of ${res.results.length} created. ${failed.length} failed.`,
        );
        const failedIds = new Set(failed.map((r) => r.template_id));
        setSelectedIds(failedIds);
      }
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to apply templates",
      );
    }
  }

  const anyPending = isPending || isBulkPending;

  const selectClassName = cn(
    "border-input bg-background text-foreground",
    "h-9 max-w-[11rem] min-w-0 flex-1 rounded-md border px-2 text-sm shadow-xs",
    "focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] focus-visible:outline-none",
    "disabled:cursor-not-allowed disabled:opacity-50",
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85dvh] flex-col sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Recommended Templates</DialogTitle>
          <DialogDescription>
            Start from a curated configuration for this connector. Use a
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
            No recommended templates are available for this connector.
          </p>
        ) : (
          <>
            <div className="bg-muted/40 space-y-3 rounded-lg border border-input p-3">
              <p className="text-sm font-semibold">Quick setup</p>
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                  <Label
                    htmlFor="quick-read"
                    className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
                  >
                    Read actions
                  </Label>
                  <select
                    id="quick-read"
                    className={selectClassName}
                    value={quickRead}
                    onChange={(e) =>
                      setQuickRead(e.target.value as ApprovalMode)
                    }
                    disabled={anyPending}
                  >
                    {approvalModeOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                  <Label
                    htmlFor="quick-write"
                    className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
                  >
                    Write actions
                  </Label>
                  <select
                    id="quick-write"
                    className={selectClassName}
                    value={quickWrite}
                    onChange={(e) =>
                      setQuickWrite(e.target.value as ApprovalMode)
                    }
                    disabled={anyPending}
                  >
                    {approvalModeOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                  <Label
                    htmlFor="quick-delete"
                    className="text-muted-foreground w-28 shrink-0 text-xs sm:text-sm"
                  >
                    Delete actions
                  </Label>
                  <select
                    id="quick-delete"
                    className={selectClassName}
                    value={quickDelete}
                    onChange={(e) =>
                      setQuickDelete(e.target.value as ApprovalMode)
                    }
                    disabled={anyPending}
                  >
                    {approvalModeOptions.map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <Button
                type="button"
                size="sm"
                variant="secondary"
                className="w-full sm:w-auto"
                onClick={handleQuickApply}
                disabled={anyPending || liveTemplates.length === 0}
              >
                Apply
              </Button>
            </div>

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
                                  approvalMode={getApprovalMode(template)}
                                  onApprovalModeChange={(mode) =>
                                    handleApprovalModeChange(template.id, mode)
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
  approvalMode,
  onApprovalModeChange,
  onUseTemplate,
  onCustomize,
  disabled,
  usePending,
}: {
  template: ActionConfigTemplate;
  selected: boolean;
  onToggleSelected: () => void;
  approvalMode: ApprovalMode;
  onApprovalModeChange: (mode: ApprovalMode) => void;
  onUseTemplate: () => void;
  onCustomize: () => void;
  disabled: boolean;
  usePending: boolean;
}) {
  const paramEntries = Object.entries(template.parameters);

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
          {paramEntries.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {paramEntries.map(([key, value]) => (
                <TemplateParamBadge key={key} name={key} value={value} />
              ))}
            </div>
          )}
          <SegmentedControl
            options={approvalModeOptions}
            value={approvalMode}
            onChange={onApprovalModeChange}
            ariaLabel="Approval mode"
          />
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
