---
id: 000650
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'aa7ae31ea75d72876271da1bb884148bffe5ed85' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T12:13:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
---

# Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call

## Problem

## Spec

Operator decision, 2026-10-17. The draft's slug line (line 1, a summary of what the slot is doing) is off by default when the wrapper is Couch-hosted. Measured by pair:2: about 70 `pair slug` spawns and about 36 Haiku calls an hour across the fleet, with ~570 MB transient memory peaks per call and about 3% of a core on average. The operator doesn't use the slug under Couch.

- When `launcher.CouchHostedEnv` is true, `wrap.go`'s `maybeSpawnSlug` returns before spawning anything: no `pair slug` process and no model call.
- `PAIR_SLUG=on` turns it back on under Couch. Outside Couch, behaviour is unchanged.
- Help, the README and the atlas say so.

## Done when

- A test: Couch-hosted env spawns nothing; with `PAIR_SLUG=on` it spawns; outside Couch it spawns as before.
- After the next restart and relaunch, no `pair slug` or `claude -p` processes appear across the fleet during normal turns. Checked live.

## Plan

Quick flow — a few lines.

- [ ] Test first (`cmd/internal/wrapcmd/slug_spawn_test.go`): couch-hosted env → `maybeSpawnSlug` spawns nothing; couch-hosted + `PAIR_SLUG=on` → spawns; non-Couch → spawns as before. `t.Setenv` pins both couch vars and `PAIR_SLUG` in every case so the ambient (hosted) test env can't leak in.
- [ ] Gate at the top of `maybeSpawnSlug` (`wrap.go`): `launcher.CouchHostedEnv(os.Getenv) && os.Getenv("PAIR_SLUG") != "on"` → return before the debounce bookkeeping or the spawn. Reuses `launcher.CouchHostedEnv` — no new env parsing (ARCH-DRY). `PAIR_SLUG` re-enables only on the exact value `on` (the knob the operator named); any other value stays off, so a hosted session fails closed to the operator's default.
- [ ] Docs: `maybeSpawnSlug`'s comment in `wrap.go`, a README line (Notifications section, next to the wrapper env knobs), and the trigger bullet in `atlas/architecture.md`'s auto-orientation slug section. `pair --help`'s usage text deliberately excludes behavior notes (its own comment says so), so "help" = the wrapper's behavior docs; flagged to the hub in the plan post.
- [ ] The live fleet check (no `pair slug` / `claude -p` after restart + relaunch) is the hub's/operator's relaunch; recorded in Log and reported with the landing.

## Log

### 2026-10-17

- Claimed via the hub's dispatch (pair:8). Quick flow confirmed: gate + test + three doc lines.
- `launcher` is already imported by `wrap.go`; `launcher.CouchHostedEnv` is the existing hosted predicate (scope or tag non-empty).
