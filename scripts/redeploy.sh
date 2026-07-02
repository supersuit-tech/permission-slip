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
# Safety: the running server is only ever replaced by a SUCCESSFUL build. A
# failed `git pull` (e.g. a transient network blip) or a failed dependency
# install is non-fatal — the script falls back to rebuilding the current
# checkout. If the build itself fails, the script aborts BEFORE restarting, so
# the service keeps serving the last known-good binary.
set -euo pipefail

SERVICE="${PS_SERVICE:-permission-slip}"
LAUNCHD_LABEL="${PS_LAUNCHD_LABEL:-com.permissionslip.server}"

# Resolve the repo root from this script's location so it works from anywhere.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

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

SHA="$(git rev-parse --short HEAD)"
echo "==> Done. Now running build $SHA — the app footer should show 'Build $SHA'."
