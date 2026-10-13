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

---

## Re-review — 2026-10-13T10:48:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 523 — broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside) |
| repo | pair |
| issue file | workshop/issues/000523-broadcast-and-are-capability-switches-on-both-casts-off-no-one-annotates-off-no-one-drives-from-outside.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 3c37f26ee6222a17e75f60ed31a59e8a1de57cc4..29ac1fe53accedd49a9f30b7798372dc390cdbb3 |
| command | sdlc milestone-close --issue 523 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T10:48:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned M2 implementation matches the reviewed capability contracts. BR-5 and BR-6 are addressed, with mutation testing confirming BR-5’s regression coverage. No new substantive findings. Confidence is limited by live Chrome failing during GPU startup before exercising the feature.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Origin generations reach operator, viewer annotation, and drive admission; ordered dispatch retains its checks. Independently removing each provenance guard made its regression fail. Removing annotation-clear gesture cancellation also failed capability-ui.test.mjs:70.
  - id: BR-6
    disposition: addressed
    note: |
      The plan revision at workshop/plans/000523-cast-capability-switches-plan.md:551 explicitly documents the 250ms unresolved-delivery retry exception, deadlines, acknowledgment behavior, context pruning, and shutdown. These match cmd/internal/webcast/chrome_operator.go:107-195.
```

1. **Strengths**

   - Origin-bound admission covers all four operator mutation kinds, all three viewer annotation routes, and drive request/release/input.
   - Controlled regressions exercise delayed ingress, queued writes, held gestures, stale completions, and replacement contexts.
   - Shared click intent preserves terminal pointer safety while adding copy-only link creation.
   - README and atlas document defaults, recovery actions, stable links, generation headers, and the new notice-copy shortcut.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None substantive.

5. **Test coverage notes**

   - Passed: 238 browser tests; all webcast Go packages; Couch and broadcast packages; race tests for webcast, Board, drive, and recorder.
   - Four independent mutation checks failed as expected, establishing that BR-5 tests detect removal of the fixes.
   - Couch required runtime-bundle generation in the disposable copy. Broadcast passed with a short temporary path.
   - Live `TestOperatorLiveAnnotation` failed twice before page creation: Chrome reported “GPU process isn't usable.” Live conformance and full `make test` are therefore not independently certified by this review.
   - Disposable copy removed; tracked checkout unchanged.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared click policy and existing capability authorities remain authoritative.
   - **ARCH-PURE — Pass:** transition and rendering logic remain directly testable; integration seams own IO.
   - **ARCH-PURPOSE — Pass:** the provenance repair covers the enumerable operator/viewer/drive mutation paths.
   - **ARCH-MOCK — Pass:** stateful fixtures exercise production boundaries; live conformance execution has the limitation above.
   - **ARCH-CONSTRAINTS — Pass:** bounded workers and delivery deadlines match the revised retry envelope.
   - **ARCH-SECURE — Pass:** missing, malformed, and retired generations fail closed without widening link roles.
   - **ARCH-ORDER — Pass:** generation/attempt checks and controlled event-order tests prevent stale authority restoration.
   - **ARCH-FUNERAL — Pass:** permission state, subscriptions, and timers have context/cast termination paths; no new durable artifact family.

7. **Plan revision recommendations:** None. The existing revisions cover provenance, retry behavior, and the approved UI refinements.
