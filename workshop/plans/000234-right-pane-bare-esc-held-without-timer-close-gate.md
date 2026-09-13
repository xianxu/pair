---
gate: boundary-review
issue: 234
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-12T16:11:23-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 3 Step 5 converts one real-timer split test; two same-class siblings remain on the real timer
          detail: |-
            TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit (run_test.go:232) and the two "split across reads" rows in TestPumpStdinHandlesTerminalTabActions (run_test.go:180-181) hold a prefix across reads through pumpStdin exactly like the named test. Convert all three to the fake timer, or delete the first two as superseded by the Task 4 generated test (ARCH-PURPOSE: the class, not the instance).
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: enumerable-siblings-swept
          round: 1
        - id: BR-2
          severity: Minor
          title: Generated case (b) cannot tell a deadline flush from the EOF flush
          detail: |-
            With autoFire the fake tick and splitReader's EOF race in the select; both paths produce exactly write:<head>. Add the same timer.resets >= 1 assertion Task 1 Step 3 adds, so the case proves the deadline armed rather than sampling one interleaving (ARCH-ORDER at-review lens).
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: oracle-distinguishes-mechanism
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-12T16:11:23-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: Plan table and atlas name `EscapeTimer`, but run.go still declares `RenameTimer` with no doc comment
          detail: Core concepts row and Task 3 Step 1 promise the interface rename plus a doc comment; atlas/architecture.md:455 already cites `EscapeTimer`. Only realEscapeTimer and the test fake were renamed. Rename the interface (two lines) or revise plan + atlas to the real name.
          family: artifact-names-real-symbol
          round: 2
        - id: BR-4
          severity: Important
          title: Done-when bullets 1-2 are pane-level claims with no live evidence; the manual Plan step is unticked
          detail: Pump tests prove the mux forwards after the deadline, not that the right pane's nvim sees one ESC. Carry the operator's smoke test (fresh split, nvim ESC / ESC j / Alt+j / Alt+t) in --verified rather than waiving the unchecked step.
          family: operator-smoke-test-before-done
          round: 2
        - id: BR-5
          severity: Minor
          title: Generated case (b) and the bare-ESC regression prove the arm, not the expiry flush
          detail: Scratch revert of the expiry branch leaves all 546 generated cases and TestBareEscapeIsForwardedAfterTheDeadline green; only the two gated tests go red. Gate the EOF behind the observed write, or rename the case to claim what it proves.
          family: oracle-distinguishes-mechanism
          round: 2
        - id: BR-6
          severity: Minor
          title: 35 ms and nvim's 50 ms ttimeoutlen add rather than nest; the "never slower than it budgets for" sentence is wrong
          detail: shortcut.go:390-391 and the plan's latency budget. The number is fine; state the combined worst case (~85 ms) instead.
          family: unsupported-performance-claim
          round: 2
        - id: BR-7
          severity: Minor
          title: Torn SGR mouse prefix now flushes raw after 35 ms with no test pinning that deliberate choice
          detail: The generated table covers chord prefixes only; one explicit mouse-prefix case would document the plan's "any held prefix" decision.
          family: enumerable-siblings-swept
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-12T16:31:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Leaky split test deleted; tab-actions rows use beforeDeadline(); every remaining pumpStdin caller is single-read or rename-mode with empty pending, so no plain prefix crosses reads on the real timer.
          round: 3
        - id: BR-2
          disposition: addressed
          note: forwardedOnTheDeadline asserts timer.resets >= 1 and gates EOF behind the observed write.
          round: 3
        - id: BR-3
          disposition: addressed
          note: run.go:354 declares EscapeTimer with the planned doc comment; RenameTimer greps to zero in Go source.
          round: 3
        - id: BR-4
          disposition: addressed
          note: 'Reproduced independently: escsmoke 6/6 on a head build, 3/6 on a base-commit control, failing the same three steps. Operator in-zellij confirmation remains requested and is declared as not run in the close text.'
          round: 3
        - id: BR-5
          disposition: addressed
          note: Scratch revert of the expiry branch now fails the generated deadline-first case, the mouse test, ESC-then-j and the real-timer test; the bare-ESC test's comment states the arm is its evidence.
          round: 3
        - id: BR-6
          disposition: addressed
          note: shortcut.go:390-393 and the plan's latency budget both state the additive ~85 ms worst case.
          round: 3
        - id: BR-7
          disposition: addressed
          note: TestATornMousePrefixMeetsTheSameDeadline covers both sides of the deadline and goes red under either scratch revert.
          round: 3
      findings:
        - id: BR-8
          severity: Minor
          title: Plan lacks a Revisions entry for the probe, the gated case (b) helper and the mouse test added after approval
          detail: CLAUDE.md asks for an appended Revisions section when a plan artifact changes mid-stream; the Core concepts and Task lists predate probes/escsmoke and the round-1 test changes.
          family: plan-records-post-approval-delta
          round: 3
      blocked: false
---

# Gate ledger — pair#234 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T16:11:23-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `enumerable-siblings-swept` Task 3 Step 5 converts one real-timer split test; two same-class siblings remain on the real timer
  TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit (run_test.go:232) and the two "split across reads" rows in TestPumpStdinHandlesTerminalTabActions (run_test.go:180-181) hold a prefix across reads through pumpStdin exactly like the named test. Convert all three to the fake timer, or delete the first two as superseded by the Task 4 generated test (ARCH-PURPOSE: the class, not the instance).
  (carried from plan-quality PQ-4, deferred to the boundary review)
- **BR-2** [Minor] `oracle-distinguishes-mechanism` Generated case (b) cannot tell a deadline flush from the EOF flush
  With autoFire the fake tick and splitReader's EOF race in the select; both paths produce exactly write:<head>. Add the same timer.resets >= 1 assertion Task 1 Step 3 adds, so the case proves the deadline armed rather than sampling one interleaving (ARCH-ORDER at-review lens).
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-09-12T16:11:23-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `artifact-names-real-symbol` Plan table and atlas name `EscapeTimer`, but run.go still declares `RenameTimer` with no doc comment
  Core concepts row and Task 3 Step 1 promise the interface rename plus a doc comment; atlas/architecture.md:455 already cites `EscapeTimer`. Only realEscapeTimer and the test fake were renamed. Rename the interface (two lines) or revise plan + atlas to the real name.
- **BR-4** [Important] `operator-smoke-test-before-done` Done-when bullets 1-2 are pane-level claims with no live evidence; the manual Plan step is unticked
  Pump tests prove the mux forwards after the deadline, not that the right pane's nvim sees one ESC. Carry the operator's smoke test (fresh split, nvim ESC / ESC j / Alt+j / Alt+t) in --verified rather than waiving the unchecked step.
- **BR-5** [Minor] `oracle-distinguishes-mechanism` Generated case (b) and the bare-ESC regression prove the arm, not the expiry flush
  Scratch revert of the expiry branch leaves all 546 generated cases and TestBareEscapeIsForwardedAfterTheDeadline green; only the two gated tests go red. Gate the EOF behind the observed write, or rename the case to claim what it proves.
- **BR-6** [Minor] `unsupported-performance-claim` 35 ms and nvim's 50 ms ttimeoutlen add rather than nest; the "never slower than it budgets for" sentence is wrong
  shortcut.go:390-391 and the plan's latency budget. The number is fine; state the combined worst case (~85 ms) instead.
- **BR-7** [Minor] `enumerable-siblings-swept` Torn SGR mouse prefix now flushes raw after 35 ms with no test pinning that deliberate choice
  The generated table covers chord prefixes only; one explicit mouse-prefix case would document the plan's "any held prefix" decision.

## Round 3 — 2026-09-12T16:31:34-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Leaky split test deleted; tab-actions rows use beforeDeadline(); every remaining pumpStdin caller is single-read or rename-mode with empty pending, so no plain prefix crosses reads on the real timer.
- BR-2 — addressed — forwardedOnTheDeadline asserts timer.resets >= 1 and gates EOF behind the observed write.
- BR-3 — addressed — run.go:354 declares EscapeTimer with the planned doc comment; RenameTimer greps to zero in Go source.
- BR-4 — addressed — Reproduced independently: escsmoke 6/6 on a head build, 3/6 on a base-commit control, failing the same three steps. Operator in-zellij confirmation remains requested and is declared as not run in the close text.
- BR-5 — addressed — Scratch revert of the expiry branch now fails the generated deadline-first case, the mouse test, ESC-then-j and the real-timer test; the bare-ESC test's comment states the arm is its evidence.
- BR-6 — addressed — shortcut.go:390-393 and the plan's latency budget both state the additive ~85 ms worst case.
- BR-7 — addressed — TestATornMousePrefixMeetsTheSameDeadline covers both sides of the deadline and goes red under either scratch revert.

### Raised

- **BR-8** [Minor] `plan-records-post-approval-delta` Plan lacks a Revisions entry for the probe, the gated case (b) helper and the mouse test added after approval
  CLAUDE.md asks for an appended Revisions section when a plan artifact changes mid-stream; the Core concepts and Task lists predate probes/escsmoke and the round-1 test changes.

## Open findings

- **BR-8** [Minor] `plan-records-post-approval-delta` Plan lacks a Revisions entry for the probe, the gated case (b) helper and the mouse test added after approval
