# Boundary Review — pair#789 (whole-issue close)

| field | value |
|-------|-------|
| issue | 789 — Couch chrome: cap slot status text at 40 cells (ellipsis) and shorten built-in status messages to fit |
| repo | pair |
| issue file | workshop/issues/000789-couch-chrome-cap-slot-status-text-at-40-cells-ellipsis-and-shorten-built-in-status-messages-to-fit.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5df9efb7d0df1a627f8d929895181fcf7b40ef73..b532c3b064bdcb1b10f02a9232591650b88970d7 |
| command | sdlc close --issue 789 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T07:19:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared display cap works in the targeted tests and preserves full diagnostics. Shipping is blocked by incomplete producer enforcement, a failing lifecycle test, and documentation and acceptance criteria that no longer match the revised behavior.

```findings
findings:
  - id: new
    severity: Critical
    family: built-in-status-enforcement
    title: |
      Catalog passes while built-in status messages still exceed 40 cells
    detail: |
      cmd/internal/couchtty/status_catalog_test.go:98-133 drops selector values and formatted substitutions, including values fixed by the program, and misses callback returns. ARCH-PURPOSE: the producer-cleanup contract remains incomplete. The affected family comprises capabilityNotice retry messages in cast_control.go:41 (41/45 cells before diagnostic suffixes); console_webcast.go:329 copy messages for view-only/pointer/drive (45/43/41), :335 not-ready messages for view-only/pointer (44/42), and :299's callback message (72); and console_castcodes.go:397/:485/:487 rotation messages containing built-in cast/link names. Sweep these producer sites and their finite variants, shorten the fixed wording, and add regression coverage that measures built-in substitutions and callback-produced notices rather than erasing them.
  - id: new
    severity: Important
    family: producer-assertion-consistency
    title: |
      Lifecycle wording change leaves its regression test failing
    detail: |
      cmd/internal/couchcmd/lifecycle_notices_test.go:684 still requires ": 1 observations", although lifecycle_notices.go:179 now emits "Lifecycle: 1 slot metadata unverified". Running go test ./cmd/internal/couchcmd -run 'Test(LifecycleDrainReportsOnlyUntargetedFacts|Supervision|Renicer)' -count=1 -timeout=60s fails at line 685 with that message. This is the remaining stale wording assertion identified in the inspected window; update it while retaining the assertion that exactly one untargeted fact is counted.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README omits the changed Ctrl+Y selection behavior
    detail: |
      cmd/internal/couchtty/console_menu.go:477 adds capability diagnostics ahead of the last feed notice. README.md:456 and :1730-1733 still promise the last notice when no selected failure applies, including repeated copies until another notice arrives. These are both affected README passages in this family. Update them to explain capability precedence and full-text recovery; README.md is unchanged in this range. The atlas update correctly describes the new behavior.
  - id: new
    severity: Important
    family: acceptance-contract-consistency
    title: |
      Done-when criteria were not reconciled with the revised contract
    detail: |
      workshop/issues/000789-couch-chrome-cap-slot-status-text-at-40-cells-ellipsis-and-shorten-built-in-status-messages-to-fit.md:44 requires every built-in status message to fit, whereas :49/:60 explicitly permit dynamic labels and errors to exceed the limit and enforce only fixed template contributions. The revised scope also adds capability-copy behavior without an explicit acceptance clause. Reconcile the second Done-when clause with the intended exemption and add capability-copy precedence and full-text recovery criteria. These are the acceptance gaps introduced by the scope revision.
```

Done-when: stale — the unconditional producer limit does not express the revised dynamic-text exemption or capability-copy contract.

1. **Strengths**

   - Both notice fields use one shared projection in `reserve.go:365/:378`.
   - `ChromeStatusText` reuses the existing sanitizer, width measurement, and fitting helpers.
   - Tests exercise ASCII, CJK, combining marks, controls, both rendering entry points, retained feed text, and full capability copying.
   - `atlas/couch.md` documents the new projection and copy behavior accurately.

2. **Critical findings**

   - Producer coverage and shortening remain incomplete; see `built-in-status-enforcement` above.

3. **Important findings**

   - Repair the failing lifecycle assertion, update both README passages, and reconcile the acceptance criteria.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Chrome, capability-recovery, and catalog selection passed.
   - Lifecycle/supervision/renicer selection failed on the stale lifecycle assertion.
   - Catalog negative controls cover several syntax forms but miss the fixed substitutions and callback path described above.
   - `git diff --check` passed. No files were modified.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both renderers share the projection and existing text utilities.
   - **ARCH-PURE: pass.** Projection remains deterministic and free of IO.
   - **ARCH-PURPOSE: flag.** The display cap is delivered; producer cleanup and enforcement are incomplete.
   - **ARCH-EVOLVE: pass.** No persisted-format semantics change.
   - No Core concepts table is present.

7. **Plan revision recommendations**

   - Add a `## Revisions` entry enumerating the fixed-value and callback producer families and their enforcement.
   - Align `## Done when` with the dynamic-text exemption and capability-copy acceptance cases.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1a716322-555f-449c-8e3f-47d698858ce5","repository":"github.com/xianxu/pair","issue_id":"000789","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/7d1307c1a5fff308","reviewed_head":"b532c3b064bdcb1b10f02a9232591650b88970d7","slot":"pair:4","requested_at":"2026-10-22T14:17:24.991459Z"},"phase":"finished","pid":77451,"started_at":"2026-10-22T14:17:25.082902Z","finished_at":"2026-10-22T14:19:30.025612Z","session_id":"01a163f9-4e64-7760-8e47-c188daa0a272","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.822864042s
head: "b532c3b064bdcb1b10f02a9232591650b88970d7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T07:32:00-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 789 — Couch chrome: cap slot status text at 40 cells (ellipsis) and shorten built-in status messages to fit |
| repo | pair |
| issue file | workshop/issues/000789-couch-chrome-cap-slot-status-text-at-40-cells-ellipsis-and-shorten-built-in-status-messages-to-fit.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5df9efb7d0df1a627f8d929895181fcf7b40ef73..b3b63b6136cb7c7a686106d74ff100f888e5a511 |
| command | sdlc close --issue 789 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T07:32:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared display cap and copy precedence pass focused tests. BR-2 through BR-4 are addressed. BR-1 remains blocking: the corrected producer sites have meaningful regression coverage, but the producer sweep still misses built-in messages supplied by other packages.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      ARCH-PURPOSE: the named sites are corrected and regression tests detect restored overlong wording, but the built-in-status-enforcement family remains incomplete. status_catalog_test.go:220-228 scans only couchtty and couchcmd, missing upstream producers. console_broadcast.go:273 combines broadcast.EndReason with fixed wording, producing 51/77/46-cell messages for three finite outcomes. console_webcast.go:221 forwards webcast/chrome.go:663's fixed 91-cell recovery message. console_continuation.go:215/:273 forwards checkpoint/miss.go:53-60 templates containing 73/103/103 fixed cells even with slot, reason and path empty. An expanded runtime matrix against the actual broadcast and continuation producers fails on these outputs while the committed catalog passes. This is a repeat in family built-in-status-enforcement: state and enforce the rule across all notice origins, including upstream helpers and fields, rather than fixing only these instances; enumerate finite outputs and shorten their chrome wording while preserving dynamic diagnostics.
  - id: BR-2
    disposition: addressed
    note: |
      lifecycle_notices_test.go:683-685 now checks the new wording while retaining the exactly-one count invariant. The focused lifecycle/supervision/renicer selection passes; restoring the old count assertion through a temporary Go overlay makes the exact regression fail.
  - id: BR-3
    disposition: addressed
    note: |
      The pinned README diff updates both passages at :456 and :1732 to describe capability precedence and full-text recovery. These match console_menu.go:468-487; the clipboard regression also verifies selected-error and held-payload precedence.
  - id: BR-4
    disposition: addressed
    note: |
      The pinned issue diff reconciles Done-when with finite built-in substitutions, dynamic diagnostic exemptions, capability-copy precedence and full-text recovery. These criteria agree with the revised Spec and the intended renderer/copy behavior.
```

1. **Strengths**

   - Both notice fields share `ChromeStatusText` through `reserve.go:365/:378`.
   - The projection reuses existing sanitization and display-width helpers without changing stored diagnostics.
   - Runtime matrices exercise actual formatters and callbacks; temporary negative controls fail as expected.
   - README and atlas describe the display cap and copy behavior.

2. **Critical findings**

   - **BR-1 remains open.** Upstream producer coverage is incomplete; concrete paths and measurements appear in its disposition above. Shorten chrome templates and cover their complete finite variants.

3. **Important findings**

   - None beyond the maintained blocking finding.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Passed chrome, copy, catalog and runtime-variant selection.
   - Passed lifecycle/supervision/renicer selection.
   - Passed webcast’s uncertain-input recovery test.
   - Expanded runtime coverage fails on the omitted broadcast and continuation templates.
   - `git diff --check` passed. Temporary overlays were removed; checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared projection and text utilities.
   - **ARCH-PURE — pass:** projection is deterministic and tested without IO.
   - **ARCH-PURPOSE — flag:** producer enforcement omits upstream sources.
   - **ARCH-MOCK — pass:** copy coverage uses controlled clipboard/browser doubles; no new external dependency.
   - **ARCH-CONSTRAINTS — pass:** bounded display output and linear processing.
   - **ARCH-SECURE — pass:** sanitization precedes rendering; full diagnostics remain separate.
   - **ARCH-ORDER — pass:** existing copy selection preserves held-payload and selected-error precedence.
   - **ARCH-FUNERAL — pass:** no new durable runtime artifacts or background lifetimes.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.
   - No Core concepts table is present.

7. **Plan revision recommendations**

   - Add a `## Revisions` entry defining the producer inventory transitively from notice consumers, including upstream helper returns and state fields.
   - Record the omitted broadcast, recovery and continuation families and their regression coverage; qualify the current claim that all finite variants fit.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6d5d19e3-351d-43e6-bea9-9853f5a49374","repository":"github.com/xianxu/pair","issue_id":"000789","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/513c6232e3ed386c","reviewed_head":"b3b63b6136cb7c7a686106d74ff100f888e5a511","slot":"pair:4","requested_at":"2026-10-22T14:28:57.510982Z"},"phase":"finished","pid":12857,"started_at":"2026-10-22T14:28:57.701927Z","finished_at":"2026-10-22T14:31:59.803653Z","session_id":"01a16403-e095-77c0-9a1f-8b8388a729a9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.089067625s
head: "b3b63b6136cb7c7a686106d74ff100f888e5a511"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
