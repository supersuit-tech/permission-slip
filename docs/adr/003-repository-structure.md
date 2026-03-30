# ADR-003: Repository Structure

**Status:** Accepted (revision 2 — replaces original three-repo decision)
**Date:** 2026-02-14
**Authors:** SuperSuit team

## Context

We originally decided on three separate repositories (`permission-slip`, `permission-slip-api`, `permission-slip`) based on the assumption that separate deploy targets and incompatible toolchains justified repo separation (see [original decision](#original-decision-three-separate-repositories) below).

After further evaluation, we've concluded the complexity costs of multi-repo outweigh the benefits for a small team building a tightly coupled product.

---

## Decision

Use a **single monorepo** where the Go backend serves the React frontend as embedded static assets.

```
permission-slip/
├── spec/openapi/          # OpenAPI schema (already exists)
├── docs/                  # ADRs, protocol docs (already exists)
├── backend/               # Go API service
│   ├── cmd/server/
│   ├── internal/
│   ├── go.mod
│   └── go.sum
├── frontend/              # React web app
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── Makefile               # Build orchestration
├── Dockerfile             # Single container
└── ...
```

The Go binary embeds the frontend build output using `//go:embed` (Go 1.16+) and serves the SPA alongside the API — one binary, one container, one port.

---

## Rationale

### Why a monorepo now

- **The API and UI are tightly coupled.** They evolve together around the same spec, the same approval flow, the same data model. Separate repos create artificial friction for changes that naturally span both.
- **Spec lives next to the code that implements it.** No syncing problem. No fetching scripts. No stale specs. The OpenAPI schema, the Go server stubs, and the TypeScript client all live in one tree.
- **Single CI pipeline.** One push triggers spec validation, backend tests, frontend tests, and a build. No cross-repo coordination, no webhook chains, no "which repo broke the contract?"
- **One deploy artifact.** `go:embed` bundles the React build into the Go binary. One Docker image, one process, one port. No CORS configuration, no reverse proxy, no separate hosting for the SPA.
- **Atomic PRs.** A feature that touches the API and the UI is one PR with one review, not two PRs that need to land in the right order.
- **Small team reality.** Multi-repo shines with large orgs and independent release cadences. For a small team shipping an MVP, the coordination overhead of three repos is pure cost.

### Why the original three-repo rationale no longer holds

| Original argument | Counterpoint |
|---|---|
| Go and JS don't share a package manager | True, but irrelevant — they share a spec, a deploy target, and a release cycle. A `Makefile` with `build-frontend` and `build-backend` targets is trivial. |
| Separate deploy targets | With `go:embed`, they deploy as one binary. Even if we later separate them, that's a packaging change, not a repo change. |
| CI simplicity | One pipeline is simpler than three. Path-based filtering (`backend/**`, `frontend/**`) is a solved problem in every CI system. |
| No cross-language build orchestration | A 10-line Makefile: `npm run build` then `go build`. This is not the complex orchestration we were worried about. |

### Why not git submodules

Git submodules were considered as a way to share the spec across repos. They add real friction:
- Requires `--recursive` on clone (easy to forget, breaks silently)
- `git submodule update --init --recursive` needed after every pull
- Submodule bumps show as cryptic hash diffs in PRs, not actual content changes
- Coordinated changes across repos means juggling commits in two places
- New contributors inevitably hit confusing detached HEAD issues

The monorepo eliminates the need for submodules entirely.

---

## Consequences

- **One repo to clone, one to CI, one to deploy.** Simpler onboarding, simpler operations.
- **The OpenAPI spec remains the contract.** Both backend and frontend consume it for code generation, but now they do so from the same tree.
- **Frontend and backend share a release cycle.** This is appropriate for an MVP where the UI and API are tightly coupled. If they need independent release cadences later, the code can be split into separate repos at that point — it's easier to split a monorepo than to merge separate repos.
- **SDK and CLI decisions remain deferred.** When the time comes, an SDK can be a separate repo (it serves external consumers) or a `packages/sdk` directory. The CLI can live in `backend/cmd/cli/`.
- **Build requires both Go and Node.js toolchains.** Developers need both installed locally. A `devcontainer.json` or similar can standardize this.

---

## Original decision: three separate repositories

> **Status:** Superseded by the monorepo decision above.

The original ADR-003 recommended three repos:

| Repository | Tech | Purpose |
|---|---|---|
| `permission-slip` | Markdown, OpenAPI | Spec, protocol docs, ADRs |
| `permission-slip-api` | Go | Backend API service |
| `permission-slip` | React | Frontend web app |

With a CI-fetch approach for syncing the spec. This was superseded after evaluating the coordination overhead for a small team building a tightly coupled product.
