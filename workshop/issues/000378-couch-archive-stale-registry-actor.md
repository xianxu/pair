---
id: 000378
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'f573ee8fec0a39f9406144ad352e612cc66346c5' # card fields mirrored from issue-cards; edit via sdlc
---

# couch archive refuses thread whose registered agent is dead

## Problem

The operator could not archive the `brain` thread (`couch-6b111ea230c149dc`, scope
`2e51fcf9799b1d8f`) to start fresh. The switcher refused:

```
error: archive couch-6b111ea230c149dc: it is live -- couch is hosting its agent; detach or park it first
```

At the same moment `couch --list` reported the row as `parked (no agent running;
resumable)`, and no `brain-couch-28` zellij session existed. The thread was
un-archivable and un-startable from the operator's seat — a bricked slot.

Cause: `~/.local/share/pair/couch/registry.json` still held **two** actor records for
that thread (`couch-0b3bf17c` pid 76476, `couch-ef593e35` pid 92434, both
`shape: warm-reattach`), and both processes were dead. `Couch.classifyForAction`
(`cmd/internal/couchcore/switchagent.go:80`) feeds every registry record with a
pid + identity into evidence as `hosted`, without a liveness check, so
`ClassifyThread` returns `ThreadLive` and `ArchivableState` refuses
(`detach.go:416` `archiveRefusal`).

Dead records are only reaped by `PruneDead` (`couch.go:1167`), whose sole caller
is `spawnResolved` (`couch.go:435`). A console that hasn't started a thread since
the agent died keeps the stale records, both in memory and on disk. Two records
for one thread also suggests that warm-reattach adds a record without retiring
the previous one.

Separately, the CLI (`--list`, which reports parked) and the switcher's action
path (which reports live) disagree about the same thread. That disagreement is
itself a symptom: the two surfaces should classify a thread from the same evidence.

## Spec

- Action admission (archive, and every other `classifyForAction` consumer) must
  not treat a registry record as hosting when its process is provably `Dead`
  (exact pid + identity). Unknown stays fail-closed (see `PruneDead`'s comment).
- Reap known-dead records on the paths that read them, not only on spawn, or
  make the classifier ignore them — pick one owner (ARCH-DRY), not both.
- Investigate why warm-reattach left two records for one thread.

## Done when

- A test: a thread with a dead registered actor (and parked record) classifies
  the same way for `classifyForAction` as for `--list`, and archive admits it.
- A test: an actor with `Unknown` liveness still refuses archive.
- Warm-reattach replaces, not appends, the thread's actor record (or the double
  record is explained and covered by a test).

## Plan

- [ ]

## Log

### 2026-10-01

- Observed live on brain. Before this, the row showed `binding lost — repairable`.
  The ledger had a launch with no binding, ever (the codex root was
  `01a0eda0-b38c-7992-988a-74ad95e6925e`). That was fixed with `pair session-repair
  codex couch-6b111ea230c149dc --scope-key 2e51fcf9799b1d8f --apply`. Only after
  that repair did the archive refusal above surface.
- Related, possibly worth separate issues: (a) why codex threads never bind;
  repair diagnostics showed 42 `parent_conflict` "Codex metadata ID disagrees
  with path ID" errors from forked subagents. (b) `pair session-repair` resolves its
  data dir from the caller's `PAIR_DATA_DIR`, not `--scope-key`, so running it
  inside another pair session looks in the wrong repo.
- Workaround: start any new thread, so that `spawnResolved` runs `PruneDead`, then archive.
