import type { RequiredCredential } from "@/hooks/useConnectorDetail";

/** One static credential field from the connector manifest (GET /connectors/:id). */
export type ResolvedCredentialField = {
  key: string;
  label: string;
  placeholder: string;
  secret: boolean;
  required: boolean;
  helpText: string;
};

function normalizeField(
  f: NonNullable<RequiredCredential["fields"]>[number],
): ResolvedCredentialField {
  return {
    key: f.key,
    label: f.label,
    placeholder: f.placeholder ?? "",
    secret: f.secret,
    required: f.required,
    helpText: f.help_text ?? "",
  };
}

/**
 * Fields to render for AddCredentialDialog (api_key / custom only).
 * Empty array means use basic-auth UI; caller handles oauth elsewhere.
 */
export function resolveStaticCredentialFields(
  credential: RequiredCredential,
  opts?: {
    credentialKey?: string;
    fieldLabel?: string;
    fieldPlaceholder?: string;
  },
): ResolvedCredentialField[] {
  if (
    credential.auth_type === "basic" ||
    credential.auth_type === "oauth2"
  ) {
    return [];
  }
  const raw = credential.fields;
  if (raw && raw.length > 0) {
    return raw.map(normalizeField);
  }
  const key = opts?.credentialKey ?? "api_key";
  const label = opts?.fieldLabel ?? "API Key";
  const placeholder =
    opts?.fieldPlaceholder ?? "Enter API key or token";
  return [
    {
      key,
      label,
      placeholder,
      secret: true,
      required: true,
      helpText: "",
    },
  ];
}

/** Label for the primary CTA when adding a static credential. */
export function staticCredentialButtonLabel(credential: RequiredCredential): string {
  if (credential.auth_type === "basic") {
    return "credentials";
  }
  if (credential.auth_type === "oauth2") {
    return "account";
  }
  const fields = credential.fields;
  if (fields && fields.length > 1) {
    return "credentials";
  }
  if (fields && fields.length === 1) {
    return fields[0]?.label.toLowerCase() ?? "credential";
  }
  return "API key";
}
