import fs from "node:fs";

export default function globalTeardown() {
  const tmpRoot = process.env["PS_CLI_TEST_TMP_ROOT"];
  if (tmpRoot) {
    fs.rmSync(tmpRoot, { recursive: true, force: true });
  }
}
