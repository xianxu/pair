---
gate: boundary-review
issue: 365
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T14:47:12-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
          detail: |-
            Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: plan-enumerates-test-cases
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-01T14:47:12-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: probes/messageidle/bin/ is not gitignored; run.sh arm drops two Go binaries a git add -A would commit
          detail: zellijcalls ships probes/zellijcalls/.gitignore with bin/; messageidle has none (git check-ignore prints nothing). Add probes/messageidle/.gitignore with bin/; consider a test that every probe bin/ dir is ignored.
          family: probe-build-output-ignored
          round: 2
        - id: BR-3
          severity: Minor
          title: realBinary in probes/messageidle/main.go copies zellijcalls realZellij (ARCH-DRY); plan said extend zellijcalls
          detail: Record the deviation in Revisions or extract the PATH-skip-self lookup into a shared probe helper.
          family: shared-helper-not-reused
          round: 2
        - id: BR-4
          severity: Minor
          title: Idle test is single-wrapper with heartbeats, not the planned three-wrapper no-traffic test; Revisions does not record it
          detail: Done-when 1 says multi-slot; add a Revisions entry noting M2's rewrite covers multi-slot.
          family: plan-revision-drift
          round: 2
        - id: BR-5
          severity: Minor
          title: TestParentNameReadsThisTestsParent reads its own pid, not its parent
          family: test-name-matches-assertion
          round: 2
        - id: BR-6
          severity: Minor
          title: messageidle trace TSV grows unbounded while armed; only arm truncates it
          family: artifact-removal-path
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-01T14:48:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan Task 2.1 still enumerates 13 named tests; compress to the invariant/permutation strategy line before M2 execution (Minor, non-blocking).
          round: 3
        - id: BR-2
          disposition: addressed
          note: Root .gitignore:163 adds /probes/*/bin/ for the whole class; git check-ignore -v confirms it matches probes/messageidle/bin/zellij.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Plan Revisions (2026-10-01 M1 boundary review) records the separate shim and the realBinary/realZellij mirror, with the reason.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Plan Revisions records the single-wrapper heartbeat form and assigns the multi-slot no-traffic rewrite to Task 2.4.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Renamed TestParentNameReadsAKnownProcess with a comment; name now matches the own-pid assertion.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: run.sh unchanged; disarm could rm the trace since window writes its summary separately (Minor).
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#365 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T14:47:12-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-enumerates-test-cases` Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
  Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-10-01T14:47:12-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `probe-build-output-ignored` probes/messageidle/bin/ is not gitignored; run.sh arm drops two Go binaries a git add -A would commit
  zellijcalls ships probes/zellijcalls/.gitignore with bin/; messageidle has none (git check-ignore prints nothing). Add probes/messageidle/.gitignore with bin/; consider a test that every probe bin/ dir is ignored.
- **BR-3** [Minor] `shared-helper-not-reused` realBinary in probes/messageidle/main.go copies zellijcalls realZellij (ARCH-DRY); plan said extend zellijcalls
  Record the deviation in Revisions or extract the PATH-skip-self lookup into a shared probe helper.
- **BR-4** [Minor] `plan-revision-drift` Idle test is single-wrapper with heartbeats, not the planned three-wrapper no-traffic test; Revisions does not record it
  Done-when 1 says multi-slot; add a Revisions entry noting M2's rewrite covers multi-slot.
- **BR-5** [Minor] `test-name-matches-assertion` TestParentNameReadsThisTestsParent reads its own pid, not its parent
- **BR-6** [Minor] `artifact-removal-path` messageidle trace TSV grows unbounded while armed; only arm truncates it

## Round 3 — 2026-10-01T14:48:44-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan Task 2.1 still enumerates 13 named tests; compress to the invariant/permutation strategy line before M2 execution (Minor, non-blocking).
- BR-2 — addressed — Root .gitignore:163 adds /probes/*/bin/ for the whole class; git check-ignore -v confirms it matches probes/messageidle/bin/zellij.
- BR-3 — addressed — Plan Revisions (2026-10-01 M1 boundary review) records the separate shim and the realBinary/realZellij mirror, with the reason.
- BR-4 — addressed — Plan Revisions records the single-wrapper heartbeat form and assigns the multi-slot no-traffic rewrite to Task 2.4.
- BR-5 — addressed — Renamed TestParentNameReadsAKnownProcess with a comment; name now matches the own-pid assertion.
- BR-6 — not-addressed — run.sh unchanged; disarm could rm the trace since window writes its summary separately (Minor).

## Open findings

- **BR-1** [Minor] `plan-enumerates-test-cases` Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
- **BR-6** [Minor] `artifact-removal-path` messageidle trace TSV grows unbounded while armed; only arm truncates it
