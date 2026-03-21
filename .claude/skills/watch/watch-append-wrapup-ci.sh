#!/usr/bin/env bash
# Append CI/audit remediation notes to the existing watch wrap-up PR comment.
#
# Usage: bash watch-append-wrapup-ci.sh --work-dir <WORK_DIR>
#
# Reads action-log.json for entries of type ci_remediation, audit_remediation,
# and ci_fix_exhausted that have not yet been appended (tracks count in work dir).

set -euo pipefail

WORK_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --work-dir) WORK_DIR="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$WORK_DIR" || ! -d "$WORK_DIR" ]]; then
  echo "ERROR: --work-dir must point to an existing watch session directory." >&2
  exit 1
fi

ACTION_LOG_FILE="$WORK_DIR/action-log.json"
WRAPUP_COMMENT_ID_FILE="$WORK_DIR/wrapup-comment-id"
PR_NUMBER_FILE="$WORK_DIR/pr-number.txt"
APPENDED_COUNT_FILE="$WORK_DIR/wrapup-ci-remediations-appended-count"

OWNER="supersuit-tech"
REPO="permission-slip"
export GH_HOST="${GH_HOST:-github.com}"
export GH_REPO="${GH_REPO:-${OWNER}/${REPO}}"

gh_api() {
  GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh api "$@"
}

if [[ ! -f "$ACTION_LOG_FILE" ]]; then
  echo "[append-wrapup-ci] No action log; nothing to append."
  exit 0
fi

PR_NUMBER=$(cat "$PR_NUMBER_FILE" 2>/dev/null || echo "")
if [[ -z "$PR_NUMBER" ]]; then
  echo "[append-wrapup-ci] ERROR: missing pr-number.txt in work dir." >&2
  exit 1
fi

COMMENT_ID=$(cat "$WRAPUP_COMMENT_ID_FILE" 2>/dev/null | tr -d '[:space:]' || echo "")
stored=0
if [[ -f "$APPENDED_COUNT_FILE" ]]; then
  stored=$(tr -d '[:space:]' < "$APPENDED_COUNT_FILE" || echo "0")
fi
[[ -z "$stored" || ! "$stored" =~ ^[0-9]+$ ]] && stored=0

remediations_json=$(jq '[.[] | select(.type == "ci_remediation" or .type == "audit_remediation" or .type == "ci_fix_exhausted")]' "$ACTION_LOG_FILE")
total=$(echo "$remediations_json" | jq 'length')

if [[ "$total" -le "$stored" ]]; then
  echo "[append-wrapup-ci] No new remediation entries to append (${stored}/${total})."
  exit 0
fi

new_slice=$(echo "$remediations_json" | jq ".[$stored:]")

append_md=$(echo "$new_slice" | jq -r '
  map(
    (if (.commit != null and .commit != "") then " (`\(.commit)`)" else "" end) as $c |
    if .type == "ci_fix_exhausted" then
      "- **\(.workflow // "workflow")** — stopped: \(.detail)"
    else
      "- **\(.workflow // "workflow")** — conclusion `\(.conclusion // "?")` — \(.detail)\($c)"
    end
  ) | join("\n")
')

block="## 🔧 CI / audit remediation

${append_md}"

posted=false
if [[ -z "$COMMENT_ID" ]]; then
  echo "[append-wrapup-ci] No wrap-up comment id; posting remediation as a new PR comment."
  if gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" -f body="$block" > /dev/null 2>&1; then
    posted=true
  fi
else
  current_body=$(gh_api "/repos/${OWNER}/${REPO}/issues/comments/${COMMENT_ID}" --jq '.body' 2>/dev/null || echo "")
  if [[ -z "$current_body" ]]; then
    echo "[append-wrapup-ci] Could not fetch comment ${COMMENT_ID}; posting new comment instead."
    if gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" -f body="$block" > /dev/null 2>&1; then
      posted=true
    fi
  else
    new_body="${current_body}

${block}"
    if gh_api "/repos/${OWNER}/${REPO}/issues/comments/${COMMENT_ID}" -X PATCH -f body="$new_body" > /dev/null 2>&1; then
      posted=true
    else
      echo "[append-wrapup-ci] PATCH failed; posting remediation as a new PR comment."
      if gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" -f body="$block" > /dev/null 2>&1; then
        posted=true
      fi
    fi
  fi
fi

if [[ "$posted" != "true" ]]; then
  echo "[append-wrapup-ci] ERROR: Could not update or post remediation on PR #${PR_NUMBER}." >&2
  exit 1
fi

echo "$total" > "$APPENDED_COUNT_FILE"
echo "[append-wrapup-ci] Appended $((total - stored)) remediation block(s). Total logged: ${total}."
