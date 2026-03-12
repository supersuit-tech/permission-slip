/**
 * Jest global setup — runs once before all test suites.
 * Creates temp dirs and sets environment variables so the store/keys
 * modules use test paths instead of ~/.permission-slip and ~/.ssh.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

export default function globalSetup() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "ps-cli-test-"));
  const sshDir = path.join(tmpDir, "ssh");
  fs.mkdirSync(sshDir, { recursive: true, mode: 0o700 });

  const privKeyFile = path.join(sshDir, "permission_slip_agent");
  const pubKeyFile = path.join(sshDir, "permission_slip_agent.pub");

  // Generate a key pair for all tests
  const { privateKey, publicKey } = crypto.generateKeyPairSync("ed25519");
  const pem = privateKey.export({ type: "pkcs8", format: "pem" }) as string;
  fs.writeFileSync(privKeyFile, pem, { mode: 0o600 });

  // Write public key in OpenSSH format
  const derBuf = publicKey.export({ type: "spki", format: "der" }) as Buffer;
  const rawPub = derBuf.slice(-32);
  const keyType = Buffer.from("ssh-ed25519");
  const wireBuf = Buffer.alloc(4 + keyType.length + 4 + rawPub.length);
  let off = 0;
  wireBuf.writeUInt32BE(keyType.length, off); off += 4;
  keyType.copy(wireBuf, off); off += keyType.length;
  wireBuf.writeUInt32BE(rawPub.length, off); off += 4;
  rawPub.copy(wireBuf, off);
  fs.writeFileSync(pubKeyFile, `ssh-ed25519 ${wireBuf.toString("base64")} test\n`, { mode: 0o644 });

  // Set env vars so the modules pick up the test paths
  process.env["PS_CLI_TEST_CONFIG_DIR"] = tmpDir;
  process.env["PS_CLI_TEST_SSH_DIR"] = sshDir;
  process.env["PS_CLI_TEST_PRIVATE_KEY"] = privKeyFile;
  process.env["PS_CLI_TEST_PUBLIC_KEY"] = pubKeyFile;
  process.env["PS_CLI_TEST_DIR"] = tmpDir;

  // Teardown: store path for globalTeardown
  process.env["PS_CLI_TEST_TMP_ROOT"] = tmpDir;
}
