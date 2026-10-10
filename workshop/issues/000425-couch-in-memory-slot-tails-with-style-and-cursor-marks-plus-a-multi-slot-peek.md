---
id: 000425
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '105fe1ad951c6770782b7d3ae68612b74d8e57bc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T11:13:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch: in-memory slot tails with style and cursor marks, plus a multi-slot peek

## Problem

A TL (in the `ops` slot) and the operator drive several slots at once, and need to know each slot's state before acting on it: is a turn running, is work still going in the background, is someone typing? Today the only window is `couch --peek`, one slot at a time, with styling stripped. Two misreads on 10-10 (ops learnings for ariadne-robustness-1, kinks 41–42):
- **Background work looked idle.** After pair:3's agent backgrounded `sleep 60` and ended its turn, the only cue was the footer text "✻ Baked for 6s · done 10:30 AM · 1 shell still running". Nothing spun. Couch's idle rules (30s quiet for messaging, Settled for #421's relaunch) read the slot as idle, and a relaunch killed the job.
- **Ghost text looked typed.** Claude Code's dim prompt suggestion ("let's work on #415") is identical to a typed draft once peek strips styling. The TL took it for operator input.

## Spec

**Principle: pair renders and never classifies.** Different coding agents, and different versions of one agent, word and draw their state differently. A binary parser of that state is fragile. Judgement belongs to the caller, an LLM or a human, who reads the tail the way a human reads the screen. Pair's job is to make that read cheap and lossless.

1. **In-memory tail.** Each wrapper keeps the last N rendered lines of its agent pane in memory (N about 50), and serves them over the existing socket. No screen dump per read.
2. **Visual cues kept, as light markup:**
   - non-default cell attributes, at least dim/faint (e.g. `‹dim›…‹/dim›`); reverse video too if cheap;
   - the cursor's position and shape (e.g. a marker at the cursor cell plus `cursor: row,col shape`).

   These are the cues a human notices; generic terminal bookkeeping, not agent knowledge.
3. **`couch --peek` reads the store**, so it's fast enough to call often (a TL polls).
4. **Multi-slot peek:** `couch --peek pair:1:2:3,ariadne:0:1:2 --lines 10` returns one snapshot, one section per slot (address, tag, tail), so the TL sees everything in one call. Slots that can't be read say why, inline.

**Out of scope:** new busy rules, and broker-side choice of a free slot. #421's existing guards stay as they are. Deciding busy or free is the caller's job (the TL skill carries the prose).

## Done when

- `couch --peek pair:3 --lines 10` returns the tail from the wrapper's memory, with dim spans and the cursor marked, in well under a second. *(Revision 2026-10-10: the live end-to-end run is deferred to the TL at rollout; see Revisions.)*
- A Claude Code ghost suggestion and a typed draft render differently in the tail (a test with captured bytes from `wrapcmd/testdata/tty/`).
- A multi-slot peek over at least 3 slots returns one sectioned snapshot, and an unreadable slot reports its reason inline.
- `atlas/couch.md` documents the tail store, the markup, and the multi-slot syntax; the `couch --skill` text tells callers that judging state is theirs.

## Plan

Durable plan: `workshop/plans/000425-couch-in-memory-slot-tails-with-style-and-cursor-marks-plus-a-multi-slot-peek-plan.md`.

- [x] terminal tail renderer (wrapper emulator → markup) + tests
- [x] endpoint + broker `tail` ops, message_service handler + tests
- [x] couchcore peek live-first, multi-slot snapshot + tests
- [x] CLI wiring/rendering + router test
- [x] atlas + couch skill docs
- [x] live check on 3+ slots (CLI + fallback; live path needs Couch restart)

## Revisions

- **2026-10-10 — Done-when 1 live check deferred to the TL (ops decision, option b).** Why: the running Couch and slot wrappers predate the new `tail` op, so an end-to-end live run needs the operator to restart Couch on this build and relaunch a slot, which this slot must not do. Delta: the issue lands with each hop proven separately: the wrapper render plus the real endpoint socket (`TestTailOverEndpointSocketIsFast`: 200-line tail over 12k lines of scrollback in about 1 ms), the broker handler (`TestTailReadsTheThreadsConnectedWrapper`), the CLI broker call (`TestReadSlotTail`) and the router (`TestMultiSlotPeekRunsThroughTheRouter`). The TL runs `couch --peek pair:N --lines 10` live at rollout, and the issue isn't counted done until that passes.

## Log

### 2026-10-10
- 2026-10-10: closed — Targeted: wrapcmd TestTail* (ghost vs typed from testdata/tty/claude/2.1.237/composer.raw + peer paste-short.raw, literal-markup escape, reverse, cursor shape, scrollback/alt, real endpoint socket; TestTailOverEndpointSocketIsFast: 200-line tail over 12k scrollback via real socket ~0.7-1.3ms; -race clean), couchmessage all, couchcmd handleTail + readSlotTail + TestMultiSlotPeekRunsThroughTheRouter (RunWithRuntime argv), couchcore live-first/fallback + PeekSnapshot + 32-slot cap. make -k test: all pass except test-pair-embedded-runtime/test-changelog, which pass with session PAIR_/COUCH_ env scrubbed (+scratch TMPDIR). go test ./...: 83 ok; 3 FAIL unrelated (couchcmd TestColdResume... fails on main too; artifactpath inventory lists many pre-existing main files; gcruntime reads live Couch slot metadata). Live: new CLI multi-slot peek on pair:0:3,ops:0,nope:9 -> 4 sections, reasons inline, falls back to recording against the pre-#425 Couch. BR-1 (Done-when 1 live e2e): explicitly deferred to the TL at rollout by ops decision, recorded as a Revision on the Done-when (as #421); each hop proven by tests incl. timed real-socket test.; review verdict: SHIP
- 2026-10-10: flow upgraded quick → full — 589 added lines in code files (limit 100); an earlier round of this close already ran the full review
- Design: the wrapper's vt emulator already keeps screen + 10k scrollback in memory, so the tail is a render on request, not a new store. Reached through an identity-free read-only broker op (`broadcast-status` precedent), because only the broker knows the wrapper endpoint's binding-hashed socket. Recording render stays as the fallback, reason inline.
- Implemented: `terminalModel.Tail` (wrapcmd/terminal_tail.go), endpoint op `tail`, identity-free broker op `tail` → `messageService.handleTail`, `couchcore.PeekSlots`/`ExpandPeekReferences`, `readSlotTail`. Ghost vs typed test uses `testdata/tty/claude/2.1.237/composer.raw` (ghost renders `‹dim›Try …‹/dim›`; the same cells unfaint and the captured `paste-short.raw` draft render without it). Claude's prompt glyph is followed by U+00A0, not a space.
- Live run caught a router gap (lesson #424 again): `runTypedOperationWithConsole` parsed the whole ref as one `repo:N` before dispatch, refusing `pair:0:3,ops:0`. My first "router" test stopped at `bindArgs`. Fixed with `singleReference` + `TestMultiSlotPeekRunsThroughTheRouter` via `RunWithRuntime`.
- Live (new CLI, running couch predates the op): `couch --peek pair:0:3,ops:0,nope:9 --lines 4` → 4 sections, `nope:9` reason inline, live tail falls back to the recording with `live tail: invalid-request: json: unknown field "TailScope"` (now mapped to "restart Couch"). The live path end to end needs a Couch restart and relaunched slots on this build.

