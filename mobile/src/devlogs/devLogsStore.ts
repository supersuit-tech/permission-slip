import { useSyncExternalStore } from "react";

/** A single request/response entry captured by the logging middleware. */
export type DevLogEntry = {
  id: string;
  method: string;
  url: string;
  /** Status code on completion, or null while the request is in-flight or threw before reaching the server. */
  status: number | null;
  /** Duration in ms from request start to response (or error). */
  durationMs: number;
  startedAt: number;
  /** Truncated response body (or error message on transport failure). */
  body: string;
  /** True when the entry represents a transport-level failure rather than an HTTP response. */
  isError: boolean;
};

const MAX_ENTRIES = 50;
const MAX_BODY_CHARS = 4096;

let entries: DevLogEntry[] = [];
const listeners = new Set<() => void>();
let counter = 0;

function notify() {
  for (const listener of listeners) listener();
}

/** Append a finalized log entry, evicting the oldest when over capacity. */
export function appendDevLog(entry: DevLogEntry): void {
  const truncated =
    entry.body.length > MAX_BODY_CHARS
      ? entry.body.slice(0, MAX_BODY_CHARS) + "\n…(truncated)"
      : entry.body;
  const next: DevLogEntry[] = [...entries, { ...entry, body: truncated }];
  if (next.length > MAX_ENTRIES) next.splice(0, next.length - MAX_ENTRIES);
  entries = next;
  notify();
}

/** Drop all captured entries; used by the overlay's Clear button. */
export function clearDevLogs(): void {
  if (entries.length === 0) return;
  entries = [];
  notify();
}

/** Monotonic-ish id for new entries (millisecond + counter, in-memory only). */
export function nextDevLogId(): string {
  counter += 1;
  return `${Date.now()}-${counter}`;
}

function getSnapshot(): DevLogEntry[] {
  return entries;
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** React hook that re-renders when new entries are appended. */
export function useDevLogs(): DevLogEntry[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

/** Test-only helper to reset module state between tests. */
export function __resetDevLogsForTests(): void {
  entries = [];
  listeners.clear();
  counter = 0;
}

/** Test-only snapshot accessor; production code reads via {@link useDevLogs}. */
export function __getDevLogsForTests(): DevLogEntry[] {
  return entries;
}
