---
gate: boundary-review
issue: 288
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T22:09:30-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Test cases enumerated in prose and the diff restated as full code blocks
          detail: |-
            Task 1 Step 1 (seven cases), Task 4 Step 2 (five scenarios with per-case assertions) and the full panebirth.go / birthwatch.go / cancellableHandoff / runCreate code blocks will be rewritten as code within the hour. Compress to named functions plus one strategy line per risky function (e.g. watchBirth over probe-answer x client-exit order -> stateful fake with scripted liveness and launch release, guarded by the four mutation checks).
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: plan-restates-executable-artifacts
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-18T22:09:30-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: The fake's cancelled-launch return cannot occur in production, so the code != 0 guard is dead in the real seam
          detail: |-
            fakeRuntime.LaunchSession returns (-1, nil) on ctx.Done and (0, nil) only
            when released before the cancel, but exec.Cmd with Cancel+WaitDelay returns
            ctx.Err() whenever Cancel succeeded and the child then exits 0 (verified:
            Run err = context canceled, ProcessState.ExitCode = 0). runBlockingHandoff
            (osruntime.go:185) turns that into (1, context.Canceled) and createflow.go:794
            handles err before the verdict at :803, so a clean quit racing the probe is
            reported as "failed to launch zellij session: context canceled" with the quit
            cleanup skipped, and a zellij client that exits 0 on SIGTERM loses the
            dead-birth diagnosis. Derive the code from cmd.ProcessState when the process
            ran, and let the fake model a clean release under cancel.
          family: fake-diverges-from-seam
          round: 2
        - id: BR-3
          severity: Important
          title: 'Neither cause guard is pinned: removing either alone leaves test 6 green 20x under -race'
          detail: |-
            In a scratch copy of the pinned tree, dropping `&& code != 0`
            (createflow.go:803) alone, or the post-probe ctx recheck
            (birthwatch.go:82) alone, leaves TestCleanQuitRacingTheProbeKeepsTheNormalPath
            passing 20/20 under -race; only removing both fails it. The test cannot
            control the stopWatch-vs-verdict interleaving, so its "holds under every
            interleaving" claim samples one ordering. Extract the dead-path decision as a
            pure predicate with a table test, and add a seam that forces
            verdict-before-stopWatch.
          family: race-guard-untested-interleaving
          round: 2
        - id: BR-4
          severity: Minor
          title: The birthUnknown loop has no cap, so a persistently failing probe repeats a machine-wide list-sessions every bound for the life of the launch
          detail: |-
            birthwatch.go:70-90 loops bound-then-probe forever while the probe errors and
            the pane never appears. That probe is itself the birth-window risk for other
            threads (ARCH-CONSTRAINTS). Cap the unknown rounds or back off.
          family: unbounded-probe-rounds
          round: 2
        - id: BR-5
          severity: Minor
          title: 'The evidence path is single-sourced but its observation is not: ModTime in the poller, FileSize in the watch'
          family: single-source-partial
          round: 2
        - id: BR-6
          severity: Minor
          title: 'SKILL.md''s "the 2026-09-18 runs predate -exit-wait" sits above #288 rows that are also 2026-09-18'
          family: doc-referent-ambiguous
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-18T22:25:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: |-
            Declined with a rationale recorded in the plan's Revisions; the predicted
            drift did occur (cancellableHandoff vs killOnCancel, livenessAfterLaunch
            vs launchProbed, the watchBirth block missing the backoff), so the RULE
            the family needs is: every divergence from a plan code block must land in
            Revisions in the same round. Minor, non-blocking.
          round: 3
        - id: BR-2
          disposition: addressed
          note: |-
            Verified by mutation: reverting runBlockingHandoff to the ExitError form
            fails TestCancelledHandoffReportsAChildsCleanExit, which drives real exec.
          round: 3
        - id: BR-3
          disposition: addressed
          note: |-
            Verified by mutation: dropping the code guard fails 2 tests and dropping
            the post-probe ctx recheck fails 1 — each alone, unlike round 1.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: |-
            The backoff ships and is reachable, but removing `wait = nextProbeWait(wait)`
            from birthwatch.go:91 leaves the suite green; only the function is pinned.
          round: 3
        - id: BR-5
          disposition: addressed
          note: |-
            panebirth.Evidence now states that birth is the file's existence and that
            each waiter stats it through its own seam; the referents (poller ModTime,
            watch FileSize) match the code.
          round: 3
        - id: BR-6
          disposition: addressed
          note: |-
            SKILL.md now says the rows above the #288 ones predate -exit-wait, which
            disambiguates the empty Hung column on the same-date rows.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: zellijLogPath re-declares zellij's tmp-dir layout that the probe already owns
          detail: |-
            birthwatch.go:131 and probes/zellijbirthrace/main.go:100 both encode
            os.TempDir()/zellij-<uid>. Same module, one external fact, two sources; a
            zellij layout change drifts one of them. Go's cmd/internal boundary blocks
            direct reuse, so this is a note for the next time either is touched.
          family: duplicated-external-tool-knowledge
          round: 3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#288 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T22:09:30-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-restates-executable-artifacts` Test cases enumerated in prose and the diff restated as full code blocks
  Task 1 Step 1 (seven cases), Task 4 Step 2 (five scenarios with per-case assertions) and the full panebirth.go / birthwatch.go / cancellableHandoff / runCreate code blocks will be rewritten as code within the hour. Compress to named functions plus one strategy line per risky function (e.g. watchBirth over probe-answer x client-exit order -> stateful fake with scripted liveness and launch release, guarded by the four mutation checks).
  (carried from plan-quality PQ-1, deferred to the boundary review)

## Round 2 — 2026-09-18T22:09:30-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `fake-diverges-from-seam` The fake's cancelled-launch return cannot occur in production, so the code != 0 guard is dead in the real seam
  fakeRuntime.LaunchSession returns (-1, nil) on ctx.Done and (0, nil) only
  when released before the cancel, but exec.Cmd with Cancel+WaitDelay returns
  ctx.Err() whenever Cancel succeeded and the child then exits 0 (verified:
  Run err = context canceled, ProcessState.ExitCode = 0). runBlockingHandoff
  (osruntime.go:185) turns that into (1, context.Canceled) and createflow.go:794
  handles err before the verdict at :803, so a clean quit racing the probe is
  reported as "failed to launch zellij session: context canceled" with the quit
  cleanup skipped, and a zellij client that exits 0 on SIGTERM loses the
  dead-birth diagnosis. Derive the code from cmd.ProcessState when the process
  ran, and let the fake model a clean release under cancel.
- **BR-3** [Important] `race-guard-untested-interleaving` Neither cause guard is pinned: removing either alone leaves test 6 green 20x under -race
  In a scratch copy of the pinned tree, dropping `&& code != 0`
  (createflow.go:803) alone, or the post-probe ctx recheck
  (birthwatch.go:82) alone, leaves TestCleanQuitRacingTheProbeKeepsTheNormalPath
  passing 20/20 under -race; only removing both fails it. The test cannot
  control the stopWatch-vs-verdict interleaving, so its "holds under every
  interleaving" claim samples one ordering. Extract the dead-path decision as a
  pure predicate with a table test, and add a seam that forces
  verdict-before-stopWatch.
- **BR-4** [Minor] `unbounded-probe-rounds` The birthUnknown loop has no cap, so a persistently failing probe repeats a machine-wide list-sessions every bound for the life of the launch
  birthwatch.go:70-90 loops bound-then-probe forever while the probe errors and
  the pane never appears. That probe is itself the birth-window risk for other
  threads (ARCH-CONSTRAINTS). Cap the unknown rounds or back off.
- **BR-5** [Minor] `single-source-partial` The evidence path is single-sourced but its observation is not: ModTime in the poller, FileSize in the watch
- **BR-6** [Minor] `doc-referent-ambiguous` SKILL.md's "the 2026-09-18 runs predate -exit-wait" sits above #288 rows that are also 2026-09-18

## Round 3 — 2026-09-18T22:25:43-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Declined with a rationale recorded in the plan's Revisions; the predicted
drift did occur (cancellableHandoff vs killOnCancel, livenessAfterLaunch
vs launchProbed, the watchBirth block missing the backoff), so the RULE
the family needs is: every divergence from a plan code block must land in
Revisions in the same round. Minor, non-blocking.
- BR-2 — addressed — Verified by mutation: reverting runBlockingHandoff to the ExitError form
fails TestCancelledHandoffReportsAChildsCleanExit, which drives real exec.
- BR-3 — addressed — Verified by mutation: dropping the code guard fails 2 tests and dropping
the post-probe ctx recheck fails 1 — each alone, unlike round 1.
- BR-4 — not-addressed — The backoff ships and is reachable, but removing `wait = nextProbeWait(wait)`
from birthwatch.go:91 leaves the suite green; only the function is pinned.
- BR-5 — addressed — panebirth.Evidence now states that birth is the file's existence and that
each waiter stats it through its own seam; the referents (poller ModTime,
watch FileSize) match the code.
- BR-6 — addressed — SKILL.md now says the rows above the #288 ones predate -exit-wait, which
disambiguates the empty Hung column on the same-date rows.

### Raised

- **BR-7** [Minor] `duplicated-external-tool-knowledge` zellijLogPath re-declares zellij's tmp-dir layout that the probe already owns
  birthwatch.go:131 and probes/zellijbirthrace/main.go:100 both encode
  os.TempDir()/zellij-<uid>. Same module, one external fact, two sources; a
  zellij layout change drifts one of them. Go's cmd/internal boundary blocks
  direct reuse, so this is a note for the next time either is touched.

## Open findings

- **BR-1** [Minor] `plan-restates-executable-artifacts` Test cases enumerated in prose and the diff restated as full code blocks
- **BR-4** [Minor] `unbounded-probe-rounds` The birthUnknown loop has no cap, so a persistently failing probe repeats a machine-wide list-sessions every bound for the life of the launch
- **BR-7** [Minor] `duplicated-external-tool-knowledge` zellijLogPath re-declares zellij's tmp-dir layout that the probe already owns
