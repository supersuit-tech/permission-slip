/**
 * Request signing for Permission Slip.
 *
 * Implements the X-Permission-Slip-Signature header as documented in docs/agents.md.
 *
 * Header format:
 *   agent_id="42", algorithm="Ed25519", timestamp="1708617600", signature="<base64url>"
 *
 * Canonical string (5 lines joined by \n):
 *   METHOD\nPATH\nQUERY\nTIMESTAMP\nBODY_HASH
 *
 * During registration, use MAX_INT64 as the agent_id placeholder.
 */

import crypto from "node:crypto";
import { loadPrivateKey } from "./keys.js";

// Placeholder agent_id used during registration before we have a real ID
export const REGISTRATION_AGENT_ID = "9223372036854775807";

export interface SignatureOptions {
  agentId: number | string;
  method: string;
  path: string;
  query?: string;
  body?: string;
  timestamp?: number;
}

/**
 * Computes the X-Permission-Slip-Signature header value.
 */
export function buildSignatureHeader(opts: SignatureOptions): string {
  const privateKey = loadPrivateKey();
  const timestamp = opts.timestamp ?? Math.floor(Date.now() / 1000);
  const method = opts.method.toUpperCase();
  const urlPath = opts.path;
  const query = canonicalizeQuery(opts.query ?? "");
  const bodyHash = hashBody(opts.body ?? "");

  const canonical = `${method}\n${urlPath}\n${query}\n${timestamp}\n${bodyHash}`;
  const sig = signCanonical(privateKey, canonical);

  return `agent_id="${opts.agentId}", algorithm="Ed25519", timestamp="${timestamp}", signature="${sig}"`;
}

function canonicalizeQuery(raw: string): string {
  if (!raw) return "";
  const pairs = new URLSearchParams(raw);
  const sorted = Array.from(pairs.entries()).sort(([a], [b]) =>
    a.localeCompare(b),
  );
  return sorted
    .map(([k, v]) => `${encodeRFC3986(k)}=${encodeRFC3986(v)}`)
    .join("&");
}

function encodeRFC3986(str: string): string {
  return encodeURIComponent(str).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`,
  );
}

function hashBody(body: string): string {
  if (!body) {
    // SHA-256 of empty string
    return "e3b0c44298fc1c149afbf4c8996fb924" +
      "27ae41e4649b934ca495991b7852b855";
  }
  return crypto.createHash("sha256").update(body, "utf-8").digest("hex");
}

function signCanonical(privateKey: crypto.KeyObject, canonical: string): string {
  const sig = crypto.sign(null, Buffer.from(canonical, "utf-8"), privateKey);
  // base64url without padding
  return sig
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}
