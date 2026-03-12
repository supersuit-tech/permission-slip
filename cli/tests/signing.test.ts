/**
 * Tests for the request signing module.
 *
 * The global setup (tests/setup.ts) writes an ephemeral key pair to a temp
 * dir and sets PS_CLI_TEST_PRIVATE_KEY so loadPrivateKey uses that key.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import { buildSignatureHeader, REGISTRATION_AGENT_ID } from "../src/auth/signing.js";

describe("REGISTRATION_AGENT_ID", () => {
  it("is max int64 as string", () => {
    expect(REGISTRATION_AGENT_ID).toBe("9223372036854775807");
  });
});

describe("buildSignatureHeader", () => {
  it("returns a properly formatted header string", () => {
    const header = buildSignatureHeader({
      agentId: 42,
      method: "GET",
      path: "/agents/me",
      timestamp: 1708617600,
    });

    expect(header).toMatch(/^agent_id="42"/);
    expect(header).toContain('algorithm="Ed25519"');
    expect(header).toContain('timestamp="1708617600"');
    expect(header).toMatch(/signature="[A-Za-z0-9_-]+"/);
  });

  it("uses REGISTRATION_AGENT_ID as placeholder", () => {
    const header = buildSignatureHeader({
      agentId: REGISTRATION_AGENT_ID,
      method: "POST",
      path: "/invite/PS-TEST-1234",
      body: JSON.stringify({ request_id: "test-uuid", public_key: "ssh-ed25519 AAAA" }),
    });
    expect(header).toContain(`agent_id="${REGISTRATION_AGENT_ID}"`);
  });

  it("produces base64url-encoded signature (no +/= chars)", () => {
    const header = buildSignatureHeader({
      agentId: 1,
      method: "POST",
      path: "/agents/1/verify",
      body: "{}",
      timestamp: 1700000000,
    });

    const match = header.match(/signature="([^"]+)"/);
    expect(match).not.toBeNull();
    const sig = match![1];
    expect(sig).toMatch(/^[A-Za-z0-9_-]+$/);
  });

  it("signature is verifiable with the corresponding public key", () => {
    const privKeyFile = process.env["PS_CLI_TEST_PRIVATE_KEY"]!;
    const pubKeyFile = process.env["PS_CLI_TEST_PUBLIC_KEY"]!;
    const privPem = fs.readFileSync(privKeyFile);
    const pubPem = fs.readFileSync(pubKeyFile, "utf-8").trim();
    // Parse openssh pub key -> extract raw bytes
    const parts = pubPem.split(/\s+/);
    const pubKeyBuf = Buffer.from(parts[1]!, "base64");
    // Skip the ssh-ed25519 length-prefixed fields to get the raw key
    const keyTypeLen = pubKeyBuf.readUInt32BE(0);
    const rawKeyOffset = 4 + keyTypeLen + 4;
    const rawPubKey = pubKeyBuf.slice(rawKeyOffset);
    const pubKeyObj = crypto.createPublicKey({
      key: {
        kty: "OKP",
        crv: "Ed25519",
        x: rawPubKey.toString("base64url"),
      },
      format: "jwk",
    });

    const timestamp = 1708617600;
    const body = '{"test":true}';
    const header = buildSignatureHeader({
      agentId: 42,
      method: "POST",
      path: "/agents/42/verify",
      body,
      timestamp,
    });

    const sigMatch = header.match(/signature="([^"]+)"/);
    expect(sigMatch).not.toBeNull();
    const sigB64 = sigMatch![1];
    const sigBuf = Buffer.from(sigB64!.replace(/-/g, "+").replace(/_/g, "/"), "base64");

    const bodyHash = crypto.createHash("sha256").update(body, "utf-8").digest("hex");
    const canonical = `POST\n/agents/42/verify\n\n${timestamp}\n${bodyHash}`;

    const valid = crypto.verify(null, Buffer.from(canonical, "utf-8"), pubKeyObj, sigBuf);
    expect(valid).toBe(true);
  });

  it("produces consistent output for the same inputs", () => {
    const opts = {
      agentId: 42,
      method: "GET" as const,
      path: "/agents/me",
      timestamp: 1708617600,
    };
    const header1 = buildSignatureHeader(opts);
    const header2 = buildSignatureHeader(opts);
    expect(header1).toBe(header2);
  });
});
