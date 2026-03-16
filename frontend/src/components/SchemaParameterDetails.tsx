import { Badge } from "@/components/ui/badge";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { friendlyTypeLabel } from "@/lib/parameterSchema";
import { formatParameterValue, humanizeKey } from "@/lib/formatValues";

export type { ParametersSchema } from "@/lib/parameterSchema";
export { parseParametersSchema } from "@/lib/parameterSchema";

interface SchemaParameterDetailsProps {
  /** Actual parameter values from the approval request. */
  parameters: Record<string, unknown>;
  /** JSON Schema describing the parameters (from connector action). */
  schema: ParametersSchema | null;
}

/**
 * Renders action parameters using the connector's JSON Schema for rich
 * display. When a schema is available, parameters are shown with their
 * human-readable descriptions as labels, type annotations, and enum context.
 * Falls back to raw key-value display when no schema is available.
 *
 * Improvements over raw display:
 * - humanizeKey for clean labels when no description provided
 * - tryFormatDateTime for timestamp values
 * - Dividers between parameter rows
 * - Hides unprovided optional parameters
 */
export function SchemaParameterDetails({
  parameters,
  schema,
}: SchemaParameterDetailsProps) {
  if (!schema?.properties) {
    return <FallbackDetails parameters={parameters} />;
  }

  const properties = schema.properties;
  const requiredSet = new Set(schema.required ?? []);

  // Render schema-known parameters first (in schema property order),
  // then any extra parameters not in the schema.
  // Hide unprovided optional params to reduce noise.
  const schemaKeys = Object.keys(properties).filter((key) => {
    const isProvided = key in parameters;
    const isRequired = requiredSet.has(key);
    return isProvided || isRequired;
  });
  const extraKeys = Object.keys(parameters).filter(
    (k) => !properties[k],
  );

  return (
    <div className="divide-border divide-y">
      {schemaKeys.map((key) => {
        // Safe to assert: schemaKeys come from Object.keys(properties).
        const prop = properties[key]!;
        const value = parameters[key];
        const isRequired = requiredSet.has(key);

        return (
          <ParameterRow
            key={key}
            name={key}
            label={prop.description ?? humanizeKey(key)}
            value={value}
            type={prop.type}
            enumValues={prop.enum}
            defaultValue={prop.default}
            isRequired={isRequired}
            isProvided={key in parameters}
          />
        );
      })}
      {extraKeys.map((key) => (
        <ParameterRow
          key={key}
          name={key}
          label={humanizeKey(key)}
          value={parameters[key]}
          isProvided
        />
      ))}
    </div>
  );
}

function ParameterRow({
  name,
  label,
  value,
  type,
  enumValues,
  defaultValue,
  isRequired,
  isProvided,
}: {
  name: string;
  label: string;
  value: unknown;
  type?: string;
  enumValues?: string[];
  defaultValue?: unknown;
  isRequired?: boolean;
  isProvided: boolean;
}) {
  const displayValue = formatParameterValue(value);
  const isDefault =
    defaultValue !== undefined && String(value) === String(defaultValue);
  const isMultiline = typeof value === "string" && value.includes("\n");
  const typeLabel = friendlyTypeLabel(type);

  return (
    <div className="space-y-1.5 py-3 first:pt-0 last:pb-0">
      {/* Label row */}
      <div className="flex items-center gap-1.5">
        <span className="text-foreground/70 text-xs font-semibold tracking-wide uppercase">
          {label !== name ? label : humanizeKey(name)}
        </span>
        {label !== name && (
          <code className="bg-muted text-muted-foreground rounded px-1 py-0.5 text-[10px] font-mono">{name}</code>
        )}
        {typeLabel && (
          <span className="text-muted-foreground text-[10px] font-mono">{typeLabel}</span>
        )}
        {isRequired && !isProvided && (
          <Badge
            variant="destructive"
            className="rounded-full text-[9px] leading-tight"
          >
            missing
          </Badge>
        )}
        {isDefault && (
          <Badge variant="secondary" className="text-[9px] leading-tight ml-auto">
            default
          </Badge>
        )}
      </div>

      {/* Value */}
      {isProvided ? (
        isMultiline ? (
          <pre className="bg-muted/60 border-border rounded-md border px-3 py-2 text-xs leading-relaxed whitespace-pre-wrap break-words font-sans text-foreground">
            {displayValue}
          </pre>
        ) : (
          <div className="flex items-center gap-2">
            <span className="bg-muted/60 border-border text-foreground inline-block rounded-md border px-2.5 py-1 text-sm font-medium break-all">
              {displayValue}
            </span>
            {enumValues && enumValues.length > 0 && (
              <span className="text-muted-foreground text-[10px]">
                one of: {enumValues.join(", ")}
              </span>
            )}
          </div>
        )
      ) : (
        <span className="text-muted-foreground text-sm italic">not provided</span>
      )}
    </div>
  );
}

function FallbackDetails({
  parameters,
}: {
  parameters: Record<string, unknown>;
}) {
  return (
    <dl className="divide-border divide-y">
      {Object.entries(parameters).map(([key, value]) => {
        const displayValue = formatParameterValue(value);
        const isMultiline = typeof value === "string" && value.includes("\n");
        return (
          <div key={key} className="space-y-1.5 py-3 first:pt-0 last:pb-0">
            <dt className="text-foreground/70 text-xs font-semibold tracking-wide uppercase">
              {humanizeKey(key)}
            </dt>
            <dd>
              {isMultiline ? (
                <pre className="bg-muted/60 border-border rounded-md border px-3 py-2 text-xs leading-relaxed whitespace-pre-wrap break-words font-sans text-foreground">
                  {displayValue}
                </pre>
              ) : (
                <span className="bg-muted/60 border-border text-foreground inline-block rounded-md border px-2.5 py-1 text-sm font-medium break-all">
                  {displayValue}
                </span>
              )}
            </dd>
          </div>
        );
      })}
    </dl>
  );
}
