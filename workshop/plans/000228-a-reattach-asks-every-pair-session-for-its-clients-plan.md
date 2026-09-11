# A Reattach Asks Every Pair Session For Its Clients — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the client-count work a reattach does scale with the threads it touches, not with the sessions that exist. Measured today at about 5.5 s per reattach (`#206` Log). After the change the critical path is 10 `list-sessions` (about 35 ms each) + 2 `list-clients` (about 250 ms each against real detached sessions) + process spawns + the attach (about 55 ms): roughly **1 s**, and, the actual promise, independent of the session count.

**Architecture:** One zellij snapshot routine gains a filter for *which sessions get asked for their clients*. Two narrow entry points use it:
- `SnapshotSessionsContext(ctx, names)` asks only the named sessions;
- `LivenessContext(ctx)` asks none, reporting non-exited sessions as a new `SessionLive` state.

Couch's detached proof asks only the candidate thread's session. `pair resume`'s launcher takes liveness only, unless its decision actually reads attach state; that is bare `pair`'s picker, and one predicate beside `DecideLaunch` owns the rule. Every claim is tested as a **count**, never as a timing. Sites 1–4 are counted at the `ZellijSource.Path` stub seam; sites 5–6 are counted at the launcher's fake-Runtime seam (`sessionsCalls` / `livenessCalls` / `probeCount`).

**Tech Stack:** Go. `cmd/internal/launcher` (zellij snapshots, `DecideLaunch`, `runOnce`), `cmd/internal/couchcore` (`DetachedSessions`), `cmd/probes/reattachcost` (the measurement, shipped here). Stub-zellij tests follow `zellij_test.go`'s `TestZellijSourceClassifiesSessions`.

Issue: `workshop/issues/000228-a-reattach-asks-every-pair-session-for-its-clients.md`. Baseline: `#206` Log, 2026-09-10.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SessionLive` | `cmd/internal/launcher/session.go` | new |
| `ZellijSource.snapshot` (filtered core) | `cmd/internal/launcher/zellij.go` | new (extracted from `SnapshotContext`) |
| `SnapshotSessionsContext`, `LivenessContext` | `cmd/internal/launcher/zellij.go` | new |
| `launchShape`, `decisionNeedsAttachState` | `cmd/internal/launcher/decision.go` | new |
| `sessionBlocksReuse` / `DecideLaunch` | `cmd/internal/launcher/decision.go` | modified |
| `ScopedThreadArtifactCollisionChecker.Zellij` | `cmd/internal/couchcore/artifactcollision.go` | new field |

- **SessionLive**: "not exited; whether a client is attached was not asked". A liveness snapshot reports it for every non-exited session. It is a separate value, not a guess at `SessionDetached`, because the consumers that care about attach state (`hasDetached`, `pick.go`) compare against `SessionDetached` and would silently misread a guess.
  - **Relationships:** produced only by `LivenessContext`. `sessionBlocksReuse` treats it as non-exited, like attached or detached.
- **`ZellijSource.snapshot(ctx, ask func(name string) bool, keep func(name string) bool)`**: one call pattern and one parser for all three entry points (`ARCH-DRY`).
  - It lists sessions twice, for names and for exited status, as today.
  - For each pair session with `keep(name)`, it runs `list-clients` only if `ask(name)` and the session is not exited. Otherwise the state is `SessionLive`, or `SessionExited`.
  - The three entry points:
    - `SnapshotContext` is `snapshot(ctx, all, all)`, unchanged in behaviour;
    - `SnapshotSessionsContext(ctx, names)` is `snapshot(ctx, in(names), in(names))`;
    - `LivenessContext` is `snapshot(ctx, none, all)`.
- **`launchShape(args LaunchArgs)`**: an enum (`shapeSelected`, `shapeForced`, `shapeExplicitArgs`, `shapeBare`), computed by ONE switch in the order `DecideLaunch` tests today:
  - `SelectedTag != ""`;
  - `ForcedTag != ""`;
  - `Agent != "" && AgentArgsExplicit`, exactly as at `decision.go:39`. It is not `AgentExplicit`: `pair -- x` parses to a defaulted agent with `AgentArgsExplicit` true (`args.go:121-142`) and must create, not pick;
  - bare.

  `DecideLaunch` **switches on `launchShape(args)`**, so its branches are that switch's arms. `decisionNeedsAttachState(args)` is `launchShape(args) == shapeBare`. The predicate and the branches are one switch, not two agreeing ones. `runOnce` reads the same predicate to choose its snapshot.
  - **The guard:** if the predicate is true and the snapshot holds any `SessionLive`, `DecideLaunch` returns an error instead of reading attach state that was never asked. That combination is unreachable by construction; the guard makes a future drift loud.
- **The checker's `Zellij` field**: a `launcher.ZellijSource` whose zero value is the real zellij. It exists so tests can point `DetachedSessions` at a stub and count `list-clients`. Production constructs the checker as today, zero value included.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ZellijOps.SessionLiveness` | `cmd/internal/launcher/runtime.go` (+ `osruntime.go`) | new | `ZellijSource.LivenessContext` |
| `pairlifecycletest.StubZellij` | `cmd/internal/pairlifecycletest/zellij_stub.go` | new (test support, shared by launcher + couchcore tests) | the zellij CLI |
| `fakeRuntime.SessionLiveness` | `cmd/internal/launcher/createflow_test.go` | new | — |
| `ScopedThreadArtifactCollisionChecker.PairSession` | `cmd/internal/couchcore/artifactcollision.go` | modified (liveness, not a full snapshot) | `Zellij.LivenessContext` |

- **`SessionLiveness`** is the launcher's liveness snapshot, used by the orphan-nvim sweep (site 4) and by `runOnce` when `decisionNeedsAttachState` is false (site 5).
  - **The fake must model it faithfully** (lesson "The double must model the state the fix reads"). It returns the fake's sessions with every non-exited state mapped to `SessionLive`. It also counts calls to `Sessions()` versus `SessionLiveness()`, so a test can assert that a forced-tag launch takes no full snapshot.
- **`StubZellij(t, sessions) (path, log string)`** and **`CountCalls(t, log, needle) int`**, in `pairlifecycletest`. That package is already test support: launcher tests import it, and it imports neither launcher nor couchcore. A stub `zellij` script answers `list-sessions --short`, `list-sessions --no-formatting` and `--session X action list-clients` from a name → state map, and logs every invocation, as `TestZellijSourceClassifiesSessions` does today. It is a non-test `.go` file under `cmd/`, so it joins `artifactpath`'s `NonArtifactSources` inventory (`manifest.go`).

### Sites after the change (the counted invariant)

Recounted by the plan review, which found two sites the first draft missed (3 and 6):

| # | site | reads | before | after |
|---|---|---|---|---|
| 1 | couchcore `ResumeContext` → `DetachedSessions` | this thread's session: live and zero clients | 2 `ls` + S `lc` | 2 `ls` + 1 `lc` |
| 2 | couchcore `confirmStillDetached` → `DetachedSessions` | the same, re-proved | 2 `ls` + S `lc` | 2 `ls` + 1 `lc` |
| 3 | couchcore `awaitResumeRegistration` → `PairSession` (`launch_existing.go`) | `State != SessionExited` only (`artifactcollision.go`) | 2 `ls` + S `lc` per poll (≥1) | 2 `ls` per poll |
| 4 | launcher orphan-nvim sweep (`createflow.go` loop top) | names only (`liveTagsForSweep` never reads `State`) | 2 `ls` + S `lc` | 2 `ls` |
| 5 | launcher `runOnce`, forced tag | exited-versus-not, and `sessionBlocksReuse` | 2 `ls` + S `lc` | 2 `ls` |
| 6 | launcher `AssignSessionName` → `ProbeSessionName` on the prior name | "does zellij accept this name's length" | 1 `lc` on the thread's own session | 0 when that name is live (a live session under it proves acceptance) |
| 7 | couchtty inventory refresh on completion (async) | the whole candidate set | 2 `ls` + S `lc` | 2 `ls` + C `lc` (C = this couch's detach candidates ≤ S) |

`ls` is `list-sessions` (about 35 ms); `lc` is `list-clients` (about 250 ms against a real detached session). The reattach's critical path goes from 10 `ls` + (5S + 1) `lc` to 10 `ls` + 2 `lc`, independent of S. The review's end-to-end test measured the couchcore side at 6 `ls` + 2 `lc` for both S=3 and S=22. The two remaining `lc` are the detached proof itself, asking only its own session twice by design. Site 7 narrows as a side effect.

Not on the warm path, and left alone:
- `createflow.go:259`: layout conflict only; couch sends no layout flag on warm reattach.
- `ProbeLiveLayout`: `list-panes`, only when the layout record is missing.
- the title poller: `list-sessions --short` only.

The failure-path `PairSession` calls (`launch_existing.go:181` `failTrackedPostAckStart`, `:220` `diagnoseRegistrationFailure`) are NOT left alone. They call the same `PairSession` that site 3 narrows, so they inherit liveness. Both read only `Name`/`Present`, so that is correct.

**All three `list-clients` producers in the tree** (plan gate PQ-2):
- `ZellijSource.clientCountContext`: the snapshot, narrowed here by the filter.
- `ProbeSessionName`: site 6, skipped for live names.
- `OSRuntime.ListSessions` (`osruntime.go:197-199`): one `list-clients` per pair session on every `pair list`. This is a **justified full scan**, because the row renders "attached (N clients)", so every session's client count is the output. It is left alone, and Task 6's comment sweep visits it only to confirm its cost comment is accurate.

`createflow.go:259` and `rename.go:171` could also take liveness; they are out of scope because they are not the reattach path. Narrowing `PairSession` (site 3) also speeds detach and park, which call it.

---

## Chunk 1

### Task 0: Enter implementation

- [ ] After approval, run `sdlc change-code --issue 228`. The probe (`cmd/probes/reattachcost`, currently uncommitted in the tree) carries into the branch.

### Task 1: One filtered snapshot, and `SessionLive`

**Files:** `cmd/internal/launcher/session.go`, `cmd/internal/launcher/zellij.go`, `cmd/internal/launcher/zellij_test.go`, new `cmd/internal/pairlifecycletest/zellij_stub.go`, `cmd/internal/artifactpath/manifest.go` (the stub's `NonArtifactSources` entry)

- [ ] **Step 1: Failing tests** (`zellij_test.go`). Create `cmd/internal/pairlifecycletest/zellij_stub.go` with `StubZellij(t, sessions map[string]string) (path, log string)` (name → `"attached"`/`"detached"`/`"exited"`) and `CountCalls(t, log, needle) int`. Add it to `artifactpath`'s `NonArtifactSources`. Then (below, `stubZellij`/`countCalls` are those two):

```go
// The counted invariant, at the seam: asking about ONE session costs one
// list-clients, however many sessions the host has (pair#228).
func TestSnapshotSessionsAsksOnlyTheNamedSessions(t *testing.T) {
	sessions := map[string]string{"pair-a": "detached", "pair-b": "detached", "pair-c": "attached", "pair-d": "exited"}
	for i := 0; i < 20; i++ {
		sessions[fmt.Sprintf("pair-x%02d", i)] = "detached"
	}
	path, log := stubZellij(t, sessions)
	got, err := ZellijSource{Path: path}.SnapshotSessionsContext(context.Background(), []string{"pair-a", "pair-c", "pair-d", "pair-absent"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{Name: "pair-a", State: SessionDetached}, {Name: "pair-c", State: SessionAttached}, {Name: "pair-d", State: SessionExited}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if n := countCalls(t, log, "list-clients"); n != 2 { // a and c; d is exited, absent is absent
		t.Fatalf("list-clients calls = %d, want 2 regardless of the 20 other sessions", n)
	}
}

// Liveness asks nobody for clients, and says so in the state it reports.
func TestLivenessAsksNoSessionForClients(t *testing.T) {
	path, log := stubZellij(t, map[string]string{"pair-a": "detached", "pair-b": "attached", "pair-c": "exited"})
	got, err := ZellijSource{Path: path}.LivenessContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{Name: "pair-a", State: SessionLive}, {Name: "pair-b", State: SessionLive}, {Name: "pair-c", State: SessionExited}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if n := countCalls(t, log, "list-clients"); n != 0 {
		t.Fatalf("list-clients calls = %d, want 0", n)
	}
}
```

  `TestZellijSourceClassifiesSessions` stays, unmodified in its assertions, as the full snapshot's pin.

- [ ] **Step 2: Run to confirm red** (the new functions and state are undefined).
- [ ] **Step 3: Implement.** In `session.go`:

```go
	// SessionLive is a session that has not exited, whose attach state was NOT
	// ASKED: a liveness snapshot reports it instead of guessing detached. It
	// exists because asking costs a list-clients per session, about 250 ms each
	// against a real detached one (pair#228), and most callers only need "not
	// exited". A consumer that needs attached-versus-detached must take a full
	// snapshot; DecideLaunch refuses to read attach state from a liveness one.
	SessionLive SessionState = "live"
```

  In `zellij.go`: move the body of `SnapshotContext` into `snapshot(ctx, ask, keep)`, as specified in Core concepts. The three entry points become one line each. Update the cost comment on `SnapshotContext` and name the two narrower forms.
- [ ] **Step 4: Run green**, including `TestZellijSourceClassifiesSessions`.
- [ ] **Step 5: Commit** `#228: one filtered zellij snapshot; asking about one session costs one list-clients`.

### Task 2: `DecideLaunch` owns which decisions need attach state

**Files:** `cmd/internal/launcher/decision.go`, its tests

- [ ] **Step 1: Failing tests.**
  - For each of `ForcedTag`, `SelectedTag` and explicit agent + args, `DecideLaunch` over a liveness snapshot decides exactly as it does over the equivalent full snapshot. This is a differential test: `SessionLive` versus `SessionDetached`/`SessionAttached`, with the same `Action`, `Tag` and `SessionName`.
  - Bare args over a liveness snapshot holding a live session return an error that names the missing attach state.
  - A table test pins `launchShape`/`decisionNeedsAttachState` against the branch each args shape takes. It includes the **`pair -- x` row** (`Agent` defaulted to claude, `AgentExplicit` false, `AgentArgsExplicit` true), which must be `shapeExplicitArgs`. The review applied the `AgentExplicit` mutation and the entire existing launcher suite stayed green, so this row is the only thing that catches it.
- [ ] **Step 2: Implement.**
  - `sessionBlocksReuse` treats `SessionLive` as non-exited.
  - Add `decisionNeedsAttachState(args)`.
  - Restructure `DecideLaunch` as `switch launchShape(args)`:
    - the `shapeSelected`, `shapeForced` and `shapeExplicitArgs` arms carry today's three early branch bodies, verbatim;
    - after the switch come the `anyLive(snap)` guard (`return LaunchDecision{}, fmt.Errorf(...)`) and then today's picker and create tail.
  - No branch body changes. `decisionNeedsAttachState(args)` is `launchShape(args) == shapeBare`.
- [ ] **Step 3: Green; commit** `#228: DecideLaunch owns which of its branches read attach state`.

### Task 3: The launcher takes liveness where that is all it needs (sites 4–6)

**Files:** `cmd/internal/launcher/runtime.go`, `osruntime.go`, `createflow.go`, `createflow_test.go` (fake)

- [ ] **Step 1: Failing tests.** The fake counts `sessionsCalls` and `livenessCalls`. `pair resume <tag>` against a live detached session:
  - makes `sessionsCalls == 0` and `livenessCalls == 2` (the sweep plus `runOnce`);
  - still decides `ActionAttach` on the same session.

  - makes `probeCount == 0` (site 6). The review verified that removing the rule survives the whole existing suite, so this test is the required red.

  Also: `AssignSessionName` with `live` holding the prior name makes no `accepts` call, and a *new* candidate name is still probed (#215's behaviour). A bare `pair` with a detached session still takes a full snapshot (`sessionsCalls ≥ 1`) and reaches the picker. Existing `RunLaunch` tests must pass unmodified. Any that construct sessions only to assert attach behaviour should pass through the fake's faithful liveness mapping; if one does not, that is a finding to record, not an assertion to loosen.
- [ ] **Step 2: Implement.**
  - Add `SessionLiveness() ([]Session, error)` to the `ZellijOps` interface (`runtime.go`). The `OSRuntime` implementation is `ZellijSource{}.LivenessContext(context.Background())`.
  - `createflow.go`'s sweep calls `rt.SessionLiveness()`. `liveTagsForSweep` never reads `State`; its doc comment gains one line saying so.
  - `runOnce`: `sessions, err := snapshotFor(rt, opts.Args)`, where `snapshotFor` returns `rt.Sessions()` when `decisionNeedsAttachState(args)` and `rt.SessionLiveness()` otherwise.
  - The fake's `SessionLiveness` maps non-exited states to `SessionLive`, counts calls, and **returns `f.sessionsErr`**. `TestRunLaunchSessionsErrorExits` uses a forced tag with `sessionsErr`; the review confirmed it fails if the liveness path drops the error.
  - **Site 6.** The rule lives *inside* `AssignSessionName`, which already receives `live`, so both of its callers (`assignLaunchSessionNames` and `assignSingleSessionName`) get it from one place (`ARCH-DRY`). The rule: a name that is a non-exited session in `live` is accepted without calling `accepts`, because zellij has already created a socket under it, which proves the length fits. It wraps `accepts` rather than feeding its `longestOK`/`shortestBad` bracket (`createflow.go:1016-1038`), so every value the bracket records is still a real probe result.
    - **Intended behaviour change, stated:** when the probe gives a *false negative* on a live name (a timeout records a refusal at that length, `osruntime.go:125-127`), today the ladder mints a shorter name and `pair resume` creates a second session. With the rule it attaches to the live one.
- [ ] **Step 3: Green; commit** `#228: pair resume takes liveness, not client counts, for a decision that reads only liveness`.

### Task 4: Couch asks only the candidate's session, and liveness where that is all it reads (sites 1–3, and 7 as a side effect)

**Files:** `cmd/internal/couchcore/artifactcollision.go`, new `cmd/internal/couchcore/artifactcollision_zellij_test.go`

- [ ] **Step 0: The sandbox every new couchcore test uses (plan gate PQ-1).** A test that builds the production checker must redirect EVERY IO seam it holds, not just the one under test. `NewScopedThreadArtifactCollisionChecker` also wires `Sessions: launcher.OSRuntime{}`, a real session deleter. On a red run, the shortened registration timeout drives `failTrackedPostAckStart` → `quiescePostAckStart` → `QuiesceThreadSession` → a real `zellij delete-session` plus a SIGKILL sweep on the developer's machine. One helper, `sandboxedChecker(t, sessions)`, used by every new test:
  - `GlobalDataDir` is `t.TempDir()`, with the index written inline;
  - `Sessions` is `&fakeSessionDeleter{}` (`artifactcollision_test.go`), and every test asserts it recorded **no deletions**;
  - `Zellij.Path` is `StubZellij`;
  - `t.Setenv("HOME"/"XDG_DATA_HOME", tmp)`, so any OS-runtime default path resolves under the temp dir;
  - **the stub's directory is prepended to `PATH`**, so any `zellij` exec from a route this list did not name hits the logging stub instead of the host.

  Every test asserts the stub log holds only `list-sessions` and `list-clients`. A `delete-session`, `kill-session` or anything else fails the test by name. The guard is the PATH shim, not the enumeration, because the enumeration is what the first draft got wrong.
- [ ] **Step 1: Failing tests** (all through `sandboxedChecker`). Build a checker over a temp data dir with a session-name index binding two addresses to `📁repo-a` and `📁repo-b`, plus `StubZellij` listing those two and 20 other live pair sessions. Write the index inline, following `artifactcollision_test.go:150-164` and `:218-233`; no existing inventory test builds one, since they use the fake checker.
  - `DetachedSessions` for `[a]` makes exactly 1 `list-clients` call, and returns `a` when its session is detached.
  - For `[a, b]` it makes exactly 2.
  - A table covers the named session being detached, attached, exited or absent: observed only when detached. That keeps `ProjectDetachedSessions`' semantics.

  - `PairSession(a)` makes **0** `list-clients` calls (site 3), and still reports present/absent/exited correctly.
  - **End to end (the test that would have caught the first draft's miss):** `newTestEnv(t, "/repo")`, then `env.Couch.Artifacts = NewScopedThreadArtifactCollisionChecker(tmp)` with `Zellij.Path` set to the stub and the index written inline. A warm `ResumeContext` then makes exactly **2 `list-clients` and 6 `list-sessions`** across the whole couchcore side (sites 1–3), at both S=3 and S=22. It counts the path, not the functions. Shorten `resumeRegistrationTimeout` in the test, so a bypass of the stub fails fast and legibly instead of after 15 s.
- [ ] **Step 2: Implement.**
  - Add the `Zellij launcher.ZellijSource` field.
  - After the bindings are resolved, collect their `SessionName`s and call `c.Zellij.SnapshotSessionsContext(ctx, names)`.
  - `PairSession` takes `c.Zellij.LivenessContext(context.Background())`. `PairSession(address)` has no ctx, and adding one would ripple through `PairSessionIO`, its fake and 9 callers. It reads only `State != SessionExited`, and `SessionLive` satisfies that. Every caller reads only `Name`/`Present` (`launch_existing.go`, `detach.go`, `park.go`; verified in review round 2).
  - `ProjectDetachedSessions` is unchanged.
  - Update the cost comment ("does not bound N once it does" becomes "asks only the candidates' own sessions").
- [ ] **Step 3: Green.** Run `go test ./cmd/internal/couchcore/` (unsandboxed; its pty tests need it). `TestResumeContextDerivesTheDetachedProofItself` and the warm-reattach tests must pass unmodified.
- [ ] **Step 4: Commit** `#228: couch's detached proof asks only the candidate thread's own session`.

### Task 5: The probe measures before and after, side by side

**Files:** `cmd/probes/reattachcost/main.go`, `Makefile.local`, `cmd/internal/artifactpath/manifest.go` (the probe's `NonArtifactSources` entry), `.gitignore` (`/reattachcost`; `TestEveryMainPackageIsIgnoredAtTheRepoRoot` fails on the untracked probe today)

- [ ] Name the probe's sessions `pair-rc<pid>-<i>`. `isPairSessionName` requires the `pair-` or `📁` prefix, so `reattachcost-…` names are invisible to every snapshot: the first draft's "new pattern" would have measured zero `list-clients`.
- [ ] Split the couch-shaped phase in two, both built from production calls and covering all seven sites:
  - **"old pattern"**: 5 × `SnapshotContext` (sites 1–5) + 1 `list-clients` on the session (site 6), as the critical path ran before this issue. Site 7 is excluded from both patterns;
  - **"new pattern"**: 2 × `SnapshotSessionsContext(ctx, [that session])` + 3 × `LivenessContext`, then the attach.

  One run measures both under identical co-tenancy. The site 7 refresh is noted as async and off the critical path, not simulated.
- [ ] Add a `make test-reattach-cost` target beside `test-zellij-repaint`, with a comment on why it is manual (it creates N+1 real sessions; sandbox off). Add `cmd/probes/reattachcost/main.go` to `NonArtifactSources`. The untracked probe fails `TestProductionArtifactReferencesAreExactlyClassified` today.
- [ ] Record the caveat with the numbers: synthetic sessions answer `list-clients` in about 53 ms, where real detached ones take about 250 ms. The probe's "old" column therefore *understates* the real before-cost, and the count (from the stub tests) is the promise. Log the host's S and the probe's own N+1 separately, because `pair-rc` sessions count in S (and appear briefly in `pair list`).
- [ ] Run with N=8. Record in #228's Log: the table, co-tenancy, and live/detached pair session counts. Commit with Task 6.

### Task 6: Docs and verification

- [ ] **Stale-claim sweep.** Update every comment and doc that states the snapshot's cost as an unqualified "per session on the host":
  - `zellij.go`;
  - `zellij_snapshot_bench_test.go`;
  - `artifactcollision.go`;
  - `actionableinventory.go:440-442` ("per session on the host");
  - the atlas, found with `git grep -n "list-clients\|per non-exited session" -- atlas cmd`.
- [ ] **Mutation sweep, named failures only** (script pattern from #183):

  | mutation | must fail |
  |---|---|
  | `SnapshotSessionsContext` asks every session | `TestSnapshotSessionsAsksOnlyTheNamedSessions` |
  | `LivenessContext` asks clients | `TestLivenessAsksNoSessionForClients` |
  | `DetachedSessions` back to `SnapshotContext` | the couchcore count test |
  | `runOnce` always takes `Sessions()` | the forced-tag count test |
  | `launchShape` drops or reorders its `ForcedTag` arm | the `launchShape` table test |
  | the `anyLive` guard removed | the bare-args error test |
  | `sessionBlocksReuse` ignores `SessionLive` | the forced-tag differential test |
  | `PairSession` takes `c.Zellij.SnapshotContext` (a full snapshot through the stub) | the `PairSession` count test and the end-to-end count test (5 `lc` at S=3, 24 at S=22 in the review) |
  | the `accepts` live-name short-circuit removed | the forced-tag `probeCount == 0` test |
  | `launchShape` checks `AgentExplicit` instead of `AgentArgsExplicit` | the `pair -- x` shape row |
  | the sandbox helper drops the PATH shim (and a test path execs `zellij delete-session`) | the stub-log "only list-sessions / list-clients" assertion; run once deliberately to prove the guard fires |

- [ ] Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test` unsandboxed; it must exit 0.
- [ ] **Operator smoke:** `make install`, then a couch detach and reattach of one thread feels faster; report it.
- [ ] Update #228's Log with the corrected seven-site table: what each site reads, and its before and after counts. That is Done-when bullet 3.
- [ ] Close with `sdlc close --issue 228 --verified '<make test, mutation sweep, probe before/after, operator smoke>'`, then run `sdlc pr` and `sdlc merge --yes`, which flips it to done.
- [ ] Then unblock #206: run `sdlc issue set-status --issue 206 working` and `sdlc issue sync --issue 206`. Record in #206's Log that startup's blocking inventory (`startup.go:132-149`) now costs O(C), not O(S).

---

## ARCH notes

- **ARCH-DRY.** One snapshot routine with two filters, not three hand-rolled zellij clients. One predicate owns "does this decision read attach state". `DecideLaunch` branches on it and `runOnce` reads it.
- **ARCH-PURE.** The classification (`snapshot`'s parse, `DecideLaunch`, `decisionNeedsAttachState`, `ProjectDetachedSessions`) is pure over the zellij output. The shell-out stays in `runContext`, behind `Path`.
- **ARCH-PURPOSE.** The purpose is the invariant, "client work scales with threads touched", on the reattach's critical path. All six critical-path sites (1–6) are narrowed. Site 7 stays out of scope with its reason: the inventory legitimately describes every candidate, and it narrows from S to C anyway.
- **ARCH-CONSTRAINTS.** This is a per-reattach path: operator gestures today, a startup pass in `#206`. Per `workbench-latency` the promise is a **count**: `list-clients` per reattach = 2, independent of S. The timing is Tier 2 and lives in the probe's Log entry with its co-tenancy.
- **ARCH-ORDER.** No new state between events. Narrowing changes which sessions are asked, not when. The proof's re-check (`confirmStillDetached`) keeps its timing.
- **ARCH-SECURE.** Names passed to `SnapshotSessionsContext` come from the session-name index. They are only compared against zellij's own listing and passed as a `--session` argv element, never through a shell. N/A beyond that.
- **ARCH-MOCK.** The seam is the existing `ZellijSource.Path` stub. The launcher fake gains a faithful `SessionLiveness` (it maps states the way the real one does). `BenchmarkZellijSnapshotLive` and the probe are the live conformance checks.

## Revisions

### 2026-09-10 — plan review round 1 (fresh context; applied Tasks 1–4 in a scratch clone)

Reason: the mechanism compiled and passed the existing suites. The reviewer
found one Critical miss, three Important issues and five Minor ones.
Delta:
- **Critical: a sixth snapshot on the critical path.** `awaitResumeRegistration`
  → `PairSession` asks every session for its clients but reads only "not
  exited". The reviewer measured 22 `list-clients` with 22 live sessions.
  - Fix: site 3, liveness. The Done-when would have failed without it.
  - Added the end-to-end `ResumeContext` count test, the one kind of test that
    catches a missed site.
- **Important:**
  - **The name probe (site 6)** is a `list-clients` outside the `Path` seam.
    It is skipped when the name is a live session, which proves acceptance, and
    asserted by the fake's `probeCount`.
  - **The probe's session names** failed `isPairSessionName`, so the "new
    pattern" would have measured nothing. They are now `pair-rc<pid>-<i>`, and
    both patterns cover all seven sites.
  - **The untracked probe fails the source inventory.** It joins
    `NonArtifactSources`, and so does the shared stub.
- **Minor:**
  - the fake's `SessionLiveness` returns `sessionsErr`;
  - the predicate is pinned as `AgentArgsExplicit`, and restated as a
    `launchShape` switch so the predicate and the branches really are one
    switch;
  - `ZellijOps`, not a nonexistent `SessionOps`;
  - where the shared stub lives;
  - the stale comment at `actionableinventory.go:440-442`;
  - the exact close and unblock commands;
  - the per-site Log step.

### 2026-09-10 — plan review round 2 (same reviewer; the new pieces built and run in the scratch clone)

Reason: the design was confirmed. The reviewer checked:
- site 3 is safe for every `PairSession` caller;
- site 6 is sound against #215's bracket and the legacy-name migration;
- `launchShape` reproduces today's order, and the full suite passes with it;
- the shared stub adds no import cycle;
- the end-to-end test builds and counts **2 `lc` + 6 `ls` at S=3 and at S=22**.

Three Important issues and six Minor ones remained.
Delta:
- **Important:**
  - `.gitignore` gets `/reattachcost` (a guard fails on the untracked probe today);
  - Task 2's step is rewritten as the `launchShape` switch (it still described two agreeing conditions);
  - the probe's old pattern is 5 snapshots + 1 `lc`, like for like with the new one.
- **Minor:**
  - the `pair -- x` row is written into Task 2's tests;
  - the site-6 `probeCount` test moved into Step 1 as the required red;
  - the mutation rows were reworded;
  - the arithmetic is now 5S + 1;
  - `PairSession` uses `context.Background()`;
  - the files lists are complete.
- **Advisories taken:**
  - the live-name rule lives inside `AssignSessionName`, so both callers share it;
  - the end-to-end test asserts `ls == 6` and shortens the registration timeout;
  - the probe logs host S separately from probe S;
  - the Architecture line says which seam counts which site;
  - site 6's false-negative behaviour change is stated as intended.

There is no round 3: every remaining fix is a replacement the reviewer itself verified.

### 2026-09-10 — plan-quality gate round 1 (PQ-1 Important, 2 Minor)

Reason: the gate found that the planned end-to-end couchcore test redirected
only the checker's zellij *reads*. Its `Sessions` field, a real session
deleter, still pointed at the host. A red run would have driven a real
`zellij delete-session` plus a SIGKILL sweep through the post-ack quiesce path.
Delta:
- **Class fixed, not the instance.** Task 4 gains Step 0, `sandboxedChecker`:
  - every IO seam is redirected (temp data dir, a fake deleter with a
    no-deletions assertion, a stub `Zellij.Path`, a temp HOME/XDG);
  - **the stub is prepended to PATH**, so an unnamed route's `zellij` exec hits
    the logging stub;
  - every test asserts the log holds only `list-sessions` and `list-clients`.

  A mutation row proves the guard fires.
- **PQ-2.** The `list-clients` producers are enumerated in full (three of
  three). `pair list`'s `ListSessions` is named as a justified full scan.
- **PQ-3.** The failure-path `PairSession` calls are recorded as inheriting
  site 3's narrowing, not as left alone.
