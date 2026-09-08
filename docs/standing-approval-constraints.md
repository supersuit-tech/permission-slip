# Standing approval constraints

Standing approvals limit what an agent may do on each execution. Constraints are validated on every auto-approve match.

## Formats

### Legacy flat map (v1)

Each key is an action parameter (or reserved namespace). Values use fixed, pattern, or wildcard syntax:

```json
{
  "repo": "supersuit-tech/permission-slip",
  "title": "*",
  "$meta": {
    "from": { "$pattern": "*@amazon.com" }
  },
  "$data_window": { "last_days": 30 }
}
```

### Structured rules (v2)

Version 2 adds **negation**, **multiple allow/deny rows per field**, and **OR across scenarios**:

```json
{
  "$version": 2,
  "match": "any",
  "groups": [
    {
      "match": "all",
      "conditions": [
        {
          "field": "channel",
          "op": "any_of",
          "values": ["#engineering", "#releases"]
        },
        {
          "field": "channel",
          "op": "none_of",
          "values": ["#executive-only"]
        },
        {
          "field": "$meta.sender",
          "op": "matches",
          "value": "boss@partner.com"
        }
      ]
    }
  ]
}
```

- **OR across groups** (`match: "any"`): the rule matches if *any scenario* (group) matches.
- **AND within a group** (`match: "all"`): every condition in the scenario must match.
- **Per-field allow/deny**: `matches` / `any_of` rows form an allow-list (OR). `does_not_match` / `none_of` rows form a deny-list (AND). A field passes when the value is allowed *and* not denied.
- **Empty allow-list** means no positive restriction (deny-list still applies).
- **Wildcard (`"*"`)** in an allow-list short-circuits that field to “any value allowed.”
- **Comparison thresholds** (`lte`, `gte`, `lt`, `gt`) apply to numeric parameters and RFC3339 datetime strings. Example: `{ "field": "limit", "op": "lte", "value": 20 }` auto-approves any request where `limit` is 20 or less.

## Comparison operators

Use structured v2 conditions — not MongoDB-style wrappers like `{"$lte": 20}` in flat maps:

```json
{
  "$version": 2,
  "match": "any",
  "groups": [
    {
      "match": "all",
      "conditions": [
        { "field": "limit", "op": "lte", "value": 20 }
      ]
    }
  ]
}
```

Supported ops: `lte` (≤), `gte` (≥), `lt` (<), `gt` (>). The web UI exposes these as “is at most”, “is at least”, etc. on number and datetime fields.

## Multi-valued fields

For array parameters and recipient metadata (`$meta.to`, `$meta.cc`, `$meta.bcc`):

- **Positive rules** (`matches`, `any_of`): at least one element must match.
- **Negation** (`does_not_match`, `none_of`): no element may match (security-safe “none of the recipients is X”).

## Example: `(repo AND title) OR channel`

Express `(repo = webapp AND title contains bug) OR (channel = #incidents)` as two groups:

```json
{
  "$version": 2,
  "match": "any",
  "groups": [
    {
      "match": "all",
      "conditions": [
        { "field": "repo", "op": "matches", "value": "supersuit-tech/webapp" },
        { "field": "title", "op": "matches", "value": { "$pattern": "*bug*" } }
      ]
    },
    {
      "match": "all",
      "conditions": [
        { "field": "channel", "op": "matches", "value": "#incidents" }
      ]
    }
  ]
}
```

## Backward compatibility

Existing flat constraints are adapted at evaluation time to a single v2 group with one `matches` condition per field. Behavior is unchanged for legacy rows.

**No database migration is required.** Legacy flat JSON continues to match identically; the Go read adapter normalizes flat maps to v2 at evaluation time. New standing approvals created in the web UI are stored as v2 structured JSON. To inspect either format:

- **CLI:** `permission-slip auto-approve format --constraints '<json>'` (add `--text` for plain output)
- **Web / mobile:** constraint summaries on standing approval cards and review dialogs

Optional future work: a one-time backfill migration to rewrite legacy rows as v2 for consistency in raw JSON exports. That is not required for correctness.

See also [ADR-002](adr/002-standing-approvals.md) and `db/constraint_validate.go` for engine details.

## Google Drive Shared Drive membership (`$meta.drive_id`)

Exact folder / spreadsheet / range allowlists cannot cover folders the agent creates later or slightly different A1 ranges. For Drive writes, list/search/get, and Sheets read/write/append/list, constrain the **verified Shared Drive** instead:

```json
{
  "folder_id": "*",
  "$meta": {
    "drive_id": "0AKbIIKZ8knmBUk9PVA"
  }
}
```

`$meta.drive_id` is resolved from the Drive API for the target folder, file, spreadsheet, or (for list/search) a verified `drive_id` parameter. It matches that drive's root **or any descendant**. Unscoped list/search (no folder or drive), My Drive, and other Shared Drives still require one-off approval. Discover the field via capabilities `meta_constraint_fields`.

The web and phone "always allow" flow proposes one workspace set covering routine Drive + Sheets work inside that Shared Drive. Constraint summaries display this as **inside Shared Drive** rather than a list of wildcard parameters.

## Google Calendar writes (`$meta.calendar_id`)

Exact `calendar_id` parameter pins miss aliases (`primary` vs the primary calendar’s email) and “always allow” copies every event field. For `google.create_calendar_event`, `google.update_calendar_event`, `google.delete_calendar_event`, and `google.create_meeting`, constrain the **verified calendar** instead:

```json
{
  "calendar_id": "*",
  "$meta": {
    "calendar_id": "work@example.com"
  }
}
```

`$meta.calendar_id` is the canonical Calendar API id from `GET /calendars/{calendarId}`. It matches that calendar regardless of whether the agent sent `primary`, omitted `calendar_id`, or used the email id. Other calendars still require one-off approval. Discover the field via capabilities `meta_constraint_fields`. Each write action still uses its own standing approval.

## Display formatting (shared)

Human-readable constraint summaries use `@permission-slip/constraints-format` in `shared/constraints/`. The same formatter powers web (`ConstraintsSummary`), mobile (standing approval detail screens), and the CLI (`auto-approve format`). When changing display semantics, update `shared/constraints/format.ts` and its tests once.
