#!/usr/bin/env bash
#
# Print a reminder that macOS Full Disk Access is required for Permission Slip's
# server process when using the iMessage connector. No-op on non-macOS hosts.
#
# When PS_CODESIGN_IDENTITY is set, prints a short note (FDA survives redeploys).
# Otherwise prints a loud banner reminding operators to grant FDA after each build.
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  exit 0
fi

if [ -t 1 ]; then
  BOLD='\033[1m'
  RESET='\033[0m'
  FG_BLACK='\033[30m'
  FG_WHITE='\033[97m'
  BG_YELLOW='\033[43m'
  BG_RED='\033[41m'
  FG_YELLOW='\033[33m'
  FG_GREEN='\033[32m'
else
  BOLD=''
  RESET=''
  FG_BLACK=''
  FG_WHITE=''
  BG_YELLOW=''
  BG_RED=''
  FG_YELLOW=''
  FG_GREEN=''
fi

banner_line() {
  local bg="$1"
  local fg="$2"
  local text="$3"
  printf '%b%b  %-74s  %b\n' "$bg" "$fg" "$text" "$RESET"
}

if [ -n "${PS_CODESIGN_IDENTITY:-}" ]; then
  printf '\n'
  printf '%b%b  Full Disk Access: bin/server is signed with a stable identity — no re-grant needed after redeploys.%b\n' "$BOLD" "$FG_GREEN" "$RESET"
  printf '%b%b  Grant FDA to bin/server once if you have not already (System Settings → Privacy & Security → Full Disk Access).%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
  printf '\n'
  exit 0
fi

printf '\n'
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
banner_line "$BG_RED" "$FG_WHITE" "!!  macOS FULL DISK ACCESS REQUIRED FOR iMESSAGE / imsg                 !!"
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
printf '\n'
banner_line "$BG_YELLOW" "$FG_BLACK" "Grant Full Disk Access to the Permission Slip server:"
banner_line "$BG_YELLOW" "$FG_BLACK" "  bin/server — the process you just restarted (this is the grant that matters)"
printf '\n'
banner_line "$BG_YELLOW" "$FG_BLACK" "For standalone imsg / Terminal runs, also grant FDA to imsg itself."
printf '\n'
printf '%b%b  System Settings → Privacy & Security → Full Disk Access%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '%b%b  Add each app/binary, then quit and relaunch so macOS picks up the grant.%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '%b%b  Without this, iMessage reads fail even when Permission Slip approvals succeed.%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '%b%b  Tip: set PS_CODESIGN_IDENTITY to keep FDA across redeploys — see docs/deployment-self-hosted.md%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '\n'
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
printf '\n'
