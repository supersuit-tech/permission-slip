#!/usr/bin/env bash
# watch-post.sh — Post-session tasks for the /watch skill.
# Handles CI triggering, waiting, auto-merge, and webhook notification.
# Invoked after the polling loop ends (idle timeout) and the agent has
# finished all its work.
#
# Usage: bash watch-post.sh [PR_URL] [--automerge]
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
AUTO_MERGE=false

# Parse arguments — PR URL is optional (auto-detected from current branch if omitted)
while [[ $# -gt 0 ]]; do
  case "$1" in
    --automerge) AUTO_MERGE=true; shift ;;
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
# Report CI results
# ---------------------------------------------------------------------------
if [[ "$ci_conclusion" == "failure" ]]; then
  echo ""
  echo "CI_FAILED"
  echo "CI_RUN_ID=${ci_run_id}"
  # Fetch failed logs for the agent
  echo "[post] Fetching failed CI logs..."
  gh_cmd run view "$ci_run_id" --log-failed 2>/dev/null > "/tmp/ci-failed-logs.txt" || true
  echo "CI_LOGS_FILE=/tmp/ci-failed-logs.txt"
  echo "AGENT_NEEDED"
  echo "REASON=CI failure needs diagnosis and fix"
  # Exit with special code — caller invokes agent to fix
  exit 101
fi

if [[ "$audit_conclusion" == "failure" ]]; then
  echo ""
  echo "AUDIT_FAILED"
  echo "AUDIT_RUN_ID=${audit_run_id}"
  echo "[post] Fetching failed audit logs..."
  gh_cmd run view "$audit_run_id" --log-failed 2>/dev/null > "/tmp/audit-failed-logs.txt" || true
  echo "AUDIT_LOGS_FILE=/tmp/audit-failed-logs.txt"
  echo "AGENT_NEEDED"
  echo "REASON=Audit failure needs diagnosis and fix"
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
# Trigger webhook notification
# ---------------------------------------------------------------------------
echo "[post] Triggering webhook notification..."
gh_cmd workflow run trigger-webhook.yml -f pr_url="${PR_URL}" 2>/dev/null || echo "[post] WARNING: Failed to trigger webhook"

echo ""
echo "POST_COMPLETE"
echo "CI=${ci_conclusion}"
echo "AUDIT=${audit_conclusion}"
echo "AUTO_MERGE=${AUTO_MERGE}"
