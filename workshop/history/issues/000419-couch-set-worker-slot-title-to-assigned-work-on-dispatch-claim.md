---
id: 000419
status: done
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-10
estimate_hours:
card_mirror: '89dd768ba860de5e2e308dc21a9ba24e2b1c7065' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T20:21:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c8835bbf", done: "9d89a3c0"}
actual_hours: 0.85
---

# couch: set worker slot title to assigned work on dispatch claim

## Problem

Slot titles today come only from each session's own conversation. A TL
dispatching work over Couch has no way to label worker slots, so the
fleet view doesn't show which slot is doing what.

Requested by ariadne:1 (TL) via Couch peer message, 2026-10-09.

## Spec

Operator decision (2026-10-09): the **recipient labels itself**, through the
existing `PublishedSummary` path. No new Couch machinery: no stored dispatch
record, no sdlc polling, no new thread field.

- The `couch --skill` protocol gains a "Receiving dispatched work" step. Once
  the recipient's `sdlc claim` of the dispatched issue succeeds, it runs
  `couch --internal publish-description --description='repo#N <short title>'`.
  The claim is the confirmation, so the label appears exactly when the
  dispatch is confirmed.
- The label shows where summaries already show (`couch --list`, switcher
  focus view). An operator `!` line later replaces it, which is acceptable
  because the operator outranks the label.
- `publish-description` printed a raw Go `ThreadRecord` dump (`%v` default
  render). Nobody saw it, because nvim discards the op's stdout (detached for
  `!` tags, waited for `!!` and clear), but an agent following the skill would. It now prints one line: `published summary for
  <tag>: <text>` / `cleared published summary for <tag>`. The rendering is keyed
  on the op name, because `Detach` also returns a bare `ThreadRecord`.
- Layering: no pair→sdlc call. The agent bridges sdlc and couch from its shell.
- ARCH-FUNERAL: creates nothing durable beyond the existing summary field, which
  is overwritten by the next publish and dies with the thread's archive.

## Done when

- `couch --skill` tells a recipient to publish `repo#N <title>` after its claim
  of dispatched work succeeds. The label names the issue's own repository, and the
  command is a single copyable line.
- `couch --internal publish-description` prints one confirmation line (test:
  `TestPublishDescriptionUsesCompositeThreadEnvironment`).
- The atlas notes the use (`atlas/couch.md`, the publish-description paragraph).
- Dogfooded: this slot published `pair#419 slot title on dispatch claim`, and
  `couch --list` shows it.

## Plan

- [x] Test: publish-description stdout is one line (set and clear).
- [x] Render publish-description's record as one line, keyed on the op name.
- [x] Skill: add the receiving-dispatched-work step.
- [x] Atlas: note the agent-shell use.

## Log

### 2026-10-09
- 2026-10-09: closed — go test ./cmd/internal/couchcmd -run TestPublishDescription passes (new one-line stdout assertions, set+clear; failed before the render fix). Full go test ./... with session env scrubbed + make build: remaining fails are pre-existing/env-only — artifactpath, launcher (5), gcruntime reproduce identically on a clean main worktree; couchcmd continuation-writer fails on missing origin remote in its fixture, unrelated to render. Dogfood: couch --internal publish-description from this agent shell set pair:2 summary; couch --list shows "pair#419 slot title on dispatch claim". Round-1 BR-1 fixed (repo-generic label).; review verdict: SHIP

- Filed only (not implemented) at ariadne:1's request.
- Design fork put to the operator. They chose recipient self-labelling over a TL
  verify-then-label verb or a Couch claim watcher, and reusing PublishedSummary
  over a new field. Smallest surface (ARCH simplicity).
- Dogfood surfaced the raw `%v` record dump from `couch --internal
  publish-description`, which is fixed here. `Detach` also returns `ThreadRecord`,
  so the render is keyed on the op name, not the type.
- Process slip: implemented before running `change-code`; I ran it afterwards.
- Close review round 1 (FIX-THEN-SHIP): BR-1, the skill hardcoded `pair#N` for a
  cross-repo label, so it now names the issue's own repository with an `ariadne#300`
  example. Minor: the command wrapped inside its quotes, so it is now a one-line
  code block. Minor: the Spec's "detached" reason is corrected.
