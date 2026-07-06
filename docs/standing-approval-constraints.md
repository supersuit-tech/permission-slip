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

See also [ADR-002](adr/002-standing-approvals.md) and `db/constraint_validate.go` for engine details.
