#!/usr/bin/env bash
# fix-ci-trigger.sh — Trigger ci.yml + audit.yml on a branch, wait, fetch logs on failure.
# Used by the /fix-ci skill for one round of remote validation.
#
# Usage: bash fix-ci-trigger.sh [--work-dir DIR]
#
# Environment:
#   GH_HOST, GH_REPO — passed through to gh (defaults match permission-slip)
#
# Exit codes:
#   0 — both workflows succeeded
#   101 — CI did not succeed (logs in CI_LOGS_FILE)
#   102 — Audit did not succeed (logs in AUDIT_LOGS_FILE)
#   1 — usage / missing branch / gh error

set -euo pipefail

WORK_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --work-dir) WORK_DIR="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

export GH_HOST="${GH_HOST:-github.com}"
export GH_REPO="${GH_REPO:-supersuit-tech/permission-slip}"

BRANCH=$(git branch --show-current 2>/dev/null || true)
if [[ -z "$BRANCH" ]]; then
  echo "ERROR: Not on a named branch (detached HEAD?). Checkout a branch first." >&2
  exit 1
fi

gh_cmd() {
  GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh "$@"
}

LOG_ROOT="${WORK_DIR:-/tmp}"
mkdir -p "$LOG_ROOT"
CI_LOGS_FILE="${LOG_ROOT}/fix-ci-ci-failed-logs.txt"
AUDIT_LOGS_FILE="${LOG_ROOT}/fix-ci-audit-failed-logs.txt"

LOCAL_SHA=$(git rev-parse HEAD)

# Fast path: latest completed CI + audit on this branch already refer to this commit and passed.
latest_completed_for_sha() {
  local workflow="$1"
  gh_cmd run list --workflow="$workflow" --branch "$BRANCH" --limit 25 \
    --json databaseId,status,conclusion,headSha 2>/dev/null \
    | jq -r --arg sha "$LOCAL_SHA" '
        [.[] | select(.headSha == $sha and .status == "completed")]
        | if length == 0 then empty
          elif .[0].conclusion == "success" then "success:\(.[0].databaseId)"
          else "failed:\(.[0].databaseId):\(.[0].conclusion)" end'
}

wait_for_workflow() {
  local workflow="$1"
  local prev_run_id="$2"
  local max_wait=600
  local elapsed=0

  sleep 5

  while [[ $elapsed -lt $max_wait ]]; do
    local result
    result=$(gh_cmd run list --workflow="$workflow" --branch "$BRANCH" --limit 1 --json databaseId,status,conclusion 2>/dev/null || echo "[]")

    local status conclusion run_id
    status=$(echo "$result" | jq -r '.[0].status // "unknown"')
    conclusion=$(echo "$result" | jq -r '.[0].conclusion // "unknown"')
    run_id=$(echo "$result" | jq -r '.[0].databaseId // "unknown"')

    if [[ "$run_id" != "unknown" && "$run_id" != "$prev_run_id" && "$status" == "completed" ]]; then
      echo "${conclusion}:${run_id}"
      return
    fi

    sleep 30
    elapsed=$((elapsed + 30))
  done

  echo "timeout:unknown"
}

ci_fast=$(latest_completed_for_sha "ci.yml" || true)
audit_fast=$(latest_completed_for_sha "audit.yml" || true)

if [[ -n "$ci_fast" && "${ci_fast%%:*}" == "success" ]]; then
  ci_fid="${ci_fast#success:}"
  if [[ -n "$audit_fast" && "${audit_fast%%:*}" == "success" ]]; then
    audit_fid="${audit_fast#success:}"
    echo "[fix-ci] CI and audit already green for HEAD ${LOCAL_SHA}; skipping trigger."
    echo ""
    echo "FIX_CI_COMPLETE"
    echo "BRANCH=${BRANCH}"
    echo "LOCAL_HEAD=${LOCAL_SHA}"
    echo "CI_RUN_URL=$(gh_cmd run view "$ci_fid" --json url --jq '.url')"
    echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_fid" --json url --jq '.url')"
    exit 0
  fi
fi

# Fast path: CI already passed but audit hasn't run yet — only trigger audit.yml
if [[ -n "$ci_fast" && "${ci_fast%%:*}" == "success" && -z "$audit_fast" ]]; then
  ci_fid="${ci_fast#success:}"
  echo "[fix-ci] CI already green for HEAD ${LOCAL_SHA}; only audit missing — triggering audit.yml only."
  echo "CI_RUN_URL=$(gh_cmd run view "$ci_fid" --json url --jq '.url')"

  audit_prev_run_id=$(gh_cmd run list --workflow="audit.yml" --branch "$BRANCH" --limit 1 --json databaseId 2>/dev/null \
    | jq -r '.[0].databaseId // "0"')

  gh_cmd workflow run audit.yml --ref "$BRANCH" 2>/dev/null || {
    echo "[fix-ci] ERROR: Failed to trigger audit.yml" >&2
    exit 1
  }

  echo "[fix-ci] Waiting for audit..."
  audit_result=$(wait_for_workflow "audit.yml" "$audit_prev_run_id")
  audit_conclusion="${audit_result%%:*}"
  audit_run_id="${audit_result##*:}"
  echo "[fix-ci] Audit result: ${audit_conclusion} (run ${audit_run_id})"

  if [[ "$audit_conclusion" == "success" ]]; then
    echo ""
    echo "FIX_CI_COMPLETE"
    echo "BRANCH=${BRANCH}"
    echo "LOCAL_HEAD=${LOCAL_SHA}"
    echo "CI_RUN_URL=$(gh_cmd run view "$ci_fid" --json url --jq '.url')"
    echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_run_id" --json url --jq '.url')"
    exit 0
  fi

  echo ""
  echo "AUDIT_FAILED"
  echo "AUDIT_RUN_ID=${audit_run_id}"
  echo "AUDIT_CONCLUSION=${audit_conclusion}"
  echo "[fix-ci] Fetching audit logs..."
  if [[ "$audit_run_id" != "unknown" ]]; then
    gh_cmd run view "$audit_run_id" --log-failed 2>/dev/null > "$AUDIT_LOGS_FILE" || true
    if [[ "$audit_conclusion" == "failure" ]]; then
      gh_cmd run view "$audit_run_id" --log 2>/dev/null >> "$AUDIT_LOGS_FILE" || true
    fi
  else
    echo "[fix-ci] Audit timed out waiting for completion. The workflow may still be running on GitHub." > "$AUDIT_LOGS_FILE"
  fi
  echo "AUDIT_LOGS_FILE=${AUDIT_LOGS_FILE}"
  if [[ -n "$audit_run_id" && "$audit_run_id" != "unknown" ]]; then
    echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_run_id" --json url --jq '.url')"
  fi
  echo "AGENT_NEEDED"
  exit 102
fi

if [[ -n "$ci_fast" && "${ci_fast%%:*}" == "failed" ]]; then
  # rest: failed:id:conclusion
  rest="${ci_fast#failed:}"
  ci_run_id="${rest%%:*}"
  ci_conclusion="${rest##*:}"
  echo "[fix-ci] Latest CI for HEAD ${LOCAL_SHA} already completed with ${ci_conclusion} (run ${ci_run_id})."
  echo ""
  echo "CI_FAILED"
  echo "CI_RUN_ID=${ci_run_id}"
  echo "CI_CONCLUSION=${ci_conclusion}"
  gh_cmd run view "$ci_run_id" --log-failed 2>/dev/null > "$CI_LOGS_FILE" || true
  gh_cmd run view "$ci_run_id" --log 2>/dev/null >> "$CI_LOGS_FILE" || true
  echo "CI_LOGS_FILE=${CI_LOGS_FILE}"
  echo "CI_RUN_URL=$(gh_cmd run view "$ci_run_id" --json url --jq '.url')"
  echo "AGENT_NEEDED"
  exit 101
fi

if [[ -n "$ci_fast" && "${ci_fast%%:*}" == "success" && -n "$audit_fast" && "${audit_fast%%:*}" == "failed" ]]; then
  rest="${audit_fast#failed:}"
  audit_run_id="${rest%%:*}"
  audit_conclusion="${rest##*:}"
  echo "[fix-ci] Latest audit for HEAD ${LOCAL_SHA} already completed with ${audit_conclusion} (run ${audit_run_id})."
  echo ""
  echo "AUDIT_FAILED"
  echo "AUDIT_RUN_ID=${audit_run_id}"
  echo "AUDIT_CONCLUSION=${audit_conclusion}"
  gh_cmd run view "$audit_run_id" --log-failed 2>/dev/null > "$AUDIT_LOGS_FILE" || true
  gh_cmd run view "$audit_run_id" --log 2>/dev/null >> "$AUDIT_LOGS_FILE" || true
  echo "AUDIT_LOGS_FILE=${AUDIT_LOGS_FILE}"
  echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_run_id" --json url --jq '.url')"
  echo "AGENT_NEEDED"
  exit 102
fi

get_latest_run_id() {
  local workflow="$1"
  gh_cmd run list --workflow="$workflow" --branch "$BRANCH" --limit 1 --json databaseId 2>/dev/null \
    | jq -r '.[0].databaseId // "0"'
}

ci_prev_run_id=$(get_latest_run_id "ci.yml")
audit_prev_run_id=$(get_latest_run_id "audit.yml")

echo "[fix-ci] Triggering ci.yml on ${BRANCH}..."
gh_cmd workflow run ci.yml --ref "$BRANCH" 2>/dev/null || {
  echo "[fix-ci] ERROR: Failed to trigger ci.yml" >&2
  exit 1
}

echo "[fix-ci] Triggering audit.yml on ${BRANCH}..."
gh_cmd workflow run audit.yml --ref "$BRANCH" 2>/dev/null || {
  echo "[fix-ci] ERROR: Failed to trigger audit.yml" >&2
  exit 1
}

echo "[fix-ci] Waiting for CI..."
ci_result=$(wait_for_workflow "ci.yml" "$ci_prev_run_id")
ci_conclusion="${ci_result%%:*}"
ci_run_id="${ci_result##*:}"
echo "[fix-ci] CI result: ${ci_conclusion} (run ${ci_run_id})"

if [[ "$ci_conclusion" != "success" ]]; then
  echo ""
  echo "CI_FAILED"
  echo "CI_RUN_ID=${ci_run_id}"
  echo "CI_CONCLUSION=${ci_conclusion}"
  echo "[fix-ci] Fetching CI logs..."
  if [[ "$ci_run_id" != "unknown" ]]; then
    gh_cmd run view "$ci_run_id" --log-failed 2>/dev/null > "$CI_LOGS_FILE" || true
    if [[ "$ci_conclusion" == "failure" ]]; then
      gh_cmd run view "$ci_run_id" --log 2>/dev/null >> "$CI_LOGS_FILE" || true
    fi
  else
    echo "[fix-ci] CI timed out waiting for completion. The workflow may still be running on GitHub." > "$CI_LOGS_FILE"
  fi
  echo "CI_LOGS_FILE=${CI_LOGS_FILE}"
  if [[ -n "$ci_run_id" && "$ci_run_id" != "unknown" ]]; then
    echo "CI_RUN_URL=$(gh_cmd run view "$ci_run_id" --json url --jq '.url')"
  fi
  echo "AGENT_NEEDED"
  exit 101
fi

echo "[fix-ci] Waiting for audit..."
audit_result=$(wait_for_workflow "audit.yml" "$audit_prev_run_id")
audit_conclusion="${audit_result%%:*}"
audit_run_id="${audit_result##*:}"
echo "[fix-ci] Audit result: ${audit_conclusion} (run ${audit_run_id})"

if [[ "$audit_conclusion" != "success" ]]; then
  echo ""
  echo "AUDIT_FAILED"
  echo "AUDIT_RUN_ID=${audit_run_id}"
  echo "AUDIT_CONCLUSION=${audit_conclusion}"
  echo "[fix-ci] Fetching audit logs..."
  if [[ "$audit_run_id" != "unknown" ]]; then
    gh_cmd run view "$audit_run_id" --log-failed 2>/dev/null > "$AUDIT_LOGS_FILE" || true
    if [[ "$audit_conclusion" == "failure" ]]; then
      gh_cmd run view "$audit_run_id" --log 2>/dev/null >> "$AUDIT_LOGS_FILE" || true
    fi
  else
    echo "[fix-ci] Audit timed out waiting for completion. The workflow may still be running on GitHub." > "$AUDIT_LOGS_FILE"
  fi
  echo "AUDIT_LOGS_FILE=${AUDIT_LOGS_FILE}"
  if [[ -n "$audit_run_id" && "$audit_run_id" != "unknown" ]]; then
    echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_run_id" --json url --jq '.url')"
  fi
  echo "AGENT_NEEDED"
  exit 102
fi

echo ""
echo "FIX_CI_COMPLETE"
echo "BRANCH=${BRANCH}"
echo "LOCAL_HEAD=$(git rev-parse HEAD)"
if [[ -n "$ci_run_id" && "$ci_run_id" != "unknown" ]]; then
  echo "CI_RUN_URL=$(gh_cmd run view "$ci_run_id" --json url --jq '.url')"
fi
if [[ -n "$audit_run_id" && "$audit_run_id" != "unknown" ]]; then
  echo "AUDIT_RUN_URL=$(gh_cmd run view "$audit_run_id" --json url --jq '.url')"
fi
