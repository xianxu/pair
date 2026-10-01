# Boundary Review — pair#365 (milestone M2)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | b36fac3d2229037e71362a9c837c1915dea2d0a6..5bc76b196ef187bd832eeaffc9c1c6dd4a27a0cf |
| command | sdlc milestone-close --issue 365 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-01T15:28:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've finished the M2 boundary review. The verdict is FIX-THEN-SHIP: one Important finding and three Minor ones, and nothing blocks the gate.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 does what it set out to do. Messaging now learns about wrappers and panes from lifecycle events instead of polling: a pure `Registry` reducer, a long-lived wrapper session socket checked against the kernel's peer PID, Console pane updates through a coalescing mailbox, and a session client that sends nothing while idle. The 1 s heartbeat, the 1 s reconcile, the 10 s verification window and messaging's own `git status` are all deleted. The use-time check (`messageAuthority.current`) spawns no processes, and the idle test passes with three slots. I ran the couchmessage, couchcmd, couchtty and wrapcmd packages outside the sandbox, which blocks pty children and `/tmp`: all four pass. couchcore only ran inside the sandbox, where its pty-child and `/tmp` tests fail with "operation not permitted", so I have no clean result for it.

The Important finding: one place in the service ignores a failed result. If `broker.Register` fails at `message_service.go:453`, the code returns silently. The Registry still reports the binding as connected and nothing ever retries. This is cheap to fix.

**Strengths**
- **The reducer is a real state machine.** One phase per session, no stray flags; state changes only through `Advance`, and the loop is the only goroutine that owns it (`registry.go:114`, `message_service.go:400`). Satisfies ARCH-ORDER.
- **Random-ordering test.** `TestRegistryInterleavingsKeepInvariants` (`registry_test.go:151`) runs 400 seeds of random event order. After each step it checks that a simulated broker, driven only by the effects, agrees with the registry; that no slot has two connected bindings; and that an older session never stays connected once a newer one is admitted. It also checks the generator actually reached every effect, so the test can't pass vacuously.
- **Expensive probe confined to admission.** The ps/zellij ownership probe now runs only at admission (`live`), while use-time checks read files only (`current`). `PairSessionName` and `RecordedSession` each have a test proving they never call zellij or the session observer (`artifactcollision_zellij_test.go`, `switchcontext_test.go`).
- **No gap at startup.** `SubscribeMessageLifecycle` replays panes that attached before the service started, under the same lock that installs the mailbox, and a reattached pane gets a fresh handle (`console_messages.go:21`).
- **Legacy wrappers cost nothing.** Old `register` requests get `unsupported` without running any checks, and a test pins that (`message_service_test.go:435`).

**Critical**
- None.

**Important**
- **Failed broker registration is dropped** (`message_service.go:453`, and `:459`/`:470` for `ReconcileObservation`). The plan's own ARCH-ORDER rule says "the IO shell executes declared effects and returns their outcomes as events"; the Connect step doesn't.
  - **How it fails:** the broker never deletes actor tombstones (no `delete(b.actors…)` anywhere; that cleanup is M3 Task 3.3). After 128 distinct wrapper launches in one Couch lifetime, `Register` fails with "actor capacity reached". The service returns silently.
  - **Result:** `isConnected` says yes, the broker has no actor, sends answer `unavailable`, and no event ever retries. Before #365 the 1 s heartbeat at least retried.
  - **Fix:** turn a `Register` failure into an event (for example an `AdmissionDone` with an error, so the session gets the normal backoff and then goes dormant), or have the worker register before `AdmissionDone`. Log it either way, and add a service test with a fake `Register` failure.

**Minor**
- **An older session can be displaced by a newer one that will never be admitted** (`registry.go:191`, `newerSessionForSlot`). Any newer session for the same slot that isn't displaced counts, including one stuck awaiting a pane, rejected, or dormant. Re-admitting an older session that was working (say after a detach/reattach) then displaces it permanently while the newer one never connects. The invariant test only checks safety, not that the slot stays reachable, so it can't catch this. Either require the newer session to be admitted or admitting, or write down that this is intended.
- **The plan has drifted from the code** (2nd instance of `plan-revision-drift`). The rule rather than the instance: at each milestone close, the Revisions entry should check every Core-concepts row and every test bullet for that milestone against the tree. In this round:
  - `ReconnectBackoff` lives in `session_protocol.go`, not the `backoff.go` the plan names.
  - Task 2.3's test that kills a helper subprocess and expects `SessionClosed` doesn't exist; client loss is only tested by cancelling a context.
  - Task 2.6's wrapper-level test against a real broker (idle sends no frames, re-hello after a broker restart) was delivered at the `SessionClient` level instead.
- **Per-send goroutine and a slot-name mismatch** (`message_service.go:542`). Each exact-target send spawns an untracked goroutine to post `SendTargeted` (it ends when the service shuts down). The match is a raw `Slot == Target`, so a send addressed by family alias (if aliases can name exact targets) won't wake a dormant session.

**Test coverage**
- The idle test sleeps 2.5 s in real time rather than using a fake clock. It's still a sound check, because no messaging timers remain.
- No test exercises a Connect that fails at the broker; the Important finding above should add one.
- No test checks that `startPeerRuntime` dials the `registry` socket. It's a one-line change, but the production wiring is untested.

**Principle-by-principle check**
| Principle | Result |
|---|---|
| ARCH-DRY | Pass. `transportListen` and `transportReadFrameLimit` are shared with the old transport; the three old freshness mechanisms are now one reducer. |
| ARCH-PURE | Pass. The Registry is pure; the service loop is a thin shell around it. |
| ARCH-PURPOSE | Pass for M2's scope. The crash suite, duplicate guard and tombstones are explicitly M3. |
| ARCH-MOCK | Pass. `messageWorld` is a stateful fake behind the authority seam, and the sockets in tests are real. |
| ARCH-CONSTRAINTS | Pass. Admissions run at most 4 at a time, sessions are capped at `MaxActors`, activity is limited to 1 frame/s, and reconnects back off up to 5 s. |
| ARCH-SECURE | Pass. Frames are strictly decoded with a size cap; the peer PID must equal `Binding.PID`; activity and submit frames apply only to the sending connection's own binding. |
| ARCH-ORDER | Flag. See the Important finding: a failed effect doesn't come back as an event. |
| ARCH-FUNERAL | Pass. `prepared` is dropped once the loop processes that admission's result (`message_service.go:412`); `workspaces` shrinks on Disconnect; sessions are deleted on close. Broker tombstones are still unbounded, a known gap scheduled for M3 Task 3.3. |

**For upcoming work**
- M3's tombstone cleanup and the Important fix should land together; together they close the "silently unreachable" case.
- The crash suite should include an event sequence where the effect fails, not just sequences where the reducer reorders events.

**Plan revisions needed**
- Add to the M2 entry: `ReconnectBackoff` is in `session_protocol.go`. Task 2.3's process-kill test was replaced by context-cancel close plus the server-restart test. Task 2.6's real-broker wrapper test is covered by `TestSessionCoalescesActivityAndSendsSubmitAtOnce` and `TestSessionClientReconnectsAfterBrokerRestart`, which don't exercise the `startPeerRuntime` wiring.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      registry_test.go:151 TestRegistryInterleavingsKeepInvariants drives 400 random event orders through Advance and asserts the stated invariants (no Admit on a stale pane, effects-driven broker equals registry, one connected binding per slot, an older session is never connected once a newer one is admitted) plus coverage floors.
findings:
  - id: new
    severity: Important
    family: effect-outcome-not-returned
    title: |
      EffectConnect drops a broker.Register failure; registry snapshot says connected, broker has no actor, nothing retries
    detail: |
      message_service.go:453 returns silently on Register error (also :459/:470 drop ReconcileObservation errors). Broker tombstones are never deleted, so after 128 distinct bindings per Couch lifetime Register fails "actor capacity reached" and the slot is silently unreachable with no retry (the old heartbeat retried). Rule (ARCH-ORDER): every IO effect's outcome returns to the reducer as an event; feed Register failure back as a failed admission (backoff, then dormant), log it, and add a service test with a failing Register.
  - id: new
    severity: Minor
    family: registry-liveness-unpinned
    title: |
      newerSessionForSlot lets a never-admitted newer session permanently displace a working older one
    detail: |
      registry.go:191 counts AwaitingPane/Rejected/Dormant newer sessions; re-admitting an older working session after a pane flicker marks it Displaced (final) while the newer session never connects. The invariant test checks only safety, not liveness. Restrict to Admitting/Admitted newer sessions, or document the choice.
  - id: new
    severity: Minor
    family: plan-revision-drift
    title: |
      M2 revision does not reconcile all Core-concepts rows and Task test bullets with the tree
    detail: |
      2nd instance of plan-revision-drift. Rule: at each milestone close the Revisions entry sweeps every Core-concepts row and every Task test bullet of that milestone against the tree. Instances: ReconnectBackoff lives in session_protocol.go, not backoff.go; Task 2.3's helper-subprocess kill test was not written; Task 2.6's wrapper-level real-broker idle/restart test was delivered at SessionClient level only, so the startPeerRuntime registry-socket wiring is untested.
  - id: new
    severity: Minor
    family: send-target-slot-match
    title: |
      Exact-send SendTargeted posts from an untracked goroutine and matches the target by raw slot string
    detail: |
      message_service.go:542; a send addressed by family alias would not wake a dormant session. Each send also spawns a goroutine that s.workers does not track; it ends at shutdown.
```
