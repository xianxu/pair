# Boundary Review — pair#228 (whole-issue close)

| field | value |
|-------|-------|
| issue | 228 — A reattach asks every pair session for its clients |
| repo | pair |
| issue file | workshop/issues/000228-a-reattach-asks-every-pair-session-for-its-clients.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4b21a6b534ef9ceb6cd327769cce1457a084a1c7..0979584e0aacfeac2e3df2b71a2a2ea8bad23bba |
| command | sdlc close --issue 228 |
| reviewer | claude |
| timestamp | 2026-09-11T09:02:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The narrowing is real, correct, and genuinely well-pinned. I verified the counted invariant independently rather than taking the Log's word for it: I rebuilt the head tree in a scratch copy and reverted four of the claimed fixes one at a time — `PairSession`→`SnapshotContext` (red: 21 `list-clients` where 0 wanted; end-to-end 5 @ S=3, 24 @ S=22), `DetachedSessions`→`SnapshotContext` (red: 22 where 1/2 wanted), `sessionsFor`→always `rt.Sessions()` (red: 1 full snapshot where 0 wanted), `acceptingLiveNames` removed (red: 1 name probe where 0 wanted), and the fake's `SessionLiveness` dropping `sessionsErr` (red: `TestRunLaunchSessionsErrorExits`). Every one died as named — none of these tests is a mock reasserting the implementation. I also re-derived the `list-clients` producer enumeration by grep and found exactly the three the plan names, and confirmed every `State`-reading consumer in `launcher` is safe under `SessionLive`. Nothing blocks the boundary. The one Important finding is a class-sweep gap: the diff added a loud refusal guard at `DecideLaunch` for "attach state that was never asked," but left the *other* pure consumer of attach state — `ProjectDetachedSessions` — with no equivalent guard, so the same drift there would silently read `SessionLive` as "not detached."

## 1. Strengths

- **`ZellijSource.snapshot(ctx, ask, keep)`** (`cmd/internal/launcher/zellij.go:67`) is the right shape: one parser, one call pattern, three one-line entry points, and the expensive decision expressed as two injected predicates instead of three hand-rolled clients. `ARCH-DRY` and `ARCH-PURE` both land here.
- **`launchShapeOf` as the single switch** (`decision.go:48`, read by `sessionsFor` at `createflow.go:878`) — `DecideLaunch`'s branches *are* that switch's arms, so the predicate and the branches cannot drift apart. The `pair -- x` row in `TestLaunchShapeNamesTheBranchEachArgsShapeTakes` (`decision_test.go:143`) is the one row that catches the plausible `AgentExplicit` mistake, and it's there.
- **`sandboxedChecker`'s PATH shim plus the `LookPath` assertion** (`artifactcollision_zellij_test.go:31-35`) — the guard for the guard. With correct code no route execs a bare `zellij`, so a dropped shim would be unobservable; asserting the shim resolves to the stub makes the sandbox itself falsifiable. This is the right answer to plan-gate PQ-1, fixing the class (every IO seam) rather than the instance.
- **`TestWarmResumeAsksTwoSessionsForClientsWhateverTheHostHas`** counts the *path*, not the functions, at two population sizes. That is the only test shape that would have caught the missed sixth site, and my mutation runs confirm it fires on both couchcore narrowings.
- **`acceptingLiveNames`** (`session_index.go:354`) wraps the acceptor instead of feeding its length bracket, so every bound `#215`'s bracket records is still a real probe result — and the resulting behaviour change (a probe false-negative on a live name no longer mints a second session) is stated in the comment as intended, not discovered later.
- `PairSessionBinding` exposing only `Name`/`Present` (`artifactcollision.go:15`) means site 3's liveness switch is safe *by the type*, not by an audit of its nine callers.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/couchcore/detachedsessions.go:68` — the `SessionLive` refusal guard was added to one consumer of attach state, not the class.** `ARCH-PURPOSE`, `ARCH-ORDER`.

`DecideLaunch` (`decision.go:89-93`) refuses loudly when a snapshot that never asked reaches a branch that reads attach state, with the comment "this fires only on drift." The tree has exactly two pure consumers of that distinction, and the second one — `ProjectDetachedSessions`, which does `state[binding.SessionName] != launcher.SessionDetached` — got nothing. Today it's unreachable (`SnapshotSessionsContext` passes `ask == keep`, so no kept session is ever `SessionLive`), but that invariant is only a comment at `artifactcollision.go:280-283`. If a later change hands `DetachedSessions` a liveness snapshot — exactly the change this issue just made to its sibling `PairSession` two functions above — every thread reads "not detached," couch's resume refuses with "no detached session," and the operator sees a *proof* that is a fabrication rather than a refusal.

Fix sketch (cheap, one seam): in `DetachedSessions`, after `c.Zellij.SnapshotSessionsContext(ctx, names)`, mirror `DecideLaunch`'s loop — `for _, s := range sessions { if s.State == launcher.SessionLive { return nil, fmt.Errorf("detached proof needs attach state, but session %q was observed for liveness only", s.Name) } }` — with a unit test on `ProjectDetachedSessions` pinning the fail direction. The stronger form (split `[]Session` into a liveness type and a classified type so the compiler refuses) is a larger ripple; see §6.

## 4. Minor findings

- `zellij.go:68-72` — `LivenessContext` still runs **two** `list-sessions`, but `--no-formatting` already carries both the names and the `EXITED` marker (`exitedSessions` parses `fields[0]`), so `--short` is redundant for every form and is 50% of liveness's entire cost. The reattach path now pays 10 `list-sessions` (~35 ms each) where 5 would do — ~175 ms of the 272–282 ms the probe measured. Not a defect and outside the issue's stated promise (which is a `list-clients` count), but it's the next obvious win on the same path.
- `atlas/architecture.md:295` — says the full form "is used by bare `pair`'s picker and `pair list`", but `pair list` goes through `OSRuntime.ListSessions`/`buildListRowsForScope` (`osruntime.go:191-206`), a *separate* `list-clients` producer that never touches `ZellijSource.snapshot`. It also omits the two remaining `SnapshotContext` callers (`rename.go:171`, `createflow.go:261`, the layout-conflict re-check). A reader grepping from the atlas will not find what it describes.
- Task 6's stale-claim sweep missed two spots: `lifecycle.go:386-388` — the plan says `liveTagsForSweep`'s doc comment "gains one line" recording that it reads names only; it didn't (the claim lives only at the caller, `createflow.go:69`), so the next person who adds a `State` read there won't see it. And `osruntime.go:111` `ProbeSessionName` has no pointer to the site-6 short-circuit that now skips it for live names.
- `session_quiescence_live_test.go:163` (`TestSessionDetachLive`, `PAIR_LIVE_COUCH=1`) is the live conformance check for `ZellijSource`, and it exercises only the full form. Adding `SnapshotSessionsContext(ctx, []string{session})` and `LivenessContext(ctx)` to the same fixture would pin the narrowed forms' classification against the real binary — which is exactly Done-when bullet 2's claim, currently asserted only against `StubZellij`. `ARCH-MOCK`.
- `cmd/probes/reattachcost/main.go:355` — `marker` is derived as the substring after the last `-` in the session name, and the names are `pair-rc<pid>-<i>`; correct today, but it silently couples the marker to the name shape two functions apart.

## 5. Test coverage notes

- Suite state I actually ran: `go build ./...` clean; `go vet` clean on `launcher`, `couchcore`, `pairlifecycletest`, `probes/reattachcost`; `go test` green on `./cmd/internal/launcher/`, `./cmd/internal/artifactpath/`, `./probes/zellijprobe/`, and every new + adjacent couchcore test (`TestDetachedSessions*`, `TestPairSessionReadsLivenessOnly`, `TestWarmResumeAsks*`, `TestProjectDetachedSessions`, `TestActionableInventoryAsksOnlyAboutDetachCandidates`, `TestResumeContextDerivesTheDetachedProofItself`). A full `./cmd/internal/couchcore/` run fails in `ptychild`/`ptyrunner`/`TestSpawnPostAckFailuresQuiesce*` with `operation not permitted` — an exec denial in this environment, untouched by this diff (the Log's unsandboxed `make test` exit 0 is consistent with that).
- The mutation sweep's claims hold. I reproduced 4 of the 12 rows independently (the two couchcore snapshot swaps, `runOnce` always-full, and the `accepts` short-circuit removal) plus the fake's `sessionsErr` row; all died as the table names.
- **One claim is asserted in a comment and cannot currently be falsified:** `artifactcollision.go:282-283` says the narrowed snapshot preserves `ProjectDetachedSessions`' duplicate-row check "since both rows of a duplicated name pass the same filter." `StubZellij` takes a `map[string]string`, so it structurally cannot emit a duplicated session row — the claim is true by inspection of `snapshot`'s per-line `keep`, but no fixture can test it. If that matters, `StubZellij` would need a raw-lines escape hatch.
- `SnapshotSessionsContext` with an empty `names` slice isn't covered (it would return nothing after 2 `list-sessions`). Unreachable today — `DetachedSessions` returns early when `bindings` is empty — but it's the kind of edge a new caller hits first.

## 6. Architectural notes for upcoming work

- **`SessionState` now conflates two kinds of fact.** `attached`/`detached`/`exited` are *observations*; `live` is "this observation was not made." Carrying both in one string type means every consumer of the enum must now be audited for the unasked case, and the diff's answer is a runtime refusal at one site. The structural answer is two types — a liveness row and a classified row — so a liveness snapshot cannot be passed where attach state is read, and `DecideLaunch`'s guard and the missing `ProjectDetachedSessions` guard both become compile errors. That's the `ARCH-ORDER` "collapse the constellation into a tagged type" move, and it's the shape worth considering before a third snapshot form appears.
- **`workshop/issues/000191-parallelize-zellij-session-snapshot.md` is now substantially stale.** Its Problem section is built on couch's blocking startup inventory paying ~1.4 s for a full `list-clients` fan-out — the exact cost this issue removed (that path now asks only candidates). #191 still has a real remainder (bare `pair`'s picker and `pair list` are genuine full scans), but its framing and its measured motivation both need rewriting against the post-#228 tree before anyone estimates it.
- **`launcher.Run` (`run.go:39`) is test-only.** It builds a `SessionSnapshot` from `SessionSource.Snapshot()` and calls `DecideLaunch` — the same decision `runOnce` makes — but nothing in production calls it, so it silently sits outside the `sessionsFor` narrowing and outside the `launchShape` ownership story. Worth checking against `#192` (production symbols reachable only from tests).
- `ARCH-SECURE` pass, worth recording as confirmed-good: session names reaching `SnapshotSessionsContext` come from the persisted session-name index (an artifact an older version may have written) and are passed as argv elements to `exec.CommandContext`, never through a shell — a hand-edited index cannot inject. The narrowing strictly *reduces* what gets queried. `ARCH-CONSTRAINTS` pass: the promise is a count asserted at two seams at S=3 and S=22, timings are quarantined in the Log with co-tenancy, and every zellij call in the probe is bounded (with the 10-minute hang recorded as the reason).

## 7. Plan revision recommendations

The plan still matches the code; two small entries would keep it honest:

- **`## Revisions` — implementation delta.** Task 3 Step 2 promises `liveTagsForSweep`'s doc comment "gains one line saying so"; the line landed at the caller (`createflow.go:68-70`) instead. Task 3 also names the helper `snapshotFor`; it shipped as `sessionsFor`. Record both so the Core-concepts table and the tree agree.
- **`## Revisions` — the guard's scope.** The Core concepts entry for `launchShape` describes the `anyLive` guard as *the* protection against reading unasked attach state. Once the Important finding above is dispositioned, record that the class has two members and which of them are guarded — otherwise the plan reads as if the hazard is closed tree-wide.

Two non-plan reminders for the close: the issue's Done-when bullet 5 (`#206` is unblocked) is still outstanding — `workshop/issues/000206-…md` reads `status: blocked`, `deps: [000228]` — and the plan correctly sequences that after `sdlc merge`, so it is deferred, not missed. And Task 6's operator smoke (`make install`, then a real couch detach/reattach) is the one Done-when item no test can stand in for.
