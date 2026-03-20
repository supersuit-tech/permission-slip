/**
 * HTTP client for the Permission Slip API.
 *
 * Transparently handles:
 *  - Request signing (X-Permission-Slip-Signature)
 *  - JSON serialization / deserialization
 *  - Error formatting
 *
 * Endpoints follow docs/agents.md. Signing paths use the router path
 * (e.g. /agents/42/verify), not the full URL including the /api/v1 prefix.
 */

import crypto from "node:crypto";
import { buildSignatureHeader, REGISTRATION_AGENT_ID } from "../auth/signing.js";

export interface ApiError {
  code: string;
  message: string;
  retryable: boolean;
  details?: Record<string, unknown>;
  trace_id?: string;
}

/** Shape returned by GET /approvals/{id}/status. */
export interface ApprovalStatusResult {
  approval_id: string;
  status: string;
  expires_at: string;
  created_at: string;
  execution_status?: string;
  execution_result?: unknown;
}

export class PermissionSlipApiError extends Error {
  constructor(
    public readonly statusCode: number,
    public readonly apiError: ApiError,
  ) {
    super(apiError.message);
    this.name = "PermissionSlipApiError";
  }
}

export interface ClientOptions {
  /** Base URL of the Permission Slip server, e.g. https://app.permissionslip.dev */
  serverUrl: string;
  /** Agent ID — use REGISTRATION_AGENT_ID during registration */
  agentId: number | string;
}

/**
 * Strips trailing slashes from a URL.
 */
function normalizeBase(url: string): string {
  return url.replace(/\/+$/, "");
}

/**
 * Extracts the router path (without /api/v1) for signing purposes.
 * The invite endpoint is at the host root (no /api/v1 prefix).
 */
function signingPath(routerPath: string): string {
  return routerPath;
}

interface RequestOptions {
  method: "GET" | "POST" | "DELETE";
  routerPath: string; // e.g. /agents/42/verify (used for signing)
  apiPath?: string;   // e.g. /api/v1/agents/42/verify (used for HTTP, defaults to /api/v1 + routerPath)
  body?: unknown;
  /** Override agentId just for this request (e.g. REGISTRATION_AGENT_ID) */
  agentIdOverride?: number | string;
  /** Whether this is a public endpoint (no signing required) */
  public?: boolean;
}

export class ApiClient {
  private base: string;
  private agentId: number | string;

  constructor(opts: ClientOptions) {
    // Reject non-http(s) schemes before ever making a request.
    let parsed: URL;
    try {
      parsed = new URL(opts.serverUrl);
    } catch {
      throw new Error(`Invalid server URL: ${opts.serverUrl}`);
    }
    if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
      throw new Error(
        `Server URL must use http or https (got ${parsed.protocol}). ` +
        "Check the --server flag.",
      );
    }
    this.base = normalizeBase(opts.serverUrl);
    this.agentId = opts.agentId;
  }

  async request<T>(opts: RequestOptions): Promise<T> {
    const bodyStr = opts.body !== undefined ? JSON.stringify(opts.body) : undefined;
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "User-Agent": "@permission-slip/cli",
    };

    if (!opts.public) {
      const agentId = opts.agentIdOverride ?? this.agentId;
      headers["X-Permission-Slip-Signature"] = buildSignatureHeader({
        agentId,
        method: opts.method,
        path: signingPath(opts.routerPath),
        body: bodyStr,
      });
    }

    const fullPath =
      opts.apiPath ?? `/api/v1${opts.routerPath}`;
    const url = `${this.base}${fullPath}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 30_000);

    let res: Response;
    try {
      res = await fetch(url, {
        method: opts.method,
        headers,
        body: bodyStr,
        signal: controller.signal,
      });
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") {
        throw new Error(`Request timed out after 30s: ${opts.method} ${fullPath}`);
      }
      throw err;
    } finally {
      clearTimeout(timeoutId);
    }

    if (!res.ok) {
      let apiError: ApiError;
      try {
        const errBody = (await res.json()) as { error?: ApiError };
        apiError = errBody.error ?? {
          code: "unknown",
          message: `HTTP ${res.status}`,
          retryable: false,
        };
      } catch {
        apiError = {
          code: "unknown",
          message: `HTTP ${res.status} ${res.statusText}`,
          retryable: false,
        };
      }
      throw new PermissionSlipApiError(res.status, apiError);
    }

    // Some responses may be empty (204)
    if (res.status === 204) {
      return undefined as T;
    }

    return res.json() as Promise<T>;
  }

  // ---------- Typed API methods ----------

  /** POST /invite/{code} — register with an invite code */
  async register(
    inviteCode: string,
    publicKey: string,
    name: string,
    version = "1.0.0",
  ) {
    const requestId = crypto.randomUUID();
    return this.request<{
      agent_id: number;
      expires_at: string;
      verification_required: boolean;
    }>({
      method: "POST",
      routerPath: `/invite/${inviteCode}`,
      apiPath: `/invite/${inviteCode}`,
      body: {
        request_id: requestId,
        public_key: publicKey,
        metadata: { name, version },
      },
      agentIdOverride: REGISTRATION_AGENT_ID,
    });
  }

  /** POST /agents/{id}/verify — verify registration with confirmation code */
  async verify(agentId: number, confirmationCode: string) {
    const requestId = crypto.randomUUID();
    return this.request<{ status: string; registered_at: string }>({
      method: "POST",
      routerPath: `/agents/${agentId}/verify`,
      body: { request_id: requestId, confirmation_code: confirmationCode },
      agentIdOverride: agentId,
    });
  }

  /** GET /agents/me — get own agent record */
  async status() {
    return this.request<{
      agent_id: number;
      status: string;
      metadata: Record<string, unknown>;
      registered_at: string;
      last_active_at: string;
      created_at: string;
    }>({
      method: "GET",
      routerPath: "/agents/me",
    });
  }

  /** GET /agents/{id}/capabilities */
  async capabilities(agentId: number) {
    return this.request<unknown>({
      method: "GET",
      routerPath: `/agents/${agentId}/capabilities`,
      agentIdOverride: agentId,
    });
  }

  /** GET /connectors — public, no auth */
  async connectors() {
    return this.request<unknown>({
      method: "GET",
      routerPath: "/connectors",
      public: true,
    });
  }

  /** GET /connectors/{id} — public, no auth */
  async connector(id: string) {
    return this.request<unknown>({
      method: "GET",
      routerPath: `/connectors/${id}`,
      public: true,
    });
  }

  /** POST /approvals/request — may auto-approve if a standing approval matches */
  async requestApproval(
    actionId: string,
    params: unknown,
    context?: { description?: string; risk_level?: string },
  ) {
    const requestId = crypto.randomUUID();
    return this.request<{
      approval_id?: string;
      approval_url?: string;
      status: string;
      expires_at?: string;
      created_at?: string;
      result?: unknown;
      standing_approval_id?: string;
      executions_remaining?: number | null;
    }>({
      method: "POST",
      routerPath: "/approvals/request",
      body: {
        request_id: requestId,
        action: { type: actionId, parameters: params },
        context: context ?? {},
      },
    });
  }

  /** GET /approvals/{id}/status — check approval and execution status */
  async approvalStatus(approvalId: string): Promise<ApprovalStatusResult> {
    return this.request<ApprovalStatusResult>({
      method: "GET",
      routerPath: `/approvals/${approvalId}/status`,
    });
  }
}
