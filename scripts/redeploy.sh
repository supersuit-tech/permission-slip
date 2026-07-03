#!/usr/bin/env bash
#
# Update a self-hosted Permission Slip install to the latest main and restart
# the running service — one command, nothing to remember. Restarts a systemd
# unit on Linux or a launchd LaunchAgent on macOS, whichever is present.
#
#   make redeploy           # from the repo root (preferred)
#   scripts/redeploy.sh     # or run it directly
#
# Linux (systemd) — override the unit name if you didn't call it "permission-slip":
#
#   PS_SERVICE=my-unit scripts/redeploy.sh
#
# Target a user-level unit (systemctl --user) instead of a system unit:
#
#   PS_SYSTEMCTL_ARGS=--user scripts/redeploy.sh
#
# macOS (launchd) — override the LaunchAgent label if you didn't use the one
# from docs/deployment-self-hosted.md ("com.permissionslip.server"):
#
#   PS_LAUNCHD_LABEL=com.example.permission-slip scripts/redeploy.sh
#
# macOS (persistent Full Disk Access) — sign bin/server with a stable identity
# so FDA survives redeploys (see docs/deployment-self-hosted.md):
#
#   PS_CODESIGN_IDENTITY="Permission Slip Signing" make redeploy
#
# Safety: the running server is only ever replaced by a SUCCESSFUL build. A
# failed `git pull` (e.g. a transient network blip) or a failed dependency
# install is non-fatal — the script falls back to rebuilding the current
# checkout. If the build itself fails, the script aborts BEFORE restarting, so
# the service keeps serving the last known-good binary.
#
# After a successful server restart, the script also publishes an over-the-air
# mobile update via EAS (`npx eas-cli@latest update --channel production` from
# mobile/). If EAS isn't configured (no mobile/ dir, no node/npx, no eas.json,
# or no EXPO_TOKEN / eas login), it prints a note and skips — never failing or
# rolling back the server redeploy. An unexpected EAS failure is likewise
# non-fatal: a warning is printed and the script still exits 0.
set -euo pipefail

SERVICE="${PS_SERVICE:-permission-slip}"
LAUNCHD_LABEL="${PS_LAUNCHD_LABEL:-com.permissionslip.server}"

# Resolve the repo root from this script's location so it works from anywhere.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Loud macOS reminder: iMessage reads need Full Disk Access on the server and imsg.
"$REPO_ROOT/scripts/print-macos-full-disk-access-notice.sh"

# nvm-installed node isn't always on PATH for non-interactive shells; pull it in
# so `make build` (which runs npm) doesn't fail with "node: command not found".
if ! command -v node >/dev/null 2>&1 && [ -s "$HOME/.nvm/nvm.sh" ]; then
  # shellcheck disable=SC1091
  . "$HOME/.nvm/nvm.sh"
fi

echo "==> Pulling latest from origin/main"
if ! git pull --ff-only; then
  echo "    WARNING: git pull failed — rebuilding the current checkout instead." >&2
fi

echo "==> Refreshing dependencies"
if ! make install; then
  echo "    WARNING: dependency install failed — continuing with existing dependencies." >&2
fi

echo "==> Building frontend + server"
# If this fails, 'set -e' aborts here, BEFORE the restart below, leaving the
# currently-running service untouched.
make build

if command -v systemctl >/dev/null 2>&1; then
  echo "==> Restarting service: $SERVICE"
  # Use sudo only when not already root and sudo is available — restarting a
  # system unit needs root, but a root deploy job (or a passwordless-sudo-less
  # box) shouldn't trip over a hardcoded `sudo`.
  SUDO=""
  if [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  fi
  # PS_SYSTEMCTL_ARGS lets you target a user unit, e.g.
  #   PS_SYSTEMCTL_ARGS=--user scripts/redeploy.sh
  $SUDO systemctl ${PS_SYSTEMCTL_ARGS:-} restart "$SERVICE"
elif command -v launchctl >/dev/null 2>&1; then
  echo "==> Restarting service: $LAUNCHD_LABEL"
  # Per-user LaunchAgent, as set up in docs/deployment-self-hosted.md Step 4.
  DOMAIN="gui/$(id -u)"
  if launchctl print "$DOMAIN/$LAUNCHD_LABEL" >/dev/null 2>&1; then
    launchctl kickstart -k "$DOMAIN/$LAUNCHD_LABEL"
  else
    echo "    WARNING: launchd service '$LAUNCHD_LABEL' isn't loaded in $DOMAIN — restart your server process manually, or bootstrap it as described in docs/deployment-self-hosted.md Step 4." >&2
  fi
else
  echo "    Neither systemctl nor launchctl found — restart your server process manually." >&2
fi

# Publish an OTA mobile update when EAS is configured. Additive only — never
# affects the server redeploy above or the script's exit code.
if [ ! -d "$REPO_ROOT/mobile" ] || [ ! -f "$REPO_ROOT/mobile/eas.json" ]; then
  echo "NOTE: mobile is not configured in this environment — skipping EAS update."
elif ! command -v node >/dev/null 2>&1 || ! command -v npx >/dev/null 2>&1; then
  echo "NOTE: mobile is not configured in this environment — skipping EAS update."
elif [ -z "${EXPO_TOKEN:-}" ] && ! (cd "$REPO_ROOT/mobile" && npx eas-cli@latest whoami >/dev/null 2>&1); then
  echo "NOTE: mobile is not configured in this environment — skipping EAS update."
else
  echo "==> Publishing EAS update (production channel)"
  if ! (cd "$REPO_ROOT/mobile" && npx eas-cli@latest update --channel production); then
    echo "    WARNING: EAS update failed — server redeploy completed successfully." >&2
  fi
fi

SHA="$(git rev-parse --short HEAD)"
echo "==> Done. Now running build $SHA — the app footer should show 'Build $SHA'."
