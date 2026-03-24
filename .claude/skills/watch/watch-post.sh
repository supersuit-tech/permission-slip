#!/usr/bin/env bash
# watch-post.sh — Post-session tasks for the /watch skill.
# Handles CI triggering, waiting, auto-merge, and webhook notification.
# Invoked after the polling loop ends (idle timeout) and the agent has
# finished all its work.
#
# Usage: bash watch-post.sh [PR_URL] [--work-dir DIR] [--no-automerge] [--skip-webhook] [--webhook-only]
#
# This script handles steps 11-13 from the original SKILL.md:
#   11. Trigger CI and audit, wait, fix failures (signals agent for fixes)
#   12. Auto-merge if enabled and checks pass
#   13. Trigger webhook notification

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
CI_LOGS_FILE="${LOG_ROOT}/ci-failed-logs.txt"
AUDIT_LOGS_FILE="${LOG_ROOT}/audit-failed-logs.txt"

# ---------------------------------------------------------------------------
# Snapshot latest run IDs before triggering (to detect new runs)
# ---------------------------------------------------------------------------
get_latest_run_id() {
  local workflow="$1"
  gh_cmd run list --workflow="$workflow" --branch "$BRANCH" --limit 1 --json databaseId 2>/dev/null \
    | jq -r '.[0].databaseId // "0"'
}

ci_prev_run_id=$(get_latest_run_id "ci.yml")
audit_prev_run_id=$(get_latest_run_id "audit.yml")

# ---------------------------------------------------------------------------
# Trigger CI and audit workflows
# ---------------------------------------------------------------------------
echo "[post] Triggering CI workflow on ${BRANCH}..."
gh_cmd workflow run ci.yml --ref "$BRANCH" 2>/dev/null || echo "[post] WARNING: Failed to trigger ci.yml"

echo "[post] Triggering audit workflow on ${BRANCH}..."
gh_cmd workflow run audit.yml --ref "$BRANCH" 2>/dev/null || echo "[post] WARNING: Failed to trigger audit.yml"

# ---------------------------------------------------------------------------
# Wait for workflows to complete
# ---------------------------------------------------------------------------
wait_for_workflow() {
  local workflow="$1"
  local prev_run_id="$2"
  local max_wait=600  # 10 minutes
  local elapsed=0

  sleep 5  # Wait for run to register

  while [[ $elapsed -lt $max_wait ]]; do
    local result
    result=$(gh_cmd run list --workflow="$workflow" --branch "$BRANCH" --limit 1 --json databaseId,status,conclusion 2>/dev/null || echo "[]")

    local status conclusion run_id
    status=$(echo "$result" | jq -r '.[0].status // "unknown"')
    conclusion=$(echo "$result" | jq -r '.[0].conclusion // "unknown"')
    run_id=$(echo "$result" | jq -r '.[0].databaseId // "unknown"')

    # Skip stale runs from before we triggered
    if [[ "$run_id" != "unknown" && "$run_id" != "$prev_run_id" && "$status" == "completed" ]]; then
      echo "${conclusion}:${run_id}"
      return
    fi

    sleep 30
    elapsed=$((elapsed + 30))
  done

  echo "timeout:unknown"
}

echo "[post] Waiting for CI workflow..."
ci_result=$(wait_for_workflow "ci.yml" "$ci_prev_run_id")
ci_conclusion="${ci_result%%:*}"
ci_run_id="${ci_result##*:}"
echo "[post] CI result: ${ci_conclusion} (run ${ci_run_id})"

echo "[post] Waiting for audit workflow..."
audit_result=$(wait_for_workflow "audit.yml" "$audit_prev_run_id")
audit_conclusion="${audit_result%%:*}"
audit_run_id="${audit_result##*:}"
echo "[post] Audit result: ${audit_conclusion} (run ${audit_run_id})"

# ---------------------------------------------------------------------------
# Report CI results (anything other than success needs remediation)
# ---------------------------------------------------------------------------
if [[ "$ci_conclusion" != "success" ]]; then
  echo ""
  echo "CI_FAILED"
  echo "CI_RUN_ID=${ci_run_id}"
  echo "CI_CONCLUSION=${ci_conclusion}"
  echo "[post] Fetching CI logs (failed jobs if any)..."
  if [[ "$ci_run_id" != "unknown" ]]; then
    gh_cmd run view "$ci_run_id" --log-failed 2>/dev/null > "$CI_LOGS_FILE" || true
    gh_cmd run view "$ci_run_id" --log 2>/dev/null >> "$CI_LOGS_FILE" || true
  else
    : > "$CI_LOGS_FILE"
  fi
  echo "CI_LOGS_FILE=${CI_LOGS_FILE}"
  echo "AGENT_NEEDED"
  if [[ "$ci_conclusion" == "timeout" ]]; then
    echo "REASON=CI workflow did not complete within wait window — re-trigger or investigate GitHub Actions"
  else
    echo "REASON=CI did not succeed (conclusion=${ci_conclusion}) — fix and re-run"
  fi
  exit 101
fi

if [[ "$audit_conclusion" != "success" ]]; then
  echo ""
  echo "AUDIT_FAILED"
  echo "AUDIT_RUN_ID=${audit_run_id}"
  echo "AUDIT_CONCLUSION=${audit_conclusion}"
  echo "[post] Fetching audit logs (failed jobs if any)..."
  if [[ "$audit_run_id" != "unknown" ]]; then
    gh_cmd run view "$audit_run_id" --log-failed 2>/dev/null > "$AUDIT_LOGS_FILE" || true
    gh_cmd run view "$audit_run_id" --log 2>/dev/null >> "$AUDIT_LOGS_FILE" || true
  else
    : > "$AUDIT_LOGS_FILE"
  fi
  echo "AUDIT_LOGS_FILE=${AUDIT_LOGS_FILE}"
  echo "AGENT_NEEDED"
  if [[ "$audit_conclusion" == "timeout" ]]; then
    echo "REASON=Audit workflow did not complete within wait window — re-trigger or investigate GitHub Actions"
  else
    echo "REASON=Audit did not succeed (conclusion=${audit_conclusion}) — fix and re-run"
  fi
  exit 102
fi

# ---------------------------------------------------------------------------
# Auto-merge if enabled and both workflows passed
# ---------------------------------------------------------------------------
if [[ "$AUTO_MERGE" == "true" && "$ci_conclusion" == "success" && "$audit_conclusion" == "success" ]]; then
  echo "[post] Auto-merge enabled and checks passed. Merging PR..."
  if ! gh_cmd pr merge "$PR_NUMBER" --squash --delete-branch 2>/dev/null; then
    echo "[post] Auto-merge failed. Posting comment..."
    gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" \
      -f body="⚠️ **Auto-merge failed.** The \`--automerge\` flag was set, but the merge could not be completed. Please merge manually." \
      > /dev/null 2>&1
  else
    echo "[post] PR merged successfully."
  fi
elif [[ "$AUTO_MERGE" == "true" ]]; then
  echo "[post] Auto-merge enabled but checks did not pass. Skipping merge."
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
echo "CI=${ci_conclusion}"
echo "AUDIT=${audit_conclusion}"
echo "AUTO_MERGE=${AUTO_MERGE}"
