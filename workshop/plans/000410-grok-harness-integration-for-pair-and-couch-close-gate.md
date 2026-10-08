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
    - "n": 2
      timestamp: "2026-10-08T00:32:11-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README lines 3/44/146/304/354/945/1008/1142 now list grok; PAIR_CODEX_ALT_SCREEN/PAIR_GROK_ALT_SCREEN documented under Command Usage (matches inlineModes in agentargs.go).
          round: 2
        - id: BR-2
          disposition: addressed
          note: sessionIDToken reads glued single-letter spellings in HasSessionID/Strip/ContextSelector; TestGluedSessionIDShortForm goes red without it.
          round: 2
        - id: BR-3
          disposition: addressed
          note: stripValuelessFlag stops at --; the opted-out inlineModeArgs case in TestStripHelpersStopAtDoubleDash goes red without it. Reader-side sibling raised as a new family finding.
          round: 2
        - id: BR-4
          disposition: addressed
          note: No-duplicates assertion added before --; removing the strip in inlineModeArgs turns it red. The test still composes helpers by hand (noted, not re-raised).
          round: 2
        - id: BR-5
          disposition: addressed
          note: Plan Core concepts and Task 7 now name ProviderGrokACPJSONLV1 grok-acp-jsonl-v1.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: hasFlag in shouldMintSessionID reads prompt text after --, so the -- boundary has no single source
          detail: '2nd finding in this family. agentargs.go:17,259-260 scan the whole argv; measured shouldMintSessionID claude with args [--, --fork-session] returns false, so prompt text suppresses the mint. Rule: every argv reader or editor sees only the flag region before the first --. Fix by exporting one FlagRegion helper (today written five ways across resumeform.beforeDoubleDash, Strip, stripValuelessFlag, stripFlagAllForms, insertBeforeDoubleDash), route hasFlag through it, and extend TestStripHelpersStopAtDoubleDash to readers.'
          family: double-dash-boundary
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-10-08T11:33:24-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: hasFlag routes through resumeform.FlagRegion (agentargs.go:17-24), every argv editor uses it, and TestStripHelpersStopAtDoubleDash now asserts shouldMintSessionID ignores --fork-session/--session-id/-s after -- for claude, qoder and grok.
          round: 3
      findings:
        - id: BR-7
          severity: Important
          title: Done-when still promises orientation auto-submit, park/cold-resume and doctor firing that did not ship; no Revisions entry
          detail: '2nd in family. Rule: every deviation from a committed contract (Done-when, plan design or Durable state) gets a Revisions entry when decided, not just Log prose. Sweep all Done-when items and plan design claims in one pass: orientation goes to #414; park/cold-resume is verified or goes to #414; the doctor rerun is logged or moved out of scope; residue-refusal reporting changed from adapt near-miss to stderr; bellAttentionHarnesses and nvim/interrupt.lua are new M2 surfaces.'
          family: plan-code-drift
          round: 3
        - id: BR-8
          severity: Minor
          title: Accepted grok update methods differ across scanner, normalizer and ParseTokenUsage
          detail: applyGrokRecord accepts any non-empty method, normalizeGrokEvent accepts session/update and _x.ai/session/update, and usage accepts only session/update. These should share one set (ARCH-DRY).
          family: grok-envelope-single-source
          round: 3
        - id: BR-9
          severity: Minor
          title: interrupt.lua current_agent duplicates the agent-file read in init.lua pair_read_saved_config
          detail: nvim/init.lua:3099-3103 reads PAIR_AGENT_PATH the same way; it could call PairInterrupt.current_agent, or both could use a shared helper (ARCH-DRY).
          family: agent-file-read-duplicated
          round: 3
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-08T11:35:47-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: 'The issue Revisions move orientation, Couch park/cold-resume and the doctor tally to #414, and #414''s Done-when carries them; the plan Revisions record the stderr residue refusal and the new M2 surfaces.'
          round: 4
        - id: BR-8
          disposition: addressed
          note: grokUpdateMethods (scan_grok.go:31) is read by applyGrokRecord, normalizeGrokEvent and ParseTokenUsage; the unknown-method scanner test fails without the fix.
          round: 4
        - id: BR-9
          disposition: addressed
          note: PairInterrupt.read_agent_file is the one reader; init.lua:3099 and current_agent both use it, and interrupt_test.lua covers the nil, missing, empty and present cases.
          round: 4
      recipe: milestone-review
      blocked: false
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

## Round 2 — 2026-10-08T00:32:11-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README lines 3/44/146/304/354/945/1008/1142 now list grok; PAIR_CODEX_ALT_SCREEN/PAIR_GROK_ALT_SCREEN documented under Command Usage (matches inlineModes in agentargs.go).
- BR-2 — addressed — sessionIDToken reads glued single-letter spellings in HasSessionID/Strip/ContextSelector; TestGluedSessionIDShortForm goes red without it.
- BR-3 — addressed — stripValuelessFlag stops at --; the opted-out inlineModeArgs case in TestStripHelpersStopAtDoubleDash goes red without it. Reader-side sibling raised as a new family finding.
- BR-4 — addressed — No-duplicates assertion added before --; removing the strip in inlineModeArgs turns it red. The test still composes helpers by hand (noted, not re-raised).
- BR-5 — addressed — Plan Core concepts and Task 7 now name ProviderGrokACPJSONLV1 grok-acp-jsonl-v1.

### Raised

- **BR-6** [Minor] `double-dash-boundary` hasFlag in shouldMintSessionID reads prompt text after --, so the -- boundary has no single source
  2nd finding in this family. agentargs.go:17,259-260 scan the whole argv; measured shouldMintSessionID claude with args [--, --fork-session] returns false, so prompt text suppresses the mint. Rule: every argv reader or editor sees only the flag region before the first --. Fix by exporting one FlagRegion helper (today written five ways across resumeform.beforeDoubleDash, Strip, stripValuelessFlag, stripFlagAllForms, insertBeforeDoubleDash), route hasFlag through it, and extend TestStripHelpersStopAtDoubleDash to readers.

## Round 3 — 2026-10-08T11:33:24-07:00 (claude) — BLOCKED

### Disposed

- BR-6 — addressed — hasFlag routes through resumeform.FlagRegion (agentargs.go:17-24), every argv editor uses it, and TestStripHelpersStopAtDoubleDash now asserts shouldMintSessionID ignores --fork-session/--session-id/-s after -- for claude, qoder and grok.

### Raised

- **BR-7** [Important] `plan-code-drift` Done-when still promises orientation auto-submit, park/cold-resume and doctor firing that did not ship; no Revisions entry
  2nd in family. Rule: every deviation from a committed contract (Done-when, plan design or Durable state) gets a Revisions entry when decided, not just Log prose. Sweep all Done-when items and plan design claims in one pass: orientation goes to #414; park/cold-resume is verified or goes to #414; the doctor rerun is logged or moved out of scope; residue-refusal reporting changed from adapt near-miss to stderr; bellAttentionHarnesses and nvim/interrupt.lua are new M2 surfaces.
- **BR-8** [Minor] `grok-envelope-single-source` Accepted grok update methods differ across scanner, normalizer and ParseTokenUsage
  applyGrokRecord accepts any non-empty method, normalizeGrokEvent accepts session/update and _x.ai/session/update, and usage accepts only session/update. These should share one set (ARCH-DRY).
- **BR-9** [Minor] `agent-file-read-duplicated` interrupt.lua current_agent duplicates the agent-file read in init.lua pair_read_saved_config
  nvim/init.lua:3099-3103 reads PAIR_AGENT_PATH the same way; it could call PairInterrupt.current_agent, or both could use a shared helper (ARCH-DRY).

## Round 4 — 2026-10-08T11:35:47-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — The issue Revisions move orientation, Couch park/cold-resume and the doctor tally to #414, and #414's Done-when carries them; the plan Revisions record the stderr residue refusal and the new M2 surfaces.
- BR-8 — addressed — grokUpdateMethods (scan_grok.go:31) is read by applyGrokRecord, normalizeGrokEvent and ParseTokenUsage; the unknown-method scanner test fails without the fix.
- BR-9 — addressed — PairInterrupt.read_agent_file is the one reader; init.lua:3099 and current_agent both use it, and interrupt_test.lua covers the nil, missing, empty and present cases.

## Open findings

(none — every finding has been disposed)
