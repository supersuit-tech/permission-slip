import { useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
import { useCreateActionConfig } from "@/hooks/useCreateActionConfig";
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
  const { createActionConfig, isPending } = useCreateActionConfig();
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

  async function handleUseTemplate(template: ActionConfigTemplate) {
    setPendingTemplateId(template.id);
    try {
      await createActionConfig({
        agent_id: agentId,
        connector_id: connectorId,
        action_type: template.action_type,
        name: template.name,
        description: template.description ?? undefined,
        parameters: template.parameters as Record<string, unknown>,
      });
      toast.success(`Configuration "${template.name}" created`);
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

  const buttonsDisabled = isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
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
          <div className="space-y-6 py-2">
            {grouped.map((group) => (
              <section key={group.actionType} className="space-y-3">
                <h3 className="text-sm font-semibold">{group.actionName}</h3>
                <div className="space-y-3">
                  {group.items.map((template) => (
                    <RecommendedTemplateCard
                      key={template.id}
                      template={template}
                      onUseTemplate={() => void handleUseTemplate(template)}
                      onCustomize={() => handleCustomize(template)}
                      useDisabled={buttonsDisabled}
                      customizeDisabled={buttonsDisabled}
                      usePending={
                        isPending && pendingTemplateId === template.id
                      }
                    />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function RecommendedTemplateCard({
  template,
  onUseTemplate,
  onCustomize,
  useDisabled,
  customizeDisabled,
  usePending,
}: {
  template: ActionConfigTemplate;
  onUseTemplate: () => void;
  onCustomize: () => void;
  useDisabled: boolean;
  customizeDisabled: boolean;
  usePending: boolean;
}) {
  const paramEntries = Object.entries(template.parameters);

  return (
    <div className="rounded-lg border border-input p-3">
      <div className="space-y-2">
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
        <div className="flex flex-wrap gap-2 pt-1">
          <Button
            type="button"
            size="sm"
            onClick={onUseTemplate}
            disabled={useDisabled}
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
            disabled={customizeDisabled}
          >
            Customize
          </Button>
        </div>
      </div>
    </div>
  );
}
