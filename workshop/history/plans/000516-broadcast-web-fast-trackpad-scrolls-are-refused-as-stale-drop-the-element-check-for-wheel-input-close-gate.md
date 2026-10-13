---
gate: boundary-review
issue: 516
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T19:24:51-07:00"
      dispatched: "2026-10-12T19:23:42-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Act and ErrStale doc comments still state the old wheel stale rule
          detail: 'The window updated Injector.Inject''s doc and the atlas but not the sibling statements of the same rule. act.go:54-56 says "for down, click and wheel: the injector refuses the act if the tab has another element there now (ErrStale)", which is now false for wheel. injector.go:40-41 defines ErrStale only as "the element the viewer aimed at isn''t at that point" and omits the wheel bounds case. Reword both to match injector.go:28-31.'
          family: doc-claims-track-rule-change
          round: 1
        - id: BR-2
          severity: Minor
          title: Done-when says the flick test fails on main; the Log shows it reproduces only intermittently
          detail: The deterministic red test is the RunInjectorContract wheel case (mutation-verified). Add a Revisions entry so Done-when 1 names the evidence that was actually delivered.
          family: done-when-matches-delivered-evidence
          round: 1
        - id: BR-3
          severity: Minor
          title: Hoisting type realCast moved TestDriveRealCast's doc comment off the test
          detail: 'live_drive_real_test.go around lines 90-96: the test''s doc and its go test -run line now sit above type realCast. Also drop the unused c binding and the _ = c at line 212.'
          family: doc-comment-follows-its-decl
          round: 1
        - id: BR-4
          severity: Minor
          title: errSpy.Inject locks the mutex twice in a row and shadows its arrival variable a
          detail: Merge the two critical sections into one. The busy counter cannot see 503s returned by the board queue before the injector; the px-injected assertion is what actually covers lost scroll.
          family: test-spy-hygiene
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T19:35:01-07:00"
      dispatched: "2026-10-12T19:33:59-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: act.go:55-56 and injector.go:40-41 now defer to Injector.Inject; onTarget (tab.go:188) scoped to down/click; atlas matches.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Revisions entry (close review) names the contract wheel case as the red-on-main evidence; mutation-verified red this round.
          round: 2
        - id: BR-3
          disposition: addressed
          note: TestDriveRealCast doc and go test -run line sit directly above the func again; the c binding and `_ = c` are gone.
          round: 2
        - id: BR-4
          disposition: addressed
          note: errSpy.Inject takes the lock once and the loop variable is renamed x; the px-injected assertion is the lost-scroll check.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: realCast fields c, ctrl, mem are set but never read
          detail: 'This is the 2nd finding in family test-spy-hygiene. The rule: every field and binding a test helper sets must have a reader. In this window the only instances are realCast.c, .ctrl and .mem (live_drive_real_test.go:32-40); spy, h and v are read. Drop the three fields or use them.'
          family: test-spy-hygiene
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 85a41d94315c0759eeec1f71a2f398adce87ef06
      blocked: false
---

# Gate ledger — pair#516 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T19:24:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-claims-track-rule-change` Act and ErrStale doc comments still state the old wheel stale rule
  The window updated Injector.Inject's doc and the atlas but not the sibling statements of the same rule. act.go:54-56 says "for down, click and wheel: the injector refuses the act if the tab has another element there now (ErrStale)", which is now false for wheel. injector.go:40-41 defines ErrStale only as "the element the viewer aimed at isn't at that point" and omits the wheel bounds case. Reword both to match injector.go:28-31.
- **BR-2** [Minor] `done-when-matches-delivered-evidence` Done-when says the flick test fails on main; the Log shows it reproduces only intermittently
  The deterministic red test is the RunInjectorContract wheel case (mutation-verified). Add a Revisions entry so Done-when 1 names the evidence that was actually delivered.
- **BR-3** [Minor] `doc-comment-follows-its-decl` Hoisting type realCast moved TestDriveRealCast's doc comment off the test
  live_drive_real_test.go around lines 90-96: the test's doc and its go test -run line now sit above type realCast. Also drop the unused c binding and the _ = c at line 212.
- **BR-4** [Minor] `test-spy-hygiene` errSpy.Inject locks the mutex twice in a row and shadows its arrival variable a
  Merge the two critical sections into one. The busy counter cannot see 503s returned by the board queue before the injector; the px-injected assertion is what actually covers lost scroll.

## Round 2 — 2026-10-12T19:35:01-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — act.go:55-56 and injector.go:40-41 now defer to Injector.Inject; onTarget (tab.go:188) scoped to down/click; atlas matches.
- BR-2 — addressed — Revisions entry (close review) names the contract wheel case as the red-on-main evidence; mutation-verified red this round.
- BR-3 — addressed — TestDriveRealCast doc and go test -run line sit directly above the func again; the c binding and `_ = c` are gone.
- BR-4 — addressed — errSpy.Inject takes the lock once and the loop variable is renamed x; the px-injected assertion is the lost-scroll check.

### Raised

- **BR-5** [Minor] `test-spy-hygiene` realCast fields c, ctrl, mem are set but never read
  This is the 2nd finding in family test-spy-hygiene. The rule: every field and binding a test helper sets must have a reader. In this window the only instances are realCast.c, .ctrl and .mem (live_drive_real_test.go:32-40); spy, h and v are read. Drop the three fields or use them.

## Open findings

- **BR-5** [Minor] `test-spy-hygiene` realCast fields c, ctrl, mem are set but never read
