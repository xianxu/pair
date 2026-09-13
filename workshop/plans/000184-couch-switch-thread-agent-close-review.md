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
