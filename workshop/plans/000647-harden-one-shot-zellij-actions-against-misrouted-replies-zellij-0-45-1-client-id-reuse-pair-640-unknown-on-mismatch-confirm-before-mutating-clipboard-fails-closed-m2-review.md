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
