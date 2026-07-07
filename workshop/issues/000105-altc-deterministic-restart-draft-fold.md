---
id: 000105
status: working
deps: []
github_issue:
created: 2026-07-06
updated: 2026-07-06
estimate_hours:
started: 2026-07-06T22:04:17-07:00
---

# alt+shift+c: deterministic writer-triggered restart + fold draft WIP into continuation NEXT ACTION

Moved from **ariadne#159** (all the code is here in `pair`). Brainstormed there 2026-07-06; blocked on #104 (single-pair-binary) until it landed (PR #70) — its landing is *why* this is doable cleanly: writer + launcher are one binary now, so the writer can trigger the restart itself instead of relying on a fragile agent step.

## Problem

`alt+shift+c` (zellij `config.kdl` `bind "Alt C"`) → `PairConfirmCompact()` (`nvim/init.lua:3327`) → `send_to_agent(COMPACT_PROMPT)` (`nvim/init.lua:3324`). `COMPACT_PROMPT` is a **two-step NL instruction to the agent**: (1) write a continuation via `pair continuation`, then (2) *itself* run `pair continue <slug>`. Step 2 is agent judgment, not code — so if the agent writes the doc but skips `pair continue`, no restart marker is written and the outer reincarnation loop (`createflow.go:63-109`) just exits. **That is the "restart stopped working" bug: the restart was never deterministic.**

Second, on a successful compaction restart the outer loop **overwrites** the draft with a seed line (`createflow.go:227-229`: `"Read workshop/continuation/%s and continue from its NEXT ACTION."`). So any WIP the operator left in the draft ("`*`") is **lost** across the restart.

## Spec

Two changes, both making the flow deterministic (no agent judgment):

1. **Writer triggers the restart (bug fix).** After `pair continuation` writes + commits + pushes the doc, it deterministically triggers the compaction restart itself using the slug it already holds — reusing the existing, tested path (`pair continue <slug>` → `runCompaction` → outer loop relaunch). Gated so it only fires in the compaction context (in-pane + `PAIR_TAG` set), never for a standalone write. `COMPACT_PROMPT` collapses to a single instruction ("write the continuation with the writer") — no step 2 for the agent to forget. Same config is preserved automatically: `PAIR_DEV` rides the env through the exec/loop, so pair vs pair-dev needs no explicit branch.

2. **Fold draft WIP into NEXT ACTION (feature).** At compaction time, read `draft-<tag>.md` (`nvim/init.lua:471` `draft_path_for_tag()`; Go path pattern already used at `createflow.go:229`, `rename.go:36`), strip comment lines (the `=== label ===` stickies — `strip_comments` at `nvim/init.lua:995` drops `^%s*===`), and if non-empty WIP remains, incorporate it into the continuation's `## NEXT ACTION` automatically — so the operator's parked draft survives the restart as part of next steps.

**Open design fork (resolve in the plan):**
- *Where the fold happens* — leaning: in the writer, before write, so the persisted+committed doc already carries the WIP (writer has `PAIR_TAG` from env; needs the data-dir → `draft-<tag>.md`). Alternative: the compaction flow edits the doc post-write (messier: re-commit an already-pushed doc).
- *How the WIP lands in NEXT ACTION* — mechanical append (fully deterministic, matches "automatically") vs. handed to the agent to weave (reintroduces judgment). Operator said "automatically" → lean mechanical.
- *Comment-strip duplication* — a Go `===`-strip is a tiny port of the Lua `strip_comments`; pin them with a drift test (one grammar, two sites — the repo's inline-copy + drift-test lesson).

## Done when

- `alt+shift+c` reliably: agent writes a continuation → the writer **automatically** restarts the tag with the same config (pair/pair-dev) + a new session → the fresh session continues from the doc's NEXT ACTION. No agent step required for the restart; hardening COMPACT_PROMPT step 2 is not the fix — removing the dependence on it is.
- When the draft had non-comment WIP at compaction time, that WIP (sans `===` comments) appears in the continuation's NEXT ACTION and thus survives into the new session.
- Standalone `pair continuation` (non-compaction) still just writes — no restart, no fold.
- Tests: writer-triggers-restart on the fake Runtime (marker written + kill sequence observed); draft-fold unit (comment strip + NEXT-ACTION insertion) + the Go/Lua strip drift test; the existing `pair continue` compaction path stays green.

## Plan

- [ ]

## Log

### 2026-07-06

Created from ariadne#159 (moved). Traced the full post-#104 flow: binding `zellij/config.kdl` → `PairConfirmCompact`/`COMPACT_PROMPT` (`nvim/init.lua:3324`) → agent → `pair continuation` writer (`cmd/internal/continuationcmd/continuationcmd.go`; assemble/NEXT-ACTION helpers in `continuation.go`) + agent-run `pair continue` → `runCompaction` (`cmd/internal/launcher/compaction.go:54`) → outer relaunch loop + draft re-seed (`createflow.go:63-109`, seed at `227-229`). Root cause of "restart stopped working": restart is agent judgment (COMPACT_PROMPT step 2), not code. Draft-fold rationale: the re-seed overwrites the draft, so parked WIP is lost without the fold. Next: `sdlc start-plan` → durable plan resolving the design fork above → change-code → implement.
