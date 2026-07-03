# Changelog

All notable changes to `@permission-slip/cli` are documented here. Agents should run
`permission-slip changelog` (or read this file) at the start of each session to learn
about new capabilities since their last use.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.22] - 2026-07-03

### Added

- **`permission-slip pending`** — heartbeat sweep listing pending approvals and those
  resolved since `--resolved-since` (default last 24h). Run on every OpenClaw heartbeat
  as a backstop when push wakes are missed ([#1371](https://github.com/supersuit-tech/permission-slip/issues/1371)).
- **`permission-slip webhook set|status|clear`** — register the OpenClaw gateway hooks
  URL + token on the agent record for server-push approval wakes. `webhook set` and
  `webhook status --test` fire a test delivery so misconfiguration is caught at setup.

### Changed

- **Pending `wait_hint`** — when `push_wake_configured` is true on `request` / `status`
  output, the hint notes that a push wake is configured and the detached watcher is optional.

## [0.1.21] - 2026-07-03

### Fixed

- **`watch` targeted session wake** — when `--session-key` is set, the default notify
  command now uses `openclaw system event --mode next-heartbeat --session-key …`
  instead of `--mode now --session-key …`. OpenClaw treats the former as an
  immediate targeted wake that bypasses the runner's not-due gate; the latter could
  return RPC ok without resuming an idle/yielded session ([#1365](https://github.com/supersuit-tech/permission-slip/issues/1365)).

### Changed

- **`watch` notify diagnostics** — JSON output now includes `notify_attempts` with
  each shell command run and whether it exited successfully. `notified: true` only
  means the notify shell command did not throw, not that the gateway delivered a
  new agent turn.

## [0.1.20] - 2026-07-03

### Added

- **`watch --session-key`** — target a specific OpenClaw session when the approval
  resolves. Without this flag, the default notify command wakes the agent's default
  session, which can miss the session that spawned the watcher (e.g. an iMessage
  session parked in `sessions_yield`).
  ```bash
  permission-slip watch appr_xxxxxxxx --session-key agent:main:imessage
  ```
  Custom `--notify-cmd` templates can use a `{session_key}` placeholder (shell-quoted).

### Changed

- **Pending wait hints** — `wait_hint` on `request` / `status` now tells agents to
  append `--session-key` when their wake channel routes by session.

### Agent guidance

- When spawning `watch` after a pending `request`, pass your OpenClaw session key via
  `--session-key` so the approval wake returns to the session that opened it.

## [0.1.19] - 2026-07-03

### Added

- **`watch` command** — poll a pending approval in a detached background process and
  wake your OpenClaw session via `openclaw system event` when the human responds.
  Run after `request` returns `pending` instead of busy-polling `status` in your turn.
  ```bash
  permission-slip watch appr_xxxxxxxx
  ```
  Flags: `--interval 5s`, `--notify-cmd '<cmd with {id} {status} placeholders>'`.

- **Non-blocking wait hints on pending approvals** — `request` and `status` JSON output
  now include `wait_hint` and `wait_command` when status is `pending`, teaching agents
  to spawn `watch` in the background and end their turn.

### Agent guidance

- When `request` returns `pending`, run `wait_command` as a detached background process,
  then end your turn. On system event wake, run `status <approval_id>` for the result.
- See [OpenClaw integration docs](../docs/integrations/openclaw.md).

## [0.1.18] - 2026-06-08

### Fixed

- **CLI publishing restored.** The npm publish workflow had been failing since 0.1.16
  because a TypeScript 6.0 bump broke Node type resolution during the `tsc` build, so
  0.1.16 and 0.1.17 never reached npm. Pinned TypeScript back to 5.7.x to unblock
  publishing; this release carries the 0.1.16/0.1.17 features (`request-bulk`,
  `changelog`, bulk group status polling) to npm.

## [0.1.17] - 2026-06-07

### Added

- **`request-bulk` command** — submit N same-type actions in one approval request so the
  user gets a single notification and reviews all items on one screen. Use this instead
  of calling `request` N times when you need multiple actions of the same type (e.g.
  creating 4 calendar events).
  ```bash
  permission-slip request-bulk --action calendar.create_event --actions '[{...},{...}]'
  ```
  Constraints: same action type only, 2–50 items, no payment actions in bulk (v1).

- **`changelog` command** — shows CLI changes since your last recorded version. Run at
  the start of every session so you always know about new commands and behavior.

- **Bulk group status polling** — `permission-slip status --group <bulk_group_id>` polls
  per-item outcomes for a bulk submission.

### Agent guidance

- Before planning multi-step workflows, run `permission-slip changelog` to check for new
  bulk or batch capabilities — prefer `request-bulk` over repeated `request` calls when
  all actions share the same type.
- After any CLI upgrade, run `permission-slip changelog --mark-read` once you've updated
  your notes/memory.

## [0.1.16] - prior releases

See git history for earlier CLI changes.
