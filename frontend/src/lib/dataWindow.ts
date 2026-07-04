export const DATA_WINDOW_NAMESPACE_KEY = "$data_window";

export interface DataWindowParams {
  start_param: string;
  end_param: string;
}

export type DataWindowMode = "last_days" | "absolute";

export interface DataWindowFormState {
  enabled: boolean;
  mode: DataWindowMode;
  lastDays: string;
  startsAt: string;
  endsAt: string;
}

export const DEFAULT_DATA_WINDOW_FORM: DataWindowFormState = {
  enabled: false,
  mode: "last_days",
  lastDays: "30",
  startsAt: "",
  endsAt: "",
};

export function supportsDataWindow(
  dataWindow: DataWindowParams | null | undefined,
): dataWindow is DataWindowParams {
  return !!(
    dataWindow?.start_param?.trim() && dataWindow?.end_param?.trim()
  );
}

export function parseDataWindowFormState(
  constraints: Record<string, unknown> | null | undefined,
): DataWindowFormState {
  if (!constraints || typeof constraints !== "object") {
    return { ...DEFAULT_DATA_WINDOW_FORM };
  }
  const raw = constraints[DATA_WINDOW_NAMESPACE_KEY];
  if (!raw || typeof raw !== "object") {
    return { ...DEFAULT_DATA_WINDOW_FORM };
  }
  const dw = raw as Record<string, unknown>;
  if (typeof dw.last_days === "number" && dw.last_days >= 1) {
    return {
      enabled: true,
      mode: "last_days",
      lastDays: String(dw.last_days),
      startsAt: "",
      endsAt: "",
    };
  }
  const startsAt =
    typeof dw.starts_at === "string" ? toDatetimeLocal(dw.starts_at) : "";
  const endsAt = typeof dw.ends_at === "string" ? toDatetimeLocal(dw.ends_at) : "";
  if (startsAt || endsAt) {
    return {
      enabled: true,
      mode: "absolute",
      lastDays: "30",
      startsAt,
      endsAt,
    };
  }
  return { ...DEFAULT_DATA_WINDOW_FORM };
}

export function buildDataWindowConstraint(
  form: DataWindowFormState,
): Record<string, unknown> | null {
  if (!form.enabled) return null;
  if (form.mode === "last_days") {
    const days = Number.parseInt(form.lastDays, 10);
    if (!Number.isFinite(days) || days < 1) return null;
    return { last_days: days };
  }
  const out: Record<string, unknown> = {};
  if (form.startsAt.trim()) {
    out.starts_at = fromDatetimeLocal(form.startsAt);
  }
  if (form.endsAt.trim()) {
    out.ends_at = fromDatetimeLocal(form.endsAt);
  }
  return Object.keys(out).length > 0 ? out : null;
}

export function formatDataWindowConstraint(
  raw: unknown,
): string | null {
  if (!raw || typeof raw !== "object") return null;
  const dw = raw as Record<string, unknown>;
  if (typeof dw.last_days === "number" && dw.last_days >= 1) {
    const n = dw.last_days;
    return `last ${n} day${n === 1 ? "" : "s"}`;
  }
  const parts: string[] = [];
  if (typeof dw.starts_at === "string" && dw.starts_at) {
    parts.push(`from ${formatHumanDate(dw.starts_at)}`);
  }
  if (typeof dw.ends_at === "string" && dw.ends_at) {
    parts.push(`until ${formatHumanDate(dw.ends_at)}`);
  }
  if (parts.length === 0) return null;
  return parts.join(" ");
}

function formatHumanDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function toDatetimeLocal(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

function fromDatetimeLocal(local: string): string {
  return new Date(local).toISOString();
}

export function dataWindowCountsAsConstraint(form: DataWindowFormState): boolean {
  return buildDataWindowConstraint(form) !== null;
}
