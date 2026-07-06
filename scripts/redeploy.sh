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
# When redeploying with PS_CODESIGN_IDENTITY set and the login keychain is
# locked (common over SSH or in detached tmux/screen sessions), the script
# prompts for your macOS login password to unlock it before building —
# codesign can't use the signing key from a locked keychain and fails with
# 'errSecInternalComponent' otherwise. Already-unlocked keychains skip the
# prompt, so GUI sessions are typically unaffected.
#
# Safety: the running server is only ever replaced by a SUCCESSFUL build. A
# failed `git pull` (e.g. a transient network blip) or a failed dependency
# install is non-fatal — the script falls back to rebuilding the current
# checkout. If the build itself fails, the script aborts BEFORE restarting, so
# the service keeps serving the last known-good binary.
#
# After a successful server restart, the script may publish an over-the-air
# mobile update via EAS (`npx eas-cli@latest update --channel production` from
# mobile/). Publishing is gated on new mobile/v* release tags: if the latest tag
# reachable from HEAD matches the last successfully published tag (recorded in
# .mobile-ota-deployed-tag), the EAS step is skipped. When no mobile/v* tag is
# reachable from HEAD yet, the script keeps the legacy always-publish behavior.
# The deploy summary compares each component's current release tag against durable
# state from the last successful redeploy (.web-deployed-version for web,
# .mobile-ota-deployed-tag for mobile) so it stays accurate even when the
# checkout was already up to date before this run started.
# Force a publish regardless of tag state:
#
#   PS_FORCE_EAS_UPDATE=1 make redeploy
#
# If EAS isn't configured (no mobile/ dir, no node/npx, no eas.json, or no
# EXPO_TOKEN / eas login), it prints a note and skips — never failing or rolling
# back the server redeploy. An unexpected EAS failure is likewise non-fatal: a
# warning is printed and the script still exits 0. The state file is only
# updated after a successful publish, so a skipped or failed EAS step retries
# on the next redeploy.
set -euo pipefail

SERVICE="${PS_SERVICE:-permission-slip}"
LAUNCHD_LABEL="${PS_LAUNCHD_LABEL:-com.permissionslip.server}"

# Resolve the repo root from this script's location so it works from anywhere.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Return the semver from the latest reachable <prefix>/v* tag, or empty when
# none exists.
release_version() {
  local prefix="$1"
  local tag
  tag="$(git tag --list "${prefix}/v*" --merged HEAD 2>/dev/null | sort -V | tail -n1 || true)"
  if [ -n "$tag" ]; then
    echo "${tag#${prefix}/v}"
  fi
}

WEB_DEPLOYED_VERSION_FILE="$REPO_ROOT/.web-deployed-version"
MOBILE_OTA_DEPLOYED_TAG_FILE="$REPO_ROOT/.mobile-ota-deployed-tag"

# Read the version recorded by the last successful redeploy (not the current
# checkout) so the deploy summary stays accurate even when the user already ran
# `git pull` before `make redeploy`.
read_recorded_version() {
  local file="$1"
  local strip_prefix="${2:-}"
  if [ ! -f "$file" ]; then
    return 0
  fi
  local raw
  raw="$(tr -d '[:space:]' < "$file")"
  if [ -z "$raw" ]; then
    return 0
  fi
  if [ -n "$strip_prefix" ]; then
    echo "${raw#${strip_prefix}}"
  else
    echo "$raw"
  fi
}

LAST_WEB_VERSION="$(read_recorded_version "$WEB_DEPLOYED_VERSION_FILE")"
LAST_MOBILE_VERSION="$(read_recorded_version "$MOBILE_OTA_DEPLOYED_TAG_FILE" "mobile/v")"

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

# Tag auto-following can miss release tags whose commits are already local
# (e.g. when a previous redeploy pulled main before the tag workflow finished).
git fetch origin 'refs/tags/mobile/v*:refs/tags/mobile/v*' 2>/dev/null || true
git fetch origin 'refs/tags/web/v*:refs/tags/web/v*' 2>/dev/null || true

echo "==> Refreshing dependencies"
if ! make install; then
  echo "    WARNING: dependency install failed — continuing with existing dependencies." >&2
fi

# codesign needs the login keychain unlocked to use PS_CODESIGN_IDENTITY's
# private key; a locked keychain fails with the cryptic
# 'errSecInternalComponent'. Don't guess the lock state from SSH env vars —
# they're missing in reattached tmux/screen sessions, and GUI keychains can
# lock on a timer. Probe the actual state: `security show-keychain-info`
# succeeds when the keychain is unlocked and fails with
# errSecInteractionNotAllowed when it's locked and can't prompt.
LOGIN_KEYCHAIN="$HOME/Library/Keychains/login.keychain-db"
if [ "$(uname -s)" = "Darwin" ] && [ -n "${PS_CODESIGN_IDENTITY:-}" ] && [ -f "$LOGIN_KEYCHAIN" ]; then
  if security show-keychain-info "$LOGIN_KEYCHAIN" >/dev/null 2>&1; then
    : # keychain already unlocked — no prompt needed
  elif [ -t 0 ]; then
    echo "==> Login keychain is locked — unlocking for codesign"
    if ! security unlock-keychain "$LOGIN_KEYCHAIN"; then
      echo "    WARNING: keychain unlock failed — codesign may fail with errSecInternalComponent." >&2
    fi
  else
    echo "    WARNING: login keychain is locked and there's no TTY to prompt — codesign may fail with errSecInternalComponent. Unlock it first: security unlock-keychain ~/Library/Keychains/login.keychain-db" >&2
  fi
fi

echo "==> Building frontend + server"
# Web is always rebuilt and redeployed on every redeploy — there is no tag-gated
# skip for web. Do NOT mirror the mobile OTA tag-gating logic below onto web.
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
# affects the server redeploy above or the script's exit code. Web redeploy is
# never gated on release tags; only mobile OTA publishing uses that pattern.
EAS_AUTH_CHECK_TIMEOUT="${PS_EAS_AUTH_CHECK_TIMEOUT:-30}"

# Return 0 when EAS credentials are available (EXPO_TOKEN or eas-cli login).
# Uses npx --yes and a closed stdin so install/login prompts can't block forever;
# optional timeout (PS_EAS_AUTH_CHECK_TIMEOUT, default 30s) caps slow networks.
eas_is_authenticated() {
  if [ -n "${EXPO_TOKEN:-}" ]; then
    return 0
  fi

  local run_whoami=(
    npx --yes eas-cli@latest whoami
  )
  if command -v timeout >/dev/null 2>&1; then
    run_whoami=(timeout "$EAS_AUTH_CHECK_TIMEOUT" "${run_whoami[@]}")
  elif command -v gtimeout >/dev/null 2>&1; then
    run_whoami=(gtimeout "$EAS_AUTH_CHECK_TIMEOUT" "${run_whoami[@]}")
  fi

  (
    cd "$REPO_ROOT/mobile"
    "${run_whoami[@]}"
  ) </dev/null >/dev/null 2>&1
}

LATEST_MOBILE_TAG="$(git tag --list 'mobile/v*' --merged HEAD | sort -V | tail -n1 || true)"
DEPLOYED_MOBILE_TAG=""
if [ -f "$MOBILE_OTA_DEPLOYED_TAG_FILE" ]; then
  DEPLOYED_MOBILE_TAG="$(tr -d '[:space:]' < "$MOBILE_OTA_DEPLOYED_TAG_FILE")"
fi

SKIP_EAS_UPDATE=0
if [ -n "$LATEST_MOBILE_TAG" ] && [ "$LATEST_MOBILE_TAG" = "$DEPLOYED_MOBILE_TAG" ] && [ "${PS_FORCE_EAS_UPDATE:-}" != "1" ]; then
  SKIP_EAS_UPDATE=1
fi

if [ ! -d "$REPO_ROOT/mobile" ] || [ ! -f "$REPO_ROOT/mobile/eas.json" ]; then
  echo "NOTE: mobile is not configured in this environment — skipping EAS update."
elif ! command -v node >/dev/null 2>&1 || ! command -v npx >/dev/null 2>&1; then
  echo "NOTE: mobile is not configured in this environment — skipping EAS update."
elif [ "$SKIP_EAS_UPDATE" -eq 1 ]; then
  echo "NOTE: EAS update skipped — mobile release tag '$LATEST_MOBILE_TAG' was already published. Set PS_FORCE_EAS_UPDATE=1 to force a publish."
elif ! eas_is_authenticated; then
  echo "NOTE: EAS update skipped — not logged in. Run 'npx eas-cli login' or set EXPO_TOKEN, then re-run 'make redeploy' to publish the OTA update."
else
  echo "==> Publishing EAS update (production channel)"
  if (cd "$REPO_ROOT/mobile" && npx --yes eas-cli@latest update --channel production); then
    if [ -n "$LATEST_MOBILE_TAG" ]; then
      printf '%s\n' "$LATEST_MOBILE_TAG" > "$MOBILE_OTA_DEPLOYED_TAG_FILE"
    fi
  else
    echo "    WARNING: EAS update failed — server redeploy completed successfully." >&2
  fi
fi

DEPLOYED_WEB_VERSION="$(release_version web)"
DEPLOYED_MOBILE_VERSION="$(release_version mobile)"
CLI_VERSION="$(release_version cli)"

format_deploy_line() {
  local name="$1"
  local last="$2"
  local current="$3"
  if [ -z "$current" ]; then
    echo "  $name: version unknown"
  elif [ -z "$last" ]; then
    echo "  $name is now at $current"
  elif [ "$last" = "$current" ]; then
    echo "  $name rebuilt and redeployed at $current"
  else
    echo "  Upgraded $name from $last to $current"
  fi
}

if [ -n "$DEPLOYED_WEB_VERSION" ]; then
  printf '%s\n' "$DEPLOYED_WEB_VERSION" > "$WEB_DEPLOYED_VERSION_FILE"
fi

echo "==> Done."
echo "  Deploy summary:"
format_deploy_line "Web" "$LAST_WEB_VERSION" "$DEPLOYED_WEB_VERSION"
format_deploy_line "Mobile" "$LAST_MOBILE_VERSION" "$DEPLOYED_MOBILE_VERSION"
if [ -n "$CLI_VERSION" ]; then
  echo "  CLI version: $CLI_VERSION"
else
  echo "  CLI version: unknown"
fi
