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

- `couch --peek pair:3 --lines 10` returns the tail from the wrapper's memory, with dim spans and the cursor marked, in well under a second.
- A Claude Code ghost suggestion and a typed draft render differently in the tail (a test with captured bytes from `wrapcmd/testdata/tty/`).
- A multi-slot peek over at least 3 slots returns one sectioned snapshot, and an unreadable slot reports its reason inline.
- `atlas/couch.md` documents the tail store, the markup, and the multi-slot syntax; the `couch --skill` text tells callers that judging state is theirs.

## Plan

Durable plan: `workshop/plans/000425-couch-in-memory-slot-tails-with-style-and-cursor-marks-plus-a-multi-slot-peek-plan.md`.

- [ ] terminal tail renderer (wrapper emulator → markup) + tests
- [ ] endpoint + broker `tail` ops, message_service handler + tests
- [ ] couchcore peek live-first, multi-slot snapshot + tests
- [ ] CLI wiring/rendering + router test
- [ ] atlas + couch skill docs
- [ ] live check on 3+ slots

## Log

### 2026-10-10
- Design: the wrapper's vt emulator already keeps screen + 10k scrollback in memory, so the tail is a render on request, not a new store. Reached through an identity-free read-only broker op (`broadcast-status` precedent), because only the broker knows the wrapper endpoint's binding-hashed socket. Recording render stays as the fallback, reason inline.
