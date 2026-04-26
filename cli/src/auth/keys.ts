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
import { execFileSync } from "node:child_process";
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
 *
 * Supports both PKCS8 PEM (`-----BEGIN PRIVATE KEY-----`, what Node natively
 * understands) and the OpenSSH native format (`-----BEGIN OPENSSH PRIVATE KEY-----`,
 * what `ssh-keygen` produces by default). For OpenSSH keys, the 32-byte ed25519
 * seed is extracted manually and converted to a JWK before construction.
 */
export function loadPrivateKey(): crypto.KeyObject {
  if (!fs.existsSync(PRIVATE_KEY_FILE)) {
    throw new Error(
      `Private key not found at ${PRIVATE_KEY_FILE}. Run 'permission-slip register' to generate one.`,
    );
  }
  const raw = fs.readFileSync(PRIVATE_KEY_FILE, "utf-8");
  if (raw.includes("BEGIN OPENSSH PRIVATE KEY")) {
    return loadOpenSSHPrivateKey(raw);
  }
  return crypto.createPrivateKey({ key: raw, format: "pem" });
}

/**
 * Parses an unencrypted OpenSSH-format ed25519 private key and returns it as a
 * Node KeyObject. Format reference: PROTOCOL.key in the OpenSSH source.
 */
function loadOpenSSHPrivateKey(pem: string): crypto.KeyObject {
  const b64 = pem
    .replace(/-----BEGIN OPENSSH PRIVATE KEY-----/, "")
    .replace(/-----END OPENSSH PRIVATE KEY-----/, "")
    .replace(/\s+/g, "");
  const buf = Buffer.from(b64, "base64");

  // Magic: "openssh-key-v1\0"
  const magic = "openssh-key-v1\0";
  if (buf.slice(0, magic.length).toString("binary") !== magic) {
    throw new Error("Not a valid OpenSSH private key (bad magic)");
  }
  let off = magic.length;

  const readString = (): Buffer => {
    const len = buf.readUInt32BE(off);
    off += 4;
    const out = buf.slice(off, off + len);
    off += len;
    return out;
  };

  const cipher = readString().toString();
  if (cipher !== "none") {
    throw new Error(
      `Encrypted OpenSSH keys are not supported (cipher: ${cipher}). Decrypt the key with 'ssh-keygen -p -P <pw> -N "" -f <key>' first.`,
    );
  }
  readString(); // kdfname
  readString(); // kdfoptions
  const numKeys = buf.readUInt32BE(off);
  off += 4;
  if (numKeys !== 1) {
    throw new Error(`Expected exactly 1 key in OpenSSH file, got ${numKeys}`);
  }
  readString(); // public key blob (we don't need it; derive from private)
  const privateBlob = readString();

  // Parse the private key blob.
  let pOff = 8; // skip check1+check2
  const readPrivString = (): Buffer => {
    const len = privateBlob.readUInt32BE(pOff);
    pOff += 4;
    const out = privateBlob.slice(pOff, pOff + len);
    pOff += len;
    return out;
  };
  const keyType = readPrivString().toString();
  if (keyType !== "ssh-ed25519") {
    throw new Error(`Unsupported key type: ${keyType} (only ssh-ed25519)`);
  }
  readPrivString(); // public key (32 bytes)
  const privKey = readPrivString(); // 64 bytes: 32-byte seed + 32-byte public
  if (privKey.length !== 64) {
    throw new Error(`Unexpected ed25519 private key length: ${privKey.length}`);
  }
  const seed = privKey.slice(0, 32);
  const pub = privKey.slice(32, 64);

  const b64url = (b: Buffer) =>
    b.toString("base64").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  return crypto.createPrivateKey({
    key: { kty: "OKP", crv: "Ed25519", d: b64url(seed), x: b64url(pub) },
    format: "jwk",
  });
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
    // Use execFileSync (not execSync) to avoid shell interpolation of the key path.
    execFileSync(
      "ssh-keygen",
      ["-t", "ed25519", "-f", PRIVATE_KEY_FILE, "-N", "", "-C", "permission-slip"],
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
 * Writes an Ed25519 private key in PKCS8 PEM format.
 * Used as a fallback when ssh-keygen is not available.
 *
 * NOTE: This produces PKCS8 PEM (`-----BEGIN PRIVATE KEY-----`), not the
 * OpenSSH native format (`-----BEGIN OPENSSH PRIVATE KEY-----`). Node's
 * `crypto.createPrivateKey` accepts both, so signing works correctly.
 * However, standard SSH tooling (e.g. `ssh-add`, `ssh-keygen -y`) will
 * not recognize PKCS8 PEM — if interoperability with SSH tools is required,
 * use a platform that has `ssh-keygen` available.
 */
function writeOpenSSHPrivateKey(
  key: crypto.KeyObject,
  filePath: string,
): void {
  const pem = key.export({ type: "pkcs8", format: "pem" }) as string;
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
