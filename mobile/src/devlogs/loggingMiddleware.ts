import type { Middleware } from "openapi-fetch";
import { isDevModeEnabled } from "../lib/devModeConfig";
import { appendDevLog, nextDevLogId, type DevLogEntry } from "./devLogsStore";

type PendingMeta = {
  id: string;
  method: string;
  url: string;
  startedAt: number;
};

/**
 * Per-request metadata, keyed by the openapi-fetch request id. Populated in
 * onRequest and consumed in onResponse so we can compute the duration and
 * carry the original (pre-rewrite) URL through to the log entry.
 */
const pending = new Map<string, PendingMeta>();

/**
 * Captures method + URL + status + response body for every API call into the
 * in-memory dev log store. Reads {@link isDevModeEnabled} on every call so the
 * user can flip the toggle without restarting the app — when disabled the
 * middleware short-circuits with negligible overhead.
 */
export const loggingMiddleware: Middleware = {
  async onRequest({ request, id }) {
    if (!isDevModeEnabled()) return request;
    pending.set(id, {
      id: nextDevLogId(),
      method: request.method,
      url: request.url,
      startedAt: Date.now(),
    });
    return request;
  },
  async onResponse({ response, id }) {
    const meta = pending.get(id);
    if (!meta) return response;
    pending.delete(id);

    // Clone to avoid draining the body the caller still needs to read.
    let body = "";
    try {
      body = await response.clone().text();
    } catch {
      body = "(body unavailable)";
    }
    const entry: DevLogEntry = {
      id: meta.id,
      method: meta.method,
      url: meta.url,
      status: response.status,
      durationMs: Date.now() - meta.startedAt,
      startedAt: meta.startedAt,
      body,
      isError: response.status >= 400,
    };
    appendDevLog(entry);
    return response;
  },
  async onError({ error, id }) {
    const meta = pending.get(id);
    if (!meta) return;
    pending.delete(id);
    const message = error instanceof Error ? error.message : String(error);
    appendDevLog({
      id: meta.id,
      method: meta.method,
      url: meta.url,
      status: null,
      durationMs: Date.now() - meta.startedAt,
      startedAt: meta.startedAt,
      body: message,
      isError: true,
    });
  },
};
