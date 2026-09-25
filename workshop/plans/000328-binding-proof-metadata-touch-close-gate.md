---
gate: boundary-review
issue: 328
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-25T10:22:26-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: No test pins the not-smaller boundary of proofAllowsFullRevalidation
          detail: 'Deleting the Size < artifact.Size guard (query.go:257) makes no query test fail, so a truncated transcript with a valid first line would be re-read and established. Add a shrink test that asserts provisional + binding_stale. Sweep the other guard clauses in that function too: stable-file-ID mismatch, and a generation token appearing where the proof had none.'
          family: fallback-guard-boundaries-untested
          round: 1
        - id: BR-2
          severity: Important
          title: Operator confirmation of alt+n relaunch after detach/reattach not recorded
          detail: Done-when requires the operator to confirm the live relaunch → detach → reattach flow. The Log records only the scratch probe, yet the Plan's Verify-live box is ticked. Get the confirmation and log it before close.
          family: done-when-clause-unevidenced
          round: 1
        - id: BR-3
          severity: Minor
          title: binding_stale proof-failure diagnostic has no reader yet
          detail: The operator asked for a debugging log, but no Couch sink reads query diagnostics. File the COUCH_TRACE wiring as a tracked follow-up issue instead of leaving it as a Log note.
          family: diagnostic-without-sink
          round: 1
        - id: BR-4
          severity: Minor
          title: Comment in agent-agnostic sessioninventory names claude specifically
          detail: query.go:230 says "a resuming claude bumps ctime"; "a resuming agent" would be clearer.
          family: agent-agnostic-wording
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-25T10:35:32-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The new table covers shrink, stable-file-ID mismatch, and generation appearance, but does not explicitly assert BindingProvisional for the shrink case.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: The pinned issue Log still records only the scratch probe, not operator confirmation of relaunch → detach → reattach.
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Issue #330 is committed in the reviewed range and explicitly tracks carrying binding_stale diagnostics into COUCH_TRACE.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: The original query.go comment now says “resuming agent”; the remaining atlas wording is a newly introduced instance.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: Atlas contract still names one agent in generic revalidation prose
          detail: This is the 2nd finding in family agent-agnostic-wording. The remaining changed instance is atlas/session-identity.md:106, which says “a resuming claude”; change it to “a resuming agent” so the generic inventory contract is not tied to one provider.
          family: agent-agnostic-wording
          round: 2
      recipe: small-diff-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-25T10:39:31-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The guard-family table now asserts BindingProvisional and binding_stale for shrink, stable-file-ID replacement, and generation appearance.
          round: 3
        - id: BR-2
          disposition: addressed
          note: The pinned issue Log records operator confirmation of the relaunch → detach → reattach smoke flow.
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Issue #330 tracks carrying binding_stale diagnostics into COUCH_TRACE.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: The query comment now uses “resuming agent”.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Atlas wording now uses “resuming agent”.
          round: 3
      findings:
        - id: BR-6
          severity: Important
          title: Exact make test verification is not green
          detail: This is the 2nd finding in family done-when-clause-unevidenced. The exact required command fails at nvim/scrollback_test.lua because editor storage protection returns operation not permitted. Re-run successfully in a permitted environment and record the evidence before close.
          family: done-when-clause-unevidenced
          round: 3
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#328 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-25T10:22:26-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fallback-guard-boundaries-untested` No test pins the not-smaller boundary of proofAllowsFullRevalidation
  Deleting the Size < artifact.Size guard (query.go:257) makes no query test fail, so a truncated transcript with a valid first line would be re-read and established. Add a shrink test that asserts provisional + binding_stale. Sweep the other guard clauses in that function too: stable-file-ID mismatch, and a generation token appearing where the proof had none.
- **BR-2** [Important] `done-when-clause-unevidenced` Operator confirmation of alt+n relaunch after detach/reattach not recorded
  Done-when requires the operator to confirm the live relaunch → detach → reattach flow. The Log records only the scratch probe, yet the Plan's Verify-live box is ticked. Get the confirmation and log it before close.
- **BR-3** [Minor] `diagnostic-without-sink` binding_stale proof-failure diagnostic has no reader yet
  The operator asked for a debugging log, but no Couch sink reads query diagnostics. File the COUCH_TRACE wiring as a tracked follow-up issue instead of leaving it as a Log note.
- **BR-4** [Minor] `agent-agnostic-wording` Comment in agent-agnostic sessioninventory names claude specifically
  query.go:230 says "a resuming claude bumps ctime"; "a resuming agent" would be clearer.

## Round 2 — 2026-09-25T10:35:32-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The new table covers shrink, stable-file-ID mismatch, and generation appearance, but does not explicitly assert BindingProvisional for the shrink case.
- BR-2 — not-addressed — The pinned issue Log still records only the scratch probe, not operator confirmation of relaunch → detach → reattach.
- BR-3 — addressed — Issue #330 is committed in the reviewed range and explicitly tracks carrying binding_stale diagnostics into COUCH_TRACE.
- BR-4 — addressed — The original query.go comment now says “resuming agent”; the remaining atlas wording is a newly introduced instance.

### Raised

- **BR-5** [Minor] `agent-agnostic-wording` Atlas contract still names one agent in generic revalidation prose
  This is the 2nd finding in family agent-agnostic-wording. The remaining changed instance is atlas/session-identity.md:106, which says “a resuming claude”; change it to “a resuming agent” so the generic inventory contract is not tied to one provider.

## Round 3 — 2026-09-25T10:39:31-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The guard-family table now asserts BindingProvisional and binding_stale for shrink, stable-file-ID replacement, and generation appearance.
- BR-2 — addressed — The pinned issue Log records operator confirmation of the relaunch → detach → reattach smoke flow.
- BR-3 — addressed — Issue #330 tracks carrying binding_stale diagnostics into COUCH_TRACE.
- BR-4 — addressed — The query comment now uses “resuming agent”.
- BR-5 — addressed — Atlas wording now uses “resuming agent”.

### Raised

- **BR-6** [Important] `done-when-clause-unevidenced` Exact make test verification is not green
  This is the 2nd finding in family done-when-clause-unevidenced. The exact required command fails at nvim/scrollback_test.lua because editor storage protection returns operation not permitted. Re-run successfully in a permitted environment and record the evidence before close.

## Open findings

- **BR-6** [Important] `done-when-clause-unevidenced` Exact make test verification is not green
