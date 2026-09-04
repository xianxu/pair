# Relaunch an Actor Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One action that replaces a thread's Pair process with the current
binary while its agent conversation continues.

**Architecture:** Relaunch is park-then-resume, and both halves already exist as
declared operations driven by `PairLifecycleController` and `ResumeContext`.
Nothing new is mechanised. What is new is the **composition's failure
semantics**, and that is the whole design: park destroys, resume can refuse, and
a relaunch that parks successfully then fails to resume has destroyed a working
session. So the resume's preconditions are evaluated **before** the park, and
the one precondition that cannot be evaluated early is named rather than hoped
over.

**Tech Stack:** Go. `cmd/internal/couchcore` (the operation, the precondition
split, the durable lifecycle), `cmd/internal/couchtty` (the switcher action and
its confirmation). `cmd/internal/launcher` is **not touched** — relaunch is
couch composing Pair operations it already drives.

**Operating envelope (ARCH-CONSTRAINTS).** Relaunch is the longest operation the
switcher will ever run, and it chains two bounded ones: park's 15s completion
timeout plus its 5s exact-child-death wait (`couch.go:119`), then the resume
spawn's 10s blocked-start acknowledgement (`launch_existing.go`). Worst case is
therefore ~30s, and the expected case is a few seconds. That is far past the
100ms feedback contract the switcher holds for park, so relaunch runs through the
same bounded, capacity-one operation queue park already uses — the switcher stays
responsive, the progress notice names which half is running ("parking…" then
"restarting…"), and the operator can navigate while it works. No new concurrency;
the queue and its coalescing already exist.

**The failure model, in full.** Relaunch has three outcomes beyond success, and
the plan is mostly about which state each leaves behind:

| where it fails | thread state after | recoverable how |
| --- | --- | --- |
| precondition, before the park | unchanged, still live | nothing happened; fix the cause |
| the park itself | `record.Park != nil` — an OPEN transaction, and Pair has already been sent its quit intent (`park.go:534`) | park's own recovery modes (`retry`, `recover`, `abandon`); NOT `Enter`, because `DecideResume` refuses `ResumeParking` |
| the resume, after a good park | verified park, no incarnation | `Enter` on the row — it is a normal parked thread |

**Park failure is the likeliest branch, not the rarest.** `PairLifecycleController.Park`
is a multi-phase transaction with six failure exits — commit deadline
(`park.go:284`), publish failure (`:504`), the 15s completion timeout (`:573`),
stale completion (`:608`), Pair cleanup failure (`:613`), child not gone
(`:616`), revision conflict (`:628`). Every one leaves an open park transaction,
which `pair#181`'s classifier renders as `ThreadBusy` / "parking…" — accurate,
but a row the operator cannot act on and whose recovery is not `Enter`. Relaunch
must not attempt the resume there, and must say which of park's recovery modes
applies rather than reporting "relaunch failed".

**Two preconditions cannot be evaluated early, and are named rather than hoped
over:**

1. **The native binding can change across the park.** `BindingEstablished` is
   validated against a scan of the agent's LIVE native-session artifacts
   (`sessioninventory/query.go:66-159`), read while the agent is still writing
   them. Established before the park does not imply established after it.
   Relaunch does not pretend otherwise: `ResumeContext` re-resolves it, and a
   change lands in the park-ok/resume-failed row of the table — recoverable, and
   reported as such. Checking early is what stops the *predictable* refusals
   from destroying a session; it cannot close a window that is open by
   construction.
2. **`soleParkableIncarnation` (`park.go:268`) is park's own precondition** and
   is absent from resume's rule set, so the extracted predicate does not cover
   it. Relaunch checks it explicitly before the park, or a thread with two
   incarnations refuses *after* the quit intent has gone out.

**The asymmetry that shapes everything.** `DecideResume` refuses a thread with
any occupied incarnation (`occupiedIncarnation`), and a relaunch target is LIVE
by definition. So relaunch cannot ask "is this resumable?" — the answer is
always no. It must ask **"would this be resumable once parked?"**, which is the
same rule set minus occupancy. Extracting that is the plan's one real design
move (ARCH-DRY): two callers, one set of rules, and the difference between them
stated in code rather than duplicated.

---

## Chunk 1: M1 — relaunch as one operation

### Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ResumePreconditions` | `cmd/internal/couchcore/resume.go` | new |
| `DecideResume` | `cmd/internal/couchcore/resume.go` | modified |
| `RelaunchRefusal` | `cmd/internal/couchcore/relaunch.go` | new |
| `RelaunchResult` | `cmd/internal/couchcore/relaunch.go` | new |
| `RelaunchOutcome` | `cmd/internal/couchcore/relaunch.go` | new |
| `menuActionItems` | `cmd/internal/couchtty/menu.go` | modified |
| `confirmationMenuItems` | `cmd/internal/couchtty/menu.go` | modified |

- **ResumePreconditions** — the resume rules that do NOT depend on the thread
  being unoccupied: a working path that still resolves, a saved launch profile
  with a supported agent and non-nil argv, and (for the cold path) an
  established native binding with a non-empty root id.
  - **Relationships:** consumed by `DecideResume` (which adds the occupancy
    refusal and the park/detached authority) and by `Relaunch` (which asks the
    same question about a thread that is currently live).
  - **DRY rationale:** without it, relaunch re-derives four rules that resume
    already owns, and they drift toward whichever cases each author thought
    about — which is exactly how the archive guard came to admit `creating`
    while resume refused it (`pair#181` M3, BR-6).
  - **Future extensions:** if a warm relaunch is ever wanted (replace the
    client, keep the session), it is this predicate plus a different authority.

- **RelaunchRefusal** — why a relaunch will not start, evaluated BEFORE any
  destructive step. It carries the `ResumeDiagnosticCode` that would have
  refused the resume, so the operator sees the real reason rather than
  "relaunch failed".
  - **DRY rationale:** reuses resume's diagnostic vocabulary rather than
    inventing a parallel one.

- **RelaunchResult / RelaunchOutcome** — which of the four states the thread
  ended in: `Relaunched`, `RefusedBeforePark` (nothing happened),
  `ParkIncomplete` (an open transaction; recover through park's modes) and
  `ParkedNotResumed` (a normal parked row; `Enter` resumes it). The outcome is
  the thing consumers switch on, so no caller has to infer state from an error
  string.
  - **DRY rationale:** follows `ArchiveResult` (`pair#181` M3) — an operation
    that mutated reports what it did and what it could not finish, rather than
    delivering a partial success on the error channel where every consumer reads
    it as total failure.
  - **Why an enum rather than two bools:** `ParkIncomplete` and
    `ParkedNotResumed` differ by which recovery works, and a boolean pair makes
    the impossible fourth combination representable.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.Relaunch` | `cmd/internal/couchcore/relaunch.go` | new | `PairLifecycle.Park` + `ResumeContext` |
| `relaunch` operation | `cmd/internal/couchcore/ops.go` | new | the declared operation surface |

- **Couch.Relaunch** — checks preconditions, parks, resumes.
  - **Injected into:** nothing pure; it is the seam. Its two effects come from
    the same injected `PairLifecycle` and `Artifacts` seams park and resume
    already use, so the stateful fakes cover it (ARCH-MOCK).
  - **Order is the property:** preconditions before park, and a precondition
    failure performs NOTHING. This is `pair#181` M3's archive lesson —
    "guard before effect, and test that a refused operation had no effects" —
    applied to a composition where the effect is destructive.

- **relaunch operation** — `ExecuteLiveOwner`, `EffectProcess`,
  `ConfirmRequired`, `ResultRelaunch`, `PresentationTUI`. Confirmed because it
  stops a running agent; live-owner because it needs the console's dispatcher.
  Declared rather than driven from the switcher, so the switcher cannot grow a
  private verb (`pair#148`'s design test).

### Task 1: split the resume preconditions from the occupancy rule

**Files:**
- Modify: `cmd/internal/couchcore/resume.go:74-144` (`DecideResume`)
- Test: `cmd/internal/couchcore/resume_test.go`

- [ ] **Step 1: Write the failing test — the two callers agree**

```go
// The rules relaunch needs are the rules resume applies, minus the one that a
// park is about to clear. Asserting the AGREEMENT is what stops them drifting:
// pair#181 M3 shipped an archive guard that admitted `creating` while resume
// refused it, from exactly this kind of parallel derivation.
func TestResumePreconditionsMatchDecideResumeMinusOccupancy(t *testing.T) {
	for _, tc := range everyResumeShape(t) { // path missing, no profile,
		// unsupported agent, bad binding, healthy — each currently LIVE
		precondition := CheckResumePreconditions(tc.record, tc.binding, tc.pathExists)
		// asParked models what the PARK will produce, which is the whole
		// question: occupancy cleared, no open transaction, a verified park
		// stamped. Clearing only the incarnations would compare against
		// ResumeLegacyUnverified and assert a false equivalence -- the two
		// would "disagree" for a reason that has nothing to do with the split.
		_, resumeErr := DecideResume(ResumeEligibilityInput{
			Thread: asParked(tc.record), WorkingPathExists: tc.pathExists,
			Binding: tc.binding,
		})
		if (precondition == nil) != (resumeErr == nil) {
			t.Fatalf("%s: precondition=%v, post-park resume=%v -- the two disagree", tc.name, precondition, resumeErr)
		}
	}
}
```

- [ ] **Step 2: Run it — FAIL, `undefined: CheckResumePreconditions`**
- [ ] **Step 3: Extract the predicate**, leaving `DecideResume` calling it so
      there is one copy of the rules. `DecideResume` keeps the occupancy
      refusal, the park/detached authority choice and the tombstone scan; those
      are about *this* resume, not about whether the thread could ever resume.
- [ ] **Step 4: Run the couchcore suite** — PASS, and the existing
      `DecideResume` tests are untouched, which is the evidence the extraction
      changed nothing.
- [ ] **Step 5: Commit.**

### Task 2: `Couch.Relaunch`, with the refusal before the park

**Files:**
- Create: `cmd/internal/couchcore/relaunch.go`, `cmd/internal/couchcore/relaunch_test.go`

- [ ] **Step 1: Write the failing test — a refusal destroys nothing**

```go
// The property that makes relaunch safe to offer. A relaunch that parks and
// then fails to resume has destroyed a working session, so a thread that could
// not be resumed must not be parked in the first place.
func TestRelaunchRefusesBeforeParkingWhenTheResumeCouldNotSucceed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		breaks  func(*testEnv, ThreadRecord)
		wantErr ResumeDiagnosticCode
	}{
		{name: "no established binding", breaks: unbindNative, wantErr: ResumeBindingProvisional},
		{name: "working path is gone", breaks: removeWorkingPath, wantErr: ResumePathMissing},
		{name: "unsupported saved agent", breaks: corruptAgent, wantErr: ResumeAgentUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, live := envWithLiveThread(t)
			tc.breaks(env, live)

			_, err := env.Couch.Relaunch(context.Background(), live.Address)

			if got := ResumeDiagnosticOf(err); got != tc.wantErr {
				t.Fatalf("Relaunch = %q, want %q", got, tc.wantErr)
			}
			// Nothing happened: no park attempt, no quit trigger, no session torn
			// down, and the thread is still live.
			if got := env.Lifecycle.trace; len(got) != 0 {
				t.Fatalf("a refused relaunch performed lifecycle work: %v", got)
			}
			if got := env.Artifacts.TriggeredQuits(); len(got) != 0 {
				t.Fatalf("a refused relaunch triggered quits: %v", got)
			}
			after, _ := env.Couch.Threads.GetThread(live.Address)
			if len(after.Incarnations) != 1 || after.VerifiedPark != nil {
				t.Fatalf("a refused relaunch changed the thread: %+v", after)
			}
		})
	}
}
```

- [ ] **Step 2: Run — FAIL, `undefined: Relaunch`**
- [ ] **Step 3: Implement**, in this order and no other: check preconditions →
      `PairLifecycle.Park` → `ResumeContext`. Every early return before the park
      performs nothing.
- [ ] **Step 4: Run — PASS. Then MUTATION-CHECK the order**: move the
      precondition check after the park and confirm the test fails. A guard that
      is not entered by a test is not a guard (`pair#181` M3, BR-12).
- [ ] **Step 5: Commit.**

### Task 3: a resume failure after a good park is recoverable, and says so

**Files:**
- Modify: `cmd/internal/couchcore/relaunch.go`
- Test: `cmd/internal/couchcore/relaunch_test.go`

- [ ] **Step 1: Write the failing test**

```go
// The window that cannot be closed: the park succeeded and the resume failed.
// The thread is PARKED — recoverable, listed, Enter resumes it — and the result
// must say so, because "relaunch failed" reads as data loss when the work is
// one keystroke away.
func TestRelaunchThatParksThenFailsToResumeLeavesARecoverableThread(t *testing.T) {
	env, live := envWithLiveThread(t)
	failResumeAfterPark(env)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if err == nil {
		t.Fatal("a failed resume reported success")
	}
	if !result.Parked || result.Resumed {
		t.Fatalf("result = %+v, want parked-but-not-resumed", result)
	}
	if !strings.Contains(err.Error(), "parked") || !strings.Contains(err.Error(), "Enter") {
		t.Fatalf("error %q does not tell the operator the work is recoverable", err)
	}
	// And the store agrees: a verified park, which the switcher lists as
	// resumable.
	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.VerifiedPark == nil {
		t.Fatalf("thread = %+v, want a verified park", after)
	}
	rows, _ := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	row, ok := findRow(rows, live.Address)
	if !ok || row.State != ThreadParked {
		t.Fatalf("row = %+v (found %v), want a parked row the operator can resume", row, ok)
	}
}
```

- [ ] **Step 2-4:** run (FAIL), implement the result + message, run (PASS).
- [ ] **Step 5: Commit.**

### Task 3b: a park failure never attempts the resume, and names its recovery

**Files:**
- Modify: `cmd/internal/couchcore/relaunch.go`
- Test: `cmd/internal/couchcore/relaunch_test.go`

The likeliest failure, and the one whose state is worst: Pair has already been
sent its quit intent, the transaction is open, and `Enter` will NOT recover the
row because `DecideResume` refuses `ResumeParking`.

- [ ] **Step 1: Write the failing test**, one case per park failure exit that a
      fake can produce (completion timeout, cleanup failure, child not gone):

```go
func TestRelaunchStopsAtAFailedParkAndNamesTheRecovery(t *testing.T) {
	env, live := envWithLiveThread(t)
	failPark(env, tc.exit)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)

	if result.Outcome != ParkIncomplete || err == nil {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	// The resume must NOT have been attempted: a thread with an open park
	// transaction is not resumable, and trying would report a second, confusing
	// failure over the first.
	if got := env.Runner.Children(); len(got) != 0 {
		t.Fatalf("a failed park still tried to start a child: %v", got)
	}
	// The message names park's recovery modes, not Enter -- `Enter` is refused
	// with ResumeParking, and telling the operator to press it is the
	// unnavigable-refusal class again (pair#181 M3).
	for _, want := range []string{"park", "retry", "recover", "abandon"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name park's recovery: missing %q", err, want)
		}
	}
	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Park == nil {
		t.Fatalf("thread = %+v, want the open park transaction the operator must recover", after)
	}
}
```

- [ ] **Step 2-4:** run (FAIL), implement, run (PASS). Mutation-check that the
      resume is genuinely skipped: make the park fail and assert no child spawn.
- [ ] **Step 5: Commit.**

### Task 3c: a successful relaunch, which is what the issue is for

**Files:**
- Test: `cmd/internal/couchcore/relaunch_test.go`

Tasks 2, 3 and 3b all test failures. The branch being built had no test at all,
which is the `done-when-untested` shape.

- [ ] **Step 1: Write the failing test** — the issue's last Done-when bullet:

```go
// The thing being built. One live incarnation at the SAME address afterwards,
// the row still in the switcher, and the same ledger identity -- the agent
// conversation continued rather than restarted, which is what distinguishes a
// relaunch from starting over.
func TestASuccessfulRelaunchKeepsTheAddressTheRowAndTheConversation(t *testing.T) {
	env, live := envWithLiveThread(t)
	before, _ := env.Couch.Threads.GetThread(live.Address)

	result, err := env.Couch.Relaunch(context.Background(), live.Address)
	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch = %+v, %v", result, err)
	}

	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Address != before.Address {
		t.Fatalf("relaunch changed the thread address: %+v -> %+v", before.Address, after.Address)
	}
	if len(after.Incarnations) != 1 || after.Incarnations[0].State != IncarnationLive {
		t.Fatalf("after = %+v, want exactly one live incarnation", after)
	}
	if after.Incarnations[0].PID == before.Incarnations[0].PID {
		t.Fatalf("the Pair process was not replaced (pid %d)", after.Incarnations[0].PID)
	}
	// The conversation continued: the resume carried the SAME native session
	// id, which is the evidence pair#181 M2 used for reattach.
	if got := requiredSessionIDOf(env, live.Address); got != nativeIDBefore {
		t.Fatalf("resumed with native id %q, want the conversation's own %q", got, nativeIDBefore)
	}
	// And the row survives, which is what the operator sees.
	rows, _ := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
	if row, ok := findRow(rows, live.Address); !ok || row.State != ThreadLive {
		t.Fatalf("row = %+v (found %v), want a live row at the same address", row, ok)
	}
}
```

- [ ] **Step 2-5:** run (FAIL), implement whatever it exposes, run (PASS), commit.

### Task 4: the declared operation and the switcher action

**Files:**
- Modify: `cmd/internal/couchcore/ops.go`, `cmd/internal/couchcore/operationdispatch.go`, `cmd/internal/couchcore/ops_declarations_test.go`, `cmd/internal/couchtty/menu.go`, `cmd/internal/couchcmd/run.go` (result rendering)
- Test: `cmd/internal/couchtty/menu_test.go`, `cmd/internal/couchcmd/run_test.go`

- [ ] **Step 1: Declare `relaunch`** and dispatch it through
      `resolveOperationThread` — the ONE thread-addressing dialect, because the
      switcher sends `{repo-scope, tag}` and reading only `ref` is how
      `Tab → archive` shipped broken (`pair#181` M3, C-1).
- [ ] **Step 2: Write the seam test FIRST**, in `couchcmd`, dispatching
      `relaunch` in the switcher's dialect through the real runtime. Neither a
      store test nor a menu test crosses that boundary.
- [ ] **Step 3: Offer it on LIVE rows only.** A parked or detached thread has no
      Pair process to replace; offering an action that always refuses teaches
      the operator to distrust the menu. Confirmed like park, and the
      confirmation says what it does: "stops and restarts Pair; the conversation
      continues".
- [ ] **Step 4: Run the whole suite**, plus `menuControls` and the README row so
      no key ships undocumented.
- [ ] **Step 5: Commit.**

### Task 5: real-stack verification with a rebuilt binary

- [ ] **Step 1:** Note the running Pair binary's mtime and the agent's current
      conversation state in one thread.
- [ ] **Step 2:** Rebuild Pair (`make` / the `pair` shell function's rebuild).
- [ ] **Step 3:** Relaunch that thread from the switcher.
- [ ] **Step 4:** Confirm BOTH halves: the new process is the rebuilt binary
      (mtime/pid changed), and the agent conversation continued rather than
      restarting (its native session id is unchanged in the ledger, which is the
      same evidence `pair#181` M2 used for reattach).
- [ ] **Step 5:** Record the observation in the issue Log — the measurement, not
      "it worked".

### Task 6: docs and close

- [ ] **Step 1:** `atlas/couch.md` — relaunch beside detach and park, and why
      its preconditions run first. Name the axis explicitly: `Alt+Shift+N`
      restarts the *conversation* and keeps the workbench; relaunch restarts the
      *code* and keeps the conversation. They are inverses and will be confused
      otherwise.
- [ ] **Step 2:** README — the action and its confirmation.
- [ ] **Step 3:** Full `env -u PAIR_SESSION_ID -u PAIR_TAG make test` — exit 0.
- [ ] **Step 4:** `sdlc milestone-close --issue 182 --milestone M1`.

---

## Open questions, recorded rather than decided

1. **Should relaunch be offered on a DETACHED row?** Its agent is running but
   couch hosts no client, so "replace the Pair process" means something
   different — reattach with the new binary, which is closer to M2's warm path
   than to park-then-resume. Left out of scope; the row can be reattached and
   then relaunched.
2. **Does relaunch need its own chord?** The plan puts it behind `Tab` with the
   other actions. A chord is cheap to add later and impossible to remove, and
   the operator has not asked for one.
3. **Is `--layout2` right on the relaunch resume?** The cold resume path sends
   it today. A relaunch is a create boundary, so it should behave exactly as a
   normal cold resume — but this is the setting that sent Pair down a
   session-deleting path in `pair#181` M2, so it is worth confirming rather than
   assuming.
