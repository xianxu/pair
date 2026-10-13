---
gate: boundary-review
issue: 496
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T20:21:21-07:00"
      dispatched: "2026-10-12T20:20:20-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Atlas runbook presents the volume as "the fix" though the issue's own measurements show no fseventsd improvement
          detail: atlas/agent-scratch-volume.md:7-11 says "The fix is configuration..." but Revision 20:25 records CPU 11.2% to 12.7% and the memory episode never recurred, so the cure is unconfirmed. Add a measured-outcome note (15 GB churn moved off Data, journal empty; no CPU change measurable; suspected remaining source is ~/workspace churn). index.md:14 is neutral; no other instances.
          family: doc-claim-exceeds-evidence
          round: 1
        - id: BR-2
          severity: Important
          title: Layout table says mktemp "${TMPDIR}" roots follow the sandbox TMPDIR; the Log shows macOS mktemp without a template ignores TMPDIR
          detail: atlas/agent-scratch-volume.md:19. Per the 17:25 Log, template-less mktemp, mktemp -d and -t all go to /var/folders/.../T, which the sandbox denies. Document the working form mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX" (as tests/pair-rename.sh:74 uses) and warn about the template-less form. The related TMPDIR claim at :59 is raised separately as Minor.
          family: doc-claim-exceeds-evidence
          round: 1
        - id: BR-3
          severity: Minor
          title: Verify step says echo $TMPDIR points into the volume; this was verified only for sandboxed commands
          detail: atlas/agent-scratch-volume.md:59. Qualify it as "in a sandboxed command".
          family: doc-claim-exceeds-evidence
          round: 1
        - id: BR-4
          severity: Minor
          title: Teardown does not undo the tmutil exclusion or the mdutil setting
          detail: atlas/agent-scratch-volume.md:72-78. Both are moot once the volume is deleted, but a one-line note would make teardown fully mirror setup.
          family: teardown-mirrors-setup
          round: 1
        - id: BR-5
          severity: Minor
          title: Plan says the runbook lists other agents' temp-dir equivalents; it only gives a generic sentence
          detail: atlas/agent-scratch-volume.md:22-24 names no Codex (or other agent) setting. Name it, or reword the Plan bullet.
          family: plan-claim-vs-delivery
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T20:22:36-07:00"
      dispatched: "2026-10-12T20:21:58-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: :9 now says "mitigation"; :13-18 adds a measured outcome matching the Log (11.2 to 12.7%, ~15 MB, episode not recurred, cure unconfirmed, ~/workspace churn suspected).
          round: 2
        - id: BR-2
          disposition: addressed
          note: The Layout row at :26 now says templated mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX"; :33-36 warns that template-less mktemp ignores TMPDIR, per the 17:25 Log.
          round: 2
        - id: BR-3
          disposition: addressed
          note: :70-71 qualifies echo $TMPDIR as "in a sandboxed command" and notes that unsandboxed commands keep the inherited TMPDIR.
          round: 2
        - id: BR-4
          disposition: addressed
          note: :90-92 says no_log, mdutil and the TM exclusion go with the volume and gives the undo commands for keeping it.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The Plan bullet is reworded to "states the requirement, none configured or verified"; runbook :31 adds the matching parenthetical.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 90589dd8787e04ad97ff09e033b4ce21a4ccf62d
      blocked: false
---

# Gate ledger — pair#496 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T20:21:21-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-claim-exceeds-evidence` Atlas runbook presents the volume as "the fix" though the issue's own measurements show no fseventsd improvement
  atlas/agent-scratch-volume.md:7-11 says "The fix is configuration..." but Revision 20:25 records CPU 11.2% to 12.7% and the memory episode never recurred, so the cure is unconfirmed. Add a measured-outcome note (15 GB churn moved off Data, journal empty; no CPU change measurable; suspected remaining source is ~/workspace churn). index.md:14 is neutral; no other instances.
- **BR-2** [Important] `doc-claim-exceeds-evidence` Layout table says mktemp "${TMPDIR}" roots follow the sandbox TMPDIR; the Log shows macOS mktemp without a template ignores TMPDIR
  atlas/agent-scratch-volume.md:19. Per the 17:25 Log, template-less mktemp, mktemp -d and -t all go to /var/folders/.../T, which the sandbox denies. Document the working form mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX" (as tests/pair-rename.sh:74 uses) and warn about the template-less form. The related TMPDIR claim at :59 is raised separately as Minor.
- **BR-3** [Minor] `doc-claim-exceeds-evidence` Verify step says echo $TMPDIR points into the volume; this was verified only for sandboxed commands
  atlas/agent-scratch-volume.md:59. Qualify it as "in a sandboxed command".
- **BR-4** [Minor] `teardown-mirrors-setup` Teardown does not undo the tmutil exclusion or the mdutil setting
  atlas/agent-scratch-volume.md:72-78. Both are moot once the volume is deleted, but a one-line note would make teardown fully mirror setup.
- **BR-5** [Minor] `plan-claim-vs-delivery` Plan says the runbook lists other agents' temp-dir equivalents; it only gives a generic sentence
  atlas/agent-scratch-volume.md:22-24 names no Codex (or other agent) setting. Name it, or reword the Plan bullet.

## Round 2 — 2026-10-12T20:22:36-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — :9 now says "mitigation"; :13-18 adds a measured outcome matching the Log (11.2 to 12.7%, ~15 MB, episode not recurred, cure unconfirmed, ~/workspace churn suspected).
- BR-2 — addressed — The Layout row at :26 now says templated mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX"; :33-36 warns that template-less mktemp ignores TMPDIR, per the 17:25 Log.
- BR-3 — addressed — :70-71 qualifies echo $TMPDIR as "in a sandboxed command" and notes that unsandboxed commands keep the inherited TMPDIR.
- BR-4 — addressed — :90-92 says no_log, mdutil and the TM exclusion go with the volume and gives the undo commands for keeping it.
- BR-5 — addressed — The Plan bullet is reworded to "states the requirement, none configured or verified"; runbook :31 adds the matching parenthetical.

## Open findings

(none — every finding has been disposed)
