---
gate: boundary-review
issue: 513
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T18:56:32-07:00"
      dispatched: "2026-10-12T18:53:26-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'Live-size rule''s display:inline-block (author origin) beats UA [hidden]{display:none}: hidden canvas/video/iframe drawn and marked'
          detail: 'render.js:192. A hidden element has no layout box, so it falls back to width/height attrs (canvas 300x150), shows as a phantom box and gets a "not captured" mark. Fix: :not([hidden]) in the selector; skip the attr fallback when a layout record lacks the id; add to the live test page.'
          family: viewer-rule-overrides-ua-hiding
          round: 1
        - id: BR-2
          severity: Important
          title: Live-size rule lives only in the document sheet; uncaptured elements inside shadow roots stay collapsed
          detail: 'drawAdopt (render.js:496) gives shadow roots no viewer sheet, and document rules don''t pierce shadow trees, so a canvas in a web component still draws as its 0x18 fallback. Fix: adopt a live-size sheet per root (n.getRootNode()); add a shadow-hosted canvas to replacedPage.'
          family: uncaptured-class-coverage
          round: 1
        - id: BR-3
          severity: Important
          title: Done when not updated for the Revisions scope (placeholders and live sizing for every uncaptured kind, iPad video case)
          detail: Done when names only the blog demo video and a failing video test; add bullets for the marks plus live-size behaviour and for the metadata-sized video.
          family: done-when-tracks-revisions
          round: 1
        - id: BR-4
          severity: Minor
          title: 'Plan item names drawCanvasSizes (code: drawLiveSizes); placeholders.js header still says the renderer sizes "a canvas"'
          family: stale-naming-in-docs
          round: 1
        - id: BR-5
          severity: Minor
          title: 'Expand not de-duplicated: repeated or stale childNodeCountUpdated each trigger a depth -1 requestChildNodes and a full-subtree Kids record'
          family: redundant-cdp-requests
          round: 1
        - id: BR-6
          severity: Minor
          title: drawLiveSizes / uncaptured() covered only by the opt-in live Chrome test; extract a pure rule builder for default-run tests
          family: renderer-logic-live-only-tests
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T19:08:41-07:00"
      dispatched: "2026-10-12T19:06:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: :not([hidden]) in liveSizeRule (render.js:56), no attr fallback once layoutKnown; pinned by placeholders.test.mjs regex and the laid-out-without-box case; live page adds hiddencv.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Per-root live sheet via getRootNode() and adoptLive/drawAdopt; live test adds shadow-hosted canvas (not runnable in this sandbox, reviewed by code).
          round: 2
        - id: BR-3
          disposition: addressed
          note: Done when gains the Revision bullet covering live size, iPad metadata video, shadow roots, hidden, and marks.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Plan names drawLiveSizes; placeholders.js header says each such element.
          round: 2
        - id: BR-5
          disposition: addressed
          note: slices.Contains de-dup while queued plus count != len(Children) guard; test asserts one Expand and no re-ask on matching count.
          round: 2
        - id: BR-6
          disposition: addressed
          note: liveSizeRule and isUncaptured exported pure with default-run node tests.
          round: 2
      findings:
        - id: BR-7
          severity: Minor
          title: '"On a tie the page''s CSS wins" holds only against adopted sheets, not style/link sheets'
          detail: 'render.js:228 and atlas/webcast.md: adopted sheets cascade after the document''s style and link sheets, so a zero-specificity page rule ties and loses to the live-size rule. Reword, or note the limit.'
          family: docs-overstate-behavior
          round: 2
        - id: BR-8
          severity: Minor
          title: layoutKnown = boxesT > 0 treats a layout at t=0 as no layout
          detail: 'This is the 2nd finding in family viewer-rule-overrides-ua-hiding. The rule covering both: the viewer''s sizing applies only to elements the operator''s layout rendered. layoutKnown should come from whether a layout record arrived (e.g. a flag), not from its timestamp.'
          family: viewer-rule-overrides-ua-hiding
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 447ed9c0c64fb362ae2ba204f20428fb74b271ff
      blocked: false
---

# Gate ledger — pair#513 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T18:56:32-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `viewer-rule-overrides-ua-hiding` Live-size rule's display:inline-block (author origin) beats UA [hidden]{display:none}: hidden canvas/video/iframe drawn and marked
  render.js:192. A hidden element has no layout box, so it falls back to width/height attrs (canvas 300x150), shows as a phantom box and gets a "not captured" mark. Fix: :not([hidden]) in the selector; skip the attr fallback when a layout record lacks the id; add to the live test page.
- **BR-2** [Important] `uncaptured-class-coverage` Live-size rule lives only in the document sheet; uncaptured elements inside shadow roots stay collapsed
  drawAdopt (render.js:496) gives shadow roots no viewer sheet, and document rules don't pierce shadow trees, so a canvas in a web component still draws as its 0x18 fallback. Fix: adopt a live-size sheet per root (n.getRootNode()); add a shadow-hosted canvas to replacedPage.
- **BR-3** [Important] `done-when-tracks-revisions` Done when not updated for the Revisions scope (placeholders and live sizing for every uncaptured kind, iPad video case)
  Done when names only the blog demo video and a failing video test; add bullets for the marks plus live-size behaviour and for the metadata-sized video.
- **BR-4** [Minor] `stale-naming-in-docs` Plan item names drawCanvasSizes (code: drawLiveSizes); placeholders.js header still says the renderer sizes "a canvas"
- **BR-5** [Minor] `redundant-cdp-requests` Expand not de-duplicated: repeated or stale childNodeCountUpdated each trigger a depth -1 requestChildNodes and a full-subtree Kids record
- **BR-6** [Minor] `renderer-logic-live-only-tests` drawLiveSizes / uncaptured() covered only by the opt-in live Chrome test; extract a pure rule builder for default-run tests

## Round 2 — 2026-10-12T19:08:41-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — :not([hidden]) in liveSizeRule (render.js:56), no attr fallback once layoutKnown; pinned by placeholders.test.mjs regex and the laid-out-without-box case; live page adds hiddencv.
- BR-2 — addressed — Per-root live sheet via getRootNode() and adoptLive/drawAdopt; live test adds shadow-hosted canvas (not runnable in this sandbox, reviewed by code).
- BR-3 — addressed — Done when gains the Revision bullet covering live size, iPad metadata video, shadow roots, hidden, and marks.
- BR-4 — addressed — Plan names drawLiveSizes; placeholders.js header says each such element.
- BR-5 — addressed — slices.Contains de-dup while queued plus count != len(Children) guard; test asserts one Expand and no re-ask on matching count.
- BR-6 — addressed — liveSizeRule and isUncaptured exported pure with default-run node tests.

### Raised

- **BR-7** [Minor] `docs-overstate-behavior` "On a tie the page's CSS wins" holds only against adopted sheets, not style/link sheets
  render.js:228 and atlas/webcast.md: adopted sheets cascade after the document's style and link sheets, so a zero-specificity page rule ties and loses to the live-size rule. Reword, or note the limit.
- **BR-8** [Minor] `viewer-rule-overrides-ua-hiding` layoutKnown = boxesT > 0 treats a layout at t=0 as no layout
  This is the 2nd finding in family viewer-rule-overrides-ua-hiding. The rule covering both: the viewer's sizing applies only to elements the operator's layout rendered. layoutKnown should come from whether a layout record arrived (e.g. a flag), not from its timestamp.

## Open findings

- **BR-7** [Minor] `docs-overstate-behavior` "On a tie the page's CSS wins" holds only against adopted sheets, not style/link sheets
- **BR-8** [Minor] `viewer-rule-overrides-ua-hiding` layoutKnown = boxesT > 0 treats a layout at t=0 as no layout
