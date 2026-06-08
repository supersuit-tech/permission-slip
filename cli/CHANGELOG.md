# Changelog

All notable changes to `@permission-slip/cli` are documented here. Agents should run
`permission-slip changelog` (or read this file) at the start of each session to learn
about new capabilities since their last use.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
