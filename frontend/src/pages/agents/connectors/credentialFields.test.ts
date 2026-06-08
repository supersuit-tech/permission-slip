import { describe, expect, it } from "vitest";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import {
  resolveStaticCredentialFields,
  staticCredentialButtonLabel,
} from "./credentialFields";

function cred(partial: Partial<RequiredCredential>): RequiredCredential {
  return {
    service: "test",
    auth_type: "api_key",
    ...partial,
  } as RequiredCredential;
}

describe("resolveStaticCredentialFields", () => {
  it("defaults api_key with no fields to single API Key field", () => {
    const fields = resolveStaticCredentialFields(cred({ fields: undefined }));
    expect(fields).toHaveLength(1);
    expect(fields[0]?.key).toBe("api_key");
    expect(fields[0]?.label).toBe("API Key");
    expect(fields[0]?.secret).toBe(true);
  });

  it("renders manifest fields for custom auth", () => {
    const fields = resolveStaticCredentialFields(
      cred({
        auth_type: "custom",
        fields: [
          {
            key: "connection_string",
            label: "Connection String",
            placeholder: "postgres://",
            secret: true,
            required: true,
          },
        ],
      }),
    );
    expect(fields).toHaveLength(1);
    expect(fields[0]?.key).toBe("connection_string");
  });

  it("returns empty for basic (handled separately in dialog)", () => {
    expect(resolveStaticCredentialFields(cred({ auth_type: "basic" }))).toEqual([]);
  });
});

describe("staticCredentialButtonLabel", () => {
  it("uses first field label for single-field custom creds", () => {
    const label = staticCredentialButtonLabel(
      cred({
        auth_type: "custom",
        fields: [
          {
            key: "url",
            label: "Redis URL",
            secret: true,
            required: true,
          },
        ],
      }),
    );
    expect(label).toBe("redis url");
  });

  it("uses credentials for multi-field", () => {
    const label = staticCredentialButtonLabel(
      cred({
        auth_type: "api_key",
        fields: [
          { key: "a", label: "A", secret: true, required: true },
          { key: "b", label: "B", secret: true, required: true },
        ],
      }),
    );
    expect(label).toBe("credentials");
  });

  it("uses credentials for Proton Mail bridge fields", () => {
    const protonFields = [
      { key: "username", label: "Bridge username", secret: false, required: true },
      { key: "password", label: "Bridge password", secret: true, required: true },
      { key: "imap_host", label: "IMAP host", secret: false, required: false },
      { key: "imap_port", label: "IMAP port", secret: false, required: false },
      { key: "smtp_host", label: "SMTP host", secret: false, required: false },
      { key: "smtp_port", label: "SMTP port", secret: false, required: false },
    ];
    const credential = cred({
      service: "protonmail",
      auth_type: "custom",
      fields: protonFields,
    });

    const fields = resolveStaticCredentialFields(credential);
    expect(fields).toHaveLength(6);
    expect(fields.map((f) => f.key)).toEqual([
      "username",
      "password",
      "imap_host",
      "imap_port",
      "smtp_host",
      "smtp_port",
    ]);
    expect(fields[0]?.secret).toBe(false);
    expect(fields[1]?.secret).toBe(true);
    expect(fields[2]?.required).toBe(false);
    expect(staticCredentialButtonLabel(credential)).toBe("credentials");
  });
});
