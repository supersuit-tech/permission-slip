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
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { useApplyActionConfigTemplate } from "@/hooks/useApplyActionConfigTemplate";
import { useBulkApplyActionConfigTemplates } from "@/hooks/useBulkApplyActionConfigTemplates";
import { Badge } from "@/components/ui/badge";
import { TemplateParamBadge } from "./TemplatePicker";

export interface RecommendedTemplatesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  onCustomize: (template: ActionConfigTemplate) => void;
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

  const liveTemplates = useMemo(
    () => templates.filter((t) => actionTypeSet.has(t.action_type)),
    [templates, actionTypeSet],
  );

  const grouped = useMemo(() => {
    const byType = new Map<string, ActionConfigTemplate[]>();
    for (const t of liveTemplates) {
      const list = byType.get(t.action_type) ?? [];
      list.push(t);
      byType.set(t.action_type, list);
    }
    const order = actions.map((a) => a.action_type);
    const groups: { actionType: string; actionName: string; items: ActionConfigTemplate[] }[] =
      [];
    for (const actionType of order) {
      const items = byType.get(actionType);
      if (items && items.length > 0) {
        groups.push({
          actionType,
          actionName: actionNameByType.get(actionType) ?? actionType,
          items,
        });
      }
    }
    return groups;
  }, [liveTemplates, actions, actionNameByType]);

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

  async function handleUseTemplate(template: ActionConfigTemplate) {
    setPendingTemplateId(template.id);
    try {
      const res = await applyTemplate({ templateId: template.id, agentId });
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
    onCustomize(template);
  }

  async function handleBulkApply() {
    const ids = Array.from(selectedIds);
    if (ids.length === 0) return;

    try {
      const res = await bulkApply({ templateIds: ids, agentId });
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
        // Keep only the failed templates selected.
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
        ) : grouped.length === 0 ? (
          <p className="text-muted-foreground py-4 text-sm">
            No recommended templates are available for this connector.
          </p>
        ) : (
          <>
            {/* Select all toggle */}
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

            {/* Scrollable template list */}
            <div className="min-h-0 flex-1 overflow-y-auto">
              <div className="space-y-6 py-2">
                {grouped.map((group) => (
                  <section key={group.actionType} className="space-y-3">
                    <h3 className="text-sm font-semibold">{group.actionName}</h3>
                    <div className="space-y-3">
                      {group.items.map((template) => (
                        <RecommendedTemplateCard
                          key={template.id}
                          template={template}
                          selected={selectedIds.has(template.id)}
                          onToggleSelected={() => toggleSelected(template.id)}
                          onUseTemplate={() => void handleUseTemplate(template)}
                          onCustomize={() => handleCustomize(template)}
                          disabled={anyPending}
                          usePending={
                            isPending && pendingTemplateId === template.id
                          }
                        />
                      ))}
                    </div>
                  </section>
                ))}
              </div>
            </div>

            {/* Sticky footer */}
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
  template: ActionConfigTemplate;
  selected: boolean;
  onToggleSelected: () => void;
  onUseTemplate: () => void;
  onCustomize: () => void;
  disabled: boolean;
  usePending: boolean;
}) {
  const paramEntries = Object.entries(template.parameters);
  const autoApproved = template.standing_approval != null;

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
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-medium">{template.name}</p>
            {autoApproved ? (
              <Badge>Auto-approved</Badge>
            ) : (
              <Badge variant="secondary" className="text-muted-foreground">
                Requires approval each time
              </Badge>
            )}
          </div>
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
