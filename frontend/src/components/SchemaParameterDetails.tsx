import { Badge } from "@/components/ui/badge";
import type { ParametersSchema } from "@/lib/parameterSchema";

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
  const schemaKeys = Object.keys(properties);
  const extraKeys = Object.keys(parameters).filter(
    (k) => !properties[k],
  );

  return (
    <div className="space-y-3">
      {schemaKeys.map((key) => {
        // Safe to assert: schemaKeys come from Object.keys(properties).
        const prop = properties[key]!;
        const value = parameters[key];
        const isRequired = requiredSet.has(key);

        return (
          <ParameterRow
            key={key}
            name={key}
            label={prop.description ?? key}
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
          label={key}
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
  const displayValue = formatValue(value);
  const isDefault =
    defaultValue !== undefined && String(value) === String(defaultValue);

  return (
    <div className="space-y-0.5">
      <div className="flex items-center gap-1.5">
        <span className="text-muted-foreground text-xs font-medium">
          {label !== name ? label : name}
        </span>
        {label !== name && (
          <code className="text-muted-foreground/60 text-[10px]">{name}</code>
        )}
        {type && (
          <span className="text-muted-foreground/50 text-[10px]">{type}</span>
        )}
        {isRequired && !isProvided && (
          <Badge
            variant="destructive"
            className="rounded-full text-[9px] leading-tight"
          >
            missing
          </Badge>
        )}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-sm break-all">
          {isProvided ? displayValue : <span className="text-muted-foreground italic">not provided</span>}
        </span>
        {isDefault && (
          <Badge variant="secondary" className="text-[9px] leading-tight">
            default
          </Badge>
        )}
        {enumValues && enumValues.length > 0 && isProvided && (
          <span className="text-muted-foreground text-[10px]">
            ({enumValues.join(" | ")})
          </span>
        )}
      </div>
    </div>
  );
}

function FallbackDetails({
  parameters,
}: {
  parameters: Record<string, unknown>;
}) {
  return (
    <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
      {Object.entries(parameters).map(([key, value]) => (
        <div key={key} className="contents">
          <dt className="text-muted-foreground font-medium">{key}</dt>
          <dd className="truncate">{formatValue(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

function formatValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) return value.join(", ");
  if (value === null || value === undefined) return "\u2014";
  return JSON.stringify(value);
}
