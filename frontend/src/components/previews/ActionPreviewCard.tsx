import type { ActionPreviewConfig } from "@/hooks/useActionSchema";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { ActionPreviewSummary } from "@/components/ActionPreviewSummary";
import { EventPreviewLayout } from "./EventPreviewLayout";
import { MessagePreviewLayout } from "./MessagePreviewLayout";
import { RecordPreviewLayout } from "./RecordPreviewLayout";

interface ActionPreviewCardProps {
  preview: ActionPreviewConfig | null;
  parameters: Record<string, unknown>;
  /** Fallback props for GenericPreviewLayout when no preview config exists. */
  actionType: string;
  schema: ParametersSchema | null;
  actionName: string | null;
  displayTemplate?: string | null;
  resourceDetails?: Record<string, unknown> | null;
}

/**
 * Dispatches to the correct layout component based on the connector's
 * preview config. Falls back to ActionPreviewSummary wrapped in a card
 * when no preview config is defined.
 */
export function ActionPreviewCard({
  preview,
  parameters,
  actionType,
  schema,
  actionName,
  displayTemplate,
  resourceDetails,
}: ActionPreviewCardProps) {
  if (preview) {
    switch (preview.layout) {
      case "event":
        return (
          <EventPreviewLayout parameters={parameters} fields={preview.fields} />
        );
      case "message":
        return (
          <MessagePreviewLayout
            parameters={parameters}
            fields={preview.fields}
          />
        );
      case "record":
        return (
          <RecordPreviewLayout
            parameters={parameters}
            fields={preview.fields}
          />
        );
    }
  }

  // Fallback: render ActionPreviewSummary in a styled card
  return (
    <div className="bg-slate-800 dark:bg-slate-900 overflow-hidden rounded-xl p-4">
      <div className="text-sm text-slate-200">
        <ActionPreviewSummary
          actionType={actionType}
          parameters={parameters}
          schema={schema}
          actionName={actionName}
          displayTemplate={displayTemplate}
          resourceDetails={resourceDetails}
        />
      </div>
    </div>
  );
}
