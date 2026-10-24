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
