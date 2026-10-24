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
