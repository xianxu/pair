---
id: 000480
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f3a21e9a22e8cbc70438b2c15d15ae2bcfce382b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T05:53:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0d487d66", done: "44b95c2e"}
actual_hours: 0.88
---

# tests: inject clocks or sync points where tests race a real timer or sleep as sync (idle floor, stdin ESC flush, wait-notice ticks, ttyio partial write, ...)

## Problem

Found by the #476 sweep (fixed real-time windows asserted under load, the #473/#476 flake family). Ranks are a reading of the code, not measured failures. Paths are under `cmd/internal/`.

These tests race a real timer, or sleep and then assert that something already happened. A slow scheduler makes them fail even though the product is right:
- wrapcmd/idle_floor_test.go:125-131: 60 ms idle floor with 15 ms sleeps between outputs; one oversleep fires a real alert, so "no alert" falsely fails.
- wrapcmd/translate_stdin_test.go:151 (5 ms sleep vs a 20 ms flush, "flushed too early"), :181 and translate_test.go:306 (second write must beat a 100 ms / 50 ms ESC flush timer).
- couchcmd/wait_notice_test.go:264 `tickAt` (~17 callers, :274-425): sleeps 150 ms as sync on socket delivery, then asserts `len(delivered)`. High.
- ttyio/writer_test.go:45: an 80 ms ctx must expire only after the pty write made partial progress (n>0). High. :238 asserts elapsed <= 4 s on a 3 s ctx; :257 asserts a cancel at 200 ms returns within 1 s.
- couchcore/registration_phase_test.go:80: the injected fault fires only within 200 ms of pane birth; a slow poll skips it.
- couchcore/never_turned_test.go:50 (refusal under 200 ms), launcher/pane_birth_test.go:223 (returns within 1 s with the production 10 s bound in play), couchmessage/broker_actors_test.go:57 (infers "no probe" from 100 ms latency; `restingProbes` already counts probes).
- broadcast/pointer_test.go:297 (sleep 100 ms, then expects a 429), hostty/hostty_test.go:236 (sleep 200 ms, then expects at least one SIGWINCH wake).

## Spec

Replace each real timer or sleep with the event or count it stands for: an injected clock or timer seam (the #473 `Couch.after` pattern), a counter the fake already keeps (`restingProbes`), or a channel signalled when the state is reached (in-flight == Max, first wake). Fix product code if the race is real there. Prove each window first by widening it with a temporary sleep. Split the issue if one item needs a product seam big enough to review alone.

## Done when

- Each listed test asserts an event, count or injected clock instead of wall time, and passes 200 runs under `-race` and parallel load.
- Each cause is recorded in the Log, measured by widening.

## Plan

- [x] Per test, prove the window by widening it with a temporary sleep; record the cause in the Log.
- [x] Replace each real timer or sleep with an injected clock/timer, an existing counter, or a state signal; fix product code if the race is there.
- [x] 200 `-race` runs of each touched test under parallel load; full `make -k test`.

## Log

### 2026-10-12
- 2026-10-12: closed — close-review advisories fixed: ttyio counts from before the reader starts and cancels before counting; idle comment corrected; ttyio 100x and idle 50x under -race green (fixed-then-ship); review verdict: SHIP
- 2026-10-12: closed — 14 tests (12 listed + TL's orientation deadline + split-shortcut) now wait on events/counts/injected timers; each window proven by widening (old fails, new passes); 200x -race soak per touched test with all packages concurrent: green (1 soak failure found+fixed, re-soaked 200x); make -k test exit 0 + go test ./... exit 0; no-atlas: new seams (loopTimer, registrationDeadline, BirthClock) are test hooks defaulting to wall clock, no new surface; review verdict: SHIP

Every cause below was proven by widening its window with a temporary sleep (the old test failed every run), and the fix was rerun with the same widening and passed. None of these races was in the product; three product seams were added so tests can cause the event themselves (ARCH-MOCK: the seam is the production timer with an injectable source, nil = wall clock).

- **couchcmd wait-notice `tickAt`** (17 callers): the broker admits a notice inside `tick` (Reserve runs on the Send path) but delivers it later on the hub's actor goroutine; the 150 ms sleep raced that delivery. Widened: 200 ms in the fake hub's `Deliver` → 4 tests failed. Fix: the fake hub records admitted IDs at Reserve (less refusals released undelivered; Release follows every delivery too) and `tickAt` waits until deliveries reach that count. A wrongly sent notice is counted synchronously, so "no notice" assertions stay exact.
- **ttyio writer** (4 tests): the 80 ms deadline expired before a late writer goroutine wrote a byte (widened 100 ms → "expected known partial progress"); the 3 s stall test bounded elapsed at 4 s; the cancel test bounded elapsed at 1 s; `drainUntilQuiet` could stop early if the reader stalled past its 300 ms quiet window. Fixes: cancel after the slave is observably readable (poll POLLIN; x/sys has no portable FIONREAD) and bound busy polling as `2*held/pollSlice+10` (busy-poll mutant: 7570 waits, caught); drop the elapsed upper bound (a deadline-ignoring write hangs on the stalled host → binary timeout); measure cancel promptness as ≤1 poll wait after cancel; drain exactly the accepted byte count, then check EAGAIN. The transient-stall test gives the caller no deadline (a transport-imposed one, #383, still fails it). `pollSlice` named in writer.go.
- **wrapcmd idle floor**: the alert test waited 400 ms for an alert to show up (widened 450 ms in the sink → 0 notifications); the output test's 15 ms writes against a 60 ms floor were raced by the scheduler (widened 200 ms gaps → false alert). A causal check on write timestamps was tried and rejected: it fails when the loop, not the writer, stalls. Fix: `proxy.newIdleTimer` (a `loopTimer` seam; `wallTimer` in production) and a `manualTimer` the test advances; the output test syncs on each chunk reaching stdout (the bump happens in the same loop step that queues the chunk). Mutant (no `bumpIdleDeadline`) fails.
- **wrapcmd held-ESC flush** (`translate_stdin_test` ×2, `translate_test` split shortcut): same seam, `proxy.newStdinFlushTimer`. The split-shortcut race could never fail (a flushed ESC then `j` gives the same bytes); it only meant the test sometimes didn't exercise the reassembly path it names. Now it always does.
- **couchcore registration phase**: two windows. (1) The fault fired only within 200 ms of birth; widened (210 ms stall in the first check) → "last ownership check: none". (2) The 150 ms budget had to hold a completed poll; the same widening gave "0 ownership polls". Fix: the fault lasts until the post-ack cleanup's first `Couch.after` call (its TERM wait, which runs before the cleanup reads the session; failing forever hangs the cleanup, measured), and a new `Couch.registrationDeadline` seam lets the test spend the budget on the second check, so exactly one poll precedes it (`1 ownership polls` asserted).
- **couchcore never-turned refusal**: "under 200 ms" meant "before the registration wait"; now asserted by the wait never taking its budget from `registrationDeadline`.
- **launcher pane birth**: "returns within 1 s with the 10 s bound" → new `LaunchOptions.BirthClock`; a stopped clock means the bound can't lapse, so returning at all proves the watch ended with its client.
- **couchmessage broker actors**: the 100 ms latency inferred "no Observe probe"; the blocking endpoint now counts Observe calls.
- **broadcast pointer slow body**: a 100 ms sleep before expecting 429, then a 7 s poll for recovery; now waits for `len(inFlight) == MaxPointInFlight`, then for the read deadline to free all slots, then expects 204.
- **hostty SIGWINCH**: a 200 ms sleep before counting wakes (n==0 if the watcher ran late; n==2 if signals were still arriving while the test drained). Now: wait for a pending wake, `Close` (joins the watcher; a blocking send would hang it), then exactly one buffered wake.

One more found by TL's runs (pair:6), added to this issue:

- **couchcore `TestOrientationSessionObservationHonorsCallerDeadline`**: the 20 ms caller context could run out while the reader was still reading the ready file, before it asked the session observer (`observed=false`). Widened (30 ms before the call) → reproduced, 2 of 3 subtests. Fix: the `spendableBudget` context from the registration test (a distant deadline the test spends), spent inside the observer, so the deadline the reader must honor lapses only once the observer holds it.

## Revisions

- 2026-10-12: TL moved the rec-blog fidelity flake (`TestLiveFidelityFixtures`, frame 0 at 6.75%) to pair#482. A replayer fix was drafted here and reverted. Findings were sent to TL for #482: reproduced 1 of 20 with 4 concurrent Chromes (plain CPU load 0 of 12); the worst crop showed the replay without the avatar image and the scrollbar; capturing after pausing at t and then `settle()` gave 40 of 40 clean.
