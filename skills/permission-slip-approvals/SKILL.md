---
name: permission-slip-approvals
description: Non-blocking approval flow for Permission Slip — request, spawn a detached watcher, end your turn, and continue when woken by a system event.
---

# Permission Slip Approvals (OpenClaw)

Use this skill when you need human approval for a Permission Slip action and must **not** block the OpenClaw session while waiting.

## The pattern

1. **Request** — run `permission-slip request ...`. If the response is `pending`, read `wait_hint` and `wait_command` from the JSON output.
2. **Watch in background** — run `wait_command` as a **detached background process** (one watcher per pending approval). Do **not** poll `permission-slip status` in a loop inside your turn.
3. **End your turn** — the main session stays responsive while the human reviews on the dashboard.
4. **On wake** — when OpenClaw delivers a system event like `Permission Slip appr_x resolved: approved — continue the task`, run `permission-slip status appr_x` (or use fields from the request response) to fetch `execution_result` and continue the task.

If `request` returns `executed: true` or `status: approved`, the action already ran — skip watching.

## Commands

```bash
# Submit an action (auto-executes when a standing approval matches)
permission-slip request --action email.send --params '{"to":"user@example.com","subject":"Hi"}'

# When pending — spawn watcher in background, then end your turn:
# Pass --session-key when your wake channel routes by session (e.g. OpenClaw iMessage):
permission-slip watch appr_xxxxxxxx --session-key <your session key>

# After wake — fetch the outcome:
permission-slip status appr_xxxxxxxx
```

## Multiple pending approvals

Run **one** `permission-slip watch <id>` per pending approval. At personal-use scale, N small background processes is fine.

## Recovery

- **Gateway restarted while watching** — the watcher process may be orphaned. On next wake, run `permission-slip status <id>` or re-run `permission-slip watch <id>`.
- **Accidentally polling `status` in a loop** — the first pending `status` response includes the same `wait_hint` / `wait_command`; switch to the watcher pattern.
- **Wake fired but wrong session answered** — pass `--session-key <your session key>` so the notify targets the session that opened the approval.

## Flags (advanced)

```bash
permission-slip watch appr_x --interval 5s
permission-slip watch appr_x --session-key agent:main:imessage
permission-slip watch appr_x --notify-cmd 'openclaw system event --text "done {id} {status}" --mode now --session-key {session_key}'
```

Default notify uses `openclaw system event` when `openclaw` is on PATH. With `--session-key`, the default template uses `--mode next-heartbeat --session-key {session_key}` for a reliable targeted wake (not `--mode now`, which can return ok without resuming an idle session).

## Further reading

- [OpenClaw integration docs](../../docs/integrations/openclaw.md)
- [Agent integration guide](../../docs/agents.md)
