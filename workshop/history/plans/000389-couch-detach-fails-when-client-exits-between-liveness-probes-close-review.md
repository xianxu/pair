# Boundary Review — pair#389 (whole-issue close)

| field | value |
|-------|-------|
| issue | 389 — couch detach fails when client exits between liveness probes |
| repo | pair |
| issue file | workshop/issues/000389-couch-detach-fails-when-client-exits-between-liveness-probes.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6bab73a96b4bccf5854f515e2bc1215cbc14f2e7..e791b5ce1dcf5a54d8f107a419d3ef1bbd87d0a7 |
| command | sdlc close --issue 389 |
| reviewer | claude |
| timestamp | 2026-10-03T12:05:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix itself is correct and well tested, but it isn't complete. `observeExactProcess` (`cmd/internal/couchcore/couch.go:999`) now checks `Exists` again when the identity read fails. A pid that has since gone (ESRCH) counts as `Dead`, and a pid still present with no identity stays `Unknown`. The new test fails without the fix, and the guard test passes. I ran `go test -run 'TestExactProcess|Detach'` and it passed. The gap is a second copy of the same two-probe logic, `Couch.Liveness` (`couch.go:1040`), which did not get the fix. The Spec says "every caller (detach, inventory, liveness) gets it", and for the `Couch.Liveness` path that is not true. It is cheap to fix by having that function call the shared helper.

1. **Strengths**
   - The fix is in the shared observation, not in detach's poll loop, so all ~30 `observeExactProcess` callers get it (detach, inventory, recovery, slot recovery, park, debris).
   - It stays conservative. Only a confirmed ESRCH becomes `Dead`. If the pid is reused between the probes, the re-check sees `Live` and the result stays `Unknown`. The "leave it alone" rule still holds.
   - The `ReapedOnIdentity` fake hook (`procops.go:154`) models the exact interleaving behind the report. The test covers both clauses of `## Done when`: `observeExactProcess` returns Dead, and `awaitExactProcessExit` returns nil.
   - `TestExactProcessPresentWithoutIdentityStaysUnknown` covers the other state, so the re-check can't quietly turn real uncertainty into an exit.

2. **Critical:** none.

3. **Important**
   - `couch.go:1040` `Couch.Liveness` copies the old `observeExactProcess` line for line and still returns `Unknown` when an identity read fails. It is called from `switchagent.go:94`, `couch.go:1177` and `couch.go:1199` (dead-actor prune/forget), so those paths can still hit the race. This contradicts the Spec's claim that "liveness" callers get the fix (ARCH-DRY, ARCH-PURPOSE). Fix: keep the `PID==0 || Identity==""` guard, then `return observeExactProcess(c.Proc, ProcessIdentity{PID: a.PID, Identity: a.Identity})`. Add a `Liveness` test using `ReapedOnIdentity`.
   - Other places that run the same Exists-then-Identity sequence by hand:
     - `supervisorlease.go:108-122` (`VerifiedOwner`)
     - `supervisor_observe.go:77-85`
     - `switchcontext.go:232-240` (orientation target)

     Each reports an error rather than `Unknown`. For the orientation target, a target reaped between the probes gives a raw "no identity token" error instead of "orientation target exited". For both supervisor sites, it gives a misleading "verify/cannot verify supervisor pid" error. These are lower impact, because failing is acceptable there, but they belong to the same family. Routing them through `observeExactProcess`, or at least the orientation one, would close it out.

4. **Minor**
   - Zombie window on darwin: `kill(pid,0)` still succeeds on a zombie. If `sysctl kern.proc.pid` doesn't give a zombie a matching `P_pid`/start time, the re-check sees `Live` and stays `Unknown`. I haven't verified this. It's worth a note in `## Log` if the symptom comes back.
   - `TestExactProcessReapedBetweenProbesIsDead` relies on `ReapedOnIdentity` staying set across the `proc.Set` re-seed. That's fine, but a one-line comment would help.

5. **Test coverage notes:** both Done-when states are tested, and the regression test is red without the fix (`Unknown`). `Couch.Liveness` has no matching test, which is why its copy of the bug wasn't caught.

6. **Architectural notes**
   - **ARCH-DRY: flag.** `Couch.Liveness` duplicates `observeExactProcess`, and the supervisor and orientation sites repeat the same probe sequence. One helper should own the two-probe sequence.
   - **ARCH-PURE: pass.** The logic stays behind the injected `ProcOps` seam, and the fake is a stateful process table, not a call mock.
   - **ARCH-PURPOSE: flag.** The stated goal of "every caller" isn't met for `Couch.Liveness`, because it copies the logic instead of deriving from the shared helper.

7. **Plan revision recommendations:** add a `## Revisions` entry to the issue saying `Couch.Liveness` now delegates to `observeExactProcess`, or narrow the Spec's "liveness" claim if that is deliberately left out. Add a Plan item for the `Liveness` test.

```findings
findings:
  - id: new
    severity: Important
    family: single-source-process-observation
    title: |
      Couch.Liveness duplicates observeExactProcess and keeps the two-probe race the fix closed
    detail: |
      couch.go:1040 Couch.Liveness repeats the pre-fix Exists-then-Identity logic line for line and still returns Unknown when an identity read fails; it is called from switchagent.go:94, couch.go:1177 and couch.go:1199, so the Spec's claim that liveness callers get the fix is false. Fix: keep the zero guard, then return observeExactProcess(c.Proc, ProcessIdentity{PID: a.PID, Identity: a.Identity}), and add a ReapedOnIdentity test for Liveness. Same family, lower impact (these report an error instead of Unknown): supervisorlease.go:108-122 VerifiedOwner, supervisor_observe.go:77-85, switchcontext.go:232-240 orientation target (a target reaped between the probes gives a raw no-identity error instead of the exited message).
  - id: new
    severity: Minor
    family: zombie-identity-window
    title: |
      darwin zombie between exit and reap may still give Unknown if sysctl lacks its identity
    detail: |
      kill(pid,0) succeeds on a zombie, so if kern.proc.pid gives a zombie no matching identity, the re-check sees Live and stays Unknown. Unverified; log it if the symptom comes back.
```

---

## Re-review — 2026-10-03T12:20:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 389 — couch detach fails when client exits between liveness probes |
| repo | pair |
| issue file | workshop/issues/000389-couch-detach-fails-when-client-exits-between-liveness-probes.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6bab73a96b4bccf5854f515e2bc1215cbc14f2e7..22fb27b2aa9bdeced83241c02374b31886d1be0d |
| command | sdlc close --issue 389 |
| reviewer | claude |
| timestamp | 2026-10-03T12:20:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change delivers what the Spec asks for. `observeExactProcess` (`cmd/internal/couchcore/couch.go:1004-1025`) is now the only place in the window that asks "is this exact process still running". It probes `Exists` again when the identity read fails, so a client reaped between the two probes reads `Dead`. A process that is still present but whose identity can't be read stays `Unknown`. BR-1 is fixed:
- `Couch.Liveness` keeps its zero guard and then delegates to `observeExactProcess`.
- The three sites that had copied the two probes now delegate too: `supervisorlease.go` `VerifiedOwner`, `supervisor_observe.go` and `switchcontext.go` `ReadOrientationStatus`.
- A fourth copy that I didn't flag last round, in `couchcmd/message_service.go`, also delegates now, through `Liveness`.

I grepped for leftover copies outside tests and found none:
- Every other `.Identity(` call reads the current process or a handle the code itself started.
- The only other `Exists(` call is `switchcontext.go:328`, which checks the pid alone (no identity) and was already there before this window.

The three new tests pass. In `couchcore` and `couchcmd`, all of the failures I saw are pty, launcher or worktree tests, and the log shows 42 `operation not permitted` lines. That is the known sandbox limit on pty child tests, not something this change caused.

1. **Strengths**
   - `couch.go:999-1020`: a single shared observation with a comment explaining why copies are banned. That fixes the cause of the bug rather than one copy of it (ARCH-DRY: pass).
   - `procops.go:200-203`: the `ReapedOnIdentity` fake hook removes the pid the moment its identity is read, which reproduces the real race window exactly. With the second probe reverted, `TestExactProcessReapedBetweenProbesIsDead` would get `Unknown` and fail, so the test really guards the fix.
   - `detach_test.go:384-393`: the guard test shows the second probe doesn't turn genuine uncertainty into `Dead`. This covers the other half of the Done-when clause.
   - The delegating callers now give clearer errors: "exited or was replaced" for `Dead` and "cannot verify" for `Unknown`, instead of a raw identity-read error. Nothing in the tree matched on the old error strings.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage**
   - All three Done-when clauses are tested: the reaped pid gives `Dead` from the observation and nil from `awaitExactProcessExit`, a present pid with no identity gives `Unknown`, and `Liveness` inherits the fix.
   - The converted callers (`VerifiedOwner`, `ObserveSupervisor`, `ReadOrientationStatus`, the message service) have no test of their own for the reaped case. That's acceptable because each is now a one-line delegation to the tested helper.
6. **Architecture**
   - ARCH-DRY: pass. One observation helper, and every exact-process check in the window goes through it.
   - ARCH-PURE: pass. The logic takes the `ProcOps` seam as a parameter and is tested against the stateful fake.
   - ARCH-PURPOSE: pass. The Spec's promise that every caller gets the fix is now true: I checked each consumer and all of them go through the helper.
7. **Plan revisions:** none needed. The Spec still matches the code; the plan checkboxes are still unticked, which the close step will handle.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Liveness delegates to observeExactProcess (couch.go:1049) with TestLivenessReapedBetweenProbesIsDead; VerifiedOwner, ObserveSupervisor, ReadOrientationStatus and message_service also delegate now; grep finds no other Exists-then-Identity copy outside tests.
  - id: BR-2
    disposition: withdrawn
    note: |
      Speculative and conditional (log it only if the symptom comes back); darwin sysctl kern.proc.pid normally still returns a zombie's start time, and nothing in this window can be changed to verify it.
```
