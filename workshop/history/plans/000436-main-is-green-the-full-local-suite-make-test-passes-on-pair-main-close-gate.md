---
gate: boundary-review
issue: 436
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T02:22:18-07:00"
      dispatched: "2026-10-11T02:20:55-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Makefile.local test-goal unexport drops operator-set PAIR_* knobs (PAIR_PROBE_SETTLE, PAIR_PROBE_N, PAIR_PERF_BUDGETS, PAIR_LIVE_*)
          detail: 'The scrub unexports every environment-origin PAIR_/COUCH_/ZELLIJ name whenever all goals are test-*. Reproduced on GNU Make 3.81: PAIR_PROBE_SETTLE=0s make test-x leaves the recipe with an empty value. Affected: the documented PAIR_PROBE_SETTLE, PAIR_PROBE_N and PAIR_PROBE_SAMPLE_SECS (Makefile.local:146,155-156; atlas/couch.md:2309), plus PAIR_PROBE_SAMPLE_TSV, PAIR_PERF_BUDGETS, PAIR_ARTIFACT_VIOLATIONS_UPDATE and PAIR_LIVE_* opt-ins. Narrow the scrub to session-identity names (or filter-out the knob prefixes) and add a make-level regression check.'
          family: test-env-scrub-drops-operator-knobs
          round: 1
        - id: BR-2
          severity: Minor
          title: OwnerFramePath comment says the missing suffix cannot be a symlink; a dangling symlink also yields ENOENT
          detail: EvalSymlinks on a dangling link returns ErrNotExist, so the loop keeps the link name as written instead of resolving it.
          family: doc-claim-from-memory
          round: 1
        - id: BR-3
          severity: Minor
          title: Artifact violation list strips line numbers and dedupes, so a second identical violation in a known file passes
          family: ratchet-dedup-masks-new-instance
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T02:33:56-07:00"
      dispatched: "2026-10-11T02:32:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TEST_ENV_KNOBS filter-out plus tests/make-test-env-test.sh; confirmed red with TEST_ENV_KNOBS emptied (PAIR_PROBE_SETTLE dropped), green as committed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: env.go Lstat refusal; lease_test.go dangling case fails pre-fix because the old loop kept the link name and returned success.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: Counted comparison looks correct, but no test exercises a duplicate violation; extract fresh/stale into a pure func and test it.
          round: 2
      findings:
        - id: BR-4
          severity: Important
          title: Scrub still drops test opt-in knobs outside TEST_ENV_KNOBS; docs-grep check cannot see them
          detail: '2nd in family. PAIR_UPDATE_GROUPED_FIXTURES, PAIR_PARK_LATENCY_SMOKE, PAIR_TERMINAL_SOAK_DURATION, PAIR_TERMINAL_NATIVE, PAIR_TERMINAL_RESOURCE_PROBE (documented as X=1 go test at terminal/resource_test.go:28) and PAIR_MENU_PERF_TARGET are read by tests, are not session identity, and get unexported under make test goals, so the test skips silently. Rule: scrub only names a session exports; a test may read only names the scrub keeps. Enforce it from the real consumers: scan os.Getenv of PAIR_/COUCH_ names in *_test.go and require each to be in TEST_ENV_KNOBS or set by the test itself. Or derive the scrub from the launcher''s exported-name list instead of prefixes. Prevalence: about 6 operator knobs missed vs 10 found by the docs grep.'
          family: test-env-scrub-drops-operator-knobs
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T02:53:31-07:00"
      dispatched: "2026-10-11T02:52:42-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: diffKnownViolations is pure and counts duplicates; TestDiffKnownViolationsCountsDuplicates pins a second identical violation as fresh, and checkKnownArtifactViolations uses it.
          round: 3
        - id: BR-4
          disposition: addressed
          note: The scrub drops only the names production mentions, so test-only names pass through; all six named knobs are absent from the production scan, and make-test-env-test.sh passes with 165 derived knobs.
          round: 3
      findings:
        - id: BR-5
          severity: Minor
          title: The production-name scan sees only literal names, and the session-drop check samples 11 hand-listed names
          detail: A session variable whose name is built at runtime would get past the scrub. No such variable exists today, so this is only a note.
          family: test-env-scrub-drops-operator-knobs
          round: 3
      recipe: milestone-review
      reviewed: 85403b9e88fbafbe6dd10b17f0902b85e90b320e
      blocked: false
---

# Gate ledger — pair#436 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T02:22:18-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `test-env-scrub-drops-operator-knobs` Makefile.local test-goal unexport drops operator-set PAIR_* knobs (PAIR_PROBE_SETTLE, PAIR_PROBE_N, PAIR_PERF_BUDGETS, PAIR_LIVE_*)
  The scrub unexports every environment-origin PAIR_/COUCH_/ZELLIJ name whenever all goals are test-*. Reproduced on GNU Make 3.81: PAIR_PROBE_SETTLE=0s make test-x leaves the recipe with an empty value. Affected: the documented PAIR_PROBE_SETTLE, PAIR_PROBE_N and PAIR_PROBE_SAMPLE_SECS (Makefile.local:146,155-156; atlas/couch.md:2309), plus PAIR_PROBE_SAMPLE_TSV, PAIR_PERF_BUDGETS, PAIR_ARTIFACT_VIOLATIONS_UPDATE and PAIR_LIVE_* opt-ins. Narrow the scrub to session-identity names (or filter-out the knob prefixes) and add a make-level regression check.
- **BR-2** [Minor] `doc-claim-from-memory` OwnerFramePath comment says the missing suffix cannot be a symlink; a dangling symlink also yields ENOENT
  EvalSymlinks on a dangling link returns ErrNotExist, so the loop keeps the link name as written instead of resolving it.
- **BR-3** [Minor] `ratchet-dedup-masks-new-instance` Artifact violation list strips line numbers and dedupes, so a second identical violation in a known file passes

## Round 2 — 2026-10-11T02:33:56-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — TEST_ENV_KNOBS filter-out plus tests/make-test-env-test.sh; confirmed red with TEST_ENV_KNOBS emptied (PAIR_PROBE_SETTLE dropped), green as committed.
- BR-2 — addressed — env.go Lstat refusal; lease_test.go dangling case fails pre-fix because the old loop kept the link name and returned success.
- BR-3 — not-addressed — Counted comparison looks correct, but no test exercises a duplicate violation; extract fresh/stale into a pure func and test it.

### Raised

- **BR-4** [Important] `test-env-scrub-drops-operator-knobs` Scrub still drops test opt-in knobs outside TEST_ENV_KNOBS; docs-grep check cannot see them
  2nd in family. PAIR_UPDATE_GROUPED_FIXTURES, PAIR_PARK_LATENCY_SMOKE, PAIR_TERMINAL_SOAK_DURATION, PAIR_TERMINAL_NATIVE, PAIR_TERMINAL_RESOURCE_PROBE (documented as X=1 go test at terminal/resource_test.go:28) and PAIR_MENU_PERF_TARGET are read by tests, are not session identity, and get unexported under make test goals, so the test skips silently. Rule: scrub only names a session exports; a test may read only names the scrub keeps. Enforce it from the real consumers: scan os.Getenv of PAIR_/COUCH_ names in *_test.go and require each to be in TEST_ENV_KNOBS or set by the test itself. Or derive the scrub from the launcher's exported-name list instead of prefixes. Prevalence: about 6 operator knobs missed vs 10 found by the docs grep.

## Round 3 — 2026-10-11T02:53:31-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — diffKnownViolations is pure and counts duplicates; TestDiffKnownViolationsCountsDuplicates pins a second identical violation as fresh, and checkKnownArtifactViolations uses it.
- BR-4 — addressed — The scrub drops only the names production mentions, so test-only names pass through; all six named knobs are absent from the production scan, and make-test-env-test.sh passes with 165 derived knobs.

### Raised

- **BR-5** [Minor] `test-env-scrub-drops-operator-knobs` The production-name scan sees only literal names, and the session-drop check samples 11 hand-listed names
  A session variable whose name is built at runtime would get past the scrub. No such variable exists today, so this is only a note.

## Open findings

- **BR-5** [Minor] `test-env-scrub-drops-operator-knobs` The production-name scan sees only literal names, and the session-drop check samples 11 hand-listed names
