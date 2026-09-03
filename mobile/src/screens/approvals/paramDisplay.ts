/**
 * Approval parameter display helpers. Intentionally duplicated from the web
 * frontend (`frontend/src/lib/resourceParameterDisplay.ts` and
 * `frontend/src/lib/binaryParamDisplay.ts`) because web and mobile share no
 * module-level code. Keep both in sync when changing the logic.
 */

const MAX_THUMBNAIL_BYTES = 512 * 1024;

/** Maps opaque IDs to human-readable names from resource_details. */
export function resolvedResourceDisplayValue(
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  if (resourceDetails == null) return null;
  if (typeof rawValue !== "string" && typeof rawValue !== "number") return null;
  const raw = String(rawValue);
  if (raw.length === 0) return null;

  const nameKey = paramKey.endsWith("_id")
    ? `${paramKey.slice(0, -3)}_name`
    : `${paramKey}_name`;
  const name = resourceDetails[nameKey];
  if (typeof name === "string" && name.length > 0 && name !== raw) {
    return `${name} (${raw})`;
  }
  return null;
}

export function isBase64ParamKey(key: string): boolean {
  return key.endsWith("_base64");
}

export function formatBinaryParamSummary(
  value: string,
  parameters: Record<string, unknown>,
): string {
  const mime = inferBinaryMime(value, parameters);
  const kind = fileKindLabel(mime);
  const size = decodedBase64Size(value);
  if (size > 0) {
    return `${kind} \u00b7 ${formatByteSize(size)}`;
  }
  return kind;
}

export function binaryThumbnailUri(
  key: string,
  value: unknown,
  parameters: Record<string, unknown>,
): string | undefined {
  if (!isBase64ParamKey(key) || typeof value !== "string" || value.length === 0) {
    return undefined;
  }
  const mime = inferBinaryMime(value, parameters);
  if (!mime.startsWith("image/")) return undefined;
  const size = decodedBase64Size(value);
  if (size <= 0 || size > MAX_THUMBNAIL_BYTES) return undefined;
  const compact = value.replace(/\s/g, "");
  return `data:${mime};base64,${compact}`;
}

export function formatApprovalParamValue(
  key: string,
  value: unknown,
  parameters: Record<string, unknown>,
  resourceDetails?: Record<string, unknown> | null,
): unknown {
  if (isBase64ParamKey(key) && typeof value === "string" && value.length > 0) {
    return formatBinaryParamSummary(value, parameters);
  }
  const resolved = resolvedResourceDisplayValue(key, value, resourceDetails);
  if (resolved != null) return resolved;
  return value;
}

function inferBinaryMime(
  value: string,
  parameters: Record<string, unknown>,
): string {
  const declared = parameters.mime_type;
  if (typeof declared === "string" && declared.trim() !== "") {
    return declared.trim();
  }
  return mimeFromMagic(value) ?? "application/octet-stream";
}

function mimeFromMagic(value: string): string | null {
  const compact = value.replace(/\s/g, "");
  if (compact.startsWith("JVBERi")) return "application/pdf";
  if (compact.startsWith("/9j/")) return "image/jpeg";
  if (compact.startsWith("iVBORw")) return "image/png";
  if (compact.startsWith("R0lGOD")) return "image/gif";
  if (compact.startsWith("UklGR")) return "image/webp";
  return null;
}

function fileKindLabel(mime: string): string {
  const lower = mime.toLowerCase();
  if (lower === "application/pdf" || lower.endsWith("/pdf")) return "PDF";
  if (lower === "image/jpeg" || lower === "image/jpg") return "JPEG image";
  if (lower === "image/png") return "PNG image";
  if (lower === "image/gif") return "GIF image";
  if (lower === "image/webp") return "WEBP image";
  if (lower.startsWith("image/")) return "Image";
  if (lower === "application/zip" || lower.endsWith("+zip")) return "ZIP archive";
  if (lower === "application/octet-stream") return "Binary file";
  return mime;
}

export function decodedBase64Size(value: string): number {
  const compact = value.replace(/\s/g, "");
  if (compact.length === 0) return 0;
  const padding = compact.endsWith("==") ? 2 : compact.endsWith("=") ? 1 : 0;
  return Math.floor((compact.length * 3) / 4) - padding;
}

export function formatByteSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) {
    const kb = bytes / 1024;
    const rounded = kb >= 10 ? Math.round(kb) : Math.round(kb * 10) / 10;
    return `${String(rounded)} KB`;
  }
  const mb = bytes / (1024 * 1024);
  const rounded = Math.round(mb * 10) / 10;
  return `${String(rounded)} MB`;
}
