---
id: 000490
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '09b1bd345b5ff37442ba14d320ca352c1dbdd075' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:29:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.81
---

# couch: say each verb's relative cost in --help and --skill (light, seconds, heavy)

## Problem

Agents (the TL included) call Couch's verbs without knowing what they cost. 10-12 the TL ran `couch --relaunch` across the fleet on a machine at load ~200, not knowing a relaunch stops the agent, cold-starts a wrapper and Claude Code, and (with PAIR_DEV) rebuilds pair; five failed (pair#489). The operator: give agents an intuition for each command's cost, concisely.

## Spec

In `couch --help` and `couch --skill`, mark each verb with a one-word relative cost and, where it matters, what makes it expensive:
- **light** (milliseconds, no process starts): e.g. `--send-to`, `--message-status`, `--peek`, `--actors`, `--broadcast-list`, `--session-sync` reads and writes.
- **seconds** (a few seconds, some work): e.g. `--prepare` (weave refresh, fresh conversation), `--reload-context`.
- **heavy** (tens of seconds, CPU-bound; avoid on a loaded machine; one at a time): e.g. `--relaunch`, `--reboot`, `--resume` (stop or start an agent and its wrapper; with PAIR_DEV, rebuild pair).
Keep it to one short tag per line, plus one sentence of guidance (run heavy verbs one at a time, check load first). Measure rather than guess: the tags come from timings on this machine, recorded in the Log.

## Done when

- Every verb in `couch --help` and `couch --skill` carries a cost tag, backed by measured timings in the Log.
- A test keeps the help and the skill in sync on the tags.

## Revisions

- 2026-10-12, close review: the measurements moved tiers from the Spec's guesses.
  light is "under a second", not milliseconds. --prepare and --reload-context are
  heavy, not seconds: both stop the agent and boot a fresh one. --adopt-store,
  --broadcast-web-query and --broadcast-web-replay are light, --broadcast-web-fidelity
  is seconds, and --recover-plan-from-sdlc is heavy (22-57 s). cost.go is the source.
  Done-when: verbs that were not timed are tagged from the path they share with a
  timed verb, as the Log says per verb: --reconcile, --reap, --reload-context,
  --prepare, --reboot, --recover, --send-to and the bare launch.

## Plan

- [x] Measure every public verb on this machine (load noted per batch); record the timings below.
- [x] One cost table (`couchcmd/cost.go`, `verbCosts`) — `couch --help` renders each verb line through `verbLine`, which panics on an untagged verb (ARCH-DRY: one source, two surfaces).
- [x] Skill section "What each verb costs": one bullet per tier plus the guidance sentence.
- [x] `cost_test.go`: help tags and skill bullets both equal the table; the guidance sentence is on both.
- [x] README + atlas lines for the new help format.

## Log

### 2026-10-12
- 2026-10-12: closed — make -k test green except test-review (wrong-conversation-ack-refused under load 22; passes on rerun, untouched area); go test ./... green after adding cost.go to the artifactpath inventory, except couchcore TestRetentionExplicitResume (load flake, passes alone); TMPDIR=scratch make test-changelog green. cost_test: help tags and skill bullets equal verbCosts; verbLine panics on an untagged verb. Timings measured on this machine in the Log; heavy slot ops on parley.nvim:3 per the TL, which is left parked as the pair#489 reproduction.; review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 110 added lines in code files (limit 100)

Timings: `couch` binary at ~/workspace/pair/bin (10:27 build), 1-min load 10-23 unless noted;
each light/seconds verb run 2-3 times, a 120 s hard cap per run (perl alarm; no `timeout` here).
- **light** (under 1 s): --help 7-9 ms, --skill 7-9 ms, --actors 10-12 ms, --message-status 12 ms,
  --broadcast-list 10-11 ms, --session-sync read --all / status 9-11 ms, --adopt-store preview 10 ms,
  --broadcast-web-query notes/dom/events 13-28 ms (rec-blog, rec-stress fixtures),
  --broadcast-web-replay prints its link in 35 ms and then serves until stopped,
  --peek one slot 0.22 s, three slots 0.19-0.58 s, --archived 0.30-0.71 s (median 0.37).
  --send-to is the same socket admission as --message-status (returns `queued` at once).
- **seconds**: --list 1.13-1.99 s (median 1.31), --peek --transcripts 5.1-6.4 s (median 5.2),
  --show 8.9-9.8 s (median 9.5), --broadcast-web-fidelity 9.7-11.7 s (median 10.5, load 67-89; a headless Chrome).
  --reconcile not run (it repairs a workspace): it computes the plan --show prints, so it is at least --show.
  --reap not run (no orphaned server to reap); tagged seconds: it ends a process tree and waits for it.
- **heavy**: --recover-plan-from-sdlc 22.3 s and 56.8 s (runs `sdlc fleet inventory` per fleet).
  On parley.nvim:3 (the TL's scratch slot; one at a time, load 10-17 before each):
  --relaunch --same-binary 15.1 s and 22.3 s succeeded; then 24.0 s and 11.6 s ended `uncertain`
  (the CLI's context deadline), and the last left the slot parked (pair#489's failure at load 11-17,
  not just 200; every receipt warned "PAIR_DEV is set, so the relaunch rebuilds pair itself").
  --resume then failed after 34.8 s: pane not born in 15 s, no Pair session live
  (claim 3.6 s, prepare 0.7 s, record+baseline 1.0 s, registration 15.0 s).
  --reload-context and --prepare were refused `not-live` on the parked slot; the TL kept it parked as
  #489's reproduction, so they are tagged heavy from the shared path (both stop the agent and boot a
  fresh one in its wrapper; --prepare also runs weave refresh). --reboot = relaunch's start plus a
  fresh session (TL: not timed). --recover runs a resume or reboot. The bare `couch` launch boots the
  supervisor and its slots' wrappers.
- Tiers: light < 1 s; seconds 1-15 s; heavy >= 15 s or boots/stops agents.

