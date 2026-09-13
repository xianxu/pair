# Boundary Review — pair#184 (whole-issue close)

| field | value |
|-------|-------|
| issue | 184 — couch: switch a thread's agent |
| repo | pair |
| issue file | workshop/issues/000184-couch-switch-thread-agent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 065929e5b5f744802881ae25bd927fcc37c98fb4..445bbefe825302fef0f1d3767806f0aa162fb771 |
| command | sdlc close --issue 184 |
| reviewer | codex |
| timestamp | 2026-09-13T13:30:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range delivers structured argv transport, fresh-session launch authority, exact parked-artifact metadata, and orientation recovery. Focused tests and the orientation race suite pass. A submission scheduling bug and missing live-switch acceptance coverage block approval.

1. **Strengths**
   - Layout → wrapper → child tests verify exact argument preservation.
   - Fresh registration checks the new launch nonce before committing preferences.
   - Park metadata survives persistence and cleanup retries without guessing an archive.
   - README and atlas document the new action and recovery behavior.

2. **Critical findings**
   - [wrap.go:1498](/Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go:1498): When the settle timer fires with a solicited terminal reply queued, the loop consumes the tick and jumps to ordinary input. Forwarding that reply neither cancels orientation nor rearms the timer. Subsequent wakeups supply `ComposerObserved`, which cannot advance `DeliverySettling`; the pasted prompt remains unsubmitted until timeout. Preserve or reschedule the pending settle event after forwarding replies. **ARCH-ORDER**.

3. **Important findings**
   - [switchagent_acceptance_test.go:22](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/switchagent_acceptance_test.go:22): The full-chain acceptance starts with `seedVerifiedPark`, bypassing live-source teardown and exact TTY archive transfer. Successful core switch tests likewise start parked. Add a successful owned-live-source test carrying the actual park-produced descriptor through launch and wrapper delivery, while checking tag-owned artifacts remain intact. **ARCH-PURPOSE, ARCH-MOCK**.

4. **Minor findings**
   - None.

5. **Test coverage**
   - All nine focused package suites passed; `git diff --check` passed.
   - Race tests passed for orientation, wrapcmd, and readiness.
   - Existing reply tests send replies before paste. Add deterministic coverage where the settle deadline and solicited reply are ready together; require exactly one submission after forwarding.

6. **Architecture**
   - **ARCH-DRY — pass:** existing launch, park, inventory, and artifact seams reused.
   - **ARCH-PURE — pass:** parsing, prompt construction, and delivery transitions separated from IO.
   - **ARCH-PURPOSE — flag:** claimed complete-chain coverage omits successful live switching.
   - **ARCH-MOCK — flag:** stateful component tests exist, but their live-switch composition is missing.
   - **ARCH-CONSTRAINTS — pass:** bounded prompts, reads, and orientation watchers.
   - **ARCH-SECURE — pass:** structured argv and scoped artifact validation.
   - **ARCH-ORDER — flag:** a consumed timer event can strand delivery.
   - **ARCH-FUNERAL — pass:** launch-only orientation is cleared; persistent metadata uses existing lifecycles.

7. **Plan revisions**
   - Append a `## Revisions` entry recording the retained-settle-event rule and added successful live-source acceptance coverage. Existing revisions explain the delivered file/type substitutions.

```findings
findings:
  - id: new
    severity: Critical
    family: input-priority-preserves-pending-events
    title: |
      Forwarding a terminal reply can lose the orientation submit deadline
    detail: |
      cmd/internal/wrapcmd/wrap.go:1498 consumes the settle tick before jumping to queued input. A solicited reply does not cancel delivery, but forwarding it never rearms the timer, leaving DeliverySettling until timeout. Preserve the pending settle event and test simultaneous reply/deadline readiness deterministically (ARCH-ORDER).
  - id: new
    severity: Important
    family: acceptance-covers-lifecycle-composition
    title: |
      Full-chain acceptance omits successful live-source switching
    detail: |
      cmd/internal/couchcmd/switchagent_acceptance_test.go:22 seeds an already verified park, bypassing teardown and exact TTY capture transfer. Add a successful owned-live-source case carrying the actual park descriptor through launch and wrapper delivery, with preserved tag-owned artifacts (ARCH-PURPOSE, ARCH-MOCK).
```

---

## Re-review — 2026-09-13T14:13:32-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 184 — couch: switch a thread's agent |
| repo | pair |
| issue file | workshop/issues/000184-couch-switch-thread-agent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 065929e5b5f744802881ae25bd927fcc37c98fb4..c70d5ca89ae51f7c0edb598b77a6c89abbbc467a |
| command | sdlc close --issue 184 |
| reviewer | codex |
| timestamp | 2026-09-13T14:13:32-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed, with passing regressions that fail when their fixes are removed through scratch overlays. The switch flow, exact argument transport, and UI recovery are well covered. One new Important issue blocks approval: retry-history lookup performs unbounded work from a persisted counter and ignores cancellation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestOrientationDueSubmitSurvivesPrioritizedInput deterministically queues input after timer receipt. It passes at HEAD and fails when pending-event preservation is removed through a scratch overlay.
  - id: BR-2
    disposition: addressed
    note: |
      The live-source acceptance exercises cleanup, actual archive preservation, verified park, launch transport and wrapper delivery. Removing completion.Scrollback transfer makes it fail on both park metadata and prompt contents.
findings:
  - id: new
    severity: Important
    family: persisted-counters-bound-work
    title: |
      Retry-history lookup is unbounded and ignores cancellation
    detail: |
      cmd/internal/pairlifecycle/store.go:174 scans every integer below the persisted attempt while holding the cleanup lock; missing files continue the scan without checking context. Any positive attempt is accepted. A bounded scratch test with attempt 1000000000 confirmed ten reads after cancellation before an injected error stopped it. Bound history discovery independently of the numeric counter, honor cancellation throughout, and add sparse-history/cancellation regressions (ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER).
```

1. **Strengths**
   - Exact archive identity survives cleanup, persistence, and delivery; the live-source acceptance detects broken transfer.
   - Both layout stanzas preserve exact argv through the actual wrapper.
   - Source revision checks, fresh-registration evidence, and stale UI-result rejection protect replacement operations.
   - README, atlas, and plan revisions describe the delivered surface.

2. **Critical findings:** None.

3. **Important findings:** The retry scan at [store.go:174](/Users/xianxu/workspace/pair/cmd/internal/pairlifecycle/store.go:174). A sparse or corrupted counter can monopolize cleanup despite cancellation. The fix must bound work as well as check cancellation.

4. **Minor findings:** None.

5. **Test coverage:** Relevant orientation, readiness, wrapper, Couch command/core/UI, launcher, lifecycle, and thread-record suites passed. Orientation race tests and pinned-range whitespace checks passed. Both prior-finding mutation checks failed as expected. The new cancellation probe failed against HEAD. Live harness smoke was not rerun.

6. **Architecture**
   - **ARCH-DRY — pass:** Existing launch, lifecycle, and artifact seams are reused.
   - **ARCH-PURE — pass:** Parameter parsing, prompt construction, delivery transitions, and form reducers remain independently testable.
   - **ARCH-PURPOSE — pass:** Acceptance now covers live-source lifecycle composition.
   - **ARCH-MOCK — pass:** Stateful process/storage doubles and actual transport boundaries provide meaningful coverage.
   - **ARCH-CONSTRAINTS — flag:** Retry lookup lacks a work bound.
   - **ARCH-SECURE — flag:** A persisted counter directly controls filesystem work.
   - **ARCH-ORDER — flag:** Cancellation does not terminate that lookup.
   - **ARCH-FUNERAL — pass:** Orientation reuses existing artifact lifecycles and bounded watcher ownership.

7. **Plan revision recommendation:** Append a `## Revisions` entry defining the retry-history lookup bound, cancellation behavior, and sparse-history regression evidence.
