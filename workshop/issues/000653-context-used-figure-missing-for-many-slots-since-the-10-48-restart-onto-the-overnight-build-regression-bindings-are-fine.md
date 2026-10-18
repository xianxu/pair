---
id: 000653
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9a6b5ae8894c15f54cc02ca0bf10b88120694c22' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T14:37:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0e44a709", done: "0499a5ac"}
actual_hours: 0.79
---

# Context-used figure missing for many slots since the 10:48 restart onto the overnight build (regression; bindings are fine)

## Problem

## Spec

Operator report, 2026-10-17 ~12:20: the per-slot "context used" figure (the agent's context-window usage, shown per slot in Couch) is missing for many slots, ops:0 included. It's been missing since the 10:48 Couch restart onto pair:0's build of the night's landings (15d84ef7, later rebuilds up to 145f38cd).

What's known:
- It isn't a binding problem, at least for ops:0. Its ledger (`repos/aef274ee2fbbcabd/ledger-1-ops-33.jsonl`) has the relaunch's launch record (resume, launch ordinal 32) and a binding record (correlation, root_native_id 20b1e863-53a4-4cba-8682-3489e12d52cc), and `--resume-candidates` reports an established native binding.
- It affects many slots, Claude and Codex both, and the relaunched and the merely restarted alike. So it looks like a regression in that build, not one slot's state.

Suspects among the landings in that build, unverified:
- pair#626 (96a28cca): the shared JSONL framer now searches only newly appended bytes. If the context reading goes through it, a record split across reads, the offset bookkeeping or the first record could be dropped.
- #625 (8013b93c): fading from Settled and last-work time.
- #632 (1606ded7): BackgroundWork on hello-v4.
- #641 (74fdb8e2): the erased-cell projection in terminalRowText, if context is read from the footer.
- #618 M2: peer turns now written into the Pair log.

## Done when

- Root cause named with file:line, and the exact build where it regressed; check against a pre-restart build if possible.
- A regression test that fails on the broken code: a slot whose transcript (or whatever the source is) grows across reads still shows its context figure.
- Fixed and landed; after the next restart, every bound slot shows its context figure. Checked live.

## Plan

- [x] RED: `TestTitleProjectionGrowthNeverStopsTitle` (admit + tail): a transcript line appended during a read must not stop the projection; usage surfaces within two polls. Red on main (admit).
- [x] Fix in the shared projection (TL placement rule, so pair#649 inherits it): growth racing admission resets and re-admits on the next Poll; every other failure stays terminal.
- [x] One growth rule, not per-site checks (close review BR-1, BR-5): `classifyAdmission` turns an admission's ErrArtifactChanged into a re-admit when every artifact it observed (stat'd before admission's first read: a proof's artifacts, or for a proofless binding the ones ValidateNamedContent's walk selects) is unchanged or only grown. Tests: `GrowthNeverStopsTitle/stale-proof` (transcript grown past its proof, empty generation token, append racing validation; red on the round-1 fix), `/proofless`, `GrowthAtEachAdmissionSite` (post-fold and boundary, proof and proofless bindings).
- [x] Rewrite `TestTitleProjectionAdmissionRejectsSnapshotDrift`: keep the invariant (no drifted pin), drop the stickiness (the next Poll re-admits from offset 0).

## Log

### 2026-10-17
- 2026-10-17: closed — RED: GrowthNeverStopsTitle/admit and /stale-proof fail on main code; /proofless fails without the proofless baseline. GREEN: go test -race sessioninventory + titlepoller pass, wrapcmd passes. Mutation-checked: the classifier and both baseline branches are each pinned. Full suite exit 0 on f9f2302f; later commits change only the title projection and its tests, and merge reruns the suite. Live: 1-pair-6 probe 2/100 terminal on main, 0/100 with the fix. Post-restart fleet check follows landing (BR-4 disputed).; review verdict: SHIP

- Root cause: #617 (PR #383, 0fe70c1c, landed 2026-10-16 19:13) replaced the stateless per-tick `contextcmd.Run` in the title poller with the stateful `sessioninventory.TitleProjection`. Its admission rejected ordinary transcript growth as `ErrArtifactChanged` (main `title_projection.go:144` `len(full) != 1` from ValidateTargetWork, `:189` `current.Size != p.rawOffset`, and the current/after fingerprint check), `stop()` (`:60`) makes the first error sticky, and `titlepoller/run.go:152` exits the poller on it; `wrapcmd/title_process.go` never respawns it. Binding confirmation follows an agent turn, so admission routinely races growth: the 10:48 restart was the first fleet-wide relaunch onto a build with #617.
- Live evidence: 1-pair-6 probe, 100 admissions against a live transcript: 2/100 terminal failures on main, 0/100 with the fix.
- Classification agreed with pair:6 (#649): growth is transient; shrink, identity change, boundary-hash mismatch, same-size fingerprint change, short read and failed root validation stay terminal. The poller's exit stays only for a lost owner identity.
- Out of scope, filed: pair#654 (Qoder has no usage parsing), pair#655 (hoprttcmd timing flake).
- Close review round 1 (FIX-THEN-SHIP): BR-1, three re-admit sites survived mutation. Growth sites: added `TestTitleProjectionGrowthAtEachAdmissionSite`. Times appends with a transcript stat/read op log from a clean control admission. Disabling each of the three growth sites now fails a test. Same-launch rebind: dropped instead of tested. `sessionledger.generation` (record.go:622) turns a second root under one launch into `Conflict`, which `owner()` already rejects, so a non-conflicting different root after admission is unreachable. The reclassification was dead code that loosened a terminal check; `owner()` is back to main's behavior, and the existing "conflict" case still pins it.
- Minors: TitleProjection comment now lists every terminal class; `onlyGrew` notes that a genuine validation failure surfaces once growth pauses. agy database growth (p.other in checkArtifacts) is still terminal; same family, outside Claude/Codex scope, filed as pair#656. The live post-restart fleet check stays open per Done when; it waits on the next Couch restart onto a build with this fix.
- Close review round 2 (REWORK): BR-5 Critical. Growth during ValidateBindingProof's fallback full read still stopped the title. That path runs when the generation token is empty (darwin st_gen 0) and the transcript has grown past its proof, the usual production state. The per-site grewFrom/onlyGrew checks are replaced by one rule, `classifyAdmission`, applied to admit's error. The baseline is the proof artifacts stat'd before validation, then the validated observations. Mutation-checked: removing the classifier, the proof pre-stat, or the observe call each turns tests red; main's code is red on admit and stale-proof. Live bindings: 101 of 103 carry a proof. The 2 proofless chosen-id ones are covered after validation; a growth racing their storage walk inside ValidateNamedContent stays terminal (no pre-read baseline exists there without a second walk). BR-6: comments now state the one rule instead of listing cases.
- Close review round 3 (FIX-THEN-SHIP): BR-7, proofless admissions had no pre-read baseline. The rule is now uniform: `baseline` stats what admission is about to validate before its first read, and the post-validation observe is gone. For a proofless binding it uses the Observe+Select that ValidateNamedContent uses, one extra walk per admission. Mutation-checked: emptying either branch of `baseline` turns its tests red.
