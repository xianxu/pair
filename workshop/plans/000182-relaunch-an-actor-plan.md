# Relaunch an Actor Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `Alt+n` in a hosted actor replaces its Pair process with the current
binary, in place, while the agent conversation continues — and the operator ends
where they started, never looking at a blank screen.

**Architecture:** Two milestones, because this is two different problems with a
real boundary between them. **M1 is the operation**: relaunch is park-then-resume,
both halves exist, and what is new is the composition's failure semantics —
provable entirely in `couchcore` with no terminal. **M2 is the gesture and the
surface**: intercepting `Alt+n` before it reaches the child, and a pane that
outlives its child so the operator's slot shows `relaunching…` rather than
vanishing. M2 holds the substantial new machinery and cannot be written until
M1's operation exists to drive.

**Tech Stack:** Go. `cmd/internal/couchcore` (M1), `cmd/internal/couchtty` (M2).
`cmd/internal/launcher` is **not touched** — couch never runs `kill-session`
itself: park publishes a nonce-bound quit request and the `QuitIntentCouch`
marker (`couchcore/park.go:502,536`), Pair runs its own cleanup, couch observes
the durable completion and CASes the incarnation away.

**Why `Alt+n` must be intercepted at all.** Un-intercepted it reaches Pair, which
handles it *inside the process couch already spawned*: `pair restart` writes a
marker and execs kill-session, the outer process unblocks, and
`createflow.go:77-86` loops back to `runOnce` **in the same process image**.
There is no re-exec, so the binary in memory is the old one. For pair development
that is worse than not working — it looks like it worked. Same argument `Alt+d`'s
interception rests on, only sharper, because this failure is silent.

**Operating envelope (ARCH-CONSTRAINTS).** Relaunch is the longest operation the
console will run: park's 15s completion timeout plus its 5s exact-child-death
wait (`couch.go:119`), then the resume spawn's 10s blocked-start acknowledgement
— ~30s worst case, a few seconds expected. It runs through the bounded,
capacity-one operation queue park already uses. What M2 adds is that the
operator's own pane, not just the switcher, has something to show for that gap.

---

## Chunk 1: M1 — relaunch as one operation

### Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `CheckResumePreconditions` | `cmd/internal/couchcore/resume.go` | new |
| `DecideResume` | `cmd/internal/couchcore/resume.go` | modified |
| `RelaunchOutcome` | `cmd/internal/couchcore/relaunch.go` | new |
| `RelaunchResult` | `cmd/internal/couchcore/relaunch.go` | new |

- **CheckResumePreconditions** — the resume rules that do NOT depend on the
  thread being unoccupied: a working path that still resolves, a saved launch
  profile with a supported agent and non-nil argv, an established native binding
  with a non-empty root id.
  - **Relationships:** consumed by `DecideResume` (which adds occupancy, the
    park/detached authority and the tombstone scan) and by `Relaunch` (which asks
    the same question about a thread that is live).
  - **DRY rationale:** without it relaunch re-derives four rules resume owns, and
    they drift toward whichever cases each author thought about — exactly how the
    archive guard came to admit `creating` while resume refused it (`pair#181`
    M3, BR-6).

- **RelaunchOutcome / RelaunchResult** — which of four states the thread ended
  in: `Relaunched`, `RefusedBeforePark` (nothing happened), `ParkIncomplete` (an
  open transaction; recover through park's modes), `ParkedNotResumed` (a normal
  parked row; `Enter` resumes it). Consumers switch on the outcome, so none has
  to infer state from an error string.
  - **DRY rationale:** follows `ArchiveResult` (`pair#181` M3) — an operation
    that mutated reports what it did and what it could not finish, rather than
    delivering a partial success on the error channel where every consumer reads
    it as total failure.
  - **Why an enum, not two bools:** `ParkIncomplete` and `ParkedNotResumed`
    differ by which recovery works, and a bool pair makes the impossible fourth
    combination representable.

**The failure model, in full.** Three outcomes beyond success, and the milestone
is mostly about which state each leaves behind:

| where it fails | thread state after | recoverable how |
| --- | --- | --- |
| precondition, before the park | unchanged, still live | nothing happened; fix the cause |
| the park itself | `record.Park != nil` — OPEN transaction, Pair already sent its quit intent (`park.go:534`) | park's `retry`/`recover`/`abandon`; NOT `Enter`, which refuses `ResumeParking` |
| the resume, after a good park | verified park, no incarnation | `Enter` on the row — a normal parked thread |

**Park failure is the likeliest branch, not the rarest.** Six failure exits —
commit deadline (`park.go:284`), publish failure (`:504`), completion timeout
(`:573`), stale completion (`:608`), Pair cleanup failure (`:613`), child not gone
(`:616`), revision conflict (`:628`) — each leaving an open transaction that
`pair#181`'s classifier renders as `ThreadBusy` / "parking…".

**Two preconditions cannot be evaluated early, and are named rather than hoped
over:**

1. **The native binding can change across the park.** `BindingEstablished` is
   validated against a scan of the agent's LIVE native-session artifacts
   (`sessioninventory/query.go:66-159`), read while the agent is still writing
   them. Established before does not imply established after. `ResumeContext`
   re-resolves it and a change lands in the park-ok/resume-failed row —
   recoverable and reported. Checking early stops the *predictable* refusals from
   destroying a session; it cannot close a window open by construction.
2. **`soleParkableIncarnation` (`park.go:268`) is park's own precondition**, not
   resume's, so the extracted predicate does not cover it. Relaunch checks it
   explicitly before the park, or a thread with two incarnations refuses *after*
   the quit intent has gone out.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.Relaunch` | `cmd/internal/couchcore/relaunch.go` | new | `PairLifecycle.Park` + `ResumeContext` |
| `relaunch` operation | `cmd/internal/couchcore/ops.go` | new | the declared operation surface |

- **Couch.Relaunch** — checks preconditions, parks, resumes. Both effects come
  from the injected `PairLifecycle` and `Artifacts` seams park and resume already
  use, so the stateful fakes cover it (ARCH-MOCK).
  - **Order is the property:** preconditions before park, and a precondition
    failure performs NOTHING — `pair#181` M3's "guard before effect, and test
    that a refused operation had no effects", where the effect is destructive.
- **relaunch operation** — `ExecuteLiveOwner`, `EffectProcess`,
  `ConfirmRequired`, `PresentationTUI`. Confirmed because it stops a running
  agent; declared so the switcher cannot grow a private verb.

### Task 1: split the resume preconditions from the occupancy rule

**Files:**
- Modify: `cmd/internal/couchcore/resume.go:74-144`
- Test: `cmd/internal/couchcore/resume_test.go`

- [x] **Step 1: Write the failing test — the two callers agree**

```go
func TestResumePreconditionsMatchDecideResumeOnAPostParkRecord(t *testing.T) {
	for _, tc := range everyResumeShape(t) { // path missing, no profile,
		// unsupported agent, bad binding, healthy — each currently LIVE
		precondition := CheckResumePreconditions(tc.record, tc.binding, tc.pathExists)
		// asParked models what the PARK will produce, which is the question
		// relaunch actually asks. Clearing only incarnations would compare
		// against ResumeLegacyUnverified and assert a false equivalence.
		_, resumeErr := DecideResume(ResumeEligibilityInput{
			Thread: asParked(tc.record), WorkingPathExists: tc.pathExists, Binding: tc.binding,
		})
		if (precondition == nil) != (resumeErr == nil) {
			t.Fatalf("%s: precondition=%v, post-park resume=%v", tc.name, precondition, resumeErr)
		}
	}
}
```

- [x] **Step 2: Run — FAIL, `undefined: CheckResumePreconditions`**
- [x] **Step 3: Extract the predicate**, with `DecideResume` calling it so there
      is one copy. `DecideResume` keeps occupancy, the authority choice and the
      tombstone scan — those are about *this* resume, not about whether the
      thread could ever resume.
- [x] **Step 4: Run the couchcore suite** — PASS with the existing `DecideResume`
      tests untouched, which is the evidence the extraction changed nothing.
- [ ] **Step 5: Commit.**

### Task 2: `Couch.Relaunch` — the refusal before the park

**Files:**
- Create: `cmd/internal/couchcore/relaunch.go`, `relaunch_test.go`

- [x] **Step 1: Write the failing test — a refusal destroys nothing**

```go
func TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		breaks  func(*testEnv, ThreadRecord)
		wantErr ResumeDiagnosticCode
	}{
		{name: "no established binding", breaks: unbindNative, wantErr: ResumeBindingProvisional},
		{name: "working path is gone", breaks: removeWorkingPath, wantErr: ResumePathMissing},
		{name: "unsupported saved agent", breaks: corruptAgent, wantErr: ResumeAgentUnsupported},
		{name: "two incarnations", breaks: addSecondIncarnation, wantErr: ResumeLive},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, live := envWithLiveThread(t)
			tc.breaks(env, live)

			result, err := env.Couch.Relaunch(context.Background(), live.Address)

			if result.Outcome != RefusedBeforePark || ResumeDiagnosticOf(err) != tc.wantErr {
				t.Fatalf("Relaunch = %+v, %v", result, err)
			}
			// NOTHING happened: no park attempt, no quit trigger, thread still live.
			if got := env.Lifecycle.trace; len(got) != 0 {
				t.Fatalf("a refused relaunch performed lifecycle work: %v", got)
			}
			if got := env.Artifacts.TriggeredQuits(); len(got) != 0 {
				t.Fatalf("a refused relaunch triggered quits: %v", got)
			}
			after, _ := env.Couch.Threads.GetThread(live.Address)
			if len(after.Incarnations) == 0 || after.VerifiedPark != nil {
				t.Fatalf("a refused relaunch changed the thread: %+v", after)
			}
		})
	}
}
```

- [x] **Step 2: Run — FAIL, `undefined: Relaunch`**
- [x] **Step 3: Implement** in this order and no other: preconditions →
      `soleParkableIncarnation` → `PairLifecycle.Park` → `ResumeContext`.
- [x] **Step 4: Run — PASS, then MUTATION-CHECK the order**: move the
      precondition check after the park and confirm the test fails. A guard no
      test enters is not a guard (`pair#181` M3, BR-12).
- [x] **Step 5: Commit.**

### Task 3: a failed park never attempts the resume, and names its recovery

**Files:**
- Modify: `cmd/internal/couchcore/relaunch.go`
- Test: `cmd/internal/couchcore/relaunch_test.go`

- [x] **Step 1: Write the failing test**, one case per park failure exit a fake
      can produce (completion timeout, cleanup failure, child not gone):

```go
func TestRelaunchStopsAtAFailedParkAndNamesTheRecovery(t *testing.T) {
	env, live := envWithLiveThread(t)
	failPark(env, tc.exit)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if result.Outcome != ParkIncomplete || err == nil {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	// The resume must NOT be attempted: an open park transaction is not
	// resumable, and trying reports a second, confusing failure over the first.
	if got := env.Runner.Children(); len(got) != 0 {
		t.Fatalf("a failed park still tried to start a child: %v", got)
	}
	// The message names PARK's recovery, not Enter — Enter is refused with
	// ResumeParking, and naming it is the unnavigable-refusal class again.
	for _, want := range []string{"retry", "recover", "abandon"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name park's recovery: missing %q", err, want)
		}
	}
	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Park == nil {
		t.Fatalf("thread = %+v, want the open transaction the operator must recover", after)
	}
}
```

- [x] **Step 2-4:** run (FAIL), implement, run (PASS). Mutation-check that the
      resume is genuinely skipped: make the park fail, assert no spawn.
- [x] **Step 5: Commit.**

### Task 4: a resume failure after a good park is recoverable, and says so

**Files:**
- Modify: `cmd/internal/couchcore/relaunch.go`
- Test: `cmd/internal/couchcore/relaunch_test.go`

- [x] **Step 1: Write the failing test**

```go
func TestRelaunchThatParksThenFailsToResumeLeavesARecoverableThread(t *testing.T) {
	env, live := envWithLiveThread(t)
	failResumeAfterPark(env)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if result.Outcome != ParkedNotResumed || err == nil {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	// "relaunch failed" reads as data loss when the work is one keystroke away.
	if !strings.Contains(err.Error(), "parked") || !strings.Contains(err.Error(), "Enter") {
		t.Fatalf("error %q does not say the work is recoverable", err)
	}
	rows, _ := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	if row, ok := findRow(rows, live.Address); !ok || row.State != ThreadParked {
		t.Fatalf("row = %+v (found %v), want a parked row Enter resumes", row, ok)
	}
}
```

- [x] **Step 2-5:** run (FAIL), implement, run (PASS), commit.

### Task 5: a successful relaunch, which is what the issue is for

**Files:**
- Test: `cmd/internal/couchcore/relaunch_test.go`

Tasks 2-4 are all failure tests. The branch being built had no test at all in
this plan's first draft, which is the `done-when-untested` shape.

- [x] **Step 1: Write the failing test** — three of the issue's Done-when bullets:

```go
func TestASuccessfulRelaunchKeepsTheAddressTheRowAndTheConversation(t *testing.T) {
	env, live := envWithLiveThread(t)
	before, _ := env.Couch.Threads.GetThread(live.Address)
	nativeBefore := nativeIDOf(env, live.Address)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch = %+v, %v", result, err)
	}

	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Address != before.Address {
		t.Fatalf("relaunch changed the address: %+v -> %+v", before.Address, after.Address)
	}
	if len(after.Incarnations) != 1 || after.Incarnations[0].State != IncarnationLive {
		t.Fatalf("after = %+v, want exactly one live incarnation", after)
	}
	if after.Incarnations[0].PID == before.Incarnations[0].PID {
		t.Fatalf("the Pair process was not replaced (pid %d)", after.Incarnations[0].PID)
	}
	// The conversation continued: the resume carried the SAME native session id,
	// which is the evidence pair#181 M2 used for reattach.
	if got := requiredSessionIDOf(env, live.Address); got != nativeBefore {
		t.Fatalf("resumed with native id %q, want the conversation's own %q", got, nativeBefore)
	}
	rows, _ := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	if row, ok := findRow(rows, live.Address); !ok || row.State != ThreadLive {
		t.Fatalf("row = %+v (found %v), want a live row at the same address", row, ok)
	}
}
```

- [x] **Step 2-5:** run (FAIL), implement whatever it exposes, run (PASS), commit.

### Task 6: declare the operation and dispatch it

**Files:**
- Modify: `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `ops_declarations_test.go`
- Test: `cmd/internal/couchcmd/run_test.go`

- [x] **Step 1: Declare `relaunch`**, dispatched through `resolveOperationThread`
      — the ONE thread-addressing dialect, because reading only `ref` is how
      `Tab → archive` shipped broken (`pair#181` M3, C-1).
- [x] **Step 2: Write the seam test FIRST**, in `couchcmd`, dispatching
      `relaunch` in the switcher's `{repo-scope, tag}` dialect through the real
      runtime. Neither a store test nor a menu test crosses that boundary.
- [x] **Step 3-5:** implement, run, commit.

### Task 7: close M1

- [x] **Step 1:** `atlas/couch.md` — relaunch beside detach and park, the
      four-outcome table, and the axis that will otherwise be confused:
      `Alt+Shift+N` restarts the *conversation* and keeps the code; relaunch
      restarts the *code* and keeps the conversation.
- [x] **Step 2:** Full `env -u PAIR_SESSION_ID -u PAIR_TAG make test` — exit 0.
- [ ] **Step 3:** `sdlc milestone-close --issue 182 --milestone M1`.

---

## Chunk 2: M2 — the gesture and a surface that outlives its child

### Core concepts

**Correction (M1 review, I-4): M1's operation is NOT reachable the moment it is
declared, and the earlier draft of this preamble asserted it was.**
`menuActionItems` returns hardcoded slices and consumes `Operations()` nowhere;
`ParseCLI` is a closed flag set. So the issue's Done-when bullet — "an actor
action `relaunch` appears alongside detach and park … reachable from the same
declared-operation surface" — had no owning task in either milestone.

**The class is six hand-maintained per-operation sites, not the two an earlier
draft named.** Every one must learn `relaunch` or it half-works:

| site | what it decides |
| --- | --- |
| `menu.go:1008` `menuActionItems` | whether the row offers it at all |
| `menu.go` `confirmationMenuItems` | the confirmation wording (it is `ConfirmRequired`) |
| `menu.go:1306` post-operation frame restore | where the switcher lands afterwards |
| `menu.go:1320` `operationNeedsProjectionRefresh` | whether the inventory re-reads |
| `console.go:1375` the `expectedExits` bridge | the completion-wins-the-race half |
| `console.go:1425` `consumeExpectedParkExitLocked` | the exit-wins-the-race half |

That enumeration is the deliverable, not the two sites: the same shape as
`pair#181` M3's occupancy finding, where a rule applied at four of five sites
left a reachable gap.

M2 adds the chord, this sweep, and a pane that does not vanish while its child is
replaced — the substantial new machinery.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `seqRelaunch` / `HitRelaunch` | `cmd/internal/couchtty/keys.go` | new |
| `paneState` | `cmd/internal/couchtty/console.go` | new |
| `RenderHoldingPane` | `cmd/internal/couchtty/holding.go` | new |
| `endsItsOwnChild` | `cmd/internal/couchtty/console.go` | new |
| `menuActionItems` | `cmd/internal/couchtty/menu.go` | modified |

- **seqRelaunch / HitRelaunch** — `Alt+n` and `Ctrl+Alt+n` consumed before the
  child sees them, encodings pulled from `workbenchshortcut.ChordEncodings`
  rather than retyped, as `ChordAltX` and `ChordAltD` already are
  (`keys.go:69-75`).
  - **Both aliases are required, not a nicety.** On newer macOS `Option+n` is a
    dead-tilde composer, which is why Pair carries `Ctrl+Alt+n` at all.
    `ChordAltN` is `\x1b[110;3u`, `ChordCtrlAltN` is `\x1b[110;7u`.
  - **`Alt+Shift+N` stays with Pair** — `\x1b[78;4u`, distinct, and it restarts
    only the agent conversation. It is the cheap in-session escape hatch that
    survives couch taking the heavier chord.
  - **Kitty-protocol edge, inherited from `ChordAltD`.** Neither chord declares a
    legacy encoding, so with the protocol off they pass through to Pair, which
    does its old in-place reload. zellij pushes the protocol, so this is a
    documented degradation — and it MUST be documented at the interception site
    the way `ChordAltD`'s is, because behaviour then differs silently by protocol
    state.
  - **Accepted cost:** inside couch the operator loses Pair's cheap in-place
    workbench reload; every `Alt+n` becomes a full process replacement. The same
    trade `Alt+d` made, with the same justification.

- **paneState** — `paneLive` or `paneHolding`. A holding pane has no child and
  renders the holding surface; it keeps its slot in `c.order`, its `c.active`
  claim, its thread address, label and actorID.
  - **Relationships:** 1:1 with a pane. Today a pane is 1:1 with a *child*
    (`console.go:31`) and `onExit` deletes the entry unconditionally
    (`console.go:723-736`), so "a pane without a child" is not a state the
    console can be in — that is the machinery this milestone adds.
  - **Why not spawn onto the existing pty:** `pty.StartWithSize` mints the
    master/slave pair per spawn (`ptychild/child.go:103`), and the screen and
    scrollback live in the `Child` (the pane renders through
    `child.ReplayThrough`). Handing a second process an existing master changes
    ptychild's contract; a pane that swaps its child changes the console's, which
    is where the problem is.

- **RenderHoldingPane** — pure: `(label, phase, spinnerPhase, size) → string`.
  **A blank page is indistinguishable from a hang**, and Pair's boot is not
  instant, so this is a status page with a live spinner and the phase name
  ("parking…" then "restarting…"), not an empty tty. Reuses the console's notice
  spinner off `spinnerC`/`syncSpinner` — no new timer.

- **endsItsOwnChild** — one predicate replacing two hand-written lists.
  `console.go:1378`'s `expectedExits` bridge and
  `consumeExpectedParkExitLocked`'s switch (`console.go:1425-1429`) each
  enumerate `park`/`detach` today, and both exist because the exit/completion
  race resolves in either order. Relaunch must appear in BOTH or the operator
  gets a spurious child-exited notice.
  - **DRY rationale:** `operationNeedsProjectionRefresh` (`menu.go:1318`) is the
    existing shape. A third hand-written list is the wrong answer — the same
    class as `pair#181` M3's occupancy finding.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `onRelaunchHotkey` | `cmd/internal/couchtty/console.go` | new | chord → operation dispatch |
| `onExit` | `cmd/internal/couchtty/console.go` | modified | child death, non-fatal for a holding pane |
| `finishOperation` | `cmd/internal/couchtty/console.go` | modified | installing the new child into the held pane |

- **onRelaunchHotkey** — **scope follows focus, with one deviation that must be
  stated.** `Alt+x` and `Alt+d` mean "what you are looking at": one actor from an
  actor, every live thread from the switcher. Relaunch has NO whole-couch form —
  that is `Alt+d`, rebuild, re-run `couch`, the symmetry this issue completes. So
  from the panel `Alt+n` relaunches the HIGHLIGHTED ROW and leaves the operator in
  the switcher; from an actor it relaunches that actor and returns to it. **The
  ending differs by caller**, which is why it belongs to the console rather than
  to the operation.
  - `processInput`'s dispatch switch is exhaustive on purpose — its comment says
    a `default` arm would turn any unhandled hit into "open the switcher".
    `HitRelaunch` is handled in both focus states, never defaulted.

### Task 8: the key layer

**Files:**
- Modify: `cmd/internal/couchtty/keys.go:35-140`
- Test: `cmd/internal/couchtty/keys_test.go`

- [ ] **Step 1: Write the failing test** — both chords decode to `HitRelaunch` at
      every read split, stay inert inside a bracketed paste, and `Alt+Shift+N`
      still passes through to the child. That last assertion protects the escape
      hatch.
- [ ] **Step 2-5:** run (FAIL), add `seqRelaunch`/`HitRelaunch` and both
      `ChordEncodings` rows with the protocol-edge comment, run (PASS), commit.

### Task 9: a pane that outlives its child

**Files:**
- Modify: `cmd/internal/couchtty/console.go:31-45`, `:721-800`
- Create: `cmd/internal/couchtty/holding.go`, `holding_test.go`
- Test: `cmd/internal/couchtty/console_test.go`

- [ ] **Step 1: Write the failing test** — the property, not the mechanism:

```go
// The operator's slot survives the gap. Today onExit deletes the pane entry
// unconditionally, so the child's death takes the pane, the slot and the
// operator's place with it.
func TestAHoldingPaneSurvivesItsChildsExit(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.beginRelaunch("c1")
	f.child.Exit(0)

	waitUpTo(t, 250*time.Millisecond, "the holding surface", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "relaunching")
	})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if _, held := f.con.panes["c1"]; !held {
		t.Fatal("the pane went with its child")
	}
	if f.con.active != "c1" {
		t.Fatalf("active = %q, want the held pane to keep the operator's slot", f.con.active)
	}
}
```

- [ ] **Step 2: Run — FAIL** (the pane is deleted; the screen is the switcher).
- [ ] **Step 3: Implement** `paneState`, the `onExit` branch that keeps a holding
      pane, and `RenderHoldingPane` driven by the existing spinner.
- [ ] **Step 4: Run — PASS.** Then assert the spinner ADVANCES: a frozen glyph is
      indistinguishable from the hang this surface exists to rule out.
- [ ] **Step 5: Commit.**

### Task 10: the three consequences, each pinned

**Files:**
- Modify: `cmd/internal/couchtty/console.go`
- Test: `cmd/internal/couchtty/console_run_menu_test.go`

- [ ] **Step 1: `previous` is not spent.** `onExit` calls `tracker.Drop`
      unconditionally (`console.go:736`), emptying `current`; the landing that
      follows copies that emptiness into `previous`. So a park/resume cycle spends
      `ctrl+backspace` even though the operator never left — contradicting
      `SwitchTracker`'s own doc comment. A holding pane dissolves it (no exit, no
      `Drop`); the test asserts the property: relaunch, then `ctrl+backspace`,
      then assert it lands where it did before.
- [ ] **Step 2: No child-exited notice.** `endsItsOwnChild` replaces both
      hand-written lists; the test drives a relaunch through the production input
      path and asserts the feed carries no exit notice.
- [ ] **Step 2b: Sweep all SIX per-operation sites** from the table above, not
      the two this step originally named, and add a test that walks
      `Operations()` asserting every `PresentationTUI` operation the switcher can
      reach appears in `menuActionItems` for some row state — so the next
      operation cannot ship declared-but-unreachable.
- [ ] **Step 3: Both focus states dispatch, with different endings.** `Alt+n`
      from an actor relaunches it and ENDS ON IT; from the panel it relaunches the
      highlighted row and stays in the switcher. Two tests, because the endings
      differ.
- [ ] **Step 4:** run the whole suite.
- [ ] **Step 5:** Commit.

### Task 11: real-stack verification with a rebuilt binary

- [ ] **Step 1:** Note the running Pair binary's mtime, the child PID, and the
      agent's conversation state in one thread.
- [ ] **Step 2:** Rebuild Pair.
- [ ] **Step 3:** `Alt+n` in that actor.
- [ ] **Step 4:** Confirm all four: the process is new (PID changed), it is the
      REBUILT binary (mtime), the conversation continued rather than restarting
      (ledger native session id unchanged — `pair#181` M2's evidence), and the
      operator never saw a blank screen.
- [ ] **Step 5:** Record the measurement in the issue Log, not "it worked".

### Task 12: close M2

- [ ] **Step 1:** `menuControls` + README rows for `Alt+n` / `Ctrl+Alt+n` so no
      key ships undocumented; atlas gets the holding pane and the
      scope-follows-focus deviation.
- [ ] **Step 2:** Full `make test` — exit 0.
- [ ] **Step 3:** `sdlc milestone-close --issue 182 --milestone M2`.

---

## Open questions, recorded rather than decided

1. **Should relaunch be offered on a DETACHED row?** Its agent runs but couch
   hosts no client, so "replace the Pair process" means
   reattach-with-the-new-binary — closer to `pair#181` M2's warm path than to
   park-then-resume. Out of scope; reattach then relaunch.
2. **Does the holding pane keep the old screen behind the status page?** Showing
   the dead child's last frame under a spinner would preserve context, but the
   scrollback belongs to the `Child` being discarded. Deferred until the surface
   exists and can be looked at.
3. **`--layout2` on the relaunch resume** is settled: relaunch behaves exactly as
   an ordinary cold resume, which sends it. Recorded because diverging here is the
   one change that would reintroduce the `pair#181` M2 session-deleting hazard.
