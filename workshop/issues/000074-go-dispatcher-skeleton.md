---
id: 000074
status: open
deps: []
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair Go dispatcher skeleton

## Problem

The target architecture needs a primary Go command, but switching the public launcher immediately would be too risky. The first code step should introduce the dispatch shape without changing user-visible behavior.

## Spec

Add a Go dispatcher skeleton that can host Pair subcommands behind an explicit development path. It should establish command parsing, help text shape, version/build metadata if needed, and an internal routing pattern for future subcommands.

The existing `bin/pair` shell launcher remains the public entrypoint for this issue. Any new Go command must be opt-in, for example a new built binary or a hidden/dev invocation, so this can merge without affecting normal sessions.

Design constraints:

- Reuse existing package structure where possible (`ARCH-DRY`).
- Keep command parsing and dispatch decision logic pure enough to unit-test (`ARCH-PURE`).
- Do not port launcher behavior yet. This issue is only the skeleton.

## Done when

- [ ] A Go dispatcher command exists and builds in the normal `make build` flow or an explicitly documented dev target.
- [ ] Dispatcher help lists the planned command families without claiming unsupported behavior works.
- [ ] Public `bin/pair` behavior is unchanged.
- [ ] Tests cover dispatch parsing and unsupported-command errors.
- [ ] Pair remains usable after merge through the existing `pair` entrypoint.

## Plan

- [ ] Choose the command location/name based on #73 inventory.
- [ ] Add pure dispatch parsing tests.
- [ ] Add the skeleton implementation.
- [ ] Wire build/install only in a non-disruptive way.
- [ ] Verify existing Pair flows still use the shell launcher.

## Log

### 2026-06-26

Created from #72 as the safe first code-bearing step toward one primary Go command.
