---
gate: boundary-review
issue: 502
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T17:09:38-07:00"
      dispatched: "2026-10-13T17:07:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: COUCH_SLOT_NICE=0 makes --actors print lowered=N though nothing was lowered
          detail: PlanRenice counts every descendant as AtLevel when level<=0 (renice.go:76), and niceColumn shows Lowered+AtLevel as "lowered". Skip counting at level 0 or hide the column; add a test.
          family: reported-facts-match-state
          round: 1
        - id: BR-2
          severity: Minor
          title: Low-agent notice compares against 0, not Couch's own niceness
          detail: renicer.go:423 fires when Agent.Nice>0. A Couch started from a niced shell gives every slot a notice whose advice (relaunch the slot) cannot help. Compare against Couch's own nice.
          family: reported-facts-match-state
          round: 1
        - id: BR-3
          severity: Minor
          title: procSource seam signature differs from the plan with no Revisions entry
          detail: The plan has Identity(pid) (string, error) and a ReniceProc type; the code has Identity(pid) string and procutil.Process.
          family: plan-matches-code
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: f4f6dfbbad2161a965dc2169f60e9fc7fce7a0fd
      blocked: false
---

# Gate ledger — pair#502 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T17:09:38-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `reported-facts-match-state` COUCH_SLOT_NICE=0 makes --actors print lowered=N though nothing was lowered
  PlanRenice counts every descendant as AtLevel when level<=0 (renice.go:76), and niceColumn shows Lowered+AtLevel as "lowered". Skip counting at level 0 or hide the column; add a test.
- **BR-2** [Minor] `reported-facts-match-state` Low-agent notice compares against 0, not Couch's own niceness
  renicer.go:423 fires when Agent.Nice>0. A Couch started from a niced shell gives every slot a notice whose advice (relaunch the slot) cannot help. Compare against Couch's own nice.
- **BR-3** [Minor] `plan-matches-code` procSource seam signature differs from the plan with no Revisions entry
  The plan has Identity(pid) (string, error) and a ReniceProc type; the code has Identity(pid) string and procutil.Process.

## Open findings

- **BR-1** [Minor] `reported-facts-match-state` COUCH_SLOT_NICE=0 makes --actors print lowered=N though nothing was lowered
- **BR-2** [Minor] `reported-facts-match-state` Low-agent notice compares against 0, not Couch's own niceness
- **BR-3** [Minor] `plan-matches-code` procSource seam signature differs from the plan with no Revisions entry
