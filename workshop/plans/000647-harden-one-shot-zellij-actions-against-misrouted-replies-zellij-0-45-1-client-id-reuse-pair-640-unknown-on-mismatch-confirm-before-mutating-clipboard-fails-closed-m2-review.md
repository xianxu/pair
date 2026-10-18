# Boundary Review — pair647-query-fix#647 (milestone M2)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair647-query-fix |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a13b19e43bff90d460caf49c3d07353fb2406661..bf0524122b46540e2db6d2e0f80d6503e4fbf1d1 |
| command | sdlc milestone-close --issue 647 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T14:52:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared validators, paired observations, explicit targets, and refusal presentation provide a solid foundation. M2 still admits note anchors without binding them to the displayed tab, permits two sidecar paths to mutate after cancellation, and leaves Couch query descendants uncollected. Temporary overlay tests reproduced these gaps; tracked files remain unchanged.

```findings
findings:
  - id: new
    severity: Critical
    family: action-authority-requires-context-binding
    title: |
      Confirmed note panes are not bound to the displayed tab
    detail: |
      cmd/internal/couchcmd/note_panes.go:127 passes every tab's panes to notePanes, which discards tab identity. AnchorFor then chooses the first overlapping rectangle. A stateful reproduction with a hidden review pane in tab 0 and the attached client focused on an agent in tab 1 produced Path:[pane work review nvim hidden.md]. Matching snapshots and the unchanged-frame check do not establish which tab those rectangles describe. Bind admission to the presenting client's tab and captured context, or retain the screen anchor when that correlation is unavailable. Add multi-tab and tab-switch regressions. ARCH-PURPOSE, ARCH-SECURE.
  - id: new
    severity: Critical
    family: unknown-observation-is-terminal
    title: |
      Sidecar fast paths issue native actions after admission cancellation
    detail: |
      cmd/internal/layoutcmd/layoutcmd.go:67 and :222 invoke native actions without checking cancellation after resolveRightTerminalContext. Canceling the context inside TerminalPaneIDs while returning a valid registered ID still issued focus-pane-id 3 and an addressed tab-switch write in deterministic tests. This is the 3rd finding in family unknown-observation-is-terminal. Do not fix only one instance: state and sweep the rule that cancellation or expiry terminates every M2 admission path, including positive local authority, before its next new effect. Check the context immediately before both mutations and add fast-path cancellation regressions. ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER.
  - id: new
    severity: Important
    family: query-processes-require-owned-collection
    title: |
      Couch's protected queries bypass the shared process collector
    detail: |
      cmd/internal/couchcmd/shortcut_focus.go:31 and cmd/internal/couchcmd/note_panes.go:42 still use cmd.Output instead of zellijexec.Run. Both query functions returned while an owned, redirected child remained alive in a private fake-binary reproduction. CommandContext and WaitDelay do not collect descendants, contrary to R2's query-lifetime contract. Route both through the existing collector and test descendant cleanup on success, cancellation and failure through these production entry points. ARCH-DRY, ARCH-ORDER, ARCH-FUNERAL.
  - id: new
    severity: Minor
    family: temporary-effects-require-cleanup
    title: |
      Cancellation after applying a flash skips its reset
    detail: |
      cmd/internal/clipcmd/run.go:250 returns on cancellation after SetPaneColor but before ResetPaneColorAfter. A deterministic test canceling during SetPaneColor reproduced an applied color with no scheduled reset. Once the flash is issued, guarantee its cleanup independently of whether further paste effects remain permitted. ARCH-FUNERAL.
```

1. **Strengths**

   - Clipboard admission validates the complete source/destination plan before staging or input.
   - Layout restoration retains completed embedding and its recovery record when later evidence becomes uncertain.
   - Refusals exercise the existing broker and wrapper presentation path.
   - Diagnostic validation checks status, exact target, body and grammar. Restoring the old substring matcher made its regression tests fail.
   - README and atlas changes cover the changed behavior and measured latency limitations.

2. **Critical findings**

   The displayed-tab binding and canceled-sidecar mutations above block M2. Both have deterministic reproductions through production logic.

3. **Important findings**

   Complete Couch’s adoption of the existing query collector. The two bypasses violate the approved ownership contract.

4. **Minor findings**

   Guarantee flash reset scheduling after an applied color, including cancellation.

5. **Test coverage notes**

   Full runs passed for layout, draft routing, clipboard, terminal, Couch TTY, wrapper, launcher, reply contracts and the stateful fake. Focused layout/clipboard/draft race checks also passed.

   The broader Couch run failed three listing-fixture tests; all three also failed on the pinned base after runtime-bundle generation. These are inherited failures. An additional diagnostic-matrix run exceeded a reviewer-imposed 60-second cap; the full launcher run passed. Native qualification was not rerun.

   Existing tests miss the reproduced multi-tab anchor, sidecar-cancellation and Couch descendant-lifetime cases. Temporary overlays and disposable baseline files were removed.

6. **Architectural notes**

   - **ARCH-DRY — flag:** reuse the existing collector at both Couch query boundaries.
   - **ARCH-PURE — pass:** shared contracts and caller projections retain a useful logic/IO separation.
   - **ARCH-PURPOSE — flag:** note anchoring lacks the promised captured-context binding; cancellation needs a complete authority-path sweep.
   - **ARCH-MOCK — pass:** stateful fixtures exercise the actual subprocess boundary and inspect effects.
   - **ARCH-CONSTRAINTS — flag:** sidecar paths can act after their admission budget expires.
   - **ARCH-SECURE — flag:** matching external rectangles lack displayed-tab provenance.
   - **ARCH-ORDER — flag:** cancellation does not consistently terminate effects, and Couch queries can leave work running.
   - **ARCH-FUNERAL — flag:** query descendants and canceled flashes lack guaranteed cleanup.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying the note-anchor tab/frame correlation rule, enumerating every M2 authority-to-effect path and its cancellation check, and naming both Couch queries as collector consumers. Clarify that cleanup of an already-issued flash survives admission cancellation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c9ec3e69-b4c4-4adf-9208-83f7ed32043a","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9893a6a93c44c1d6","reviewed_head":"bf0524122b46540e2db6d2e0f80d6503e4fbf1d1","slot":"pair:1","requested_at":"2026-10-17T21:45:51.843065Z"},"phase":"finished","pid":27829,"started_at":"2026-10-17T21:45:52.051494Z","finished_at":"2026-10-17T21:52:53.66775Z","session_id":"01a14bd4-13d3-78a3-87cb-c9b972817aa6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.010926334s
head: "bf0524122b46540e2db6d2e0f80d6503e4fbf1d1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T23:56:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a13b19e43bff90d460caf49c3d07353fb2406661..106554eb937654a2fa75901b248b3d709f762b59 |
| command | sdlc milestone-close --issue 647 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T23:56:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All four prior findings are addressed, with regression evidence that fails when the relevant correction is removed. One Important defect remains: cosmetic title changes can refuse otherwise valid layout actions, contrary to the plan’s decision-specific comparison contract.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Native note enrichment, its wiring, async resolver and setter chain are removed. Broadcast acceptance preserves screen data without native queries; restoring the old composition wiring fails TestCouchCompositionHasNoNativeNoteAdapter. README and atlas document the approved screen-only contract.
  - id: BR-6
    disposition: addressed
    note: |
      Both sidecar actions use Effect immediately before dispatch, and the manifest covers the enumerated M2 sinks. Disabling the shared cancellation check makes TestSidecarCancellationRefusesEveryTerminalEffect fail for both focus and addressed tab input.
  - id: BR-7
    disposition: addressed
    note: |
      queryShortcutClients uses zellijexec.Run; the note query is deleted. Replacing the collector with cmd.Run makes TestCouchQueriesCollectOwnedDescendants fail on success, failure, cancellation and pipe-holder completion.
  - id: BR-8
    disposition: addressed
    note: |
      flashAdmitted schedules reset after every attempted application independently of cancellation. Reintroducing the cancellation return before reset makes TestFlashCancellationStillOwesReset fail with an applied color and no reset.
findings:
  - id: new
    severity: Important
    kind: defect
    family: admission-compares-only-decision-facts
    title: |
      Cosmetic pane titles can refuse valid layout actions
    detail: |
      cmd/internal/layoutcmd/admission.go:55-66 serializes raw Title values into the confirmation projection, including titles whose pane role is already established by terminal_command. A disposable regression through RunToggleFocusedContext passes with stable replies but returns code 1 and ErrMismatch when only pane 2's title changes; excluding that cosmetic field makes it pass. Compare derived role and actual decision facts instead of raw cosmetic titles, preserving title-based role changes where relevant. Add this regression and sweep consumers of the shared helper. ARCH-PURPOSE: the plan explicitly excludes cosmetic titles from admission comparisons.
```

1. **Strengths**

   - Cancellation protection now covers local-authority fast paths and individual effects, with structural enforcement and behavioral tests.
   - Query ownership tests independently observe descendant lifetime across four completion paths.
   - Note enrichment is removed completely, including orphaned helpers; README and atlas explain the resulting contract.

2. **Critical findings**

   None.

3. **Important findings**

   The title comparison above affects the shared admission helper used by layout, terminal fallback and draft routing. Project derived roles and relevant state rather than serializing whole panes.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full package tests passed for `layoutcmd`, `termcmd`, `draftroute`, `clipcmd` and `zellijreply`. Targeted Couch, broadcast, launcher, wrapper and artifact tests passed, as did focused race tests across all seven M2 packages.

   Disposable mutation checks verified the four dispositions. The additional production-toggle regression reproduced the title defect. Disposable clones were removed; the reviewed checkout remains unchanged. Native qualification was inspected but not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared effect gate and process collector replace duplicated boundary handling.
   - **ARCH-PURE — pass:** selectors remain separated from runtime IO; new integration glue is classified accordingly.
   - **ARCH-PURPOSE — flag:** cosmetic comparison violates the promised decision-specific projection.
   - **ARCH-MOCK — pass:** stateful fixtures exercise production seams and record effects.
   - **ARCH-CONSTRAINTS — pass:** shared admission budgets and bounded restore traversal remain enforced.
   - **ARCH-SECURE — pass:** unproven note provenance is removed; malformed or contradictory authority refuses.
   - **ARCH-ORDER — pass:** cancellation and completed-effect-prefix tests cover controllable ordering.
   - **ARCH-FUNERAL — pass:** query collection, owed flash cleanup and removal of the async note owner have explicit endings.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the projection correction and cosmetic-title regression. Preserve the existing requirement that role, target, tab and relevant state changes still refuse.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c4d61848-76b4-4f4f-a34b-3cc5750943a9","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8bf54d54118d6e0d","reviewed_head":"106554eb937654a2fa75901b248b3d709f762b59","slot":"pair:1","requested_at":"2026-10-18T06:51:59.657606Z"},"phase":"finished","pid":16786,"started_at":"2026-10-18T06:51:59.91283Z","finished_at":"2026-10-18T06:56:01.541884Z","session_id":"01a14dc8-15ee-7842-ab3d-fc5f5f965752","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.427845417s
head: "106554eb937654a2fa75901b248b3d709f762b59"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T00:08:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a13b19e43bff90d460caf49c3d07353fb2406661..50e85e14617d41a620c64441a01f9aeee60030b1 |
| command | sdlc milestone-close --issue 647 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T00:08:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M2 is ready to close. BR-9 is addressed by a shared decision-fact projection, and executable regressions demonstrate that restoring raw title comparison breaks valid layout and terminal actions. No new blocking findings. M3’s editor and native fault qualification remain outside this boundary.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      admission.go:63-88 compares derived role and explicit decision facts rather than cosmetic titles. Layout, terminal, restoration and draft consumers use the shared helper. Both production-path regression tests pass at head and fail with ErrMismatch when raw pane serialization is restored in disposable checkouts, which were removed afterward.
```

1. **Strengths**

   - The [shared projection](/Users/xianxu/Library/Caches/sdlc/review-checkouts/cf7803dcf844e3a1/cmd/internal/layoutcmd/admission.go:63) preserves identity, native order, tab, state, required geometry, launch template and derived role while excluding cosmetic fields.
   - Stateful tests exercise production query/action seams and inspect effects, targets and recovery records.
   - Cancellation guards preserve completed progress; attempted flashes retain their cleanup obligation.
   - README and atlas changes document refusal behavior, timing limits and the approved removal of native note inference.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None raised.

5. **Test coverage notes**

   - Complete tests passed for layout, terminal, draft routing, clipboard, reply contracts, Couch, Couch TTY, artifact inventory, launcher and wrapper.
   - Focused race checks passed across the five action/admission packages.
   - BR-9 mutation controls failed as expected for cosmetic and registered-terminal title changes.
   - Broadcast’s full run had one unrelated private-socket failure. The pinned base reproduces it under the review’s long `TMPDIR`; head passes that test with `TMPDIR=/tmp`. Its implementation and test are unchanged in this range.
   - `git diff --check` passed. Native opt-in tests were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Consumers share projection, admission and effect guards.
   - **ARCH-PURE — pass:** Pure selectors/planners remain separate from injected runtime IO.
   - **ARCH-PURPOSE — pass:** BR-9’s shared consumers were swept; M2 delivers its approved scope.
   - **ARCH-MOCK — pass:** Stateful fake queries and effects use production seams.
   - **ARCH-CONSTRAINTS — pass:** Shared admission deadlines and bounded query counts remain enforced.
   - **ARCH-SECURE — pass:** Strict parsing and context binding precede query-derived effects; refusal notices exclude content.
   - **ARCH-ORDER — pass:** Confirmation, effect ordering, cancellation and partial-progress behavior have deterministic controls.
   - **ARCH-FUNERAL — pass:** Query collection and owed flash cleanup retain explicit owners; obsolete note workers are removed.

7. **Plan revision recommendations:** None required for M2.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b65e8bb8-1c9b-4ec8-bcd0-bbc1478faf13","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/cf7803dcf844e3a1","reviewed_head":"50e85e14617d41a620c64441a01f9aeee60030b1","slot":"pair:1","requested_at":"2026-10-18T07:04:37.019217Z"},"phase":"finished","pid":91781,"started_at":"2026-10-18T07:04:37.308059Z","finished_at":"2026-10-18T07:08:31.637259Z","session_id":"01a14dd3-a32d-7972-a2eb-4c7c7770b003","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.006435917s
head: "50e85e14617d41a620c64441a01f9aeee60030b1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
