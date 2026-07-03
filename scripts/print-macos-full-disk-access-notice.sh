#!/usr/bin/env bash
#
# Print a loud, colored reminder that macOS Full Disk Access is required for
# Permission Slip's server process and for imsg when using the iMessage connector.
# No-op on non-macOS hosts (e.g. Linux self-hosted redeploys).
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
else
  BOLD=''
  RESET=''
  FG_BLACK=''
  FG_WHITE=''
  BG_YELLOW=''
  BG_RED=''
  FG_YELLOW=''
fi

banner_line() {
  local bg="$1"
  local fg="$2"
  local text="$3"
  printf '%b%b  %-74s  %b\n' "$bg" "$fg" "$text" "$RESET"
}

printf '\n'
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
banner_line "$BG_RED" "$FG_WHITE" "!!  macOS FULL DISK ACCESS REQUIRED FOR iMESSAGE / imsg                 !!"
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
printf '\n'
banner_line "$BG_YELLOW" "$FG_BLACK" "Grant Full Disk Access to BOTH of these processes:"
banner_line "$BG_YELLOW" "$FG_BLACK" "  1. Permission Slip server  (bin/server — the process you just restarted)"
banner_line "$BG_YELLOW" "$FG_BLACK" "  2. imsg                    (brew install steipete/tap/imsg)"
printf '\n'
printf '%b%b  System Settings → Privacy & Security → Full Disk Access%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '%b%b  Add each app/binary, then quit and relaunch so macOS picks up the grant.%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '%b%b  Without this, iMessage reads fail even when Permission Slip approvals succeed.%b\n' "$BOLD" "$FG_YELLOW" "$RESET"
printf '\n'
banner_line "$BG_RED" "$FG_WHITE" "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
printf '\n'
