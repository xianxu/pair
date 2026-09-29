---
gate: plan-quality
issue: 247
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-28T22:09:13-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Title-poller migration drops the draft with a zero floor, which silently disables Pair's frame meter and heat prefix on new sessions
          detail: titlepoller/run.go:151 skips updateFrameTitles and updateWorkspaceTitle while activityMTime is zero. Attach-created draft mtime used to supply a non-zero time at startup. With the draft dropped and CreatedAt passed as zero, a fresh session shows no frame meter or cmux prefix until its first transcript or log write. Give the poller a real floor (launch or pane-birth time), or state the change and pin it with a test.
          family: shared-seam-migration-behavior-change
          round: 1
        - id: PQ-2
          severity: Minor
          title: ActivityProbe, the Task 8 snippet and threadactivity.Latest disagree on signature
          detail: The probe is declared (time.Time, error), but the snippet returns (time.Time, bool, error) and calls Latest with different arguments from its definition. Pick one signature.
          family: plan-signature-consistency
          round: 1
        - id: PQ-3
          severity: Minor
          title: CreatedAt floor is recovery time for recovered threads, fabricating freshness
          detail: slotrecovery.go:326 and :437 stamp CreatedAt with Clock.Now() on recovery, so a long-idle recovered thread renders fresh for a day.
          family: activity-floor-provenance
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
          family: test-prose-enumeration
          round: 1
        - id: PQ-5
          severity: Minor
          title: Unknown-palette fallback renders both faded levels as the same SGR 90; document that levels are indistinguishable there
          family: fallback-level-collapse
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-28T22:11:31-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Pane-birth mtime is the floor; verified createflow.go:769-783 clears it per launch and titlepoller/run.go:125 awaits it before the :151 zero check.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: One Latest(ctx, rt, Thread) time.Time; the probe returns (time.Time, error) and Task 8 matches.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: CreatedAt floor dropped in favor of pane birth; ActionableThreadSummary unchanged.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Strategy lines added, but the prose case lists remain under Tasks 3, 4, 6 and 7. Minor, carried to close review.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Decision 5 now states the two faded levels look the same under the SGR 90 fallback and that the docs say so.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-28T22:13:19-07:00"
      agent: claude
      dispose:
        - id: PQ-4
          disposition: not-addressed
          note: Test-strategy lines added to Tasks 3/4/6/7, but the Step 1 prose case lists remain beside them; delete the lists (rule - one strategy line per risky function, cases live in code). Minor, carried to close.
          round: 3
      blocked: false
content_hash: cc12bb0b1260ae3723116cdd2a826ad65b6a8211e379abf3ce3dda140f9be305
---

# Gate ledger — pair#247 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T22:09:13-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `shared-seam-migration-behavior-change` Title-poller migration drops the draft with a zero floor, which silently disables Pair's frame meter and heat prefix on new sessions
  titlepoller/run.go:151 skips updateFrameTitles and updateWorkspaceTitle while activityMTime is zero. Attach-created draft mtime used to supply a non-zero time at startup. With the draft dropped and CreatedAt passed as zero, a fresh session shows no frame meter or cmux prefix until its first transcript or log write. Give the poller a real floor (launch or pane-birth time), or state the change and pin it with a test.
- **PQ-2** [Minor] `plan-signature-consistency` ActivityProbe, the Task 8 snippet and threadactivity.Latest disagree on signature
  The probe is declared (time.Time, error), but the snippet returns (time.Time, bool, error) and calls Latest with different arguments from its definition. Pick one signature.
- **PQ-3** [Minor] `activity-floor-provenance` CreatedAt floor is recovery time for recovered threads, fabricating freshness
  slotrecovery.go:326 and :437 stamp CreatedAt with Clock.Now() on recovery, so a long-idle recovered thread renders fresh for a day.
- **PQ-4** [Minor] `test-prose-enumeration` Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
- **PQ-5** [Minor] `fallback-level-collapse` Unknown-palette fallback renders both faded levels as the same SGR 90; document that levels are indistinguishable there

## Round 2 — 2026-09-28T22:11:31-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Pane-birth mtime is the floor; verified createflow.go:769-783 clears it per launch and titlepoller/run.go:125 awaits it before the :151 zero check.
- PQ-2 — addressed — One Latest(ctx, rt, Thread) time.Time; the probe returns (time.Time, error) and Task 8 matches.
- PQ-3 — addressed — CreatedAt floor dropped in favor of pane birth; ActionableThreadSummary unchanged.
- PQ-4 — not-addressed — Strategy lines added, but the prose case lists remain under Tasks 3, 4, 6 and 7. Minor, carried to close review.
- PQ-5 — addressed — Decision 5 now states the two faded levels look the same under the SGR 90 fallback and that the docs say so.

## Round 3 — 2026-09-28T22:13:19-07:00 (claude) — passed

### Disposed

- PQ-4 — not-addressed — Test-strategy lines added to Tasks 3/4/6/7, but the Step 1 prose case lists remain beside them; delete the lists (rule - one strategy line per risky function, cases live in code). Minor, carried to close.

## Open findings

- **PQ-4** [Minor] `test-prose-enumeration` Tasks 3, 4, 6 and 7 enumerate test cases in prose instead of one strategy line per risky function
