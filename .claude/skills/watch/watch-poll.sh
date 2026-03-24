#!/usr/bin/env bash
# watch-poll.sh — Deterministic polling loop for the /watch skill.
# Handles all mechanical bookkeeping (fetching, deduplication, idle timeout)
# and only invokes the Claude agent when there's actionable work.
#
# Usage: bash watch-poll.sh [PR_URL] [--no-automerge] [--work-dir <path>] [--max-turns <N>]
#
# Options:
#   --no-automerge    Disable auto-merge (enabled by default)
#   --work-dir <path> Reuse an existing work directory (preserves state
#                     across invocations: last-seen IDs, action log, cache)
#   --max-turns <N>   Maximum number of agent turns before ending the session.
#                     Each turn = one AGENT_NEEDED exit (one round of AI work).
#                     The turn counter increments on each AGENT_NEEDED; wrap-up and
#                     IDLE_TIMEOUT happen on the *next* run when turns >= N (or on
#                     idle timeout). Re-invoke with the same --work-dir after each
#                     AGENT_NEEDED — stopping after the first AGENT_NEEDED skips
#                     wrap-up and watch-post.sh (CI/merge).
#
# Environment:
#   GH_HOST, GH_REPO — set by caller (defaults provided below)
#
# Outputs (in WORK_DIR):
#   work-items.json  — new items for the agent to process
#   action-log.json  — agent appends its actions here; used for wrap-up
#   wrap-up.md       — generated session summary posted to the PR

set -euo pipefail

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------
PR_URL=""
AUTO_MERGE=true
EXISTING_WORK_DIR=""
MAX_TURNS=0  # 0 = unlimited

# Parse arguments — PR URL is optional (auto-detected from current branch if omitted)
while [[ $# -gt 0 ]]; do
  case "$1" in
    --automerge) AUTO_MERGE=true; shift ;;  # kept for backwards compat
    --no-automerge) AUTO_MERGE=false; shift ;;
    --work-dir) EXISTING_WORK_DIR="$2"; shift 2 ;;
    --max-turns) MAX_TURNS="$2"; shift 2 ;;
    https://*) PR_URL="$1"; shift ;;
    *) shift ;;
  esac
done

# Auto-detect PR URL from current branch if not provided
if [[ -z "$PR_URL" ]]; then
  echo "[setup] No PR URL provided, detecting from current branch..."
  PR_URL=$(GH_HOST="${GH_HOST:-github.com}" GH_REPO="${GH_REPO:-supersuit-tech/permission-slip}" gh pr view --json url --jq '.url' 2>/dev/null || echo "")
  if [[ -z "$PR_URL" ]]; then
    echo "ERROR: No PR URL provided and could not detect a PR for the current branch." >&2
    echo "Either pass a PR URL or run this from a branch with an open PR." >&2
    exit 1
  fi
  echo "[setup] Detected PR: ${PR_URL}"
fi

PR_NUMBER=$(echo "$PR_URL" | grep -oP '/pull/\K[0-9]+')
if [[ -z "$PR_NUMBER" ]]; then
  echo "ERROR: Could not parse PR number from URL: $PR_URL" >&2
  exit 1
fi

OWNER="supersuit-tech"
REPO="permission-slip"
export GH_HOST="${GH_HOST:-github.com}"
export GH_REPO="${GH_REPO:-${OWNER}/${REPO}}"
BRANCH=$(git branch --show-current)

# ---------------------------------------------------------------------------
# If the PR is a draft, mark it as ready for review
# ---------------------------------------------------------------------------
PR_META=$(GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}" --jq '{draft: .draft, node_id: .node_id}' 2>/dev/null || echo '{"draft":false}')
IS_DRAFT=$(echo "$PR_META" | jq -r '.draft')
if [[ "$IS_DRAFT" == "true" ]]; then
  echo "[setup] PR is a draft — marking as ready for review..."
  PR_NODE_ID=$(echo "$PR_META" | jq -r '.node_id')
  if [[ -n "$PR_NODE_ID" && "$PR_NODE_ID" != "null" ]]; then
    GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh api graphql -f query="
      mutation {
        markPullRequestReadyForReview(input: {pullRequestId: \"${PR_NODE_ID}\"}) {
          pullRequest {
            isDraft
          }
        }
      }" > /dev/null 2>&1 && echo "[setup] PR marked as ready for review." || echo "[setup] WARNING: Failed to mark PR as ready for review." >&2
  else
    echo "[setup] WARNING: Could not fetch PR node ID to mark as ready." >&2
  fi
fi

# ---------------------------------------------------------------------------
# Working directory for session state
# ---------------------------------------------------------------------------
if [[ -n "$EXISTING_WORK_DIR" && -d "$EXISTING_WORK_DIR" ]]; then
  WORK_DIR="$EXISTING_WORK_DIR"
else
  WORK_DIR=$(mktemp -d "/tmp/watch-session-XXXXXX")
fi
# No EXIT trap — the caller (SKILL.md) is responsible for cleanup
# since work-items.json and action-log.json must survive across script exits.

# State files
LAST_REVIEW_ID_FILE="$WORK_DIR/last-review-id"
LAST_REVIEW_COMMENT_ID_FILE="$WORK_DIR/last-review-comment-id"
LAST_ISSUE_COMMENT_ID_FILE="$WORK_DIR/last-issue-comment-id"
WORK_ITEMS_FILE="$WORK_DIR/work-items.json"
ACTION_LOG_FILE="$WORK_DIR/action-log.json"
CHECKLIST_CACHE_FILE="$WORK_DIR/checklist-cache.txt"
TURNS_FILE="$WORK_DIR/turns-count"
WRAPUP_COMMENT_ID_FILE="$WORK_DIR/wrapup-comment-id"
PR_NUMBER_FILE="$WORK_DIR/pr-number.txt"

# Only initialize state files if they don't already exist (fresh session)
[[ -f "$LAST_REVIEW_ID_FILE" ]] || echo "0" > "$LAST_REVIEW_ID_FILE"
[[ -f "$LAST_REVIEW_COMMENT_ID_FILE" ]] || echo "0" > "$LAST_REVIEW_COMMENT_ID_FILE"
[[ -f "$LAST_ISSUE_COMMENT_ID_FILE" ]] || echo "0" > "$LAST_ISSUE_COMMENT_ID_FILE"
[[ -f "$ACTION_LOG_FILE" ]] || echo "[]" > "$ACTION_LOG_FILE"
[[ -f "$TURNS_FILE" ]] || echo "0" > "$TURNS_FILE"
echo "$PR_NUMBER" > "$PR_NUMBER_FILE"

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
POLL_INTERVAL=60
MAX_IDLE_CYCLES=6
IDLE_COUNTER=0
CYCLE=0

# ---------------------------------------------------------------------------
# Helper: run gh with correct env
# ---------------------------------------------------------------------------
gh_api() {
  GH_HOST="${GH_HOST}" GH_REPO="${GH_REPO}" gh api "$@"
}

# ---------------------------------------------------------------------------
# Fetch new comments from all three endpoints
# Returns JSON array of work items
# ---------------------------------------------------------------------------
fetch_new_comments() {
  local last_review_id last_rc_id last_ic_id
  last_review_id=$(cat "$LAST_REVIEW_ID_FILE")
  last_rc_id=$(cat "$LAST_REVIEW_COMMENT_ID_FILE")
  last_ic_id=$(cat "$LAST_ISSUE_COMMENT_ID_FILE")

  local items="[]"

  # --- PR Reviews ---
  local reviews
  reviews=$(gh_api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}/reviews?per_page=100" 2>/dev/null || echo "[]")

  local new_reviews
  new_reviews=$(echo "$reviews" | jq -r --argjson last "$last_review_id" '
    [.[] | select(
      .id > $last
      and .body != null and .body != ""
      and .state != "PENDING"
    )] | sort_by(.id)')

  local max_review_id
  max_review_id=$(echo "$new_reviews" | jq -r '[.[].id] | max // 0')
  if [[ "$max_review_id" != "0" && "$max_review_id" != "null" ]]; then
    echo "$max_review_id" > "$LAST_REVIEW_ID_FILE"
  fi

  # Map reviews to work items
  items=$(echo "$items" | jq --argjson revs "$new_reviews" '
    . + [$revs[] | {
      type: "review",
      id: .id,
      author: .user.login,
      state: .state,
      body: .body,
      submitted_at: .submitted_at,
      node_id: .node_id
    }]')

  # --- Review Comments (inline on diffs) ---
  local review_comments
  review_comments=$(gh_api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}/comments?per_page=100" 2>/dev/null || echo "[]")

  local new_rcs
  new_rcs=$(echo "$review_comments" | jq -r --argjson last "$last_rc_id" '
    [.[] | select(.id > $last and .in_reply_to_id == null)] | sort_by(.id)')

  local max_rc_id
  max_rc_id=$(echo "$new_rcs" | jq -r '[.[].id] | max // 0')
  if [[ "$max_rc_id" != "0" && "$max_rc_id" != "null" ]]; then
    echo "$max_rc_id" > "$LAST_REVIEW_COMMENT_ID_FILE"
  fi

  items=$(echo "$items" | jq --argjson rcs "$new_rcs" '
    . + [$rcs[] | {
      type: "review_comment",
      id: .id,
      author: .user.login,
      body: .body,
      path: .path,
      line: (.line // .original_line),
      diff_hunk: .diff_hunk,
      created_at: .created_at,
      node_id: .node_id,
      in_reply_to_id: .in_reply_to_id
    }]')

  # --- Issue Comments (general PR conversation) ---
  local issue_comments
  issue_comments=$(gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments?per_page=100" 2>/dev/null || echo "[]")

  local new_ics
  new_ics=$(echo "$issue_comments" | jq -r --argjson last "$last_ic_id" '
    [.[] | select(.id > $last)] | sort_by(.id)')

  local max_ic_id
  max_ic_id=$(echo "$new_ics" | jq -r '[.[].id] | max // 0')
  if [[ "$max_ic_id" != "0" && "$max_ic_id" != "null" ]]; then
    echo "$max_ic_id" > "$LAST_ISSUE_COMMENT_ID_FILE"
  fi

  items=$(echo "$items" | jq --argjson ics "$new_ics" '
    . + [$ics[] | {
      type: "issue_comment",
      id: .id,
      author: .user.login,
      body: .body,
      created_at: .created_at,
      node_id: .node_id
    }]')

  echo "$items"
}

# ---------------------------------------------------------------------------
# Merge from main, detect conflicts
# Returns: "clean", "updated", or "conflict"
# ---------------------------------------------------------------------------
merge_from_main() {
  git fetch origin main 2>/dev/null

  local merge_output
  if merge_output=$(git merge origin/main --no-edit 2>&1); then
    if echo "$merge_output" | grep -q "Already up to date"; then
      echo "clean"
    else
      echo "updated"
    fi
  else
    # Merge failed — conflicts
    echo "conflict"
  fi
}

# ---------------------------------------------------------------------------
# Get conflicted files (when merge status is "conflict")
# ---------------------------------------------------------------------------
get_conflict_files() {
  git diff --name-only --diff-filter=U 2>/dev/null || true
}

# ---------------------------------------------------------------------------
# Parse PR body for unchecked Claude Code checklist items
# ---------------------------------------------------------------------------
fetch_checklist_items() {
  local pr_body
  pr_body=$(gh_api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}" --jq '.body' 2>/dev/null || echo "")

  if [[ -z "$pr_body" ]]; then
    echo "[]"
    return
  fi

  # Save full body for later use
  echo "$pr_body" > "$WORK_DIR/pr-body.txt"

  # Extract unchecked items under Claude Code section
  # We look for lines matching "- [ ] <text>" that appear after a "### Claude Code" heading
  # and before the next heading of equal or higher level
  local in_claude_section=false
  local items="[]"

  while IFS= read -r line; do
    # Detect section headers
    if [[ "$line" =~ ^###[[:space:]] ]]; then
      if [[ "$line" =~ [Cc]laude[[:space:]]*[Cc]ode ]] || [[ "$line" =~ [Aa]utomated ]]; then
        in_claude_section=true
        continue
      else
        in_claude_section=false
        continue
      fi
    fi
    # Higher-level heading exits any section
    if [[ "$line" =~ ^##[[:space:]] ]] && [[ ! "$line" =~ ^###[[:space:]] ]]; then
      in_claude_section=false
      continue
    fi

    # Collect unchecked items in Claude Code section
    if $in_claude_section && [[ "$line" =~ ^[[:space:]]*-[[:space:]]\[[[:space:]]\][[:space:]](.+) ]]; then
      local item_text="${BASH_REMATCH[1]}"
      items=$(echo "$items" | jq --arg text "$item_text" '. + [{type: "checklist", text: $text}]')
    fi
  done <<< "$pr_body"

  # Compare with cache to only return new/unprocessed items
  local cached=""
  if [[ -f "$CHECKLIST_CACHE_FILE" ]]; then
    cached=$(cat "$CHECKLIST_CACHE_FILE")
  fi

  # Filter out already-processed items using jq (avoids pipe subshell issues)
  if [[ -z "$cached" ]]; then
    echo "$items"
  else
    # Build a jq filter that excludes exact matches against cached lines
    echo "$items" | jq --arg cache "$cached" '
      ($cache | split("\n") | map(select(length > 0))) as $done |
      [.[] | select(.text as $t | $done | index($t) | not)]'
  fi
}

# ---------------------------------------------------------------------------
# Mark a checklist item as done in the PR body
# ---------------------------------------------------------------------------
check_off_item() {
  local item_text="$1"
  local current_body
  current_body=$(gh_api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}" --jq '.body' 2>/dev/null || echo "")

  # Use python3 for literal string replacement (avoids sed regex escaping issues)
  local updated_body
  updated_body=$(python3 -c "
import sys
body = sys.stdin.read()
old = '- [ ] ' + sys.argv[1]
new = '- [x] ' + sys.argv[1]
print(body.replace(old, new, 1), end='')
" "$item_text" <<< "$current_body")

  gh_api "/repos/${OWNER}/${REPO}/pulls/${PR_NUMBER}" -X PATCH -f body="$updated_body" > /dev/null 2>&1

  # Update cache
  echo "$item_text" >> "$CHECKLIST_CACHE_FILE"
}

# ---------------------------------------------------------------------------
# Build work-items.json for the agent
# ---------------------------------------------------------------------------
build_work_items() {
  local comments="$1"
  local merge_status="$2"
  local conflict_files="$3"
  local checklist_items="$4"

  local work_items
  work_items=$(jq -n \
    --argjson comments "$comments" \
    --arg merge_status "$merge_status" \
    --arg conflict_files "$conflict_files" \
    --argjson checklist "$checklist_items" \
    --arg pr_number "$PR_NUMBER" \
    --arg pr_url "$PR_URL" \
    --arg branch "$BRANCH" \
    --argjson cycle "$CYCLE" \
    '{
      pr_number: $pr_number,
      pr_url: $pr_url,
      branch: $branch,
      cycle: $cycle,
      comments: $comments,
      merge_status: $merge_status,
      conflict_files: ($conflict_files | split("\n") | map(select(. != ""))),
      checklist_items: $checklist
    }')

  echo "$work_items" > "$WORK_ITEMS_FILE"
}

# ---------------------------------------------------------------------------
# Generate wrap-up comment from action log
# ---------------------------------------------------------------------------
generate_wrapup() {
  local total_cycles="$1"
  local action_log
  action_log=$(cat "$ACTION_LOG_FILE")

  local changes conflicts checklist_done checklist_skipped
  local implemented declined judgments open_questions

  changes=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "change")] |
    if length == 0 then "No changes made during this session."
    else map("- **`" + .description + "`** (`" + .commit + "`) — " + .detail) | join("\n")
    end')

  conflicts=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "conflict_resolution")] |
    if length == 0 then ""
    else "### Merge Conflict Resolutions\n" + (map("- **`" + .file + "`** — " + .detail + " (`" + .commit + "`)") | join("\n"))
    end')

  checklist_done=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "checklist_done")] |
    if length == 0 then ""
    else map("- ✅ **`" + .text + "`** — " + .detail + " (`" + .commit + "`)") | join("\n")
    end')

  checklist_skipped=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "checklist_skipped")] |
    if length == 0 then ""
    else map("- ⏭️ **`" + .text + "`** — skipped (" + .reason + ")") | join("\n")
    end')

  implemented=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "implemented")] |
    if length == 0 then "No review comments acted on."
    else map("- " + .author + " asked for " + .request + " → implemented in `" + .commit + "`") | join("\n")
    end')

  declined=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "declined")] |
    if length == 0 then ""
    else map("- " + .author + " suggested " + .request + " → declined because " + .reason) | join("\n")
    end')

  judgments=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "judgment")] |
    if length == 0 then ""
    else map("- " + .description + " → chose " + .choice + " because " + .reason) | join("\n")
    end')

  open_questions=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "open_question")] |
    if length == 0 then ""
    else map("- " + .description) | join("\n")
    end')

  local ci_audit_fixes ci_fix_exhausted
  ci_audit_fixes=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "ci_remediation" or .type == "audit_remediation")] |
    if length == 0 then ""
    else "### CI / audit fixes (logged)\n" + (map(
        "- **" + .workflow + "** (" + .conclusion + ")"
        + (if .commit != null and .commit != "" then " — `" + .commit + "`" else "" end)
        + " — " + .detail
      ) | join("\n"))
    end')

  ci_fix_exhausted=$(echo "$action_log" | jq -r '
    [.[] | select(.type == "ci_fix_exhausted")] |
    if length == 0 then ""
    else "### CI / audit remediation stopped\n" + (map(
        "- **" + .workflow + "** — " + .detail
      ) | join("\n"))
    end')

  local comment_count
  comment_count=$(echo "$action_log" | jq '[.[] | select(.type == "implemented" or .type == "declined")] | length')

  # Build the markdown
  local wrapup="## 🤖 Watch Session Summary

### Changes Made
${changes}
"

  if [[ -n "$conflicts" ]]; then
    wrapup="${wrapup}
${conflicts}
"
  fi

  local has_checklist=false
  if [[ -n "$checklist_done" || -n "$checklist_skipped" ]]; then
    has_checklist=true
    wrapup="${wrapup}
### PR Checklist Items
"
    [[ -n "$checklist_done" ]] && wrapup="${wrapup}${checklist_done}
"
    [[ -n "$checklist_skipped" ]] && wrapup="${wrapup}${checklist_skipped}
"
  fi

  wrapup="${wrapup}
### Decision Log

#### ✅ Implemented
${implemented}
"

  if [[ -n "$declined" ]]; then
    wrapup="${wrapup}
#### ❌ Declined
${declined}
"
  fi

  if [[ -n "$judgments" ]]; then
    wrapup="${wrapup}
#### ⚖️ Judgment Calls
${judgments}
"
  fi

  if [[ -n "$open_questions" ]]; then
    wrapup="${wrapup}
#### ❓ Open Questions
${open_questions}
"
  fi

  if [[ -n "$ci_audit_fixes" ]]; then
    wrapup="${wrapup}
${ci_audit_fixes}
"
  fi

  if [[ -n "$ci_fix_exhausted" ]]; then
    wrapup="${wrapup}
${ci_fix_exhausted}
"
  fi

  wrapup="${wrapup}
---
*Watch session ended after $((MAX_IDLE_CYCLES * POLL_INTERVAL / 60)) minutes of inactivity. Processed ${comment_count} comments across ${total_cycles} poll cycles.*"

  echo "$wrapup" > "$WORK_DIR/wrap-up.md"
  echo "$wrapup"
}

# ---------------------------------------------------------------------------
# Post wrap-up comment to PR
# ---------------------------------------------------------------------------
post_wrapup_comment() {
  local body="$1"
  local resp
  if ! resp=$(gh_api "/repos/${OWNER}/${REPO}/issues/${PR_NUMBER}/comments" -f body="$body" 2>/dev/null); then
    echo "[wrap-up] WARNING: Failed to post wrap-up comment" >&2
    echo -n "" > "$WRAPUP_COMMENT_ID_FILE"
    return 1
  fi
  echo "$resp" | jq -r '.id // empty' | tr -d '\n' > "$WRAPUP_COMMENT_ID_FILE"
}

# ---------------------------------------------------------------------------
# Main output: print session context for the agent to use
# ---------------------------------------------------------------------------
print_session_context() {
  cat <<EOF
{
  "pr_url": "${PR_URL}",
  "pr_number": "${PR_NUMBER}",
  "branch": "${BRANCH}",
  "auto_merge": ${AUTO_MERGE},
  "work_dir": "${WORK_DIR}",
  "work_items_file": "${WORK_ITEMS_FILE}",
  "action_log_file": "${ACTION_LOG_FILE}",
  "wrapup_comment_id_file": "${WRAPUP_COMMENT_ID_FILE}",
  "pr_number_file": "${PR_NUMBER_FILE}"
}
EOF
}

# ---------------------------------------------------------------------------
# MAIN LOOP
# ---------------------------------------------------------------------------
CURRENT_TURNS=$(cat "$TURNS_FILE")

echo "=== Watch session started ==="
echo "PR: ${PR_URL} (#${PR_NUMBER})"
echo "Branch: ${BRANCH}"
echo "Auto-merge: ${AUTO_MERGE}"
echo "Max turns: ${MAX_TURNS} (0=unlimited)"
echo "Turns used: ${CURRENT_TURNS}"
echo "Work dir: ${WORK_DIR}"
echo ""

# --- Check turn limit before doing anything ---
if [[ "$MAX_TURNS" -gt 0 && "$CURRENT_TURNS" -ge "$MAX_TURNS" ]]; then
  echo "=== Turn limit reached (${CURRENT_TURNS}/${MAX_TURNS}) ==="

  # Print session context for the calling agent
  print_session_context

  # Generate and post wrap-up
  echo "[wrap-up] Generating session summary..."
  wrapup=$(generate_wrapup "$CURRENT_TURNS")
  echo "[wrap-up] Posting to PR..."
  post_wrapup_comment "$wrapup" || true

  echo "IDLE_TIMEOUT"
  echo "REASON=turn limit reached (${CURRENT_TURNS}/${MAX_TURNS})"
  echo "WRAPUP_POSTED=true"
  echo "AUTO_MERGE=${AUTO_MERGE}"
  echo "TOTAL_CYCLES=${CURRENT_TURNS}"

  exit 0
fi

# Print session context for the calling agent
print_session_context

# --- Pre-poll merge from main ---
echo "[pre-poll] Merging from main..."
pre_merge=$(merge_from_main)
if [[ "$pre_merge" == "conflict" ]]; then
  conflict_files=$(get_conflict_files)
  echo "[pre-poll] CONFLICTS detected in: ${conflict_files}"
  # Build work items with just the conflict
  build_work_items "[]" "conflict" "$conflict_files" "[]"
  # Increment turns counter
  echo $((CURRENT_TURNS + 1)) > "$TURNS_FILE"
  echo "AGENT_NEEDED"
  echo "REASON=pre-poll merge conflict"
  echo "WORK_ITEMS_FILE=${WORK_ITEMS_FILE}"
  # Agent will be invoked by the caller; we wait for it to finish
  # then continue the loop. For now, just signal and let the skill handle it.
  exit 100  # Special exit code: agent needed before loop starts
else
  echo "[pre-poll] Merge: ${pre_merge}"
fi

# --- Polling loop ---
while true; do
  CYCLE=$((CYCLE + 1))
  echo ""
  echo "=== Poll cycle ${CYCLE} ==="

  had_activity=false

  # 1. Fetch new comments
  echo "[${CYCLE}] Fetching comments..."
  new_comments=$(fetch_new_comments)
  comment_count=$(echo "$new_comments" | jq 'length')

  if [[ "$comment_count" -gt 0 ]]; then
    echo "[${CYCLE}] Found ${comment_count} new comments/reviews"
    had_activity=true
  fi

  # 2. Merge from main
  echo "[${CYCLE}] Merging from main..."
  merge_status=$(merge_from_main)
  conflict_files=""
  if [[ "$merge_status" == "conflict" ]]; then
    conflict_files=$(get_conflict_files)
    echo "[${CYCLE}] CONFLICTS: ${conflict_files}"
    had_activity=true
  elif [[ "$merge_status" == "updated" ]]; then
    echo "[${CYCLE}] Branch updated from main"
    had_activity=true
  else
    echo "[${CYCLE}] Already up to date"
  fi

  # 3. Check PR body checklist
  echo "[${CYCLE}] Checking PR body checklist..."
  checklist_items=$(fetch_checklist_items)
  checklist_count=$(echo "$checklist_items" | jq 'length')

  if [[ "$checklist_count" -gt 0 ]]; then
    echo "[${CYCLE}] Found ${checklist_count} unchecked checklist items"
    had_activity=true
  fi

  # 4. Decide whether to invoke the agent
  if $had_activity; then
    IDLE_COUNTER=0

    build_work_items "$new_comments" "$merge_status" "$conflict_files" "$checklist_items"

    # Increment turns counter
    CURRENT_TURNS=$(cat "$TURNS_FILE")
    echo $((CURRENT_TURNS + 1)) > "$TURNS_FILE"

    echo "AGENT_NEEDED"
    echo "REASON=cycle ${CYCLE}: ${comment_count} comments, merge=${merge_status}, ${checklist_count} checklist items"
    echo "WORK_ITEMS_FILE=${WORK_ITEMS_FILE}"
    echo "ACTION_LOG_FILE=${ACTION_LOG_FILE}"

    # The caller (SKILL.md) reads these signals and invokes the agent.
    # After the agent finishes, it calls this script again with --resume
    # to continue the loop. For simplicity, we exit here and let the
    # skill orchestrate the agent invocation.
    exit 0
  else
    IDLE_COUNTER=$((IDLE_COUNTER + 1))
    echo "[${CYCLE}] No activity. Idle counter: ${IDLE_COUNTER}/${MAX_IDLE_CYCLES}"
  fi

  # 5. Check idle timeout
  if [[ $IDLE_COUNTER -ge $MAX_IDLE_CYCLES ]]; then
    echo ""
    echo "=== Idle timeout reached (${MAX_IDLE_CYCLES} cycles) ==="

    # Generate and post wrap-up
    echo "[wrap-up] Generating session summary..."
    wrapup=$(generate_wrapup "$CYCLE")
    echo "[wrap-up] Posting to PR..."
    post_wrapup_comment "$wrapup" || true

    # CI triggering is handled by watch-post.sh (Step 5 in SKILL.md)
    echo "IDLE_TIMEOUT"
    echo "WRAPUP_POSTED=true"
    echo "AUTO_MERGE=${AUTO_MERGE}"
    echo "TOTAL_CYCLES=${CYCLE}"

    exit 0
  fi

  # 6. Sleep before next poll
  echo "[${CYCLE}] Sleeping ${POLL_INTERVAL}s..."
  sleep "$POLL_INTERVAL"
done
