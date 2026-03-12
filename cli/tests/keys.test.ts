/**
 * Tests for key management.
 *
 * Uses PS_CLI_TEST_PRIVATE_KEY / PS_CLI_TEST_PUBLIC_KEY (set by global setup)
 * so we don't touch ~/.ssh.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import {
  keyPairExists,
  readPublicKey,
  loadPrivateKey,
} from "../src/auth/keys.js";
import { PRIVATE_KEY_FILE, PUBLIC_KEY_FILE } from "../src/config/store.js";

describe("paths point to test temp dir", () => {
  it("private key file uses test path", () => {
    expect(PRIVATE_KEY_FILE).toContain("ps-cli-test-");
  });
  it("public key file uses test path", () => {
    expect(PUBLIC_KEY_FILE).toContain("ps-cli-test-");
  });
});

describe("keyPairExists", () => {
  it("returns true because global setup generated keys", () => {
    expect(keyPairExists()).toBe(true);
  });
});

describe("readPublicKey", () => {
  it("returns an ssh-ed25519 key string", () => {
    const pub = readPublicKey();
    expect(pub).toMatch(/^ssh-ed25519 [A-Za-z0-9+/=]+$/);
  });

  it("strips the comment field", () => {
    const pub = readPublicKey();
    // Should be exactly 2 space-separated parts: type + base64
    const parts = pub.split(" ");
    expect(parts).toHaveLength(2);
  });
});

describe("loadPrivateKey", () => {
  it("loads the private key as a KeyObject", () => {
    const key = loadPrivateKey();
    expect(key.type).toBe("private");
    expect(key.asymmetricKeyType).toBe("ed25519");
  });

  it("can sign data that is verifiable with the public key", () => {
    const privKey = loadPrivateKey();
    const data = Buffer.from("test canonical string\nwith multiple\nlines");
    const sig = crypto.sign(null, data, privKey);
    expect(sig).toHaveLength(64);

    // Verify with public key read from file
    const pubPem = fs.readFileSync(PUBLIC_KEY_FILE, "utf-8").trim();
    const parts = pubPem.split(/\s+/);
    const pubBuf = Buffer.from(parts[1]!, "base64");
    const keyTypeLen = pubBuf.readUInt32BE(0);
    const rawKeyOffset = 4 + keyTypeLen + 4;
    const rawPub = pubBuf.slice(rawKeyOffset);
    const pubKeyObj = crypto.createPublicKey({
      key: { kty: "OKP", crv: "Ed25519", x: rawPub.toString("base64url") },
      format: "jwk",
    });

    expect(crypto.verify(null, data, pubKeyObj, sig)).toBe(true);
  });
});
