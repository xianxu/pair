---
id: 000490
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '01884db062fc4c711d3abf14471c6337ff8f6aa1' # card fields mirrored from issue-cards; edit via sdlc
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

## Plan

- [ ]

## Log

### 2026-10-12
