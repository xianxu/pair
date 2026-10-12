---
gate: boundary-review
issue: 462
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T02:38:13-07:00"
      dispatched: "2026-10-12T02:35:38-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: query.Select matches notes-model anchors (>>>, >) more loosely than notes defines them; anchors are ambiguous
          detail: 'selector.go reads >>> as a plain descendant, and > crosses shadow and frame boundaries via ParentElement. notes/selector.go''s matcher is tree-scoped. Probe: a blog astro-dev-toolbar >>> style:nth-of-type(1) anchor matches 6 elements; html:nth-of-type(1) matches 2 on stress. Delegate to an exported notes matcher, or make >>> and > tree-aware.'
          family: duplicated-selector-semantics
          round: 1
        - id: BR-2
          severity: Important
          title: TestModelSelectorsSelectTheirElement checks slices.Contains, so ambiguous anchors pass
          detail: Assert exactly one match wherever the model's own matcher resolves the anchor uniquely.
          family: test-asserts-containment-not-identity
          round: 1
        - id: BR-3
          severity: Minor
          title: FakeSurface reports note state at note time, Querier the final state; RunSurfaceContract pins neither
          family: fake-real-contract-gap
          round: 1
        - id: BR-4
          severity: Minor
          title: Querier.Events duplicates FakeSurface.Events; the window check and the reducer-error format are repeated in answer() and events.go
          family: duplicated-logic
          round: 1
        - id: BR-5
          severity: Minor
          title: events --json ignores --max-lines; DOMAt drops further matches with no trailing line when one page fills the budget exactly
          family: output-budget-gaps
          round: 1
        - id: BR-6
          severity: Minor
          title: Recorded page text is printed unquoted (render.go text lines), so control sequences pass through to stdout
          family: untrusted-text-to-terminal
          round: 1
        - id: BR-7
          severity: Minor
          title: UserAction (#457 seam) widened to seg/stop, and NoteView gained presentation fields, without the TL report the issue asks for
          family: seam-change-unreported
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#462 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T02:38:13-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `duplicated-selector-semantics` query.Select matches notes-model anchors (>>>, >) more loosely than notes defines them; anchors are ambiguous
  selector.go reads >>> as a plain descendant, and > crosses shadow and frame boundaries via ParentElement. notes/selector.go's matcher is tree-scoped. Probe: a blog astro-dev-toolbar >>> style:nth-of-type(1) anchor matches 6 elements; html:nth-of-type(1) matches 2 on stress. Delegate to an exported notes matcher, or make >>> and > tree-aware.
- **BR-2** [Important] `test-asserts-containment-not-identity` TestModelSelectorsSelectTheirElement checks slices.Contains, so ambiguous anchors pass
  Assert exactly one match wherever the model's own matcher resolves the anchor uniquely.
- **BR-3** [Minor] `fake-real-contract-gap` FakeSurface reports note state at note time, Querier the final state; RunSurfaceContract pins neither
- **BR-4** [Minor] `duplicated-logic` Querier.Events duplicates FakeSurface.Events; the window check and the reducer-error format are repeated in answer() and events.go
- **BR-5** [Minor] `output-budget-gaps` events --json ignores --max-lines; DOMAt drops further matches with no trailing line when one page fills the budget exactly
- **BR-6** [Minor] `untrusted-text-to-terminal` Recorded page text is printed unquoted (render.go text lines), so control sequences pass through to stdout
- **BR-7** [Minor] `seam-change-unreported` UserAction (#457 seam) widened to seg/stop, and NoteView gained presentation fields, without the TL report the issue asks for

## Open findings

- **BR-1** [Important] `duplicated-selector-semantics` query.Select matches notes-model anchors (>>>, >) more loosely than notes defines them; anchors are ambiguous
- **BR-2** [Important] `test-asserts-containment-not-identity` TestModelSelectorsSelectTheirElement checks slices.Contains, so ambiguous anchors pass
- **BR-3** [Minor] `fake-real-contract-gap` FakeSurface reports note state at note time, Querier the final state; RunSurfaceContract pins neither
- **BR-4** [Minor] `duplicated-logic` Querier.Events duplicates FakeSurface.Events; the window check and the reducer-error format are repeated in answer() and events.go
- **BR-5** [Minor] `output-budget-gaps` events --json ignores --max-lines; DOMAt drops further matches with no trailing line when one page fills the budget exactly
- **BR-6** [Minor] `untrusted-text-to-terminal` Recorded page text is printed unquoted (render.go text lines), so control sequences pass through to stdout
- **BR-7** [Minor] `seam-change-unreported` UserAction (#457 seam) widened to seg/stop, and NoteView gained presentation fields, without the TL report the issue asks for
