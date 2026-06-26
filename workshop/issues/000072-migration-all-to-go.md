---
id: 000072
status: working
deps: []
github_issue:
created: 2026-06-25
updated: 2026-06-26
started: 2026-06-26T10:06:43-07:00
estimate_hours:
---

# migration all to Go

## Problem

Pair started as a set of shell scripts, then performance- and correctness-sensitive pieces moved to Go (`pair-wrap`, `pair-scrollback-render`, `pair-slug`, `pair-changelog`, `pair-continuation`, `pair-context`, `pair-scribe`). The remaining installed surface is now a mixed shell/Go/Lua/zellij asset tree. That works for development, but it makes packaging and distribution harder than the desired end state: a primary `pair` binary that can be installed, upgraded, and reasoned about as one command.

The question is not "rewrite everything because Go is nicer." The question is whether a packaging-led Go consolidation is worth doing, and if so, how to sequence it so every intermediate merge leaves Pair working.

## Spec

### Decision

Yes, move toward a single primary Go `pair` binary, but do it as a staged packaging architecture rather than a blanket rewrite.

The primary driver is packaging/distribution. Reliability/testability is the secondary driver. Portability matters, but should fall out of reducing shell-specific behavior rather than becoming a separate rewrite goal.

### Target architecture

- `pair` is the installed command and owns CLI dispatch, session lifecycle, data/config path resolution, asset discovery, restart/quit/continue flows, and subprocess orchestration.
- Existing Go command surfaces become internal subcommands or dispatch modes behind the primary binary: `pair wrap`, `pair slug`, `pair context`, `pair scrollback-render`, `pair changelog`, `pair continuation`, and `pair scribe`.
- Native runtime assets stay native. `nvim/*.lua` remains the Neovim integration layer; `zellij/*.kdl` remains the zellij layout/config layer. Packaging may embed or install these assets beside the binary, but migration should not force them into Go.
- Shell remains only for irreducible platform glue or short compatibility shims. Stateful logic and installed command behavior should move behind the Go binary when that improves packaging or reliability.
- Every migration step must be merge-safe: after each sub-issue lands, the public `pair` command, `pair-dev`, keybindings, scrollback, changelog, continuation, and review flows still work.

Architecture principles:

- `ARCH-PURPOSE` — the point is a packaging roadmap and executable issue sequence, not a token code port. A sub-issue is valid only if it moves the repo toward the single-primary-binary target while preserving Pair's current behavior.
- `ARCH-DRY` — existing Go implementations should be reused behind dispatch rather than copied into parallel binaries.
- `ARCH-PURE` — new Go migration work should extract pure decision logic from IO-heavy shell behavior, then keep subprocess/zellij/filesystem interaction in thin seams with process-level tests.

### Migration sequence

Created follow-up issues:

- #73 — inventory the installed surface and packaging contract.
- #74 — add a Go command dispatcher skeleton without changing the public entrypoint.
- #76 — fold existing Go helpers behind the dispatcher while keeping old command names working.
- #75 — prototype the Go launcher behind an alternate or guarded path.
- #77 — switch the public `pair` entrypoint to Go with fallback/compatibility protection.
- #78 — port stateful shell glue where packaging/reliability payoff is clear.
- #79 — consolidate packaging, asset handling, and obsolete compatibility shims.

The ordering above is intentional even though the allocated issue IDs are not strictly chronological: the issue files were allocated while this roadmap was being created. Dependencies should be set or restated when each sub-issue is claimed.

## Done when

- [x] Target architecture is recorded in this issue.
- [x] Migration sequence is recorded in this issue.
- [x] Sub-issues exist for each independently executable milestone.
- [x] Each sub-issue states that Pair must remain usable after its merge.

## Plan

- [x] Brainstorm whether migrating all scripts to Go is worth doing.
- [x] Choose the packaging-led target architecture.
- [x] Create independently executable follow-up issues.
- [x] Record merge-safe acceptance criteria.

## Log

### 2026-06-25

Issue opened to evaluate whether Pair should move "all to Go" for packaging/distribution.

### 2026-06-26

Claimed and entered planning. Brainstorm conclusion: pursue a single primary Go `pair` binary as the target architecture, but do not treat "all to Go" as literal removal of Lua/zellij native assets or a blanket shell rewrite. The migration should be staged through merge-safe sub-issues so Pair remains usable at every intermediate state.
