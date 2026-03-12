/**
 * Ed25519 key management for Permission Slip.
 *
 * Keys are stored in OpenSSH format:
 *   Private: ~/.ssh/permission_slip_agent
 *   Public:  ~/.ssh/permission_slip_agent.pub
 *
 * We use Node's built-in `crypto` module and `child_process` (ssh-keygen) to
 * generate the key pair, matching the existing manual flow agents used to do.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import { execSync } from "node:child_process";
import { PUBLIC_KEY_FILE, PRIVATE_KEY_FILE, SSH_DIR } from "../config/store.js";

export interface KeyPair {
  privateKeyFile: string;
  publicKey: string; // "ssh-ed25519 AAAA..."
}

/**
 * Returns true if a Permission Slip key pair already exists.
 */
export function keyPairExists(): boolean {
  return (
    fs.existsSync(PRIVATE_KEY_FILE) && fs.existsSync(PUBLIC_KEY_FILE)
  );
}

/**
 * Reads the public key from disk.
 * Returns the full "ssh-ed25519 AAAA..." string (just key type + base64, no comment).
 */
export function readPublicKey(): string {
  if (!fs.existsSync(PUBLIC_KEY_FILE)) {
    throw new Error(
      `Public key not found at ${PUBLIC_KEY_FILE}. Run 'permission-slip register' to generate one.`,
    );
  }
  const raw = fs.readFileSync(PUBLIC_KEY_FILE, "utf-8").trim();
  // Take only type + key, drop optional comment
  const parts = raw.split(/\s+/);
  if (parts.length < 2) {
    throw new Error(`Invalid public key format in ${PUBLIC_KEY_FILE}`);
  }
  return `${parts[0]} ${parts[1]}`;
}

/**
 * Loads the private key from disk as a Node KeyObject.
 */
export function loadPrivateKey(): crypto.KeyObject {
  if (!fs.existsSync(PRIVATE_KEY_FILE)) {
    throw new Error(
      `Private key not found at ${PRIVATE_KEY_FILE}. Run 'permission-slip register' to generate one.`,
    );
  }
  const pem = fs.readFileSync(PRIVATE_KEY_FILE);
  return crypto.createPrivateKey({ key: pem, format: "pem" });
}

/**
 * Generates a new Ed25519 key pair using ssh-keygen and saves it to
 * ~/.ssh/permission_slip_agent{,.pub}.
 *
 * Throws if ssh-keygen is not available or if keys already exist and
 * overwrite is false.
 */
export function generateKeyPair(overwrite = false): KeyPair {
  if (!overwrite && keyPairExists()) {
    return {
      privateKeyFile: PRIVATE_KEY_FILE,
      publicKey: readPublicKey(),
    };
  }

  if (!fs.existsSync(SSH_DIR)) {
    fs.mkdirSync(SSH_DIR, { recursive: true, mode: 0o700 });
  }

  // Remove existing key files if overwriting to avoid ssh-keygen prompt
  if (overwrite) {
    if (fs.existsSync(PRIVATE_KEY_FILE)) fs.unlinkSync(PRIVATE_KEY_FILE);
    if (fs.existsSync(PUBLIC_KEY_FILE)) fs.unlinkSync(PUBLIC_KEY_FILE);
  }

  try {
    execSync(
      `ssh-keygen -t ed25519 -f ${JSON.stringify(PRIVATE_KEY_FILE)} -N "" -C "permission-slip"`,
      { stdio: "pipe" },
    );
  } catch (err) {
    // Fallback: generate key using Node crypto and write in OpenSSH format
    const { privateKey, publicKey } = crypto.generateKeyPairSync("ed25519");
    writeOpenSSHPrivateKey(privateKey, PRIVATE_KEY_FILE);
    writeOpenSSHPublicKey(publicKey, PUBLIC_KEY_FILE);
  }

  return {
    privateKeyFile: PRIVATE_KEY_FILE,
    publicKey: readPublicKey(),
  };
}

/**
 * Writes an Ed25519 private key in OpenSSH PEM format.
 * Used as a fallback when ssh-keygen is not available.
 */
function writeOpenSSHPrivateKey(
  key: crypto.KeyObject,
  filePath: string,
): void {
  const pem = key.export({ type: "pkcs8", format: "pem" }) as string;
  // Wrap in OpenSSH-style PEM (Node generates PKCS8, which OpenSSH can load)
  fs.writeFileSync(filePath, pem, { mode: 0o600 });
}

/**
 * Writes the public key in OpenSSH authorized_keys format: "ssh-ed25519 <base64> permission-slip"
 */
function writeOpenSSHPublicKey(key: crypto.KeyObject, filePath: string): void {
  // Export as SubjectPublicKeyInfo DER, then convert to OpenSSH wire format
  const derBuf = key.export({ type: "spki", format: "der" }) as Buffer;
  // The last 32 bytes of the SPKI DER for ed25519 are the raw public key bytes
  const rawPubKey = derBuf.slice(-32);

  // Build OpenSSH wire format: length-prefixed "ssh-ed25519" + length-prefixed key bytes
  const keyType = Buffer.from("ssh-ed25519");
  const buf = Buffer.alloc(4 + keyType.length + 4 + rawPubKey.length);
  let offset = 0;
  buf.writeUInt32BE(keyType.length, offset);
  offset += 4;
  keyType.copy(buf, offset);
  offset += keyType.length;
  buf.writeUInt32BE(rawPubKey.length, offset);
  offset += 4;
  rawPubKey.copy(buf, offset);

  const b64 = buf.toString("base64");
  fs.writeFileSync(filePath, `ssh-ed25519 ${b64} permission-slip\n`, {
    mode: 0o644,
  });
}

/**
 * Returns a path relative to home for display purposes.
 */
export function displayPath(absPath: string): string {
  const home = os.homedir();
  return absPath.startsWith(home) ? absPath.replace(home, "~") : absPath;
}
