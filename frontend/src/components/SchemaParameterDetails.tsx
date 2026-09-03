import { useState } from "react";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { ParametersSchema, SchemaProperty } from "@/lib/parameterSchema";
import { friendlyTypeLabel } from "@/lib/parameterSchema";
import { formatParameterValue, humanizeKey } from "@/lib/formatValues";
import { resolvedResourceDisplayValue } from "@/lib/resourceParameterDisplay";
import {
  isBase64ParamKey,
  formatBinaryParamSummary,
  binaryThumbnailSrc,
} from "@/lib/binaryParamDisplay";
import { KeyValueList, type KeyValueEntry } from "@/components/KeyValueList";

export type { ParametersSchema } from "@/lib/parameterSchema";
export { parseParametersSchema } from "@/lib/parameterSchema";

interface SchemaParameterDetailsProps {
  /** Actual parameter values from the approval request. */
  parameters: Record<string, unknown>;
  /** JSON Schema describing the parameters (from connector action). */
  schema: ParametersSchema | null;
  /** Accepted for caller compatibility; resource names come from resourceDetails. */
  actionType?: string;
  /** Resolved names from the backend (channel_name, user_name, …). */
  resourceDetails?: Record<string, unknown> | null;
}

function parameterDisplayLabel(key: string, prop?: SchemaProperty): string {
  const uiLabel = prop?.["x-ui"]?.label;
  return uiLabel ?? humanizeKey(key);
}

function resolveDisplayValue(
  key: string,
  value: unknown,
  parameters: Record<string, unknown>,
  resourceDetails?: Record<string, unknown> | null,
): string {
  if (isBase64ParamKey(key) && typeof value === "string" && value.length > 0) {
    return formatBinaryParamSummary(value, parameters);
  }
  const resolved = resolvedResourceDisplayValue(key, value, resourceDetails);
  return resolved ?? formatParameterValue(value);
}

interface ParameterRowData {
  key: string;
  label: string;
  value: unknown;
  prop?: SchemaProperty;
  isRequired?: boolean;
  isProvided: boolean;
}

function collectParameterRows(
  parameters: Record<string, unknown>,
  schema: ParametersSchema | null,
): ParameterRowData[] {
  if (!schema?.properties) {
    return Object.entries(parameters).map(([key, value]) => ({
      key,
      label: humanizeKey(key),
      value,
      isProvided: true,
    }));
  }

  const properties = schema.properties;
  const requiredSet = new Set(schema.required ?? []);

  const schemaKeys = Object.keys(properties).filter((key) => {
    const isProvided = key in parameters;
    const isRequired = requiredSet.has(key);
    return isProvided || isRequired;
  });
  const extraKeys = Object.keys(parameters).filter((k) => !properties[k]);

  const rows: ParameterRowData[] = schemaKeys.map((key) => {
    const prop = properties[key]!;
    return {
      key,
      label: parameterDisplayLabel(key, prop),
      value: parameters[key],
      prop,
      isRequired: requiredSet.has(key),
      isProvided: key in parameters,
    };
  });

  for (const key of extraKeys) {
    rows.push({
      key,
      label: humanizeKey(key),
      value: parameters[key],
      isProvided: true,
    });
  }

  return rows;
}

/**
 * Renders action parameters as a clean key/value list for approvers.
 * Full schema metadata (descriptions, types, enums) is available behind
 * a collapsed "Developer details" toggle.
 */
export function SchemaParameterDetails({
  parameters,
  schema,
  resourceDetails,
}: SchemaParameterDetailsProps) {
  const [developerOpen, setDeveloperOpen] = useState(false);
  const rows = collectParameterRows(parameters, schema);

  const entries: KeyValueEntry[] = rows
    .filter((row) => row.isProvided)
    .map((row) => ({
      label: row.label,
      value: resolveDisplayValue(row.key, row.value, parameters, resourceDetails),
      thumbnailSrc: binaryThumbnailSrc(row.key, row.value, parameters),
    }));

  const hasDeveloperDetails = schema?.properties != null;

  return (
    <div className="space-y-3">
      {entries.length > 0 ? (
        <KeyValueList entries={entries} />
      ) : (
        <p className="text-muted-foreground text-sm">No parameters provided.</p>
      )}

      {hasDeveloperDetails && (
        <div className="border-border border-t pt-2">
          <button
            type="button"
            onClick={() => setDeveloperOpen((open) => !open)}
            className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
            aria-expanded={developerOpen}
          >
            <ChevronRight
              className={`size-3 shrink-0 transition-transform duration-150 ${developerOpen ? "rotate-90" : ""}`}
              aria-hidden="true"
            />
            Developer details
          </button>
          {developerOpen && (
            <div className="divide-border mt-2 divide-y">
              {rows.map((row) => (
                <DeveloperParameterRow
                  key={row.key}
                  name={row.key}
                  label={row.label}
                  description={row.prop?.description}
                  value={row.value}
                  type={row.prop?.type}
                  enumValues={row.prop?.enum}
                  defaultValue={row.prop?.default}
                  isRequired={row.isRequired}
                  isProvided={row.isProvided}
                  parameters={parameters}
                  resourceDetails={resourceDetails}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function DeveloperParameterRow({
  name,
  label,
  description,
  value,
  type,
  enumValues,
  defaultValue,
  isRequired,
  isProvided,
  parameters,
  resourceDetails,
}: {
  name: string;
  label: string;
  description?: string;
  value: unknown;
  type?: string;
  enumValues?: string[];
  defaultValue?: unknown;
  isRequired?: boolean;
  isProvided: boolean;
  parameters: Record<string, unknown>;
  resourceDetails?: Record<string, unknown> | null;
}) {
  const displayValue = resolveDisplayValue(name, value, parameters, resourceDetails);
  const thumbnailSrc = binaryThumbnailSrc(name, value, parameters);
  const isDefault =
    defaultValue !== undefined && String(value) === String(defaultValue);
  const isBinary = isBase64ParamKey(name);
  const isMultiline = !isBinary && typeof value === "string" && value.includes("\n");
  const typeLabel = friendlyTypeLabel(type);

  return (
    <div className="space-y-1.5 py-3 first:pt-0 last:pb-0">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-foreground text-xs font-semibold">{label}</span>
        <code className="bg-muted text-muted-foreground rounded px-1 py-0.5 font-mono text-[10px]">
          {name}
        </code>
        {typeLabel && (
          <span className="text-muted-foreground font-mono text-[10px]">{typeLabel}</span>
        )}
        {isRequired && !isProvided && (
          <Badge variant="destructive" className="rounded-full text-[9px] leading-tight">
            missing
          </Badge>
        )}
        {isDefault && (
          <Badge variant="secondary" className="ml-auto text-[9px] leading-tight">
            default
          </Badge>
        )}
      </div>

      {description && (
        <p className="text-muted-foreground text-xs leading-relaxed">{description}</p>
      )}

      {isProvided ? (
        <div className="space-y-2">
          {thumbnailSrc && (
            <img
              src={thumbnailSrc}
              alt=""
              className="border-border max-h-24 rounded-md border object-contain"
            />
          )}
          {isMultiline ? (
            <pre className="bg-muted/60 border-border text-foreground rounded-md border px-3 py-2 font-sans text-xs leading-relaxed break-words whitespace-pre-wrap">
              {displayValue}
            </pre>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              <span className="bg-muted/60 border-border text-foreground inline-block rounded-md border px-2.5 py-1 text-sm font-medium break-all">
                {displayValue}
              </span>
              {enumValues && enumValues.length > 0 && (
                <span className="text-muted-foreground text-[10px]">
                  one of: {enumValues.join(", ")}
                </span>
              )}
            </div>
          )}
        </div>
      ) : (
        <span className="text-muted-foreground text-sm italic">not provided</span>
      )}
    </div>
  );
}
