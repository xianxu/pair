---
gate: boundary-review
issue: 469
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T03:14:05-07:00"
      dispatched: "2026-10-12T03:12:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: query/render.go spacedText is a second visible-text walker that disagrees with RenderedText
          detail: 'render.go:277 skips only script/style (not template/noscript/hidden) and splits inline text nodes; it feeds the "context: within" label at render.go:269 beside the RenderedText-based text line. Replace with n.RenderedText() and regenerate the goldens.'
          family: single-rendered-text-walker
          round: 1
        - id: BR-2
          severity: Minor
          title: visibility:hidden drops visible descendants; every select option counts as rendered text
          detail: dom.go:97 treats visibility:hidden as hiding the whole subtree, unlike CSS; blockTags includes option, so a closed select contributes every option (see the stress golden's "alpha beta gamma").
          family: rendered-text-fidelity
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T03:17:18-07:00"
      dispatched: "2026-10-12T03:16:24-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: spacedText removed; Describe uses n.RenderedText() (render.go:269); blog.events golden and TestSelect "AB" row pin it and go red if the old walker is restored.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'visibility now inherits and a descendant can set it back (dom.go:62); the dom_test case fails under the old subtree-skip, confirmed by mutation. Listing every select option is a documented decision: the live choice is not in the DOM.'
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 9a8b36a714da56bebd08ea85f4aa14920697b151
      blocked: false
---

# Gate ledger — pair#469 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T03:14:05-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `single-rendered-text-walker` query/render.go spacedText is a second visible-text walker that disagrees with RenderedText
  render.go:277 skips only script/style (not template/noscript/hidden) and splits inline text nodes; it feeds the "context: within" label at render.go:269 beside the RenderedText-based text line. Replace with n.RenderedText() and regenerate the goldens.
- **BR-2** [Minor] `rendered-text-fidelity` visibility:hidden drops visible descendants; every select option counts as rendered text
  dom.go:97 treats visibility:hidden as hiding the whole subtree, unlike CSS; blockTags includes option, so a closed select contributes every option (see the stress golden's "alpha beta gamma").

## Round 2 — 2026-10-12T03:17:18-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — spacedText removed; Describe uses n.RenderedText() (render.go:269); blog.events golden and TestSelect "AB" row pin it and go red if the old walker is restored.
- BR-2 — addressed — visibility now inherits and a descendant can set it back (dom.go:62); the dom_test case fails under the old subtree-skip, confirmed by mutation. Listing every select option is a documented decision: the live choice is not in the DOM.

## Open findings

(none — every finding has been disposed)
