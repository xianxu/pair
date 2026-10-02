---
id: 000385
status: open
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: '2e92f8ac2d209bb3982f54a3f9bd491a647ca07b' # card fields mirrored from issue-cards; edit via sdlc
---

# review helpers renamed by #104 still referenced by old names

## Problem

#104 M3 (`c4518af9`, "collapse to a single pair binary; stop bundling helpers")
folded `pair-review-readiness` and `pair-review-target` into the dispatcher as
`pair review readiness` / `pair review target`, with no shims. Callers still use
the old names, so an agent following them hits `command not found` during review
prep and is left to either reverse-engineer the dispatcher or hand-write the
review-target JSON, which the skill forbids because the CLI stamps the session.

Observed 2026-10-02 in a xianxu.dev couch session preparing
`ariadne-1-building-blocks-of-ai-coding.md`: both names missing from PATH; found
the new spelling only by reading `cmd/internal/dispatcher/dispatcher.go`.

Stale references:
- `ariadne/construct/local/fix/SKILL.md:289` (`pair-review-readiness <abs>`) and
  `:303` (`pair-review-target <abs> ready`) — the xx-fix skill every review
  agent loads.
- `workshop/targets/review-protocol.md:255,282` (`pair-review-readiness --prepare`).
- `atlas/go-migration-inventory.md` rows still describe them as standalone
  `cmd/pair-review-*` binaries.

Discoverability makes it worse: `pair review --help` (and bare `pair review`)
fails with `unknown command "review"`, and `pair help` does not list the
`review` group, so the new names can't be found from the CLI either.

## Spec

## Done when

- Agents following the xx-fix skill and the review protocol invoke a command
  that exists (update the references, or ship thin `pair-review-*` shims —
  decide which; a rename that breaks a consumer contract wants one or the other).
- `pair review --help` lists `readiness`, `target` and `open` with usage.
- A test pins that every `pair-review-*` name the skill/protocol references
  resolves (guards the next rename).

## Plan

- [ ]

## Log

### 2026-10-02
