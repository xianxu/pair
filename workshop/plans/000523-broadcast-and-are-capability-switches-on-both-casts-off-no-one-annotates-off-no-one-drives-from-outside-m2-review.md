# Boundary Review — pair#523 (milestone M2)

| field | value |
|-------|-------|
| issue | 523 — broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside) |
| repo | pair |
| issue file | workshop/issues/000523-broadcast-and-are-capability-switches-on-both-casts-off-no-one-annotates-off-no-one-drives-from-outside.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 891801b15a49644fd0e463d9cfd6440b06034265..46af2350d254fc620ea7e9806ee23ac5e01b2f82 |
| command | sdlc milestone-close --issue 523 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T09:58:03-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared controls, server admission checks, and documentation are substantially complete. One reproduced ordering bug blocks M2: an annotation created before OFF can acquire fresh authority when delivered after OFF/ON. Existing tests cover delays after recorder ingress, but miss delays before it.

1. **Strengths:** Shared click intent preserves right-copy semantics across all three controls. Operator writes recheck admission without taking the cleanup command lock, avoiding the tested deadlock. README and atlas cover the new controls, defaults, recovery, and Ctrl+Y.

2. **Critical findings:** At `cmd/internal/webcast/recorder/recorder.go:290`, ingress assigns the *current* permit to an incoming operator message. A disposable regression accepted stroke request 1, retained request 2 before OFF, completed OFF/ON, then delivered request 2. Result: `Outcome:accepted`, with one stroke restored after clearing. Messages carry no originating capability generation. Separately, `web/annotate.js:596` clears rendered strokes without cancelling a held gesture; delayed capability delivery lets that gesture emit another stroke. **ARCH-ORDER / ARCH-PURPOSE:** bind mutations to the generation observed when created, reject retired generations at ingress and dispatch, and cancel gestures on the clear barrier. Sweep both operator and viewer mutation paths.

3. **Important findings:** None additional.

4. **Minor findings:** `chrome_operator.go:107` introduces a 250 ms retry ticker, contradicting the plan’s explicit “No new periodic polling” envelope at line 177. Document its retry lifetime and termination policy, or use readiness events.

5. **Test coverage:** All 229 browser tests passed. Board, recorder, webcast, broadcast, and Couch Go suites passed; the latter two required a disposable copy with generated assets and a short temporary path. Capability, Board, and recorder race suites passed. The new delayed-ingress regression failed as described above. Headless operator conformance failed before exercising behavior because Chrome’s GPU process crashed. Full `make test` was not rerun. Disposable copies were removed; the reviewed checkout remains unchanged.

6. **Architecture:** **ARCH-DRY: pass**—shared click policy and authorities. **ARCH-PURE: pass**—pure policy with separate integration boundaries. **ARCH-PURPOSE: flag**—held gestures can survive OFF/ON. **ARCH-MOCK: pass structurally**—stateful seams support controlled ordering; live conformance remains unverified here. **ARCH-CONSTRAINTS: flag**—retry polling contradicts the documented envelope. **ARCH-SECURE: flag**—incoming messages receive fresh authority without originating-generation evidence. **ARCH-ORDER: flag**—the pre-ingress interleaving is unprotected. **ARCH-FUNERAL: pass**—subscriptions, workers, and context state have cast-owned cleanup.

7. **Plan revisions:** Add a `## Revisions` entry defining end-to-end generation provenance across browser creation, transport, ingress, and dispatch, with delayed-delivery tests for operator and viewer mutations. Reconcile the polling implementation with the operating envelope.

```findings
findings:
  - id: new
    severity: Critical
    family: capability-generation-provenance
    title: |
      Pre-OFF annotations can acquire fresh authority after OFF/ON
    detail: |
      cmd/internal/webcast/recorder/recorder.go:290 assigns current authority at ingress. A disposable regression retained a valid second stroke before OFF, completed OFF/ON, then delivered it: the recorder accepted it and restored one stroke. web/annotate.js:596 also leaves a held gesture active after annotation-clear when capability delivery is delayed. ARCH-ORDER, ARCH-SECURE, ARCH-PURPOSE: carry the originating capability generation through operator and viewer mutations, reject retired generations at ingress and dispatch, and cancel held gestures at the clear barrier. Add controlled pre-ingress delay regressions across these consumers.
  - id: new
    severity: Minor
    family: operating-envelope-traceability
    title: |
      Permission retry polling contradicts the documented operating envelope
    detail: |
      cmd/internal/webcast/chrome_operator.go:107 introduces a 250 ms retry ticker, while workshop/plans/000523-cast-capability-switches-plan.md:177 promises no new periodic polling. ARCH-CONSTRAINTS: document the retry lifetime, bounds, and termination policy in a Revisions entry, or replace polling with readiness-driven delivery.
```
