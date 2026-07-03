#!/usr/bin/env bash
#
# Re-sign bin/server with a stable code-signing identity so macOS Full Disk
# Access survives redeploys. TCC keys grants on the binary's signature; ad-hoc
# Go builds get a new cdhash every rebuild, but a real identity + fixed
# identifier satisfies the stored designated requirement across future builds.
#
# No-op unless the host is Darwin and PS_CODESIGN_IDENTITY is set.
set -euo pipefail

BINARY="${1:-bin/server}"
IDENTIFIER="${PS_LAUNCHD_LABEL:-com.permissionslip.server}"

if [ "$(uname -s)" != "Darwin" ]; then
  exit 0
fi

if [ -z "${PS_CODESIGN_IDENTITY:-}" ]; then
  exit 0
fi

if [ ! -f "$BINARY" ]; then
  echo "codesign-server: $BINARY not found" >&2
  exit 1
fi

echo "==> Signing $BINARY with identity '$PS_CODESIGN_IDENTITY' (identifier: $IDENTIFIER)"
if ! codesign --force --sign "$PS_CODESIGN_IDENTITY" --identifier "$IDENTIFIER" "$BINARY"; then
  echo "ERROR: codesign failed — check PS_CODESIGN_IDENTITY and that your login keychain is unlocked." >&2
  exit 1
fi
