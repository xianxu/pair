---
id: 000378
status: working
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'ab4791c630310a385f67f97acc42cef53e469ff6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T21:44:28-07:00
flow: {kind: quick, provenance: inferred, spec: "14fe2079", done: "fb2b11fe"}
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

The registry is liveness-recomputed by design (`Couch.Liveness`, "Liveness is
recomputed rather than stored, so the registry accumulates records"). Nothing
removes an actor record when its child exits normally: `Forget` runs only from
`Stop`, and `PruneDead` only from `spawnResolved`. So the contract is that
**every reader checks liveness**. `classifyForAction` is the one reader that
doesn't: it passes every record with a pid and identity as positive `hosted`
proof. The switcher menu builds its hosted list from panes whose child is still
running (`snapshotMenuObservationsLocked`), and that difference is why the
switcher's archive action said "live" while `--list` said "parked".

Warm-reattach's double record is the same root cause, not a separate bug.
`launchTrackedThread` inserts a fresh record each time (`launch_existing.go`),
the previous reattach's record stays until a fresh spawn prunes it, and the
resume path never prunes.

Changes:
1. `classifyForAction` counts a registry actor of the target thread as hosted
   only when `Liveness == Live`. A `Dead` record is skipped. `Unknown` fails
   closed as `unusable/unknown`, which archive refuses with "retry" and
   switch-agent refuses too, so a probe that can't answer never unlocks an
   action.
2. ARCH-FUNERAL: actor records are created by `spawnResolved` and
   `launchTrackedThread`, the last thing that needs one is its process, and
   `PruneDead` removes them. Today only fresh spawn prunes, so a console that only
   resumes grows the registry by one record per resume. `launchTrackedThread`
   should prune too, before it inserts, which bounds the registry to the
   live actors plus records whose liveness is unknown.

## Done when

- A test: a thread whose only registry actor is dead and whose record is parked
  classifies as parked through `classifyForAction`, and archive admits it.
- A test: an actor with `Unknown` liveness classifies as `unusable/unknown`, and
  archive refuses it.
- A test: launching an existing thread prunes a dead prior record, so the
  registry holds one record for that thread rather than two.
- The existing hosted-actor test marks its actor live, so it still covers the
  live refusal.

## Plan

- [x] Write failing tests: dead actor → archive admits; unknown actor → refuses as unknown; resume prunes the dead record
- [x] Filter by liveness in `classifyForAction`; prune in `launchTrackedThread`
- [x] `go test ./cmd/internal/couchcore/` passes, then the full `make test`

## Log

### 2026-10-01
- 2026-10-01: closed — TDD: 4 couchcore tests (parked thread + dead actor classifies parked and archives; unknown actor refused as unusable/unknown; live+unknown actors classify live; resume reaps dead prior record) fail before, pass after; couchcore/couchtty/couchcmd green; deadsymbols green; test-changelog green (scratchpad TMPDIR); remaining go failures (TestBareCouchInstalledCommand, TestProductionArtifactReferencesAreExactlyClassified, TestCouchReferencesLocalArchiveLocatorRoundTrip) reproduce on origin/main; review verdict: SHIP

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
- Implemented. The three new tests failed before the fix (dead actor read as
  `live`, unknown actor read as `live`, and resume left `[previous-launch,
  couch-ah8d]`), and pass after it. ARCH-DRY: `withoutDead` is the single prune
  rule. `PruneDead` and `launchTrackedThread` share it, and the separate prune
  in `spawnResolved` is gone, because spawn launches through
  `launchTrackedThread`. The two existing hosted-actor fixtures (archive and
  switch-agent) were "live" only because of this bug, since their pid was never
  marked alive. They now share `registeredActorFixture` and set the pid live.
- The full suite caught `PruneDead` left with no production caller
  (`TestNoProductionSymbolIsReferencedOnlyByTests`). I deleted it. `withoutDead`
  is the only prune, and the unknown-fails-closed test now targets it.
  Failures present on `origin/main` as well, checked in a main worktree:
  `TestBareCouchInstalledCommand`, `TestProductionArtifactReferencesAreExactlyClassified`,
  `TestCouchReferencesLocalArchiveLocatorRoundTrip`.
- Close review round 1 (FIX-THEN-SHIP). BR-1: the early `unknown` return
  skipped `ClassifyThread`'s precedence, under which live and busy outrank
  Unproven. Unknown registry actors now join `evidence.Unproven`, so precedence
  lives in one place, and `TestALiveRegistryActorOutranksAnUnprovableOne` pins
  it. BR-2: the dead-actor test now uses the real parked fixture
  (`createParkedThreadInCouch` plus an established binding) and asserts
  `ThreadParked`. BR-3: the atlas now limits its claim to readers that take a
  record as hosting proof, and the `withoutDead` comment states the bound holds
  per launch.
  Out of scope: `Couch.Liveness` and `observeExactProcess` overlap, which
  predates this issue.
