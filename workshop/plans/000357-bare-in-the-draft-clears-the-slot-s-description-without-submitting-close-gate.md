---
gate: boundary-review
issue: 357
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T13:29:32-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: previous_description now returns nil for legacy logged !! / !! text entries, unpinned by a test
          detail: nvim/bang_tag.lua:67 changed `if tag and tag.agent_text` to `if tag then text = tag.agent_text or ''`, which also flips `!!`-family entries (only possible in pre-#358 logs) from self-description to nil. Add check_previous cases for '!!' and '!! text' to pin the intended result. This is the only instance in the window.
          family: behavior-change-needs-pinning-test
          round: 1
        - id: BR-2
          severity: Minor
          title: atlas/couch.md list missing a comma after "(#337)"
          detail: '"a `!` tag line (#337) a `!!` describe line (#358), and a bare `!` clear line" needs a comma after (#337). Also, the header comment line at nvim/bang_tag.lua:4 is much longer than the lines around it.'
          family: doc-prose-typo
          round: 1
      recipe: small-diff-review
      blocked: false
    - "n": 2
      timestamp: "2026-09-30T15:24:36-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: nvim/bang_tag_test.lua now covers logged !! and !! text. Both expect nil; evaluating the prior implementation confirms both assertions would fail.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: atlas/couch.md:284 now has the missing comma. The long header comment at nvim/bang_tag.lua:4 remains; wrap it to finish this Minor advisory.
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: Clear-description coverage omits the explicitly promised no-fallback state
          detail: 'cmd/internal/couchcmd/run_test.go:653 exercises only a populated operator Description. Done when promises fallback description “or none”; no test exercises clearing with Description empty. This is the 2nd finding in family behavior-change-needs-pinning-test. Apply the rule “exercise every explicitly named outcome”: parameterize the CLI test over populated and empty Description, seed PublishedSummary in both, clear through --description=, and assert both persisted and displayed results. Enumeration: fallback present is covered; fallback absent is the sole missing state in this clause.'
          family: behavior-change-needs-pinning-test
          round: 2
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#357 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T13:29:32-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `behavior-change-needs-pinning-test` previous_description now returns nil for legacy logged !! / !! text entries, unpinned by a test
  nvim/bang_tag.lua:67 changed `if tag and tag.agent_text` to `if tag then text = tag.agent_text or ''`, which also flips `!!`-family entries (only possible in pre-#358 logs) from self-description to nil. Add check_previous cases for '!!' and '!! text' to pin the intended result. This is the only instance in the window.
- **BR-2** [Minor] `doc-prose-typo` atlas/couch.md list missing a comma after "(#337)"
  "a `!` tag line (#337) a `!!` describe line (#358), and a bare `!` clear line" needs a comma after (#337). Also, the header comment line at nvim/bang_tag.lua:4 is much longer than the lines around it.

## Round 2 — 2026-09-30T15:24:36-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — nvim/bang_tag_test.lua now covers logged !! and !! text. Both expect nil; evaluating the prior implementation confirms both assertions would fail.
- BR-2 — not-addressed — atlas/couch.md:284 now has the missing comma. The long header comment at nvim/bang_tag.lua:4 remains; wrap it to finish this Minor advisory.

### Raised

- **BR-3** [Important] `behavior-change-needs-pinning-test` Clear-description coverage omits the explicitly promised no-fallback state
  cmd/internal/couchcmd/run_test.go:653 exercises only a populated operator Description. Done when promises fallback description “or none”; no test exercises clearing with Description empty. This is the 2nd finding in family behavior-change-needs-pinning-test. Apply the rule “exercise every explicitly named outcome”: parameterize the CLI test over populated and empty Description, seed PublishedSummary in both, clear through --description=, and assert both persisted and displayed results. Enumeration: fallback present is covered; fallback absent is the sole missing state in this clause.

## Open findings

- **BR-2** [Minor] `doc-prose-typo` atlas/couch.md list missing a comma after "(#337)"
- **BR-3** [Important] `behavior-change-needs-pinning-test` Clear-description coverage omits the explicitly promised no-fallback state
