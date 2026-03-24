#!/usr/bin/env bash
# watch-poll-test.sh — Regression test for watch-poll.sh reliability fixes.
#
# Simulates the failure mode from issue #847:
#   "comment IDs A,B seen → AGENT_NEEDED → new C arrives → idle cycles"
#   and asserts wrap-up does NOT fire until C is handled or explicitly skipped.
#
# This test mocks the GitHub API calls and verifies the script's behavior
# by checking exit codes, output signals, and state files.
#
# Usage: bash watch-poll-test.sh
#   Exit 0 = all tests pass
#   Exit 1 = test failure (details printed to stderr)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POLL_SCRIPT="$SCRIPT_DIR/watch-poll.sh"

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# ---------------------------------------------------------------------------
# Test framework
# ---------------------------------------------------------------------------
pass() {
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo "  ✓ $1"
}

fail() {
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo "  ✗ $1" >&2
  echo "    $2" >&2
}

assert_contains() {
  local haystack="$1" needle="$2" msg="$3"
  TESTS_RUN=$((TESTS_RUN + 1))
  if echo "$haystack" | grep -qF "$needle"; then
    pass "$msg"
  else
    fail "$msg" "Expected output to contain: '$needle'"
  fi
}

assert_not_contains() {
  local haystack="$1" needle="$2" msg="$3"
  TESTS_RUN=$((TESTS_RUN + 1))
  if echo "$haystack" | grep -qF "$needle"; then
    fail "$msg" "Expected output NOT to contain: '$needle'"
  else
    pass "$msg"
  fi
}

assert_file_exists() {
  local path="$1" msg="$2"
  TESTS_RUN=$((TESTS_RUN + 1))
  if [[ -f "$path" ]]; then
    pass "$msg"
  else
    fail "$msg" "File not found: $path"
  fi
}

assert_file_not_exists() {
  local path="$1" msg="$2"
  TESTS_RUN=$((TESTS_RUN + 1))
  if [[ -f "$path" ]]; then
    fail "$msg" "File should not exist: $path"
  else
    pass "$msg"
  fi
}

assert_equals() {
  local actual="$1" expected="$2" msg="$3"
  TESTS_RUN=$((TESTS_RUN + 1))
  if [[ "$actual" == "$expected" ]]; then
    pass "$msg"
  else
    fail "$msg" "Expected '$expected', got '$actual'"
  fi
}

# ---------------------------------------------------------------------------
# Setup: create a mock gh command that returns canned responses
# ---------------------------------------------------------------------------
setup_mock_env() {
  local test_dir
  test_dir=$(mktemp -d "/tmp/watch-poll-test-XXXXXX")
  local mock_bin="$test_dir/mock-bin"
  mkdir -p "$mock_bin"

  # Mock gh command
  cat > "$mock_bin/gh" << 'MOCK_GH'
#!/usr/bin/env bash
# Mock gh for watch-poll tests.
# Reads MOCK_DIR env var for response files.
if [[ "$*" == *"pulls/"*"/reviews"* ]]; then
  cat "$MOCK_DIR/reviews-response.json" 2>/dev/null || echo "[]"
elif [[ "$*" == *"pulls/"*"/comments"* ]]; then
  cat "$MOCK_DIR/review-comments-response.json" 2>/dev/null || echo "[]"
elif [[ "$*" == *"issues/"*"/comments"* && "$1" == "api" && ! "$*" == *"-f body="* ]]; then
  cat "$MOCK_DIR/issue-comments-response.json" 2>/dev/null || echo "[]"
elif [[ "$*" == *"issues/"*"/comments"* && "$*" == *"-f body="* ]]; then
  echo '{"id": 99999}'  # wrap-up comment post
elif [[ "$*" == *"pulls/"* && "$*" == *"--jq"* && "$*" == *".body"* ]]; then
  echo ""  # empty PR body (no checklist)
elif [[ "$*" == *"pulls/"* && "$*" == *"--jq"* && "$*" == *"draft"* ]]; then
  echo '{"draft":false,"node_id":"fake"}'
elif [[ "$*" == *"pr view"* ]]; then
  echo "https://github.com/supersuit-tech/permission-slip/pull/999"
elif [[ "$*" == *"graphql"* ]]; then
  cat "$MOCK_DIR/graphql-response.json" 2>/dev/null || echo '{"data":{"repository":{"pullRequest":{"merged":false,"reviewThreads":{"nodes":[]}}}}}'
else
  echo "[]"
fi
MOCK_GH
  chmod +x "$mock_bin/gh"

  # Mock git commands (merge from main is a no-op in tests)
  cat > "$mock_bin/git" << 'MOCK_GIT'
#!/usr/bin/env bash
case "$*" in
  "fetch origin main") exit 0 ;;
  "merge origin/main --no-edit") echo "Already up to date." ;;
  "branch --show-current") echo "test-branch" ;;
  "diff --name-only --diff-filter=U") echo "" ;;
  *) /usr/bin/git "$@" ;;
esac
MOCK_GIT
  chmod +x "$mock_bin/git"

  echo "$test_dir"
}

write_mock_responses() {
  local mock_dir="$1"
  local reviews="${2:-[]}"
  local review_comments="${3:-[]}"
  local issue_comments="${4:-[]}"
  local graphql="${5:-}"

  echo "$reviews" > "$mock_dir/reviews-response.json"
  echo "$review_comments" > "$mock_dir/review-comments-response.json"
  echo "$issue_comments" > "$mock_dir/issue-comments-response.json"

  if [[ -n "$graphql" ]]; then
    echo "$graphql" > "$mock_dir/graphql-response.json"
  fi
}

# ---------------------------------------------------------------------------
# Run watch-poll.sh with mocked environment
# Returns output; exit code saved in LAST_EXIT_CODE
# ---------------------------------------------------------------------------
LAST_EXIT_CODE=0
run_poll() {
  local test_dir="$1"
  shift
  local mock_bin="$test_dir/mock-bin"

  # Override PATH to use mock commands, set MOCK_DIR for mock gh
  set +e
  local output
  output=$(
    PATH="$mock_bin:$PATH" \
    MOCK_DIR="$test_dir" \
    GH_HOST="github.com" \
    GH_REPO="supersuit-tech/permission-slip" \
    bash "$POLL_SCRIPT" "$@" 2>&1
  )
  LAST_EXIT_CODE=$?
  set -e
  echo "$output"
}

# ---------------------------------------------------------------------------
# Test 1: Deferred ID advancement — IDs not advanced until work items built
# ---------------------------------------------------------------------------
test_deferred_id_advancement() {
  echo ""
  echo "=== Test 1: Deferred ID advancement ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  # Set up initial state
  echo "0" > "$work_dir/last-review-id"
  echo "0" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo "[]" > "$work_dir/action-log.json"
  echo "0" > "$work_dir/turns-count"

  # Mock: return 2 new review comments (IDs 100, 200)
  write_mock_responses "$test_dir" "[]" '[
    {"id": 100, "user": {"login": "reviewer"}, "body": "Fix A", "path": "a.go", "line": 1, "diff_hunk": "", "created_at": "2026-01-01", "node_id": "N1", "in_reply_to_id": null},
    {"id": 200, "user": {"login": "reviewer"}, "body": "Fix B", "path": "b.go", "line": 2, "diff_hunk": "", "created_at": "2026-01-01", "node_id": "N2", "in_reply_to_id": null}
  ]' "[]"

  local output
  output=$(run_poll "$test_dir" "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "$work_dir")

  assert_contains "$output" "AGENT_NEEDED" "First run exits AGENT_NEEDED"
  assert_contains "$output" "2 comments" "Detects 2 new comments"

  # Check that last-seen IDs were committed (work items were built)
  local last_rc_id
  last_rc_id=$(cat "$work_dir/last-review-comment-id")
  assert_equals "$last_rc_id" "200" "Last review-comment ID committed to 200"

  # Check staged files are cleaned up
  assert_file_not_exists "$work_dir/.staged-review-comment-id" "Staged file cleaned up after commit"

  # Check pending items file exists
  assert_file_exists "$work_dir/pending-items.json" "Pending items file created"

  # Check work-items.json has the comments
  local work_comment_count
  work_comment_count=$(jq '.comments | length' "$work_dir/work-items.json")
  assert_equals "$work_comment_count" "2" "work-items.json has 2 comments"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Test 2: Pending queue — unprocessed items re-included on re-invocation
# ---------------------------------------------------------------------------
test_pending_queue_reprocessing() {
  echo ""
  echo "=== Test 2: Pending queue re-processing ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  # Simulate state after a previous AGENT_NEEDED exit where agent did NOT process items
  echo "200" > "$work_dir/last-review-id"
  echo "200" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo "[]" > "$work_dir/action-log.json"  # Agent didn't add anything
  echo "1" > "$work_dir/turns-count"
  echo "999" > "$work_dir/pr-number.txt"

  # Pending items from previous run (action_log_length was 0 when saved)
  echo '[
    {"type": "pending_metadata", "action_log_length": 0},
    {"type": "review_comment", "id": 100, "author": "reviewer", "body": "Fix A"},
    {"type": "review_comment", "id": 200, "author": "reviewer", "body": "Fix B"}
  ]' > "$work_dir/pending-items.json"

  # No NEW comments from API (IDs haven't advanced, but we test pending re-inclusion)
  write_mock_responses "$test_dir" "[]" "[]" "[]"

  local output
  output=$(run_poll "$test_dir" "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "$work_dir")

  assert_contains "$output" "AGENT_NEEDED" "Re-invocation exits AGENT_NEEDED for pending items"
  assert_contains "$output" "pending items from previous run" "Logs pending item detection"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Test 3: Pending queue cleared when action log grows
# ---------------------------------------------------------------------------
test_pending_queue_cleared_on_processing() {
  echo ""
  echo "=== Test 3: Pending queue cleared when agent processes items ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  echo "200" > "$work_dir/last-review-id"
  echo "200" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo "1" > "$work_dir/turns-count"
  echo "999" > "$work_dir/pr-number.txt"

  # Agent DID process: action log has entries now (was 0 at pending save time)
  echo '[{"type": "implemented", "author": "reviewer", "request": "Fix A", "commit": "abc123"}]' > "$work_dir/action-log.json"

  # Pending items from previous run (action_log_length was 0)
  echo '[
    {"type": "pending_metadata", "action_log_length": 0},
    {"type": "review_comment", "id": 100, "author": "reviewer", "body": "Fix A"}
  ]' > "$work_dir/pending-items.json"

  # No new comments from API
  write_mock_responses "$test_dir" "[]" "[]" "[]"

  # This should detect that action log grew and clear pending
  # Then with no new comments, it should enter idle cycles
  # We use max-turns to force a quick exit (turn limit already at 1 with 1 used)
  local output
  output=$(run_poll "$test_dir" "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "$work_dir" --max-turns 1)

  # Should see IDLE_TIMEOUT since turns limit reached and pending was cleared
  assert_contains "$output" "IDLE_TIMEOUT" "Exits IDLE_TIMEOUT when pending cleared and turn limit reached"
  assert_file_not_exists "$work_dir/pending-items.json" "Pending file cleaned up"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Test 4: New comments arriving after AGENT_NEEDED are picked up
# ---------------------------------------------------------------------------
test_new_comments_after_agent_needed() {
  echo ""
  echo "=== Test 4: New comments arriving after AGENT_NEEDED are detected ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  # State: previous run saw IDs up to 200 and agent processed them
  echo "0" > "$work_dir/last-review-id"
  echo "200" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo '[{"type": "implemented", "author": "reviewer", "request": "Fix B", "commit": "def456"}]' > "$work_dir/action-log.json"
  echo "1" > "$work_dir/turns-count"
  echo "999" > "$work_dir/pr-number.txt"

  # New comment C arrived (ID 300) — this is the scenario from issue #847
  write_mock_responses "$test_dir" "[]" '[
    {"id": 100, "user": {"login": "reviewer"}, "body": "Fix A", "path": "a.go", "line": 1, "diff_hunk": "", "created_at": "2026-01-01", "node_id": "N1", "in_reply_to_id": null},
    {"id": 200, "user": {"login": "reviewer"}, "body": "Fix B", "path": "b.go", "line": 2, "diff_hunk": "", "created_at": "2026-01-01", "node_id": "N2", "in_reply_to_id": null},
    {"id": 300, "user": {"login": "greptile"}, "body": "Fix C", "path": "c.go", "line": 3, "diff_hunk": "", "created_at": "2026-01-02", "node_id": "N3", "in_reply_to_id": null}
  ]' "[]"

  local output
  output=$(run_poll "$test_dir" "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "$work_dir")

  assert_contains "$output" "AGENT_NEEDED" "Detects new comment C and exits AGENT_NEEDED"
  assert_contains "$output" "1 comments" "Finds exactly 1 new comment (C, ID > 200)"
  assert_not_contains "$output" "IDLE_TIMEOUT" "Does NOT idle timeout"

  # Verify last-seen ID advanced to 300
  local last_rc_id
  last_rc_id=$(cat "$work_dir/last-review-comment-id")
  assert_equals "$last_rc_id" "300" "Last review-comment ID advanced to 300"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Test 5: Post-idle drain catches unresolved threads
# ---------------------------------------------------------------------------
test_post_idle_drain() {
  echo ""
  echo "=== Test 5: Post-idle drain catches unresolved threads ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  echo "0" > "$work_dir/last-review-id"
  echo "0" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo "[]" > "$work_dir/action-log.json"
  echo "0" > "$work_dir/turns-count"
  echo "999" > "$work_dir/pr-number.txt"

  # No new comments from API endpoints
  write_mock_responses "$test_dir" "[]" "[]" "[]"

  # But GraphQL shows unresolved review threads
  local graphql_response='{"data":{"repository":{"pullRequest":{"merged":true,"reviewThreads":{"nodes":[{"id":"THREAD1","isResolved":false,"comments":{"nodes":[{"databaseId":500,"body":"Unresolved comment","author":{"login":"greptile"}}]}}]}}}}}'
  echo "$graphql_response" > "$test_dir/graphql-response.json"

  # Run with MAX_IDLE_CYCLES=1 to trigger drain quickly
  # We need to override the constant in the script for testing
  local output
  output=$(
    PATH="$test_dir/mock-bin:$PATH" \
    MOCK_DIR="$test_dir" \
    GH_HOST="github.com" \
    GH_REPO="supersuit-tech/permission-slip" \
    bash -c '
      # Patch MAX_IDLE_CYCLES to 1 for fast test
      sed "s/MAX_IDLE_CYCLES=6/MAX_IDLE_CYCLES=1/" "'"$POLL_SCRIPT"'" > /tmp/poll-patched.sh
      bash /tmp/poll-patched.sh "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "'"$work_dir"'"
    ' 2>&1
  )

  assert_contains "$output" "unresolved review threads" "Detects unresolved threads during drain"
  assert_contains "$output" "AGENT_NEEDED" "Exits AGENT_NEEDED for drain pass"
  assert_not_contains "$output" "IDLE_TIMEOUT" "Does NOT emit IDLE_TIMEOUT when drain items exist"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Test 6: Idle counter suppressed when pending queue non-empty
# ---------------------------------------------------------------------------
test_idle_counter_suppressed_by_pending() {
  echo ""
  echo "=== Test 6: Idle counter suppressed when pending queue non-empty ==="

  local test_dir
  test_dir=$(setup_mock_env)
  local work_dir="$test_dir/work"
  mkdir -p "$work_dir"

  echo "200" > "$work_dir/last-review-id"
  echo "200" > "$work_dir/last-review-comment-id"
  echo "0" > "$work_dir/last-issue-comment-id"
  echo "[]" > "$work_dir/action-log.json"  # Agent didn't process
  echo "1" > "$work_dir/turns-count"
  echo "999" > "$work_dir/pr-number.txt"

  # Pending items exist but action log didn't grow
  echo '[
    {"type": "pending_metadata", "action_log_length": 0},
    {"type": "review_comment", "id": 100, "author": "reviewer", "body": "Fix A"}
  ]' > "$work_dir/pending-items.json"

  # No new comments from API
  write_mock_responses "$test_dir" "[]" "[]" "[]"

  local output
  output=$(run_poll "$test_dir" "https://github.com/supersuit-tech/permission-slip/pull/999" --work-dir "$work_dir")

  # Should re-include pending items and exit AGENT_NEEDED, not idle
  assert_contains "$output" "AGENT_NEEDED" "Pending items trigger AGENT_NEEDED, not idle"
  assert_not_contains "$output" "IDLE_TIMEOUT" "No idle timeout while pending queue non-empty"

  rm -rf "$test_dir"
}

# ---------------------------------------------------------------------------
# Run all tests
# ---------------------------------------------------------------------------
echo "Running watch-poll.sh regression tests..."
echo "(Simulating issue #847 failure scenarios)"

test_deferred_id_advancement
test_pending_queue_reprocessing
test_pending_queue_cleared_on_processing
test_new_comments_after_agent_needed
test_post_idle_drain
test_idle_counter_suppressed_by_pending

echo ""
echo "=== Results ==="
echo "Tests run: ${TESTS_RUN}"
echo "Passed:    ${TESTS_PASSED}"
echo "Failed:    ${TESTS_FAILED}"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
  echo ""
  echo "FAILED — ${TESTS_FAILED} test(s) failed"
  exit 1
fi

echo ""
echo "ALL TESTS PASSED"
exit 0
