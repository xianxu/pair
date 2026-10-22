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

---

## Re-review — 2026-10-22T14:59:32-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | d66407a76dc45888fd82e589c0a999f7f0fd5f65..435148c06892bc8db5d30aef95c70b1ca005ed93 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:59:32-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The interdiff changes the approved folding policy after close: groups now fold only when at least three quiet members qualify. This is explicitly recorded as superseding earlier behavior, rather than correcting a prior finding. The narrow fix-review contract therefore requires a full `sdlc close`. Focused tests pass, and README/atlas updates accompany the changes.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: boundary-review-scope
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      workshop/plans/000796-collapse-tab-style-plan.md:151-153 introduces a three-member folding threshold and explicitly supersedes the earlier any-positive-count policy; cmd/internal/couchtty/status_collapse.go:49-52 implements it. Neither disposed prior finding requested this behavior change. Operator approval does not expand this fix-only review's coverage. Run a full `sdlc close` against the revised Spec and Plan before crossing the boundary.
```

1. **Prior findings**

   No open findings require disposition. BR-1 and BR-2 remain previously addressed; neither is reopened.

2. **Strengths**

   - Grapheme tests compare measurement against physical `terminal.StyledRows` cells, providing an independent geometry oracle.
   - `rowtext.Fit` delegates to shared grapheme-safe fitting, with allocation checks covering ordinary prefixes.
   - New tests exercise the folding threshold, exact group identity, parked membership, focus-independent anchors, and count clicks.
   - README and atlas both document the changed surface.

3. **Critical findings**

   - The scope change above blocks admission through this fix-only review. Remedy: run a full `sdlc close`.

4. **Important and Minor findings**

   None raised.

5. **Test coverage notes**

   Independently passed:

   - `go test ./cmd/internal/textwidth ./cmd/internal/rowtext ./cmd/internal/couchtty -run 'Test(Grapheme|Fit|Collapse)' -count=1`
   - Pinned-range `git diff --check`

   No mutation tests or full repository suite were run. These results do not establish full-close acceptance of the expanded scope.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Fitting shares one implementation and uses the existing terminal grapheme metric.
   - **ARCH-PURE — pass:** Candidate selection and text measurement remain deterministic; projection changes stay in the Console boundary.
   - **ARCH-PURPOSE — flag:** The newly selected policy requires full-boundary validation; the previous close cannot establish its acceptance.
   - **ARCH-EVOLVE — pass:** This interdiff introduces no persisted-format changes.

7. **Plan revision recommendations**

   The new Revisions entry already records the policy change. Reconcile the main Spec/decision text with that threshold during the required full close; it still describes remaining quiet members becoming counts without stating the minimum.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"82b5b282-e73f-4b17-8691-736d4de0104a","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c4111e20918b4711","reviewed_head":"435148c06892bc8db5d30aef95c70b1ca005ed93","slot":"pair:4","requested_at":"2026-10-22T21:58:36.519519Z"},"phase":"finished","pid":17849,"started_at":"2026-10-22T21:58:36.727598Z","finished_at":"2026-10-22T21:59:30.987526Z","session_id":"01a1659f-8b6a-7ef3-a40b-c3bf6deeec16","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.279532458s
head: "435148c06892bc8db5d30aef95c70b1ca005ed93"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T15:14:11-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..78bccaf9c228a780a1fd8d3dbd120254072301db |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:14:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The revised policy received full-range review, resolving BR-3’s scope concern. Default wiring, folding thresholds, scoped clicks, geometry, and documentation are supported by passing tests. One contract regression blocks SHIP: collapse hides queued/loading reattachment placeholders.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Completed full-range review against the reconciled Spec and Plan, including the three-member threshold, live membership, launch anchors, and physical count clicks. Removing the threshold through a temporary overlay makes its regression test fail in all four one/two-member cases.
  - id: BR-1
    disposition: addressed
    note: |
      Exact StatusGroupID aggregation remains implemented; the primary/numbered hub and current/inactive regression matrix passes.
  - id: BR-2
    disposition: addressed
    note: |
      Inspected the corrected Core concepts tables against production declarations and the pinned diff; locations and new/modified/reused classifications agree.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: collapse-candidate-state-policy
    title: |
      Collapse filters out legitimate queued and loading placeholders
    detail: |
      cmd/internal/couchtty/console_presentation.go:83 rejects every known non-live row, including detached rows whose queued/loading placeholders were constructed at lines 63–74. Pending inventory remains detached until attachment succeeds, so default collapse silently removes these chips despite the Spec and Plan requiring explicit placeholders (ARCH-PURPOSE). Running TestTheStatusModelCarriesAPlaceholderPerPendingThread with only its fixture style changed to collapse fails with "placeholders = [], want one per pending thread"; exempting placeholders in a temporary projection overlay makes it pass alongside the parked-member exclusion test. Preserve legitimate pending placeholders while excluding non-live attached members, and run this behavioral regression under both styles.
```

1. **Strengths**

   - Collapse reuses the measured packer and drawing pass, including weighted overflow counts.
   - Count-click tests derive coordinates from physical terminal cells and exercise every character of `+4` and `+12` after an emoji alias.
   - Native tests cover collapse/multiline geometry for Claude and Codex.
   - README and atlas document the new option, default, counts, and scope recovery.

2. **Critical findings**

   The placeholder admission defect above at `cmd/internal/couchtty/console_presentation.go:83`. Add both-style coverage to the existing placeholder regression while retaining parked-member exclusion.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full package tests passed: `couchtty` (50.579s), `couchcmd` (114.246s), `textwidth`, `rowtext`, and `artifactpath`. Focused tests and diff whitespace checks also passed.

   Temporary overlays reproduced the placeholder failure, verified the proposed exemption, and confirmed the threshold test fails without its guard. Overlays were removed; tracked files remain unchanged. The full repository suite was not run.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass — shared packing, drawing, and presentation identity. |
   | ARCH-PURE | Pass — policy and scope filtering remain pure. |
   | ARCH-PURPOSE | **Flag** — pending-placeholder contract is not delivered. |
   | ARCH-MOCK | Pass — existing stateful doubles and native harness reused. |
   | ARCH-CONSTRAINTS | Pass — counted work and allocation checks pass. |
   | ARCH-SECURE | Pass — sanitization and confirmed-revision fencing retained. |
   | ARCH-ORDER | Pass — scope refresh, disappearance, nested return, and Escape transitions tested. |
   | ARCH-FUNERAL | Pass — new state is frame- or Console-scoped. |
   | ARCH-EVOLVE | Pass — no production persisted-format change. |

7. **Plan revision recommendations**

   None required: the plan already requires explicit pending placeholders; implementation and regression coverage need correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0fa3eabc-e563-4499-9278-cdf91e65b4f7","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bc4467ed6cec7008","reviewed_head":"78bccaf9c228a780a1fd8d3dbd120254072301db","slot":"pair:4","requested_at":"2026-10-22T22:10:52.011133Z"},"phase":"finished","pid":58247,"started_at":"2026-10-22T22:10:52.147326Z","finished_at":"2026-10-22T22:14:09.963461Z","session_id":"01a165aa-c3c4-7b92-9f48-bdc237cf4204","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.773852042s
head: "78bccaf9c228a780a1fd8d3dbd120254072301db"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T15:20:29-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..22b6e77554789861539120c0e5ca28d4c180ddc5 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:20:29-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the revised Spec and Plan. BR-4 is verified addressed: both styles preserve queued/loading placeholders, while collapse still excludes parked attached members. No new blocking findings. This review does not certify the separately required recorded-suite or staging gates.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      console_presentation.go:83 exempts placeholders from non-live exclusion. The regression at console_reattach_test.go:356 passes for both styles; removing the exemption through a temporary Go overlay makes collapse fail with "placeholders = [], want one per pending thread".
  - id: BR-1
    disposition: addressed
    note: |
      Exact owner/group aggregation and CollapseAnchor preserve the hub-role policy. The primary/numbered current/inactive regression matrix passes.
  - id: BR-2
    disposition: addressed
    note: |
      The plan's Core concepts tables and BR-2 Revision identify the actual declarations, paths and change statuses; unchanged geometry and menu machinery are explicitly described as reuse.
  - id: BR-3
    disposition: addressed
    note: |
      This review inspected the complete pinned base-to-head window against the revised Spec and Plan, including threshold, live membership, grapheme measurement and count navigation changes.
```

1. **Strengths**

   - Collapse reuses the measured packer and drawing pass, keeping rendering and hit spans consistent.
   - Counts preserve actor cardinality and exact owner/repository identity; conservation, threshold and scope regressions pass.
   - Physical-cell click tests cover every digit after emoji aliases. Native PTY tests confirm collapse geometry and multiline compatibility.
   - README and atlas document the default, option, count interaction and scope recovery.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed focused collapse, placeholder, scoped-menu, CLI/runtime wiring, grapheme/fitting, allocation-budget, native geometry, stale-revision, broadcast and annotation checks. BR-4’s negative control failed as expected. `git diff --check` passed; the checkout remained unchanged. The complete repository suite was not rerun.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** Shared packing, clipping, presentation identity and style default.
   - **ARCH-PURE — pass:** Candidate selection, measurement and scope filtering remain directly testable without IO.
   - **ARCH-PURPOSE — pass:** Default wiring, folding, navigation and stable geometry ship together.
   - **ARCH-MOCK — pass:** Existing fake hosts/children and native PTY conformance tests cover the integration boundary.
   - **ARCH-CONSTRAINTS — pass:** Counted N/2N work bounds and allocation checks pass.
   - **ARCH-SECURE — pass:** Label sanitization, confirmed-revision fencing and broadcast fail-closed behavior remain intact.
   - **ARCH-ORDER — pass:** Scope refresh, removal, nested return, Escape and ordinary reopen have explicit tested behavior.
   - **ARCH-FUNERAL — pass:** New data is frame- or session-local; no durable artifact family is introduced.
   - **ARCH-EVOLVE — pass:** No persisted-format change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f80d466a-3886-48f5-a6ef-4dff2e8c3f22","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/2367b06cb85946d6","reviewed_head":"22b6e77554789861539120c0e5ca28d4c180ddc5","slot":"pair:4","requested_at":"2026-10-22T22:19:02.688445Z"},"phase":"finished","pid":34232,"started_at":"2026-10-22T22:19:02.899163Z","finished_at":"2026-10-22T22:20:28.466189Z","session_id":"01a165b2-4196-7fd2-b5ab-a40b7ca7e9d1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.657755s
head: "22b6e77554789861539120c0e5ca28d4c180ddc5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-22T15:39:30-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..c3b09ebd717a6794f686bb72c1daea8d2b25d746 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T15:39:30-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the revised Spec and Plan. Default collapse, exact group counts, scoped navigation, one-row geometry, and explicit multiline compatibility are supported by code inspection and passing tests. No new blocking findings were identified.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Exact identity aggregation and hub-role regressions pass; restoring the pre-fix collapse implementation makes the regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      The corrected Core concepts tables match production declarations and diff statuses; the Revisions passage correctly identifies unchanged geometry and menu machinery as reuse.
  - id: BR-3
    disposition: addressed
    note: |
      This review inspected the complete pinned range against the reconciled Spec and Plan, including threshold, live membership, launch anchors, physical clicks, and ownership styling.
  - id: BR-4
    disposition: addressed
    note: |
      Queued/loading placeholder regressions pass in both styles; restoring the pre-fix projection makes the collapse test fail with no placeholders.
```

1. **Strengths**

   - Collapse aggregates by exact owner/repository identity and preserves actor cardinality through overflow.
   - Count hit spans come from the drawing pass and use the existing confirmed-revision input fence.
   - Physical-cell click tests cover every character of `+4` and `+12` after an emoji alias.
   - README and atlas describe the new option, default, scope recovery, and geometry behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed whole packages: `couchtty`, `couchcmd`, `textwidth`, `rowtext`, and `artifactpath`. Native geometry and multiline compatibility tests passed within those packages. Negative controls failed as expected for numbered hubs, pending placeholders, and inactive-owner styling. `git diff --check` passed; the checkout remained unchanged.

   This review does not establish a full repository suite result or operator acceptance.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: shared packing, drawing, width measurement, and ownership styling. |
   | ARCH-PURE | Pass: candidate preparation, packing, and scope filtering remain deterministic. |
   | ARCH-PURPOSE | Pass: CLI wiring, counts, clicks, scoping, and geometry ship together. |
   | ARCH-MOCK | Pass: existing injected doubles and native PTY tests cover the changed integration. |
   | ARCH-CONSTRAINTS | Pass: instrumented work bounds cover N/2N workloads; no new background work. |
   | ARCH-SECURE | Pass: sanitization and confirmed-geometry fences remain in the production path. |
   | ARCH-ORDER | Pass: scope refresh, nested return, Escape, ordinary reopen, and stale-click behavior are exercised. |
   | ARCH-FUNERAL | Pass: frame data is replaced and temporary menu scope has explicit reset paths; no durable artifacts added. |
   | ARCH-EVOLVE | Pass: no persisted format changes. |

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7ade9039-b3f0-40d2-a8a2-a3739f67520b","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ec524abfb93fec17","reviewed_head":"c3b09ebd717a6794f686bb72c1daea8d2b25d746","slot":"pair:4","requested_at":"2026-10-22T22:37:20.105026Z"},"phase":"finished","pid":9239,"started_at":"2026-10-22T22:37:20.235638Z","finished_at":"2026-10-22T22:39:29.931683Z","session_id":"01a165c2-ff34-7980-8c20-04250ac23580","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.767267833s
head: "c3b09ebd717a6794f686bb72c1daea8d2b25d746"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
