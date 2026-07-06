# ADR 010: Remove action configurations; open action requests + standing approvals

## Status

Accepted (2026-07-06)

## Context

Permission Slip originally required users to create an **action configuration** for every connector action before an agent could request it. Standing approvals could optionally link back to a source configuration for labeling and account scope. This model was cumbersome: new connector actions required manual setup, and mobile/web flows broke when trying to match approvals to configs (#1284, #1388, #1395).

The approval pipeline already treated action configurations as optional at request time — standing-approval matching never consulted configs. The hard gate was discovery (`GET /capabilities` advertised configs) and UX that assumed configs existed.

## Decision

1. **Remove action configurations** (`action_configurations`, `action_config_templates` tables and CRUD APIs).
2. **Agents may request any action** on an enabled connector with schema-valid parameters. Default path is one-off human approval.
3. **Standing approvals are the only user configuration** — self-describing with `name` and `description`, optional `connector_instance_id` scope, constraint semantics unchanged.
4. **Templates become standing-approval templates** (`standing_approval_templates`, `/standing-approval-templates/*`) — presets for pre-authorizing actions, not for gating requests.
5. **`POST /approvals/request`**: `configuration` field deprecated — accepted and ignored for one release, then removed from the spec.
6. **Capabilities**: drop `action_configurations` per action; include `name` on standing approval entries.

## Consequences

- Simpler agent decision tree: credentials ready → standing approval match → else prompt.
- Existing standing approvals continue to work; names backfilled from linked configs during migration.
- Action configurations are **not** auto-converted to standing approvals (would silently escalate to auto-approve).
- Frontend/mobile/CLI must migrate to standing-approval-centric UX (Phases 2–3 of #1425).

## Amends

- ADR 001 (email MVP): removes "configuration as permission" framing.
- ADR 002 (standing approvals): data model drops `source_action_configuration_id`; rules carry their own name/description.
