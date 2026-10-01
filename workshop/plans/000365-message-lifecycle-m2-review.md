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

---

## Re-review — 2026-10-01T15:32:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | b36fac3d2229037e71362a9c837c1915dea2d0a6..74ad84c127aca3b25b801b9e077c8b359f70224e |
| command | sdlc milestone-close --issue 365 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-01T15:32:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

**Verdict: SHIP.** The one Important finding still open, BR-7, is fixed and covered by tests that fail without the fix. The other open findings (BR-8, BR-9, BR-10) are Minor and this round's commit didn't touch them, so they stay open. They don't block the gate. I found one new Minor issue: a capacity limit on retained disconnected actors that this boundary leaves in place. `go test ./cmd/internal/couchmessage/ ./cmd/internal/couchcmd/ -count=1` passes. I had to run it outside the sandbox, because inside it `mkdir /tmp/...` fails with "operation not permitted".

**How BR-7 was fixed (`74ad84c1`)**
- **The loop:** `execute` now returns a follow-up event when an effect fails (`cmd/internal/couchcmd/message_service.go:445-487`). `step` processes those events before reading any new input (`message_service.go:421-425`), so the registry can't get ahead of what the broker holds.
- **The registry:** a new `ConnectFailed` event is handled in `cmd/internal/couchmessage/registry.go:200-205`. It ignores a stale failure whose phase or pane doesn't match. Otherwise it goes through the shared `fail` helper (`registry.go:246-256`), which the old `AdmissionDone` error branch now also uses (ARCH-DRY).
- **Tests:**
  - `registry_test.go:137` checks that a failure for the wrong pane is ignored and a matching one schedules a retry.
  - The randomized interleaving test now injects broker refusals (`registry_test.go:249`).
  - `TestMessageBrokerRefusalIsNotConnectedAndRetries` fills the broker's 128 actor slots for real. Without the fix, Register's error would be dropped, the registry would show the binding as connected, and no retry would be scheduled, so the test would fail at "scheduled no retry". The fix is reachable and the test exercises it.

**Strengths**
- The rule BR-7 stated was fixed as a class, not just at one call site. Every effect that can fail now reports back, and the comment at `message_service.go:445-449` lists each effect kind and why it can or can't fail (ARCH-PURPOSE).
- The interleaving test now covers a failed external effect, not only successful orderings (ARCH-ORDER).
- `fail` consolidates the backoff ladder in one place.

**Critical:** none.

**Important:** none.

**Minor**
- **BR-7 leftover:** a refused Connect still isn't logged anywhere, so an operator never sees why a slot went dormant.
- **BR-8 (still open):** `registry.go:266` still counts never-admitted newer sessions when deciding to displace an older one.
- **BR-9 (still open):**
  - The plan revision still doesn't say `ReconnectBackoff` lives in `session_protocol.go`; there is no `backoff.go`.
  - Task 2.3's helper-subprocess kill test still isn't written.
  - Task 2.6's real-broker test still runs only at the session-client level.
- **BR-10 (still open):** `message_service.go:542` still sends from an untracked goroutine and matches the target by the raw slot string.
- **New (ARCH-FUNERAL):** the broker keeps every disconnected actor as a tombstone and never removes it. Every wrapper launch produces a new binding, so once 128 have accumulated in one Couch run, every new session for a new binding fails Register, works through the retry ladder, and goes dormant with nothing logged. This predates M2 (the capacity check is at base `broker.go:136`), but M2 now ends in permanent dormancy where the old heartbeat kept retrying.

**Test coverage notes**
- BR-7 is pinned at both the pure-registry level and the service level, against a real broker.
- Still missing, from BR-9: the real-broker test through the wrapper's registry-socket wiring.

**Architecture**
| Principle | Result |
|---|---|
| ARCH-DRY | pass |
| ARCH-PURE | pass. The registry is a pure reducer; the service runs its effects. |
| ARCH-PURPOSE | pass for BR-7 |
| ARCH-MOCK | pass. The service test uses the real in-process broker. |
| ARCH-CONSTRAINTS | pass |
| ARCH-SECURE | pass. No new untrusted input. |
| ARCH-ORDER | pass for BR-7. BR-8 is a remaining liveness gap. |
| ARCH-FUNERAL | flag: the tombstone growth above |

**Plan revisions needed:** BR-9's entries are still owed: correct the `ReconnectBackoff` location, and record the Task 2.3 kill test and Task 2.6 wrapper-level test as either not delivered or deferred.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      ConnectFailed is fed back via execute's return and stepped before new input; registry and service tests go red without it. Still no log line for a refused Connect.
  - id: BR-8
    disposition: not-addressed
    note: |
      registry.go:266 newerSessionForSlot still counts any non-Displaced newer session.
  - id: BR-9
    disposition: not-addressed
    note: |
      No new plan Revisions entry this round; backoff.go still named, 2.3/2.6 gaps unrecorded.
  - id: BR-10
    disposition: not-addressed
    note: |
      message_service.go:542 unchanged.
findings:
  - id: new
    severity: Minor
    family: artifact-removal-path
    title: |
      Broker actor tombstones are never evicted, so after 128 bindings per Couch lifetime new sessions go dormant
    detail: |
      2nd finding in family artifact-removal-path. Rule: every bounded table a component writes must name its removal path at the writer, not just a cap. Here, a tombstone whose slot and repository have re-registered under a newer binding is superseded (the registry marks it Displaced, which is final), so evict it on that Register. Pre-existing at base broker.go:136, but M2 turns the failure into silent permanent dormancy where the old heartbeat kept retrying.
```
