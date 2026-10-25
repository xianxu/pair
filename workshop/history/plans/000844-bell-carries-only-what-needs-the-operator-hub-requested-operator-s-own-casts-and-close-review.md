# Boundary Review — pair#844 (whole-issue close)

| field | value |
|-------|-------|
| issue | 844 — Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub |
| repo | pair |
| issue file | workshop/issues/000844-bell-carries-only-what-needs-the-operator-hub-requested-operator-s-own-casts-and.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ebb6f2fb37088078c205242b11a1631422eb98c..93261df696424b545731f1161d5a25f124850155 |
| command | sdlc close --issue 844 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T11:52:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation establishes a clear routing policy, bounded delivery state, and broad producer coverage. Focused tests, race tests, and the reader inventory pass. Shipping is blocked by an ownership-transition bug: queued diagnostics can disappear when their owning hub disconnects, instead of reaching the bell. README descriptions also retain the previous behavior.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: notice-owner-loss-fallback
    title: |
      Losing the owning hub can silently retire undelivered diagnostics
    detail: |
      cmd/internal/couchcmd/notice_delivery.go:239 treats every epoch difference as condition clearance. With four admissions occupied and a fifth notice pending under an enabled owner with a nonzero epoch, disconnecting that owner makes resolveNoticeOwner return Unavailable with epoch zero (notice_delivery_backend.go:106); reconcile deletes the fifth notice before RouteNotice can emit its required bell fallback. Notices already marked Sending have the same loss when DeliverNotice returns Refused for an unavailable owner and the reducer removes them at couchcore/notice_delivery.go:179. ARCH-ORDER / ARCH-PURPOSE: distinguish proven owner absence from automation invalidation, and route confirmed unadmitted occurrences through the unavailable-owner policy without resending uncertain admissions. Add deterministic tests for both orderings.
  - id: new
    severity: Important
    kind: defect
    family: user-docs-behavior-parity
    title: |
      README still promises the previous bell contents
    detail: |
      README.md:1265 and README.md:1533 say actionable asynchronous warnings, blocked prompts, and judged questions appear in the bell without the new owning-hub qualification. README.md is unchanged in this range, although the implementation and atlas now route these events to connected owners and suppress fallback for automation-OFF owners. Update these user-facing descriptions to explain operator-owned exceptions, owning-hub routing, and unavailable-owner fallback.
  - id: new
    severity: Minor
    kind: defect
    family: shared-analysis-single-source
    title: |
      Producer inventory analysis is duplicated across packages
    detail: |
      cmd/internal/couchcmd/notice_class_inventory_test.go:44 and cmd/internal/couchtty/notice_class_inventory_test.go:44 contain near-identical AST walkers, receiver recognition, and rejection fixtures. ARCH-DRY: extract the common analysis into a shared test helper, passing package-specific audited projections and forwarders as configuration.
```

1. **Strengths**

   - `couchcore/notice_route.go` centralizes eligibility and conservatively handles unknown classes.
   - Delivery uses fixed workers, bounded bodies, canonical receipt IDs, and explicit shutdown ownership.
   - Startup integration tests exercise real registry/broker paths; condition tests verify unchanged picker/question sets across owner changes.
   - Atlas updates explain ownership, automation, capacity, and ephemeral lifecycle behavior.

2. **Critical findings**

   The owner-loss finding above blocks shipping. Preserve the distinction between automation being disabled and the recipient becoming unavailable. A confirmed pre-admission refusal can safely generate fallback attention; uncertain admission must retain its no-resend rule.

3. **Important findings**

   Update the two README passages above to match the implemented policy and atlas.

4. **Minor findings**

   Consolidate the duplicated inventory analyzer when convenient.

5. **Test coverage notes**

   Passed:

   - Focused notice, condition, picker, wait, archive, and automatic-input-stall tests across all three affected packages.
   - Race-enabled routing, delivery, owner, inventory, Console, and condition tests.
   - `make test-affected-inventory`.
   - Pinned-range `git diff --check`.

   The owner-loss defect follows directly from the production control flow. Existing tests separately cover unavailable owners and epoch invalidation, but omit their interaction with queued and admission-starting notices. No files were changed.

6. **Architectural notes**

   - **ARCH-DRY — flag:** duplicated inventory analysis.
   - **ARCH-PURE — pass:** routing and reducer are directly tested without IO; integration stays behind a backend seam.
   - **ARCH-PURPOSE — flag:** producer coverage is broad, but owner loss breaks the promised fallback.
   - **ARCH-MOCK — pass:** stateful endpoints and existing registry/broker fixtures share the production boundary.
   - **ARCH-CONSTRAINTS — pass:** worker, entry, body, and expiry bounds have focused coverage.
   - **ARCH-SECURE — pass:** authenticated notify admission and exact binding/automation checks remain; message bodies are sanitized.
   - **ARCH-ORDER — flag:** owner absence and automation invalidation collapse into the same retirement outcome.
   - **ARCH-FUNERAL — pass:** pending state expires or dies with Console; workers cancel and join.
   - **ARCH-EVOLVE — pass:** new routing fields are ephemeral; no persisted-format change was found.

   Core-concept entities exist at their documented paths, and their PURE/INTEGRATION classifications match their roles.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing unavailable-owner fallback from automation-generation invalidation. Extend the executable test contract with deterministic owner-disconnect sequences before admission starts and immediately before guarded admission. Preserve the approved no-replay and uncertain-admission rules.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4b2c1112-b94e-4723-84b7-9a8d3e6e5170","repository":"github.com/xianxu/pair","issue_id":"000844","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/159f9e002bf86c09","reviewed_head":"93261df696424b545731f1161d5a25f124850155","slot":"pair:1","requested_at":"2026-10-24T18:50:14.665211Z"},"phase":"finished","pid":48342,"started_at":"2026-10-24T18:50:14.824197Z","finished_at":"2026-10-24T18:52:51.293993Z","session_id":"01a16f3f-cf73-7621-9284-a57dc355bada","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 31.924441667s
head: "93261df696424b545731f1161d5a25f124850155"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413

---

## Re-review — 2026-10-24T12:06:02-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 844 — Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub |
| repo | pair |
| issue file | workshop/issues/000844-bell-carries-only-what-needs-the-operator-hub-requested-operator-s-own-casts-and.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ebb6f2fb37088078c205242b11a1631422eb98c..0d4186bad3b010c4985057fd932cc61f2e056f56 |
| command | sdlc close --issue 844 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T12:06:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The original owner-disconnect regressions are repaired and have meaningful regression tests. README corrections and shared inventory analysis are also delivered. BR-1 remains blocking because its epoch-clearance rule still loses pending diagnostics when ownership moves between hubs with different automation counters.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Original disconnect orderings are repaired; restoring pre-fix production code through a temporary Go overlay makes their regressions fail. The same clearance rule remains incomplete at cmd/internal/couchcmd/notice_delivery.go:247: epochs are scoped per hub root, but reconciliation compares counters across different owners. A deterministic real-service fixture holding four admissions, queuing a fifth notice under /repo1 at epoch 2, then moving its source to /repo0 at epoch 0 deletes the notice. Disconnecting /repo0 after the move also produces zero fallback projections. Preserve scoped generation provenance and distinguish owner handoff from automation revocation across pending and confirmed-refused paths.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1264-1268 and :1536-1543 now describe owning-hub routing, operator-owned exceptions, unavailable-owner fallback and connected-OFF suppression/no replay. These passages match couchcore.RouteNotice and the condition projections in attention.go and message_service.go.
  - id: BR-3
    disposition: addressed
    note: |
      Both package inventories now delegate AST walking, receiver recognition and rejection fixtures to cmd/internal/testnotice/inventory.go, passing package-specific Projection and Forward configuration. The shared helper tests pass.
```

1. **Strengths**

   - Routing policy is centralized in `couchcore.RouteNotice`; unknown classes follow the conservative owning-hub policy.
   - Delivery uses bounded state and four lifetime-owned workers, with explicit shutdown and uncertain-receipt handling.
   - BR-1 tests exercise guarded admissions through the real service seam. Restoring pre-fix production code makes the reported regressions fail.
   - README and atlas changes explain the new routing surface consistently.

2. **Critical findings**

   **BR-1 remains open — scoped automation generations are compared across owners.** At `cmd/internal/couchcmd/notice_delivery.go:247`, unequal epochs clear a pending notice, although `automation.go:91` stores epochs separately by hub root.

   Reproduction: hold four admissions; queue a fifth diagnostic under `/repo1` at epoch 2; move its source to enabled `/repo0` at epoch 0. Reconciliation leaves four entries, silently deleting the fifth. If `/repo0` disconnects after the move, reconciliation also produces **zero fallback projections**. This contradicts the plan’s pending-owner-move transition at line 114. **ARCH-ORDER / ARCH-PURPOSE.**

   **This is the 2nd finding in family `notice-owner-loss-fallback`.** Retained under BR-1: fix the rule across owner transitions. Validate generation provenance within its owning root, distinguish handoff from revocation, and sweep pending and confirmed-refused transitions while preserving uncertain-send non-replay.

3. **Important findings**

   None additional. BR-2 is addressed.

4. **Minor findings**

   None additional. BR-3 is addressed.

5. **Test coverage notes**

   - Focused routing, delivery, condition, picker, wait, archive and stall tests passed.
   - Focused race tests passed across `couchcore`, `couchtty` and `couchcmd`.
   - Shared analyzer tests, affected-reader inventory and diff whitespace checks passed.
   - Temporary overlay tests reproduced both remaining handoff failures. Overlay files were removed; the checkout remains clean.
   - A complete repository suite was not run during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared analyzer replaces duplicated analysis.
   - **ARCH-PURE — pass:** policy and delivery transitions have direct, IO-free tests; listed entities exist at their documented paths.
   - **ARCH-PURPOSE — flag:** BR-1 still drops an actionable routing obligation.
   - **ARCH-MOCK — pass:** production broker seams are exercised through stateful fixtures.
   - **ARCH-CONSTRAINTS — pass:** queue/body/admission bounds and cancellation are tested.
   - **ARCH-SECURE — pass:** exact binding and automation checks remain at ingress.
   - **ARCH-ORDER — flag:** a different owner’s counter is treated as evidence of revocation.
   - **ARCH-FUNERAL — pass:** ephemeral tracking expires or ends with Console; workers join.
   - **ARCH-EVOLVE — pass:** no persisted-format change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining owner-scoped generation validation across handoffs. Enumerate pending and confirmed-refused cases across same/moved owner, enabled/disabled/unavailable destination, and valid/revoked original generation. Add deterministic regressions for connected handoff and handoff followed by destination loss.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3146359b-b465-48ce-a838-aba546717993","repository":"github.com/xianxu/pair","issue_id":"000844","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/091905e0529f6c0e","reviewed_head":"0d4186bad3b010c4985057fd932cc61f2e056f56","slot":"pair:1","requested_at":"2026-10-24T19:02:06.902029Z"},"phase":"finished","pid":72536,"started_at":"2026-10-24T19:02:07.00413Z","finished_at":"2026-10-24T19:05:57.039641Z","session_id":"01a16f4a-acce-77b2-8017-7f7d68907ce3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.0673895s
head: "0d4186bad3b010c4985057fd932cc61f2e056f56"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-24T12:26:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 844 — Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub |
| repo | pair |
| issue file | workshop/issues/000844-bell-carries-only-what-needs-the-operator-hub-requested-operator-s-own-casts-and.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ebb6f2fb37088078c205242b11a1631422eb98c..d7a07ecba6348d1d1a9d9de3c3704c45926c19c6 |
| command | sdlc close --issue 844 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T12:26:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised Spec and Plan. BR-1 is addressed across pending owner loss, confirmed refusal, cross-hub handoff, and unknown readiness. Regression tests exercise production transitions, and mutation checks turn them red when key corrections are removed. No additional findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Owner generation is validated within its retained root. Pending owner loss and confirmed refusal receive unavailable-owner fallback; unknown readiness preserves the obligation, while admitted/uncertain sends never resend. Production-path regressions pass, and mutations removing scoped revocation or witness retention fail.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1264 and :1536 replace unconditional warning/prompt eligibility with owning-hub routing, operator exceptions, unavailable-owner fallback, and connected-OFF suppression, consistent with couchcore/notice_route.go:73.
  - id: BR-3
    disposition: addressed
    note: |
      Both package inventories use cmd/internal/testnotice/inventory.go:21 for analysis and its shared rejection fixtures, retaining package-specific projection and forwarding declarations.
```

1. **Strengths**

   - `couchcore/notice_route.go` centralizes eligibility without interpreting warning text.
   - `couchcore/notice_delivery.go:199` distinguishes confirmed refusal from uncertain admission; `AwaitingDisposition` cannot admit again.
   - `couchcmd/notice_delivery.go:237` fences stale observations by occurrence, owner, token, and phase.
   - Stateful integration fixtures exercise real registry/broker admission for both Claude and Codex.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused race suite passed across `couchcore`, `couchtty`, `couchcmd`, and `testnotice`.
   - Broader routing/producer/startup race selection passed on rerun. Its first run failed at `TestFreshStartupRunSerializesLaunchThroughAdoption`; that test passed independently. A disposable base checkout separately failed the existing startup-cancellation assertion. These observations do not establish the same failure on base.
   - Three temporary source mutations failed the expected regressions.
   - Reader inventory and pinned-range whitespace checks passed. The reviewed checkout remains unchanged.
   - Full recorded verification and coordinated live acceptance remain the Plan’s pre-landing postconditions.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass — shared policy and inventory analysis. |
   | ARCH-PURE | Pass — routing/reducer tests run without IO; backend effects remain behind the shell. |
   | ARCH-PURPOSE | Pass — producer and bypass paths follow the documented routing contract. |
   | ARCH-MOCK | Pass — stateful fixtures share production admission seams; existing external dependency seams remain intact. |
   | ARCH-CONSTRAINTS | Pass — bounded entries, bodies, workers, admission contexts, and expiry are exercised. |
   | ARCH-SECURE | Pass — verified ownership, guarded ingress, and body sanitization are preserved. |
   | ARCH-ORDER | Pass — explicit phases, scoped generation witnesses, stale-event fencing, and controlled completion ordering. |
   | ARCH-FUNERAL | Pass — local tracking expires or ends with Console; workers cancel and join. |
   | ARCH-EVOLVE | Pass — new fields are ephemeral; no persisted-format change. |

7. **Plan revision recommendations:** None. Existing revisions describe the delivered owner-scoped generation and refusal-disposition behavior.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e3613c05-525b-451c-b562-ab804901230c","repository":"github.com/xianxu/pair","issue_id":"000844","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/474ca958dbc663fe","reviewed_head":"d7a07ecba6348d1d1a9d9de3c3704c45926c19c6","slot":"pair:1","requested_at":"2026-10-24T19:20:34.7575Z"},"phase":"finished","pid":87997,"started_at":"2026-10-24T19:20:34.930069Z","finished_at":"2026-10-24T19:26:30.075655Z","session_id":"01a16f5b-968f-7643-ad9e-c04f0a48b495","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.280794459s
head: "d7a07ecba6348d1d1a9d9de3c3704c45926c19c6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T13:46:02-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 844 — Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub |
| repo | pair |
| issue file | workshop/issues/000844-bell-carries-only-what-needs-the-operator-hub-requested-operator-s-own-casts-and.md |
| boundary | whole-issue close |
| milestone | — |
| window | ef4e44efd8fe3532c8aa0f6780fc9a0713c8ca2f..f7f4ec084657b6f4bfcd669254f40f694a2b835a |
| command | sdlc close --issue 844 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T13:46:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The checker changes pass the artifactpath package tests and affected-reader inventory. However, this interdiff introduces executable checker behavior beyond the previously disposed findings and adds a corresponding plan revision. The explicit fix-review scope rule therefore requires a full `sdlc close`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Previously accepted routing corrections and regression tests are unchanged in this window; no regression introduced by the interdiff was found.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1264-1268 and :1536-1543 retain the corrected owning-hub, operator-exception and automation-OFF descriptions, consistent with couchcore/notice_route.go.
  - id: BR-3
    disposition: addressed
    note: |
      Both notice_class_inventory_test.go consumers still delegate analysis and rejection fixtures to cmd/internal/testnotice/inventory.go with package-specific configuration.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: close-review-scope-integrity
    title: |
      Scope changed after close — run a full `sdlc close`
    detail: |
      workshop/plans/000844-human-attention-routing-plan.md:258-262 adds a checker-design revision beyond recording any prior finding's fix. cmd/internal/artifactpath/manifest.go:32 introduces GoTypedConstantVocabulary, and coverage_test.go:755-762 adds executable acceptance behavior for it. None of BR-1, BR-2 or BR-3 requested this behavior. Although the focused checks pass and the plan records approval, the supplied narrow-review rule requires a full sdlc close for this expanded scope.
```

1. **Strengths**
   - Typed-constant allowances pin the type, symbol, literal and occurrence count.
   - Adversarial tests cover implicit, computed, local, mismatched and duplicate declarations.
   - Atlas documentation describes the new checker contract.

2. **Critical findings**
   - The scope finding above blocks this narrow gate. Run a full `sdlc close` against the final head.

3. **Important findings**
   - None.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Passed `go test ./cmd/internal/artifactpath -count=1`.
   - Passed `make test-affected-inventory`.
   - Passed pinned-range `git diff --check`.
   - No files changed; working tree remains clean.

6. **Architectural notes**
   - **ARCH-DRY — pass:** new recognition reuses the existing allowance and coverage machinery.
   - **ARCH-PURE — pass:** AST recognition operates on supplied syntax; repository scanning remains in the existing integration shell.
   - **ARCH-PURPOSE — pass within the inspected implementation:** all six added source classifications are covered by the production inventory test.
   - **ARCH-EVOLVE — pass:** no persisted schema or field meaning changes appear in this window.

7. **Plan revision recommendations**
   - Correct line 262’s instruction to perform a “narrow re-close”: this added executable contract requires full-close review under the supplied boundary rules.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1c5e0bac-b06f-4c1d-8f31-7da57791f76b","repository":"github.com/xianxu/pair","issue_id":"000844","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/98fa17fd279a15ba","reviewed_head":"f7f4ec084657b6f4bfcd669254f40f694a2b835a","slot":"pair:1","requested_at":"2026-10-24T20:45:07.53001Z"},"phase":"finished","pid":74943,"started_at":"2026-10-24T20:45:07.64089Z","finished_at":"2026-10-24T20:46:01.595562Z","session_id":"01a16fa8-fc42-7a00-a5f1-01b9b0ad3cf6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.306326584s
head: "f7f4ec084657b6f4bfcd669254f40f694a2b835a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T13:48:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 844 — Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub |
| repo | pair |
| issue file | workshop/issues/000844-bell-carries-only-what-needs-the-operator-hub-requested-operator-s-own-casts-and.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5ebb6f2fb37088078c205242b11a1631422eb98c..6724410b13b26b54f352dff9d6f698ff57cb3835 |
| command | sdlc close --issue 844 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T13:48:55-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the documented routing policy and bounded delivery model. This full-range review satisfies BR-4’s scope requirement, including inspection and testing of the typed-constant checker change. No blocking defects found. Recorded-suite completion and coordinated live acceptance remain separate pre-landing postconditions.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      This review inspected the complete pinned range against Spec and Plan, including the executable checker extension. Plan lines 264–266 supersede narrow re-close; artifactpath tests, routing race tests and reader inventory pass.
  - id: BR-1
    disposition: addressed
    note: |
      Retained-root generation checks and AwaitingDisposition preserve unavailable-owner fallback without re-admission. Owner-loss, unknown-readiness and controlled handoff regressions pass under race detection.
  - id: BR-2
    disposition: addressed
    note: |
      README passages at lines 1264 and 1536 now describe owning-hub routing, operator exceptions, unavailable-owner fallback and automation-OFF behavior consistently with RouteNotice.
  - id: BR-3
    disposition: addressed
    note: |
      Both package inventories consume testnotice.Findings and CheckRejections, retaining only package-specific projection and forwarder configuration. Shared analyzer and inventory tests pass.
```

1. **Strengths**

   - `couchcore/notice_route.go` centralizes eligibility independently of display text and feed kind.
   - `couchcore/notice_delivery.go:199` distinguishes confirmed refusal from admitted or uncertain outcomes; refused entries cannot admit again.
   - Controlled handoff fixtures exercise real guarded ingress with independently scoped owner epochs.
   - README and atlas document routing, capacity loss, expiry and shutdown behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Complete `artifactpath` and `testnotice` package tests.
   - Race-enabled routing, delivery, startup, handoff, condition, picker, wait, archive and automatic-stall selections across core, tty and command packages.
   - Additional race-enabled producer tests for lifecycle diagnostics, hub conflicts, renicer, scratch warnings and supervision.
   - `make test-affected-inventory` and pinned-range `git diff --check`.

   Pure routing/reducer tests use explicit inputs without IO. Integration tests use injected stateful backends and controlled completions. I did not rerun the complete repository suite or live acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared policy and inventory analyzer; package-specific exceptions remain configuration.
   - **ARCH-PURE — pass:** Deterministic routing/transitions remain separate from delivery IO.
   - **ARCH-PURPOSE — pass:** Producer and bypass sweep covers conditions, archive, stalls, startup and direct warning paths.
   - **ARCH-MOCK — pass:** Stateful doubles share the production backend seam; no new external dependency introduced.
   - **ARCH-CONSTRAINTS — pass:** Entry/body limits, fixed workers, deadlines and observation bounds have executable coverage.
   - **ARCH-SECURE — pass:** Exact recipient and automation generation are revalidated; broker-bound text remains sanitized.
   - **ARCH-ORDER — pass:** Explicit phases preserve uncertainty; controlled tests exercise refusal, handoff and late completion.
   - **ARCH-FUNERAL — pass:** Tracking expires or ends with Console; shutdown cancels and joins workers.
   - **ARCH-EVOLVE — pass:** Routing facts are ephemeral; no persisted-format change found.

7. **Plan revision recommendations:** None. The existing revisions cover the final implementation and expanded review scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"afa2bf74-06bd-49ad-a688-58ed564bd15b","repository":"github.com/xianxu/pair","issue_id":"000844","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8f2d8b34f788648e","reviewed_head":"6724410b13b26b54f352dff9d6f698ff57cb3835","slot":"pair:1","requested_at":"2026-10-24T20:47:14.991932Z"},"phase":"finished","pid":99051,"started_at":"2026-10-24T20:47:15.119119Z","finished_at":"2026-10-24T20:48:53.573724Z","session_id":"01a16faa-ee40-7cf3-b558-5c5b9a826661","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.248733084s
head: "6724410b13b26b54f352dff9d6f698ff57cb3835"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
