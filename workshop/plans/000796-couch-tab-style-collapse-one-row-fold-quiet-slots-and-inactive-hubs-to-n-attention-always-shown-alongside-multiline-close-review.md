# Boundary Review — pair#796 (whole-issue close)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..bccaa4617a2fa49cf6a530cfab72fc04bb581b04 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:28:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The launch wiring, shared renderer, scoped switcher, and geometry tests are sound. One verified policy defect blocks shipping: numbered hubs do not receive the anchor preservation and combined folding promised by the plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: collapse-hub-role-policy
    title: |
      Numbered hubs violate the approved collapse policy
    detail: |
      cmd/internal/couchtty/status_collapse.go:13-39 uses legacy chipGroups runs and preserves quiet anchors only when SlotNumber == 0. A temporary regression overlay at width 100 produced "repo:1 +1  [:2]" for a quiet current hub at slot 1 with active worker 2, folding the current hub despite ample space. Making both members quiet under an inactive hub produced "repo:1 +1  repo:2 +1", with two count spans for the same (OwnerHub, GroupKey). The plan at lines 21-23 requires preserving the current hub when space allows and folding quiet inactive hubs with their workers into one count per identity. Preserve anchors by hub role and aggregate collapse counts by exact identity without changing multiline grouping; add primary/numbered × current/inactive regression cases.
```

1. **Strengths**

   - CLI validation and actual Console configuration are tested, including opaque message arguments.
   - Count spans come from the shared drawing pass and use the existing confirmed-revision mouse fence.
   - Scoped-menu tests cover owner collisions, disappearing membership, nested return, and Escape recovery.
   - Native tests retain kernel PTY-size and physical bottom-row assertions across both styles. README and atlas describe the new surface.

2. **Critical findings**

   - **Numbered-hub policy:** [status_collapse.go:13](/Volumes/TestScratch/review-checkouts/0802444ba0bdba70/cmd/internal/couchtty/status_collapse.go:13), with the anchor predicate at line 27. **ARCH-PURPOSE:** legacy display runs are insufficient for the promised collapse identity. Preserve the numbered current hub and consolidate inactive members sharing the exact owner/group.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Targeted collapse, scope, parser, and runtime-wiring tests passed.
   - Status, broadcast, privacy, annotation, chrome, and native wrap selections passed.
   - A temporary Go overlay reproduced both numbered-hub failures; it was removed and the repository remains clean.
   - Existing inactive-hub coverage uses slot zero, missing the numbered-hub cases. No prior findings required disposition.
   - Recorded-suite success and staged operator acceptance remain outstanding acceptance gates.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared packing, drawing, and presentation authority.
   - **ARCH-PURE — pass:** collapse selection and scope filtering remain pure.
   - **ARCH-PURPOSE — flag:** numbered-hub behavior underdelivers the approved policy.
   - **ARCH-MOCK — pass:** existing terminal/process doubles and native conformance harness are reused.
   - **ARCH-CONSTRAINTS — pass:** production-loop work counters exercise bounded planning.
   - **ARCH-SECURE — pass:** shared sanitization and confirmed-geometry fencing remain in place.
   - **ARCH-ORDER — pass:** scope recovery and refresh reconciliation use the existing menu model.
   - **ARCH-FUNERAL — pass:** no new durable artifacts or background owners.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying numbered-hub coverage and distinguishing legacy multiline display runs from collapse’s exact owner/group aggregation. Retain the approved anchor and folding contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"862aa34a-53af-4ca3-babd-4f4ccbcf6019","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0802444ba0bdba70","reviewed_head":"bccaa4617a2fa49cf6a530cfab72fc04bb581b04","slot":"pair:4","requested_at":"2026-10-22T21:26:26.57925Z"},"phase":"finished","pid":68160,"started_at":"2026-10-22T21:26:27.096314Z","finished_at":"2026-10-22T21:28:28.174404Z","session_id":"01a16582-1b94-7642-af75-73c61154c955","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 5.360233666s
head: "bccaa4617a2fa49cf6a530cfab72fc04bb581b04"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T14:37:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..40d868b02b1034ac96bc85830ca7324778947e27 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:37:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is addressed with meaningful regression evidence, and the focused behavior checks pass. The remaining blocker is a Core concepts table contradiction, classified Critical under this review’s explicit cross-check rule. No new runtime defect was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Exact-identity aggregation and hub-role retention are exercised by TestCollapseHubRoleAndIdentityIndependentOfSlotNumber. The pinned head passes; substituting pre-fix status_collapse.go through a temporary Go overlay makes all four numbered-hub cases fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      Core concepts table names incorrect implementation locations and change statuses
    detail: |
      workshop/plans/000796-collapse-tab-style-plan.md:53-55 lists the scoped switcher opener as modified in console.go/console_menu.go, but openGroupSwitcher is newly implemented in console_group_menu.go:6 and console_menu.go is unchanged. It also lists console_chrome_layout.go as modified although that file is reused unchanged. Correct the table to distinguish new, modified, and reused components and add a Revisions entry; the Core concepts cross-check explicitly makes these contradictions blocking.
```

1. **Strengths**

   - Collapse now aggregates by exact `(OwnerHub, GroupKey)` while preserving current hubs by role ([status_collapse.go:8](cmd/internal/couchtty/status_collapse.go:8)).
   - Count spans come from the drawing pass and share confirmed-revision mouse fencing.
   - Scoped selection covers ownership changes, disappearing groups, nested returns, and Escape recovery.
   - README and atlas both document the default, explicit multiline option, and count interaction.

2. **Critical findings**

   - Correct the Core concepts table at [plan:53](workshop/plans/000796-collapse-tab-style-plan.md:53) as detailed above. This is a documentation correction; wording-presence tests are unnecessary.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Focused collapse, group-menu, status, and style tests.
   - CLI extraction, validation, help, and Console wiring tests.
   - Native collapse/multiline PTY geometry, stale/pending chrome input, broadcast fail-closed/privacy, annotation anchoring, and tiny-host routing checks.
   - Pinned-range `git diff --check`.

   BR-1’s regression fails without the fix. Temporary overlay files were removed. The full repository suite and operator acceptance were not independently verified.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared packing, drawing, geometry, and presentation authority.
   - **ARCH-PURE — pass:** collapse preparation and scope filtering remain deterministic.
   - **ARCH-PURPOSE — pass:** default wiring, folding, navigation, and geometry are delivered together.
   - **ARCH-MOCK — pass:** existing stateful doubles and native PTY seams cover changed IO.
   - **ARCH-CONSTRAINTS — pass:** production-loop work counters exercise N/2N bounds.
   - **ARCH-SECURE — pass:** sanitization and confirmed-geometry fences remain intact.
   - **ARCH-ORDER — pass:** scope recovery and revision ordering have executable coverage.
   - **ARCH-FUNERAL — pass:** new candidates/spans are frame-local; scope has explicit clearing paths.
   - **ARCH-EVOLVE — pass:** no persisted-format change.

7. **Plan revision recommendation**

   Add a `## Revisions` entry recording the actual ownership: new `console_group_menu.go` opener, modified ordinary-reopen handling in `console.go`, and unchanged geometry machinery reused through the modified status-model projection.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"89a5ce5b-7f54-4d2d-89a5-2d560619d725","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b99615f570325050","reviewed_head":"40d868b02b1034ac96bc85830ca7324778947e27","slot":"pair:4","requested_at":"2026-10-22T21:35:32.349266Z"},"phase":"finished","pid":28296,"started_at":"2026-10-22T21:35:32.457368Z","finished_at":"2026-10-22T21:37:23.60825Z","session_id":"01a1658a-6b8e-78e1-9160-d441b7f434e5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.168133917s
head: "40d868b02b1034ac96bc85830ca7324778947e27"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T14:42:58-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..1c75450c44a2e7e423dbc04a64352a15f943d9b3 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:42:58-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the approved collapse policy, and BR-2’s documentation correction matches the actual declarations and diff. Focused rendering, CLI, scope, native geometry, and safety checks passed. No new findings. Recorded-suite completion and operator acceptance remain separate landing gates.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Inspected the pinned before/after plan diff. The table at workshop/plans/000796-collapse-tab-style-plan.md:55-62 now identifies openGroupSwitcher as new in console_group_menu.go, where its declaration exists at line 6, and explicitly lists console_chrome_layout.go and console_menu.go as reused unchanged, consistent with the base-to-head diff. The Revisions entry at lines 144-146 records the correction. Remaining table declarations and change statuses also match the implementation. This is a prose-only correction requiring no behavioral regression test.
```

1. **Strengths**

   - Collapse reuses measured packing and drawing; count spans come from the actual drawing pass.
   - Exact owner/group scoping precedes text filtering, and removed membership remains empty rather than broadening the switcher.
   - Native tests cover both styles, selected and background PTY sizes, kernel geometry, and physical bottom-row output.
   - README, help, and atlas explain the default, explicit multiline option, count navigation, and Escape recovery.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Collapse, numbered-hub role matrix, group menu, style parsing, and Console wiring tests.
   - Legacy measured-layout tests, native style matrix, stale/pending chrome input guards, broadcast privacy and indicator checks, annotation anchors, and pointer geometry checks.
   - `make test-affected-inventory` and pinned-range `git diff --check`.

   This review did not rerun the whole repository suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared packer, clipping, presentation membership, and default constant.
   - **ARCH-PURE — pass:** Candidate preparation and scoped filtering remain deterministic; Console handles effects.
   - **ARCH-PURPOSE — pass:** Default wiring, folding, count navigation, and stable geometry are delivered together.
   - **ARCH-MOCK — pass:** Existing injected terminal/process doubles and native conformance fixtures cover the affected boundaries.
   - **ARCH-CONSTRAINTS — pass:** Counted N/2N work tests exercise bounded planning; no new asynchronous work.
   - **ARCH-SECURE — pass:** Sanitization and confirmed-revision input guards remain in the production path.
   - **ARCH-ORDER — pass:** Existing menu transitions preserve scope across refresh/nested return and explicitly clear it on Escape or ordinary reopen.
   - **ARCH-FUNERAL — pass:** New frame data and session-local scope have bounded lifetimes; no durable artifacts introduced.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.

7. **Plan revision recommendations:** None; the corrected concepts table matches the code.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"16ff7c30-3638-4aee-8ad0-46d2668a79aa","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/85ca065272aafa4b","reviewed_head":"1c75450c44a2e7e423dbc04a64352a15f943d9b3","slot":"pair:4","requested_at":"2026-10-22T21:41:30.242351Z"},"phase":"finished","pid":13710,"started_at":"2026-10-22T21:41:30.523486Z","finished_at":"2026-10-22T21:42:55.80049Z","session_id":"01a1658f-e3ec-7e82-92e0-45920910b160","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 8.2861955s
head: "1c75450c44a2e7e423dbc04a64352a15f943d9b3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
