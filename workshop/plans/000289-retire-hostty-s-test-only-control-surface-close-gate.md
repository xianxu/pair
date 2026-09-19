---
gate: boundary-review
issue: 289
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T18:16:14-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Count/date claims in new prose disagree with the tree (a week vs three days; ten vs nine; five more vs six)
          detail: 'All instances in window: deadsymbols_test.go:23 and workshop/lessons.md:10 say "for a week", but #255 M3 (f32bb4cf) is 2026-09-15 and the issue Problem itself says three days. Issue 000289 Plan line 86 says "ten hostty orphans"; replaying the guard on base flags nine. Issue 000281 line 22 says "setRegion and five more in reserve.go"; reserve.go holds seven sequences, so six more.'
          family: prose-count-claim-unmeasured
          round: 1
        - id: BR-2
          severity: Minor
          title: Open issues 217 and 241 still cite deleted hostty symbols as live mechanism
          detail: 000217:36 claims hostty.ResetInteractiveModes includes 1004 in teardown, which was false after 255 M3 and the symbol is now deleted. 000241:42 directs generalising PrivateModes in hostty, which contradicts the new package doc. 000207 mentions are measured history and fine. A one-line note per open issue closes the class.
          family: stale-prose-names-retired-surface
          round: 1
        - id: BR-3
          severity: Minor
          title: Dead-symbol guard skips fake files as declarations but counts them as production references
          detail: productionIdentifierCounts still counts fake.go and *_fake.go, so a production symbol reached only by fakes reads as live. Excluding fakes on the reference side surfaces couchcore strings.go joinArgs, used only by git_fake.go and runner_fake.go. Pre-existing, but this window widened the fake-skip rule; hostty has no hidden orphan today.
          family: dead-symbol-guard-fake-asymmetry
          round: 1
        - id: BR-4
          severity: Minor
          title: The guard's new const/var, fake.go and iota-zero rules have no fixture test of their own
          detail: isIotaZero is a pure function and productionDeclarations could run over a testdata dir. Without that, a regression that widens the iota or fake skip silently hides orphans. Red-before-fix was shown manually only.
          family: guard-rules-unpinned
          round: 1
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#289 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T18:16:14-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `prose-count-claim-unmeasured` Count/date claims in new prose disagree with the tree (a week vs three days; ten vs nine; five more vs six)
  All instances in window: deadsymbols_test.go:23 and workshop/lessons.md:10 say "for a week", but #255 M3 (f32bb4cf) is 2026-09-15 and the issue Problem itself says three days. Issue 000289 Plan line 86 says "ten hostty orphans"; replaying the guard on base flags nine. Issue 000281 line 22 says "setRegion and five more in reserve.go"; reserve.go holds seven sequences, so six more.
- **BR-2** [Minor] `stale-prose-names-retired-surface` Open issues 217 and 241 still cite deleted hostty symbols as live mechanism
  000217:36 claims hostty.ResetInteractiveModes includes 1004 in teardown, which was false after 255 M3 and the symbol is now deleted. 000241:42 directs generalising PrivateModes in hostty, which contradicts the new package doc. 000207 mentions are measured history and fine. A one-line note per open issue closes the class.
- **BR-3** [Minor] `dead-symbol-guard-fake-asymmetry` Dead-symbol guard skips fake files as declarations but counts them as production references
  productionIdentifierCounts still counts fake.go and *_fake.go, so a production symbol reached only by fakes reads as live. Excluding fakes on the reference side surfaces couchcore strings.go joinArgs, used only by git_fake.go and runner_fake.go. Pre-existing, but this window widened the fake-skip rule; hostty has no hidden orphan today.
- **BR-4** [Minor] `guard-rules-unpinned` The guard's new const/var, fake.go and iota-zero rules have no fixture test of their own
  isIotaZero is a pure function and productionDeclarations could run over a testdata dir. Without that, a regression that widens the iota or fake skip silently hides orphans. Red-before-fix was shown manually only.

## Open findings

- **BR-1** [Minor] `prose-count-claim-unmeasured` Count/date claims in new prose disagree with the tree (a week vs three days; ten vs nine; five more vs six)
- **BR-2** [Minor] `stale-prose-names-retired-surface` Open issues 217 and 241 still cite deleted hostty symbols as live mechanism
- **BR-3** [Minor] `dead-symbol-guard-fake-asymmetry` Dead-symbol guard skips fake files as declarations but counts them as production references
- **BR-4** [Minor] `guard-rules-unpinned` The guard's new const/var, fake.go and iota-zero rules have no fixture test of their own
