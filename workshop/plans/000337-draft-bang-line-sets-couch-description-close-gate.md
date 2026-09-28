---
gate: boundary-review
issue: 337
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T11:34:41-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: README omits the new draft bang syntax.
          detail: 'README.md:213 documents draft comments but has no bang-line documentation, and README is unchanged in the pinned range. Document the single new syntax family introduced at nvim/init.lua:803: single-line stripping and Couch tagging, standalone behavior, bare/multiline handling, and the intentional draft bash-mode compatibility change.'
          family: user-facing-surface-documentation
          round: 1
        - id: BR-2
          severity: Important
          title: Bang integration tests do not exercise publication or dispatch failures.
          detail: tests/bang-tag-nvim-test.sh:12 always supplies an immediately successful Couch stub, and nvim/bang_tag_integration_test.lua:15 always succeeds at dispatch. Enumerate and test unavailable executable, nonzero publisher exit, slow publisher, failed dispatch with no publication, and successful retry with exactly one publication. Existing BODY-only transaction tests cannot verify these bang-specific guarantees.
          family: side-effect-failure-contract-coverage
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#337 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T11:34:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-facing-surface-documentation` README omits the new draft bang syntax.
  README.md:213 documents draft comments but has no bang-line documentation, and README is unchanged in the pinned range. Document the single new syntax family introduced at nvim/init.lua:803: single-line stripping and Couch tagging, standalone behavior, bare/multiline handling, and the intentional draft bash-mode compatibility change.
- **BR-2** [Important] `side-effect-failure-contract-coverage` Bang integration tests do not exercise publication or dispatch failures.
  tests/bang-tag-nvim-test.sh:12 always supplies an immediately successful Couch stub, and nvim/bang_tag_integration_test.lua:15 always succeeds at dispatch. Enumerate and test unavailable executable, nonzero publisher exit, slow publisher, failed dispatch with no publication, and successful retry with exactly one publication. Existing BODY-only transaction tests cannot verify these bang-specific guarantees.

## Open findings

- **BR-1** [Important] `user-facing-surface-documentation` README omits the new draft bang syntax.
- **BR-2** [Important] `side-effect-failure-contract-coverage` Bang integration tests do not exercise publication or dispatch failures.
