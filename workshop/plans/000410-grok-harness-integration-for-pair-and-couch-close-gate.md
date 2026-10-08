---
gate: boundary-review
issue: 410
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-08T00:24:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README rosters and env docs omit grok and PAIR_GROK_ALT_SCREEN though grok is usable at M1
          detail: README.md:44,304,354,945,1137 still list five agents; the new per-harness opt-out env is undocumented. Add the roster entries now (cheap) or record the M2 deferral in Revisions.
          family: docs-lag-new-surface
          round: 1
        - id: BR-2
          severity: Minor
          title: Glued short form -s<uuid> is not seen by HasSessionID or Strip
          detail: pair grok -s<uuid> would also get a minted --session-id (likely a duplicate-argument error from grok) and the glued token would persist into saved args; the fresh-launch validator does refuse it.
          family: resumeform-spelling-coverage
          round: 1
        - id: BR-3
          severity: Minor
          title: inlineModeArgs strips --no-alt-screen from user prompt text after --
          detail: stripValuelessFlag ignores the -- boundary that insertBeforeDoubleDash now honors; the strip should stop at the first --.
          family: double-dash-boundary
          round: 1
        - id: BR-4
          severity: Minor
          title: TestPairInsertedTokensPrecedeDoubleDash composes helpers by hand and skips the no-duplicate invariant
          detail: The plan promised a no-flag-twice assertion across fresh/resume/restart; the test checks only the tail after --.
          family: test-restates-composition
          round: 1
        - id: BR-5
          severity: Minor
          title: Plan names ProviderGrokACPV1 grok-acp-v1; code ships ProviderGrokACPJSONLV1 grok-acp-jsonl-v1
          family: plan-code-drift
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#410 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T00:24:28-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-lag-new-surface` README rosters and env docs omit grok and PAIR_GROK_ALT_SCREEN though grok is usable at M1
  README.md:44,304,354,945,1137 still list five agents; the new per-harness opt-out env is undocumented. Add the roster entries now (cheap) or record the M2 deferral in Revisions.
- **BR-2** [Minor] `resumeform-spelling-coverage` Glued short form -s<uuid> is not seen by HasSessionID or Strip
  pair grok -s<uuid> would also get a minted --session-id (likely a duplicate-argument error from grok) and the glued token would persist into saved args; the fresh-launch validator does refuse it.
- **BR-3** [Minor] `double-dash-boundary` inlineModeArgs strips --no-alt-screen from user prompt text after --
  stripValuelessFlag ignores the -- boundary that insertBeforeDoubleDash now honors; the strip should stop at the first --.
- **BR-4** [Minor] `test-restates-composition` TestPairInsertedTokensPrecedeDoubleDash composes helpers by hand and skips the no-duplicate invariant
  The plan promised a no-flag-twice assertion across fresh/resume/restart; the test checks only the tail after --.
- **BR-5** [Minor] `plan-code-drift` Plan names ProviderGrokACPV1 grok-acp-v1; code ships ProviderGrokACPJSONLV1 grok-acp-jsonl-v1

## Open findings

- **BR-1** [Important] `docs-lag-new-surface` README rosters and env docs omit grok and PAIR_GROK_ALT_SCREEN though grok is usable at M1
- **BR-2** [Minor] `resumeform-spelling-coverage` Glued short form -s<uuid> is not seen by HasSessionID or Strip
- **BR-3** [Minor] `double-dash-boundary` inlineModeArgs strips --no-alt-screen from user prompt text after --
- **BR-4** [Minor] `test-restates-composition` TestPairInsertedTokensPrecedeDoubleDash composes helpers by hand and skips the no-duplicate invariant
- **BR-5** [Minor] `plan-code-drift` Plan names ProviderGrokACPV1 grok-acp-v1; code ships ProviderGrokACPJSONLV1 grok-acp-jsonl-v1
