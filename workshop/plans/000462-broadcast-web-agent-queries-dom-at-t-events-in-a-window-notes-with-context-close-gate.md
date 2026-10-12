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
    - "n": 2
      timestamp: "2026-10-12T02:46:49-07:00"
      dispatched: "2026-10-12T02:44:07-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Select delegates to notes.Match and scopes CSS combinators to one tree. Reverting either one turns TestModelSelectorsSelectTheirElement or TestSelect red; the cross-time probe on all fixtures found no CSS-fallback mismatch.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The test now requires exactly one match wherever Structural.Resolve resolves uniquely; turning the notes.Match hand-off off makes it fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The contract recording now resolves n1 after the note and RunSurfaceContract asserts NoteResolved; the fake was changed to match.
          round: 2
        - id: BR-4
          disposition: addressed
          note: checkWindow and unreadable are shared helpers. The separate Events loops are a deliberate choice to keep the fake independent of the real implementation.
          round: 2
        - id: BR-5
          disposition: addressed
          note: --json with --max-lines is refused (cli_test cases), and the trailing more-matches line is pinned by TestDOMSaysWhenMatchesAreCut.
          round: 2
        - id: BR-6
          disposition: addressed
          note: The scrubber on stdout covers every text path; turning it off makes TestMainScrubsControlCharacters fail.
          round: 2
        - id: BR-7
          disposition: addressed
          note: 'Prose-only: the issue Log records the seam additions and the report to the TL. The report itself is the operator''s word and was not checked further.'
          round: 2
      findings:
        - id: BR-8
          severity: Important
          title: The atlas still says selectors search through shadow roots, frames and templates; after round 1 only >>> crosses trees
          detail: atlas/webcast-format.md:384-388 contradicts the tree-scoped matching (TestSelect pins .badge and "#card span" as matching nothing). Restate SelectorGrammar, or point to it, instead of keeping a hand-written copy.
          family: doc-restates-changed-semantics
          round: 2
        - id: BR-9
          severity: Minor
          title: FakeSurface.Notes panics on an empty recording (fake.go:54 indexes Records[len-1]); Querier.Notes returns nil
          detail: 'Brought in by the BR-3 fix and reproduced. This is the 2nd finding in this family. Rule: any input the real surface accepts, the fake must answer the same way, and RunSurfaceContract is the list of those inputs, so add the empty recording there rather than only guarding the fake.'
          family: fake-real-contract-gap
          round: 2
        - id: BR-10
          severity: Minor
          title: The matchSelector doc comment now sits above Match in notes/selector.go, so godoc for Match starts "matchSelector is"
          family: misplaced-doc-comment
          round: 2
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

## Round 2 — 2026-10-12T02:46:49-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Select delegates to notes.Match and scopes CSS combinators to one tree. Reverting either one turns TestModelSelectorsSelectTheirElement or TestSelect red; the cross-time probe on all fixtures found no CSS-fallback mismatch.
- BR-2 — addressed — The test now requires exactly one match wherever Structural.Resolve resolves uniquely; turning the notes.Match hand-off off makes it fail.
- BR-3 — addressed — The contract recording now resolves n1 after the note and RunSurfaceContract asserts NoteResolved; the fake was changed to match.
- BR-4 — addressed — checkWindow and unreadable are shared helpers. The separate Events loops are a deliberate choice to keep the fake independent of the real implementation.
- BR-5 — addressed — --json with --max-lines is refused (cli_test cases), and the trailing more-matches line is pinned by TestDOMSaysWhenMatchesAreCut.
- BR-6 — addressed — The scrubber on stdout covers every text path; turning it off makes TestMainScrubsControlCharacters fail.
- BR-7 — addressed — Prose-only: the issue Log records the seam additions and the report to the TL. The report itself is the operator's word and was not checked further.

### Raised

- **BR-8** [Important] `doc-restates-changed-semantics` The atlas still says selectors search through shadow roots, frames and templates; after round 1 only >>> crosses trees
  atlas/webcast-format.md:384-388 contradicts the tree-scoped matching (TestSelect pins .badge and "#card span" as matching nothing). Restate SelectorGrammar, or point to it, instead of keeping a hand-written copy.
- **BR-9** [Minor] `fake-real-contract-gap` FakeSurface.Notes panics on an empty recording (fake.go:54 indexes Records[len-1]); Querier.Notes returns nil
  Brought in by the BR-3 fix and reproduced. This is the 2nd finding in this family. Rule: any input the real surface accepts, the fake must answer the same way, and RunSurfaceContract is the list of those inputs, so add the empty recording there rather than only guarding the fake.
- **BR-10** [Minor] `misplaced-doc-comment` The matchSelector doc comment now sits above Match in notes/selector.go, so godoc for Match starts "matchSelector is"

## Open findings

- **BR-8** [Important] `doc-restates-changed-semantics` The atlas still says selectors search through shadow roots, frames and templates; after round 1 only >>> crosses trees
- **BR-9** [Minor] `fake-real-contract-gap` FakeSurface.Notes panics on an empty recording (fake.go:54 indexes Records[len-1]); Querier.Notes returns nil
- **BR-10** [Minor] `misplaced-doc-comment` The matchSelector doc comment now sits above Match in notes/selector.go, so godoc for Match starts "matchSelector is"
