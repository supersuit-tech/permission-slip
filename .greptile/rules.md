# Greptile Review Rules

These rules define automated review passes that run on every pull request. Each rule corresponds to a review phase with a specific lens and timing.

---

## Rule 1: Security & Principal Engineer Review

**Trigger:** On PR open and on every push to a PR branch.

Review the PR changes through the lens of a principal security engineer. Focus on:

- **Authentication & authorization gaps** — Are there endpoints, pages, or data paths missing auth checks? Are RLS policies applied to new tables?
- **Injection vulnerabilities** — SQL injection, XSS, command injection, path traversal, or any OWASP Top 10 concern in the changed code.
- **Secrets & credentials** — Are API keys, tokens, passwords, or connection strings hardcoded or logged? Are `.env` files or sensitive configs being committed?
- **Input validation** — Is user input validated and sanitized at system boundaries (API handlers, form submissions, URL params)?
- **Unsafe type assertions** — Are there `any` casts, unchecked type assertions, or `as` casts without justification?
- **Error leakage** — Do error responses expose internal details (stack traces, SQL errors, file paths) to the client?
- **Dependency risks** — Are new dependencies well-maintained and necessary? Do they introduce known vulnerabilities?
- **Database security** — Do new migrations grant appropriate permissions to `app_backend`? Are RLS policies in place? Are extension-managed objects explicitly granted? Are extension-schema grants wrapped in `IF EXISTS` guards for CI compatibility (plain Postgres lacks `vault`/`pgsodium` schemas)? Do `SECURITY INVOKER` views have transitive grants on every object they touch?

**Important:** Only flag real, concrete issues visible in the diff. Do not speculate or fabricate concerns. If nothing is found, say so.

---

## Rule 2: Code Quality, DRY & Maintainability

**Trigger:** 5 minutes after Rule 1 completes.

Review the PR changes for long-term maintainability. Focus on:

- **DRY violations** — Is logic duplicated across files or functions that should share a common abstraction? Are there copy-pasted blocks that differ by only a value or two?
- **Mega files** — Are any files growing too large? Would splitting improve readability and reduce merge conflicts? Files over ~300 lines of logic deserve scrutiny.
- **Component decomposition** — Are there large JSX blocks with inline comments labeling sections? Those should be extracted into named components.
- **Test race conditions** — Do tests share mutable state, use hardcoded ports, or rely on timing? Are database tests isolated with proper setup/teardown?
- **Future-proofing** — Are there magic strings or numbers that should be constants? Are switch statements missing default cases? Are there assumptions about data shape that could break as the schema evolves?
- **Error handling** — Do API hooks handle loading, error, and empty states? Are errors typed? Is there rollback logic for optimistic updates?
- **Refactor opportunities** — Could a complex function be broken into smaller, testable units? Are there opportunities to leverage existing utilities instead of reinventing?
- **Merge conflict risk** — Are changes inserting into the middle of shared files (route registries, config arrays) instead of appending? Are unrelated formatting changes mixed in?

**Important:** Be specific. Reference file names, line numbers, and concrete suggestions. Don't flag theoretical concerns — focus on what's actually in the diff.

---

## Rule 3: Documentation & Comments

**Trigger:** 5 minutes after Rule 2 completes.

Review the PR changes for documentation completeness. Focus on:

- **Code comments** — Is complex logic explained? Are non-obvious "why" decisions documented? Are there misleading or outdated comments that the changes should have updated?
- **README updates** — If the PR changes setup steps, environment variables, project structure, or user-facing behavior, is the README updated to match?
- **OpenAPI spec** — If new or modified API endpoints are introduced, is the OpenAPI spec (`spec/openapi/`) updated? Are request/response schemas accurate and complete?
- **Migration documentation** — Do new migrations have clear comments explaining the schema change and its purpose?
- **Inline documentation** — Are new exported functions, types, and interfaces documented? Are parameter constraints and return value semantics clear?
- **docs/ folder** — If a README section is growing large, should it be split into a dedicated doc page?

**Important:** Don't demand documentation for self-evident code. Focus on places where a new developer would be confused without context.

---

## Rule 4: Honesty & Completeness Audit

**Trigger:** 5 minutes after Rule 3 completes.

Audit the PR for gaps between what was claimed and what was actually done. Focus on:

- **Promised but missing** — Does the PR description or linked issue claim changes that aren't reflected in the diff? Are there checklist items marked done that aren't actually implemented?
- **Incomplete implementations** — Are there TODO comments, placeholder values, or stubbed-out functions that should have been completed?
- **Test coverage gaps** — Are new code paths covered by tests? Are edge cases and error paths tested, not just the happy path?
- **Skipped validations** — Were linting, type-checking, or build steps skipped? Are there `// @ts-ignore`, `eslint-disable`, or `nolint` directives that bypass checks without justification?
- **Silent failures** — Are errors swallowed silently? Are catch blocks empty or logging without re-throwing when they should propagate?

**Important:** This is not a gotcha — it's a safety net. Flag genuine gaps without judgment. The goal is to catch honest mistakes before they ship.

---

## Rule 5: Final Polish & Overlooked Improvements

**Trigger:** 5 minutes after Rule 4 completes.

Take one final look at the PR for anything that may have been skipped or seems too hard. Focus on:

- **Edge cases** — Are there boundary conditions (empty arrays, null values, zero-length strings, concurrent requests) that aren't handled?
- **Performance** — Are there N+1 queries, unnecessary re-renders, missing database indexes for new query patterns, or unbounded list fetches?
- **Accessibility** — Do new UI components have proper ARIA attributes, keyboard navigation, and screen reader support?
- **Consistency** — Do the changes follow existing patterns in the codebase, or do they introduce a new convention without migrating the old one?
- **Hard problems deferred** — Is there something in the diff that looks like a shortcut around a harder problem? Would addressing it now prevent tech debt?

**Important:** Be constructive. If suggesting a harder fix, explain why it's worth doing now rather than deferring. Acknowledge when the current approach is acceptable for the scope of the PR.
