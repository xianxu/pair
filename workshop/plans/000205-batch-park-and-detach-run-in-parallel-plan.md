# Parallel lifecycle operations (pair#205) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restarting couch at ~18 slots stops scaling linearly. Quit (`leave`)
and the startup reattach pass act on several threads at once, under a
per-thread guard that refuses a second lifecycle operation on a busy thread.

**Architecture:** M1 adds one in-memory `ThreadGate` to `Couch`, held by every
couchcore entry point that changes a thread's lifecycle. Composite operations
re-enter through a hold carried in their `context.Context`. M2 then removes the
three serial points behind it, all bounded by one shared constant:
- `Couch.Leave`'s loop;
- `parkWorker`'s capacity of 1;
- the startup reattach pass's single in-flight slot, plus the console's single
  queue worker.

**Tech Stack:** Go; packages `cmd/internal/couchcore` (domain) and
`cmd/internal/couchtty` (console). Tests use the existing `newTestEnv` /
`envWithLiveThread` fakes.

Issue: `workshop/issues/000205-batch-park-and-detach-run-in-parallel.md`
(Spec, "Per-thread admission guard" and the 2026-10-06 Log are the inputs).

---

## Design decisions (read first)

**D1. The guard lives in couchcore, not in the console queue.** The queue
cannot always name the thread:
- a remote socket job learns its address only after `prepare` runs on the
  worker (`couchtty/console_remote.go`);
- path-addressed `resume` and `reboot` resolve their thread inside couchcore;
- `leave` acts on every thread from one job.

Every lifecycle path converges on a small set of couchcore entries, so one gate
there covers all callers: switcher, reattach pass, continuations, remote,
startup resume, slot recovery, and the startup park-recovery worker.
(ARCH-DRY, ARCH-PURPOSE.)

**D2. Hold at every public entry; re-enter through the context.** Composites call
leaves: `Relaunch` → `ParkExpected` + `ResumeContext`; `SwitchAgent` and
`Continue` → `ParkExpected`; `RetryContinuation` → `Recover` + `ResumeContextWith`.
Guarding only the leaves would release between park and resume. That gap is
exactly the window #214 named. So a composite acquires first, and its leaves
see the hold in `ctx` and pass through.

**D3. Refuse, do not queue (operator decision, 2026-09-08)** for gestures and
background work. **Two drains wait instead:**
- `Leave`'s per-thread step;
- `RecoverActiveParks`.

Each waits for the thread's holder, bounded by its own `ctx`. Quit must not
silently skip a thread because a reattach was mid-flight, and boot recovery must
not fail because the pass touched the thread first.

**D4. Nothing outlives the hold.** `PairLifecycleController.submit` now returns
only after the park work finishes, not as soon as `ctx` is cancelled. The work
receives the same `ctx` and stops promptly; `CompletionTimeout` bounds it.
Without this, a cancelled caller would release the thread while its park is
still running (ARCH-ORDER extent).

**D5. One concurrency bound, `couchcore.LifecycleParallelism = 4`, with this
reason.** The #206 probe measured raw `zellij attach` as nearly free under
concurrency (8 at once: 121 ms in total, `zellij action` p95 unchanged). A
*couch* reattach, however, still issues one `list-clients` per live session
(190–350 ms each against a detached session). So N parallel reattaches put N×S
calls in flight. A bound of 4 keeps that to 4×S, about 72 calls at 18 slots,
instead of 18×S.

The constant is shared by `Leave`, `parkWorker` and the console's queue workers.
If measurement says otherwise, it changes in one place. (ARCH-CONSTRAINTS.)

**D6. `Leave` no longer stops at the first failure.** Each thread's outcome is
independent. `Leave` finishes the others and returns the partial `LeaveResult`
plus a joined error naming every failed thread. The result already "preserves
partial progress" by contract, and this widens that to every sibling.

### ARCH-ORDER: the interleavings that apply

The state that lives between events is the gate's `held` map and the pass's
in-flight set. Durable thread state stays under `ThreadStore`'s revision CAS,
unchanged.

| Event (cannot be blocked by the caller) | Policy | Rollback |
|---|---|---|
| Second lifecycle op on a held thread (pass vs remote vs continuation vs operator) | **Refuse** with `ThreadBusyError{Address, Running}` | none: nothing ran |
| `Leave` reaches a thread a reattach attempt holds | **Wait** on the gate (ctx-bounded), then detach or park | none |
| `RecoverActiveParks` reaches a thread an operation holds | **Wait**, then recover | none |
| Operator quits mid-pass | Pass holds new attempts (cell 10, unchanged). At most `LifecycleParallelism` attempts are in flight; `Leave` waits on each thread it needs | none |
| Caller `ctx` cancelled mid-park | `submit` waits for the work to finish (D4); the hold is released after it | park's own recovery modes, unchanged |
| Pass completions arrive out of order | `finishReattach` matches by attempt number in the in-flight map, not "the" attempt | none |
| One of N `Leave` threads fails | Others continue; joined error plus partial result (D6) | none: already-detached threads stay detached |
| couch process dies mid-`Leave` | Unchanged from today: per-thread durable transitions; the next startup reconciles | n/a |
| A thread's agent exits on its own while its park is in flight | Unchanged: park already treats child death as completion evidence (`awaitCompletionAndChildDeath`). The gate stays held until park returns, so no other operation sees the half-torn-down thread | park's recovery modes |
| `Leave` waited behind a holder, so its snapshot row is stale | `leaveOne` takes `holdWait` **first**, then re-reads the record with `GetThread` and decides from that. Never from the snapshot | none |
| Completion-side effects run after release (`finishOperation` → `attach`/`AbortStarted` on the console goroutine) | These act on the exact actor record and process identity the operation returned, never on "the thread's current state". A newer holder's incarnation has a different identity and is untouched. Test in Task 10 | none |
| Boot: `StartInteractive` resumes a thread `RecoverActiveParks` holds | Startup **waits** (`ResumeOptions.WaitForHolder`). The operator's first frame must not fail because boot recovery touched the same thread | none |

**Most likely to be mishandled:** a composite that releases its hold before its
inner work finishes (D4), and its twin, a drain that decides from an observation
taken before it waited (`Leave`'s snapshot). Both have their own tests.

**Nondeterminism:** scheduler order of goroutines. Tests control it with blocking
fakes (a park whose `TriggerQuitHook` waits on a channel) and assert outcomes
that hold under any order. They don't assert sleeps.

The issue's Done-when requires these answers to be stated **in the issue**. Task 5
copies this table into its `## Spec` under "The interleaving policy has to be
written down", replacing the open questions there.

### ARCH-FUNERAL

Creates nothing durable. The gate's `held` entries are in memory, each removed by
its holder's `release` (deferred at the acquiring frame), and the map is bounded
by the number of threads. The pass's in-flight map is bounded by
`LifecycleParallelism` and emptied by `finishReattach`.

### ARCH-SECURE

No new trust boundary. Addresses come from the same resolvers as today
(`resolveOperationThread`, path resolution). The gate trusts only its own map.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `gateDecision` | `cmd/internal/couchcore/threadgate.go` | new |
| `ThreadBusyError` | `cmd/internal/couchcore/threadgate.go` | new |
| `LifecycleParallelism` | `cmd/internal/couchcore/threadgate.go` | new |
| `ReattachPass` | `cmd/internal/couchtty/menu_reattach.go` | modified |

- **gateDecision** — `(held, holder) → admit | reenter | busy(running op)`. It
  has no locking or IO, and the table test covers it exhaustively.
  - **Relationships:** consumed only by `ThreadGate`.
  - **DRY rationale:** it is the one per-address admission rule.
    `parkWorker`'s own `active` map stays only for nonce de-duplication (one
    future per park transaction). Its "another park transaction owns this
    address" refusal becomes unreachable through `Couch`, and is kept as the
    worker's internal invariant.
  - **Future extensions:** a relaxed rule (for example, read-only operations
    co-holding) widens this function, not its callers.
- **ThreadBusyError** — names the address and the running operation. Its message
  reads `"<tag> is busy: <op> is already running on it"`. It's a typed error so
  the console and remote reporting can recognise it.
- **LifecycleParallelism** — the one bound from D5.
- **ReattachPass** — `Loading`/`LoadingAttempt` become
  `InFlight map[uint64]couchcore.ThreadAddress` (attempt → thread).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadGate` | `cmd/internal/couchcore/threadgate.go` | new | `sync.Mutex` + `context` values |
| `Couch.hold` / `Couch.holdWait` | `cmd/internal/couchcore/threadgate.go` | new | the gate, lazily created on `Couch` |
| `Couch.Park` | `cmd/internal/couchcore/park.go` | new | `PairLifecycleController` park modes, under the gate |
| `PairLifecycleController.submit` | `cmd/internal/couchcore/park.go` | modified | park future: waits for completion (D4) |
| `Couch.Leave` | `cmd/internal/couchcore/park.go` | modified | bounded fan-out (M2) |
| `operationQueue` | `cmd/internal/couchtty/operation_queue.go` | modified | N workers (M2) |

- **ThreadGate** — `acquire(ctx, address, op, wait bool) (context.Context, func(), error)`.
  - **Injected into:** every gated `Couch` entry, via `c.hold` / `c.holdWait`.
    It is in-memory, so tests use the real gate (no fake needed). Ordering is
    controlled through blocking fakes on the work, not on the gate.
  - **Future extensions:** a cross-process lease if CLI operations ever run
    beside a console. Today the supervisor lease prevents that
    (`couchcmd/run.go`).

---

## Chunk 1: M1 — per-thread guard (single worker unchanged)

### Task 1: The gate and its decision rule

**Files:**
- Create: `cmd/internal/couchcore/threadgate.go`
- Test: `cmd/internal/couchcore/threadgate_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func gateAddr(tag string) ThreadAddress { return ThreadAddress{RepoScope: "s", Tag: ThreadTag(tag)} }

func TestGateDecision(t *testing.T) {
	a, b := gateAddr("a"), gateAddr("b")
	held := map[ThreadAddress]string{a: "relaunch"}
	cases := []struct {
		name    string
		holds   []ThreadAddress
		address ThreadAddress
		want    gateVerdict
		running string
	}{
		{"free thread admits", nil, b, gateAdmit, ""},
		{"held thread refuses a stranger", nil, a, gateBusy, "relaunch"},
		{"holder re-enters its own thread", []ThreadAddress{a}, a, gateReenter, ""},
		{"a released hold in a surviving context does not bypass", []ThreadAddress{b}, b, gateAdmit, ""},
		{"a holder of another thread is still refused", []ThreadAddress{b}, a, gateBusy, "relaunch"},
	}
	for _, tc := range cases {
		got, running := gateDecision(held, tc.holds, tc.address)
		if got != tc.want || running != tc.running {
			t.Errorf("%s: got %v/%q, want %v/%q", tc.name, got, running, tc.want, tc.running)
		}
	}
}

func TestThreadGateRefusesNamingTheRunningOperationAndReleases(t *testing.T) {
	var g ThreadGate
	ctx, release, err := g.acquire(context.Background(), gateAddr("a"), "relaunch", false)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = g.acquire(context.Background(), gateAddr("a"), "resume", false)
	var busy *ThreadBusyError
	if !errors.As(err, &busy) || busy.Running != "relaunch" || busy.Address != gateAddr("a") {
		t.Fatalf("want ThreadBusyError naming relaunch, got %v", err)
	}
	// Re-entry through the holder's context neither refuses nor releases.
	_, inner, err := g.acquire(ctx, gateAddr("a"), "resume", false)
	if err != nil {
		t.Fatal(err)
	}
	inner()
	if _, _, err := g.acquire(context.Background(), gateAddr("a"), "resume", false); err == nil {
		t.Fatal("inner release must not free the outer hold")
	}
	// A different thread is unaffected.
	if _, r, err := g.acquire(context.Background(), gateAddr("b"), "resume", false); err != nil {
		t.Fatal(err)
	} else {
		r()
	}
	release()
	if _, r, err := g.acquire(context.Background(), gateAddr("a"), "resume", false); err != nil {
		t.Fatalf("released thread must admit: %v", err)
	} else {
		r()
	}
}

func TestThreadGateWaitAdmitsAfterReleaseAndHonoursContext(t *testing.T) {
	var g ThreadGate
	_, release, _ := g.acquire(context.Background(), gateAddr("a"), "resume", false)
	done := make(chan error, 1)
	go func() {
		_, r, err := g.acquire(context.Background(), gateAddr("a"), "leave", true)
		if err == nil {
			r()
		}
		done <- err
	}()
	select {
	case <-done:
		t.Fatal("wait must block while held")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	_, release, _ = g.acquire(context.Background(), gateAddr("a"), "resume", false)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := g.acquire(ctx, gateAddr("a"), "leave", true); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait must return ctx error, got %v", err)
	}
}
```

- [ ] **Step 2: Run, expect a compile failure.** `go test ./cmd/internal/couchcore -run 'Gate' -count=1`

- [ ] **Step 3: Implement**

```go
package couchcore

import (
	"context"
	"fmt"
	"sync"
)

// LifecycleParallelism bounds how many threads couch drives through a
// lifecycle operation at once: Leave's fan-out, the park worker and the
// console's queue workers (pair#205 D5). Each couch reattach still asks every
// live session for its clients, so N parallel attempts put N×S zellij calls in
// flight; 4 keeps that burst bounded at the slot counts in use.
const LifecycleParallelism = 4

// ThreadBusyError refuses a lifecycle operation on a thread another operation
// holds. Refused, not queued (pair#214 operator decision, carried by #205).
type ThreadBusyError struct {
	Address ThreadAddress
	Running string
}

func (e *ThreadBusyError) Error() string {
	return fmt.Sprintf("%s is busy: %s is already running on it", e.Address.Tag, e.Running)
}

type gateVerdict uint8

const (
	gateAdmit gateVerdict = iota
	gateReenter
	gateBusy
)

// gateDecision is the whole admission rule: one lifecycle operation per
// thread, re-entered by its own holder.
func gateDecision(held map[ThreadAddress]string, holds []ThreadAddress, address ThreadAddress) (gateVerdict, string) {
	running, isHeld := held[address]
	if !isHeld {
		// Also covers a context that outlived its own release: re-entry needs
		// the hold to still exist, or a stale ctx would bypass the gate.
		return gateAdmit, ""
	}
	for _, h := range holds {
		if h == address {
			return gateReenter, ""
		}
	}
	return gateBusy, running
}

type gateHoldsKey struct{}

func gateHolds(ctx context.Context) []ThreadAddress {
	holds, _ := ctx.Value(gateHoldsKey{}).([]ThreadAddress)
	return holds
}

// ThreadGate admits one lifecycle operation per thread. The zero value is
// ready. Holds live in memory only and die with their holder's release.
type ThreadGate struct {
	mu      sync.Mutex
	held    map[ThreadAddress]string
	changed chan struct{} // closed and replaced on every release
}

func (g *ThreadGate) acquire(ctx context.Context, address ThreadAddress, op string, wait bool) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		g.mu.Lock()
		if g.held == nil {
			g.held = map[ThreadAddress]string{}
			g.changed = make(chan struct{})
		}
		verdict, running := gateDecision(g.held, gateHolds(ctx), address)
		switch verdict {
		case gateReenter:
			g.mu.Unlock()
			return ctx, func() {}, nil
		case gateAdmit:
			g.held[address] = op
			g.mu.Unlock()
			holds := append(append([]ThreadAddress(nil), gateHolds(ctx)...), address)
			var once sync.Once
			return context.WithValue(ctx, gateHoldsKey{}, holds), func() {
				once.Do(func() {
					g.mu.Lock()
					delete(g.held, address)
					close(g.changed)
					g.changed = make(chan struct{})
					g.mu.Unlock()
				})
			}, nil
		}
		changed := g.changed
		g.mu.Unlock()
		if !wait {
			return ctx, nil, &ThreadBusyError{Address: address, Running: running}
		}
		select {
		case <-ctx.Done():
			return ctx, nil, ctx.Err()
		case <-changed:
		}
	}
}

// hold acquires the thread for op or refuses with ThreadBusyError; holdWait
// waits for the current holder instead (the drains: Leave, RecoverActiveParks).
func (c *Couch) hold(ctx context.Context, address ThreadAddress, op string) (context.Context, func(), error) {
	return c.gate().acquire(ctx, address, op, false)
}

func (c *Couch) holdWait(ctx context.Context, address ThreadAddress, op string) (context.Context, func(), error) {
	return c.gate().acquire(ctx, address, op, true)
}
```

Add the field `threadGate ThreadGate` to the `Couch` struct (`couch.go`). It
is a plain value, since the zero value is ready, and `Couch` is always used by
pointer:

```go
func (c *Couch) gate() *ThreadGate { return &c.threadGate }
```

- [ ] **Step 4: Run, expect PASS.** Same command, plus `-race`.
- [ ] **Step 5: Commit.** `#205 M1: couchcore: per-thread lifecycle gate`

### Task 2: Nothing outlives its hold (D4)

**Files:** Modify `cmd/internal/couchcore/park.go` (`submit`, ~:538). Test in
`cmd/internal/couchcore/parkworker_test.go` (create it if it's absent).

- [ ] **Step 1: Failing test** `TestSubmitReturnsOnlyAfterTheParkWorkEnds`. The
  work blocks on a channel and records when it ended; cancel `ctx`. Assert that
  `submit` hasn't returned until the channel is released, and that the work
  observed `ctx.Done()`.
- [ ] **Step 2: Run, expect FAIL** (`Await` returns on `ctx.Done()`).
- [ ] **Step 3: Implement.** In `submit`, after `future.Await(ctx)` returns a
  `ctx` error, block on `<-future.done` and then return the work's result
  joined with the `ctx` error. Leave `parkFuture.Await` unchanged for
  `reconcileActive`, which already awaits under its own loop. Comment the
  reason: a hold must not end while the effect it guards is still running.
- [ ] **Step 3b: `parkWorker` releases its address before signalling done.**
  `parkworker.go:79-83` closes `future.done` and *then* deletes `active[address]`.
  So after `submit` returns, the worker still briefly counts the address. A
  following `Park` on that address with a new nonce is refused "another park
  transaction already owns this address", and at full capacity a sibling gets
  `ErrParkWorkerOverloaded`. Reorder to delete under `w.mu`, then close `done`.
  Test: `TestParkWorkerFreesTheAddressBeforeDone`. After `Await` returns, an
  immediate `Submit` on the same address with a new nonce is admitted (loop it
  1000× under `-race`).
- [ ] **Step 4: Run, expect PASS** (`go test ./cmd/internal/couchcore -run 'Submit|ParkWorker' -race -count=1`).
  D4 lengthens console shutdown by at most the time the park work takes to see
  cancellation, which `CompletionTimeout` bounds. `c.workers.Wait()`
  (`couchtty/console.go:936`) already waits on the workers, so nothing hangs.
  Check that the existing quit tests still pass.
- [ ] **Step 5: Commit.**

### Task 3: Gate every lifecycle entry

The acquire goes **where the address is final**, as the first effectful
statement after validation. The idiom:

```go
ctx, release, err := c.hold(ctx, address, "relaunch")
if err != nil {
	return refused, err
}
defer release()
```

| Entry | File | Op name | Mode |
|---|---|---|---|
| `ResumeContextWith` | `resume.go:422` | `resume` | refuse |
| `Relaunch` | `relaunch.go:71` (before `GetThread`) | `relaunch` | refuse |
| `Detach` | `detach.go:46` (before `GetThread`) | `detach` | refuse |
| `SwitchAgent` | `switchagent.go:258` (address from the request) | `switch-agent` | refuse |
| `Continue` | `continuation.go:185` | `continue-thread` | refuse |
| `RetryContinuation` | `continuation_recovery.go:231` | `retry-continuation` | refuse |
| `ReconcileContinuation` | `continuation_recovery.go:182` | `continuation-status` | refuse. The console's continuation scan treats busy as "not yet" and retries on its next scan, silently (Task 4) |
| `rebootPrimary` / `rebootSlot` | `reboot.go`. In `rebootSlot`, hold **after** the `switch` at :226-229, so `ctx` is not shadowed inside a case. The unreadable-record branch has no address and is accepted unheld: it archives a file couch cannot decode | `reboot` | refuse |
| `RecoverThread` | `recovery_execute.go:213`. A composite: it writes (`reconcileRecoveryHelper` :257, `prepareAbsentContinuation` :279) before calling `ResumeContextWith` / `RetryContinuation` | `recover` | refuse |
| `Stop` | `couch.go` (`Stop(record)`: hold `record.Thread` when it is non-zero) | `stop` | refuse |
| **new** `Couch.Park(ctx, address, mode)` | `park.go` | `park` | refuse |
| `Leave`'s per-record step | `park.go:194` loop body, extracted here into `leaveOne(ctx, address, disposition)`: `holdWait`, then `GetThread`, then today's decision | `leave` | **wait** |
| `RecoverActiveParks`'s per-record step | `couch.go:89` loop body | `park-recovery` | **wait** |

`Couch.Park` holds the gate and switches on `mode` to
`PairLifecycle.Park/Retry/Recover/Abandon`. The executor's `"park"` case
(`operationdispatch.go:358`) calls it instead of reaching `c.PairLifecycle`
directly. So the executor no longer holds lifecycle logic; it routes, like every
other case. `ResumeTarget` (path or address) needs no hold of its own. Every
route (`resume_route.go:180-205`) ends in a gated entry *before any write*:
`ResumeContextWith`, `RetryContinuation`, or `RecoverThread`, which is gated
above because it writes first.

**Not gated, with reasons:**
- `DismissContinuation` and `RequestContinuation` are single revision-CAS writes
  to the record, with no process effect.
- `StartFreshSlot` and `start` mint a new address.
- `ArchiveThread` is CAS-guarded and has no live caller.
- `ReconcileActiveParks` has no interactive caller; `RecoverActiveParks` is the
  one used.

`ResumeContextWith` gains `ResumeOptions.WaitForHolder`. `StartInteractive`
(`startup.go:241`) sets it, so the boot resume waits behind `RecoverActiveParks`
instead of failing the operator's first frame.

- [ ] **Step 1: Failing tests** in `cmd/internal/couchcore/threadgate_couch_test.go`,
  built on `envWithLiveThread`:
  - `TestARelaunchDuringAResumeIsRefused`: the Done-when's resume-then-relaunch
    order. Block a cold resume in its launch (`env.Runner` acknowledge hook),
    call `Relaunch`, and assert `*ThreadBusyError` with `Running == "resume"`.
    Once the hook is released, the resume completes and the thread is live.
  - `TestAResumeDuringARelaunchIsRefusedAndTheThreadStaysResumable` (the
    2026-09-08 incident's order):
    1. Set `TriggerQuitHook` to block on a channel. Start `Relaunch` in a
       goroutine and wait until the hook is entered.
    2. Call `ResumeContext` on the same address and assert `*ThreadBusyError`
       with `Running == "relaunch"`.
    3. Unblock the hook. `Relaunch` succeeds (`Outcome == Relaunched`).
    4. The ledger has exactly one new launch since the park.
    5. Then `Detach` and `ResumeContext` succeed. This is #214's Done-when,
       carried over.
  - `TestTheGateDoesNotSerialiseDifferentThreads`: two threads, a blocked
    relaunch on A; `Detach(B)` completes.
  - `TestEveryLifecycleEntryRefusesAHeldThread`: table over the entries above
    in refuse mode, `RecoverThread` and `Stop` included. Hold the address with
    `c.hold(ctx, addr, "test")`, call each entry and assert `*ThreadBusyError`. This test is the enumeration that
    keeps a future entry from skipping the gate (ARCH-PURPOSE: the class, not
    the instance).
  - `TestLeaveAndParkRecoveryWaitForAHolder`: hold A, start `Leave(LeaveDetach)`
    in a goroutine, assert it hasn't returned, release, assert A is in
    `Detached`.
  - `TestLeaveDecidesFromTheRecordAfterWaiting`: A is detached in the snapshot.
    Hold A, start `Leave`, then make A live with a new incarnation (a warm
    resume) and release. Assert `Leave` detached A rather than skipping it on
    the stale snapshot row. In Task 3 `leaveOne` already takes `holdWait`
    first and then re-reads with `GetThread`; Task 7 only adds the fan-out.
  - `TestStartupResumeWaitsBehindParkRecovery`: hold A as `park-recovery`, call
    `ResumeContextWith(WaitForHolder)` in a goroutine, release, and assert it
    proceeds.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** the table above, one entry per commit-sized edit.
- [ ] **Step 4: Run** `go test ./cmd/internal/couchcore -race -count=1`. Expect PASS,
  including the existing relaunch, switch-agent, continuation and park suites
  (re-entry keeps the composites working).
- [ ] **Step 5: Commit.**

### Task 4: The refusal reaches the operator

**Files:** `cmd/internal/couchtty/console.go` (`finishOperation` notice path),
`cmd/internal/couchtty/menu_reattach.go` (`finishReattach`),
`cmd/internal/couchtty/console_remote.go`. Tests sit beside each.

- [ ] **Step 1: Failing tests.**
  - An operator operation whose dispatcher returns `*ThreadBusyError` shows a
    notice containing `"is busy: relaunch is already running"`.
  - A background reattach attempt refused busy is **skipped silently** like
    cell 6, not marked `reattach failed`. The thread is being acted on by
    someone else, so its row is not a failure.
  - A remote job refused busy reports the error text to its caller.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement.** `finishReattach` sees only `event.Diagnostic`,
  which `console.go:1844` sets from `couchcore.ResumeDiagnosticOf(err)`, so
  busy has to arrive as a diagnostic code:
  - Add `ResumeThreadBusy` to the resume diagnostic codes.
  - Teach `ResumeDiagnosticOf` to return it for a `*ThreadBusyError` (via
    `errors.As`).
  - In `finishReattach`, add it to the cell-6 skip case alongside
    `ResumeNotDetached` and `ResumeSessionGone`.
  - In the console's continuation scan, an operation refused busy leaves
    `watch.queued` false, so the next scan retries it without a notice.
  - The operator notice and the remote path already carry `err.Error()`;
    assert rather than change them unless a test shows otherwise.
- [ ] **Step 4: Run** `go test ./cmd/internal/couchtty -race -count=1`. Expect PASS.
- [ ] **Step 5: Commit.**

### Task 5: M1 close

- [ ] Atlas: add the gate to `atlas/couch.md`'s lifecycle section: one paragraph
  covering the rule, the drains, re-entry, and a pointer to `threadgate.go`.
- [ ] Copy the ARCH-ORDER table into the issue's `## Spec`, answering the open
  interleaving cells. Log it in the issue Revisions.
- [ ] Full verification per the repo's test memory: unsandboxed `make -k test`,
  then `go test ./...`. Record the results in the issue Log.
- [ ] `sdlc milestone-close --issue 205 --milestone M1 --verified '<evidence>'`.

---

## Chunk 2: M2 — bounded parallelism

### Task 6: Measure the baseline first

- [ ] Record one traced restart at the live slot count before any M2 change:
  `COUCH_TRACE=<scratchpad>/before.trace` on quit and on the next start, plus
  `PAIR_PROBE_SAMPLE_SECS=60 make test-reattach-cost` in sample mode across it
  (`cmd/probes/reattachcost`).
- [ ] Log the following in `## Log`, with the load average and agent count
  (`workshop/targets/workbench-latency.md`):
  - the quit duration;
  - first-frame time;
  - the pass wall-clock;
  - `zellij action` p50, p95 and max.
- [ ] The operator runs quit and start (zellij needs the sandbox off). Ask before
  starting.

### Task 7: `Leave` fans out (D6)

**Files:** `cmd/internal/couchcore/park.go` (`Leave`). Tests in a new
`cmd/internal/couchcore/leave_test.go`.

- [ ] **Step 1: Failing tests.**
  - `TestLeaveDetachesThreadsConcurrentlyUpToTheBound`: 6 live threads whose
    detach blocks on a shared barrier. Assert exactly `LifecycleParallelism`
    are in flight at the peak, and all 6 end up `Detached`.
  - `TestLeaveFinishesSiblingsWhenOneFails`: one detach errors. The others
    are in `Detached`, and the error names the failed tag.
  - `TestLeaveResultIsInSnapshotOrder`: fan-out must not reorder the operator's
    report.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement.**
  - The loop body is already `leaveOne` from Task 3, which takes `holdWait`, then
    re-reads with `GetThread`, then decides. Task 7 only drives it
    concurrently.
  - Run it over the snapshot with a semaphore of `LifecycleParallelism`.
  - Collect into a slice indexed by record position, then build the
    `LeaveResult` lists in snapshot order. Return `errors.Join` of the
    per-thread errors.
  - `ctx` cancellation stops *starting* records; started ones finish (D4).
  - Replace the "Serial by choice" comment with the new reason and D5's bound.
- [ ] **Step 4: Run** `-race`. Expect PASS, including the existing leave tests in
  `couchtty`.
- [ ] **Step 5: Commit.**

### Task 8: The park worker admits the bound

- [ ] `submitFuture`: change `newParkWorker(1)` to
  `newParkWorker(LifecycleParallelism)`.
- [ ] Test: N parks on distinct threads run concurrently up to the bound, and
  the bound+1th gets `ErrParkWorkerOverloaded`. Then decide: should
  `Leave(LeavePark)` with more threads than the bound queue inside `leaveOne`
  instead of failing? Its semaphore already caps it at the bound, so assert
  that `Leave` never sees `ErrParkWorkerOverloaded`.
- [ ] Commit.

### Task 8b: The actor registry is safe under concurrent operations

**Files:** `cmd/internal/couchcore/couch.go`, `launch_existing.go`. Test in
`cmd/internal/couchcore/registry_concurrency_test.go`.

`c.reg` and `c.names` are read-modify-written with no lock:
- `launch_existing.go:260-263` (`withoutDead`, `Insert`, `Store.Save`);
- `couch.go:815` (`AbortStarted`);
- `couch.go:1110` (`Forget`, which the console goroutine calls via `SetForget`,
  `couchcmd/run.go:749`).

With parallel workers, two inserts can start from the same old registry, and
one actor record is lost both in memory and on disk. A small version of this
race exists today, between `Forget` on the console goroutine and the single
worker.

- [ ] **Step 1: Failing test.** `TestConcurrentLaunchesKeepEveryActorRecord`
  runs 8 goroutines through the registry insert path on distinct threads, with
  a concurrent `Forget` loop, under `-race`. Assert the race detector stays
  clean, the saved registry holds all 8 actors, and the in-memory registry
  holds all 8.
- [ ] **Step 2: Implement.**
  - Add `regMu sync.Mutex` to `Couch`. Every read-modify-write of
    `c.reg`/`c.names` and its `Store.Save` happens under it, as one critical
    section per writer.
  - Readers take the lock to copy the value (the registry is an immutable
    value type, so a copied value is safe to use after unlocking).
  - Route all ~20 `c.reg`/`c.names` uses through two helpers,
    `c.registry() (Registry, Names)` and
    `c.mutateRegistry(func(Registry, Names) (Registry, Names, error)) error`,
    so a future writer cannot skip the lock (ARCH-DRY).
- [ ] **Step 3: Run** `go test ./cmd/internal/couchcore -race -count=1`. Expect PASS.
- [ ] **Step 4: Commit.**

### Task 9: The reattach pass keeps up to the bound in flight

**Files:** `cmd/internal/couchtty/menu_reattach.go`, `console.go:687` (the one
external reader of `Loading`), `menu_reattach_test.go`.

- [ ] **Step 1: Failing tests.**
  - `advanceReattach` emits up to `LifecycleParallelism` effects when the queue
    is long. It emits none while the operator holds `InFlight` (cell 10,
    existing test kept).
  - `finishReattach` with completions in reverse order resolves each by attempt
    number.
  - A completion for an unknown attempt is ignored (cell 8).
  - Generated-sequence test: random interleavings of seed, advance, finish
    (success, failure or busy) and operator in-flight on/off, under fixed
    seeds. Invariants:
    1. In-flight size never exceeds the bound.
    2. Every queued thread is attempted at most once.
    3. The pass reaches `ReattachDone` exactly when the queue and in-flight
       set are both empty.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement.**
  - Replace `Loading`/`LoadingAttempt` with `InFlight map[uint64]couchcore.ThreadAddress`.
  - `advanceReattach` loops while `len(InFlight) < bound` and the queue is
    non-empty.
  - `PassView`'s `PassLoading` comes from membership in `InFlight`'s values.
  - `pendingPlaceholders` (`menu_reattach.go:291-307`) builds its chips as the
    in-flight threads **sorted by attempt number**, then the queue. Map order
    would jitter the chip columns every frame. Add a test that renders twice
    and compares the chip order.
  - Update the other consumers of loading state: the status-tick check at
    `console.go:687`, and the "the one placeholder" comment at `reserve.go:45`.
- [ ] **Step 4: Run** `go test ./cmd/internal/couchtty -race -count=1`.
- [ ] **Step 5: Commit.**

### Task 10: The console queue runs the bound's worth of workers

**Files:** `cmd/internal/couchtty/operation_queue.go`, `console.go:629`.

- [ ] **Step 1: Failing test** in `operation_queue_test.go`: with
  `Run` started `LifecycleParallelism` times, that many blocking requests are
  in flight together, and results are still delivered through `q.results`.
- [ ] **Step 2: Implement.** Start `couchcore.LifecycleParallelism` `Run`
  goroutines under `c.workers`. `Run` is already safe for concurrent workers
  (`pending` is under `q.mu`; the channels are shared).
- [ ] **Step 3: Re-run the console suites with `-race`.** Expect these to pass
  unmodified:
  - `TestAnOperatorSwitchWaitsBehindAtMostTheRunningAttempt`. It must still hold:
    with the pass bounded, the operator waits behind at most the in-flight
    attempts. If the test assumed exactly one, update its *statement* to the
    bound and record why in the Log.
  - `TestAReattachedChildKeepsItsTrackingMode` (#196) unchanged, plus a new
    variant that completes N reattaches in one burst.
  - New test, `TestALateCompletionDoesNotTouchANewerHolder`. An attempt on A
    completes and is adopted. Its failed sibling's `AbortStarted` runs after a
    new operation has taken A's hold. Assert that the abort removes only its own
    actor ID and never the newer incarnation.
- [ ] **Step 4: Commit.**

### Task 11: Counted invariant and measurement

- [ ] Add to the #204 counting suite: `Leave` of N threads issues O(N) store
  writes, and the pass issues at most N resume calls.
- [ ] Repeat Task 6's measurement at the same slot count. Log the before/after
  table with co-tenancy. Done-when needs a material reduction in quit duration
  and pass wall-clock, and no `zellij action` p95 regression beyond the quiet
  baseline's spread.
- [ ] Ask the operator for a live smoke test: quit with 18 slots, restart, and
  check that every live thread reattaches with no `reattach failed` rows.

### Task 12: M2 close

- [ ] Atlas: update `atlas/couch.md` on leave, the reattach pass and the bound.
- [ ] Full verification as in Task 5.
- [ ] `sdlc close --issue 205 --verified '<evidence>'`.
