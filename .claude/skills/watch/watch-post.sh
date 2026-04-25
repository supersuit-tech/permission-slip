#!/usr/bin/env bash
# watch-post.sh — Post-session tasks for the /watch skill.
# Handles optional auto-merge and webhook notification.
# Invoked after the polling loop ends (idle timeout) and the agent has
# finished all its work.
#
# CI and audit run on every push to a PR via GitHub Actions; they are not
# required checks for merge. This script does not wait on or gate merge on
# workflow results (use /fix-ci to drive CI to green when needed).
#
# Usage: bash watch-post.sh [PR_URL] [--work-dir DIR] [--no-automerge] [--skip-webhook] [--webhook-only]
#
# This script handles:
#   - Auto-merge when enabled (merge is not blocked on CI/audit here)
#   - Webhook notification (legacy flags; webhook currently disabled)

set -euo pipefail

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------
PR_URL=""
AUTO_MERGE=true
NO_NOTIFY=false
SKIP_WEBHOOK=false
WEBHOOK_ONLY=false
WORK_DIR=""

# Parse arguments — PR URL is optional (auto-detected from current branch if omitted)
while [[ $# -gt 0 ]]; do
  case "$1" in
    --automerge) AUTO_MERGE=true; shift ;;  # kept for backwards compat
    --no-automerge) AUTO_MERGE=false; shift ;;
    --no-notify) NO_NOTIFY=true; shift ;;
    --skip-webhook) SKIP_WEBHOOK=true; shift ;;
    --webhook-only) WEBHOOK_ONLY=true; shift ;;
    --work-dir) WORK_DIR="$2"; shift 2 ;;
    https://*) PR_URL="$1"; shift ;;
    *) shift ;;
  esac
done

# Auto-detect PR URL from current branch if not provided
if [[ -z "$PR_URL" ]]; then
  echo "[post] No PR URL provided, detecting from current branch..."
  PR_URL=$(GH_HOST="${GH_HOST:-github.com}" GH_REPO="${GH_REPO:-supersuit-tech/permission-slip}" gh pr view --json url --jq '.url' 2>/dev/null || echo "")
  if [[ -z "$PR_URL" ]]; then
    echo "ERROR: No PR URL provided and could not detect a PR for the current branch." >&2
    exit 1
  fi
  echo "[post] Detected PR: ${PR_URL}"
fi

PR_NUMBER=$(echo "$PR_URL" | grep -oP '/pull/\K[0-9]+')
OWNER="supersuit-tech"
REPO="permission-slip"
export GH_HOST="${GH_HOST:-github.com}"
export GH_REPO="${GH_REPO:-${OWNER}/${REPO}}"

BRANCH=$(git branch --show-current)

gh_api() {
  GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh api "$@"
}

gh_cmd() {
  GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh "$@"
}

# ---------------------------------------------------------------------------
# Webhook-only mode (after a successful --skip-webhook run)
# ---------------------------------------------------------------------------
if [[ "$WEBHOOK_ONLY" == "true" ]]; then
  if [[ "$NO_NOTIFY" == "true" ]]; then
    echo "[post] --webhook-only with --no-notify: nothing to do."
    exit 0
  fi
  pending_file="${WORK_DIR}/post-webhook-pending"
  if [[ -z "$WORK_DIR" || ! -f "$pending_file" ]]; then
    echo "[post] ERROR: --webhook-only requires --work-dir and a prior successful watch-post run with --skip-webhook (missing ${pending_file})." >&2
    exit 2
  fi
  echo "[post] Webhook notifications disabled — skipping."
  rm -f "$pending_file"
  echo "POST_WEBHOOK_ONLY=true"
  exit 0
fi

LOG_ROOT="${WORK_DIR:-/tmp}"
mkdir -p "$LOG_ROOT"

echo "[post] CI and audit run on push/PR via Actions; not waiting on workflow results before merge."

# ---------------------------------------------------------------------------
# Auto-merge if enabled (does not wait on or require CI/audit success)
# ---------------------------------------------------------------------------
if [[ "$AUTO_MERGE" == "true" ]]; then
  echo "[post] Auto-merge enabled. Merging PR..."
  if ! gh_cmd pr merge "$PR_NUMBER" --squash --delete-branch 2>/dev/null; then
    echo "[post] Auto-merge failed. Posting comment..."
    gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" \
      -f body="⚠️ **Auto-merge failed.** The \`--automerge\` flag was set, but the merge could not be completed. Please merge manually." \
      > /dev/null 2>&1
  else
    echo "[post] PR merged successfully."
  fi
fi

# ---------------------------------------------------------------------------
# Trigger webhook notification (skipped with --no-notify or --skip-webhook)
# ---------------------------------------------------------------------------
# Webhook notifications disabled — trigger-webhook.yml is no longer in use.
# The flags (--no-notify, --skip-webhook) are kept for backwards compatibility.
echo "[post] Webhook notifications disabled — skipping."

# post-webhook-pending sentinel no longer needed (webhook disabled)

echo ""
echo "POST_COMPLETE"
echo "AUTO_MERGE=${AUTO_MERGE}"
