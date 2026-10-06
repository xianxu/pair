---
gate: boundary-review
issue: 387
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-05T18:32:11-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
          detail: (carried from plan-quality PQ-3, deferred to the boundary review)
          family: plan-enumerates-test-cases
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-05T18:32:11-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: --show builds the slot layout from conventionalSlot's guessed common dir and an unresolved user path, while observation compares against git's resolved paths
          detail: slotreport.go:63 uses conventionalSlot (common = primary/.git) and never runs EvalSymlinks on the ref. Git resolves symlinks in worktree list, --git-common-dir and --absolute-git-dir, so a primary reached through a symlink, or one with a gitfile .git, reads registration absent, branch broken (elsewhere) and host mismatched. Take the identity from Discover (RepoIdentity) and resolve the path; add a symlinked-primary fixture.
          family: slot-identity-from-git-not-convention
          round: 2
        - id: BR-3
          severity: Important
          title: observeDep marks a git-readable clone unreadable when its git dir is not <path>/.git
          detail: slotobserve.go:444 checks filepath.Dir(absolute-git-dir) != d.Path, so a gitfile-backed clone (linked worktree, separate-git-dir) becomes broken/unreadable, the reading that triggers set-aside. That violates R2. Use rev-parse --show-toplevel == d.Path and add a gitfile dependency test.
          family: broken-only-on-positive-evidence
          round: 2
        - id: BR-4
          severity: Important
          title: README not updated for couch --show repo:N, the slot resources and plan, and showing a slot with no thread
          detail: README.md:383 and :528 still describe --show as one thread by tag or path.
          family: readme-tracks-user-surface
          round: 2
        - id: BR-5
          severity: Minor
          title: A non-layer dependency hides every other declared dependency; plan step 1.3 still says the others are reported
          detail: slotobserve.go:405 replaces the list with the NotLayer alone. When it is outside the env, in-env missing dependencies go unseen and setup can read present. Record this in Revision (j).
          family: plan-tracks-implementation
          round: 2
        - id: BR-6
          severity: Minor
          title: A held weave lock makes a converged slot report a retryable stop
          detail: slotplan.go:340 stops on SubLockHeld whatever the setup state, so --show of a healthy slot reports a stop while an agent compiles. Settle this before M2 assigns it a severity.
          family: stop-only-when-work-pending
          round: 2
        - id: BR-7
          severity: Minor
          title: An unknown agent is reported with the live-agent hold reason
          family: stop-reason-precision
          round: 2
        - id: BR-8
          severity: Minor
          title: slotOfShowReference drops WorkspaceReferencePath's error, so --show repo:N of a missing slot shows the generic not-found message
          family: error-surface-preserved
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-05T18:44:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Tasks 2.5 and 3.1 still list test cases in prose; Minor, can be compressed when M2/M3 start.
          round: 3
        - id: BR-2
          disposition: addressed
          note: slotreport.go:68-108 resolves through retainedPhysicalPath and takes the identity from Discover; TestShowThroughASymlinkedFleetReadsGitsIdentity fails on the reverted code (registration absent, host mismatched, branch elsewhere).
          round: 3
        - id: BR-3
          disposition: addressed
          note: slotobserve.go:447 uses rev-parse --show-toplevel == path; TestObserveGitfileDependencyIsReadable fails on the reverted code (broken/unreadable).
          round: 3
        - id: BR-4
          disposition: addressed
          note: README.md:383 and :541-551 now document --show repo:N, the resource states and readings, the plan line, and a slot with no thread; this matches slotreport/couchcmd rendering.
          round: 3
        - id: BR-5
          disposition: addressed
          note: 'Revision (o) amends (j): the walk stops at a non-layer dependency, the consequence is bounded, and Task 1.3''s claim is retracted.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: slotplan.go:235 stops on a held lock only when setup is not present; the test "a held weave lock stops only pending setup work" checks both sides.
          round: 3
        - id: BR-7
          disposition: addressed
          note: StopReasonAgentUnknown is new and the test "an unknown agent holds with its own reason" pins it.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: 'The code now returns slotErr, but no test drives show of repo:N for a missing slot and asserts the "does not exist (existing: …)" message.'
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: --show still drops slot-resolution errors whenever thread resolution succeeds, or the physical path probe fails
          detail: 'This is the 2nd finding in family error-surface-preserved; do not fix only this site. Rule: every error on --show''s slot-resolution path (retainedPhysicalPath when the error is not not-exist, Discover, WorkspaceReferencePath) either fails the command or appears on the ShowResult as a slot-report error line. It never reduces the result to "not a slot". Today slotreport.go:73 returns (false, nil) on a path error, and operationdispatch.go:165-170 reads slotErr only when the thread lookup also failed. Make slotOfShowReference''s error always reach the result (e.g. a SlotReport carrying PlanError) and add one test per error source.'
          family: error-surface-preserved
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#387 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-05T18:32:11-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-10-05T18:32:11-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `slot-identity-from-git-not-convention` --show builds the slot layout from conventionalSlot's guessed common dir and an unresolved user path, while observation compares against git's resolved paths
  slotreport.go:63 uses conventionalSlot (common = primary/.git) and never runs EvalSymlinks on the ref. Git resolves symlinks in worktree list, --git-common-dir and --absolute-git-dir, so a primary reached through a symlink, or one with a gitfile .git, reads registration absent, branch broken (elsewhere) and host mismatched. Take the identity from Discover (RepoIdentity) and resolve the path; add a symlinked-primary fixture.
- **BR-3** [Important] `broken-only-on-positive-evidence` observeDep marks a git-readable clone unreadable when its git dir is not <path>/.git
  slotobserve.go:444 checks filepath.Dir(absolute-git-dir) != d.Path, so a gitfile-backed clone (linked worktree, separate-git-dir) becomes broken/unreadable, the reading that triggers set-aside. That violates R2. Use rev-parse --show-toplevel == d.Path and add a gitfile dependency test.
- **BR-4** [Important] `readme-tracks-user-surface` README not updated for couch --show repo:N, the slot resources and plan, and showing a slot with no thread
  README.md:383 and :528 still describe --show as one thread by tag or path.
- **BR-5** [Minor] `plan-tracks-implementation` A non-layer dependency hides every other declared dependency; plan step 1.3 still says the others are reported
  slotobserve.go:405 replaces the list with the NotLayer alone. When it is outside the env, in-env missing dependencies go unseen and setup can read present. Record this in Revision (j).
- **BR-6** [Minor] `stop-only-when-work-pending` A held weave lock makes a converged slot report a retryable stop
  slotplan.go:340 stops on SubLockHeld whatever the setup state, so --show of a healthy slot reports a stop while an agent compiles. Settle this before M2 assigns it a severity.
- **BR-7** [Minor] `stop-reason-precision` An unknown agent is reported with the live-agent hold reason
- **BR-8** [Minor] `error-surface-preserved` slotOfShowReference drops WorkspaceReferencePath's error, so --show repo:N of a missing slot shows the generic not-found message

## Round 3 — 2026-10-05T18:44:55-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Tasks 2.5 and 3.1 still list test cases in prose; Minor, can be compressed when M2/M3 start.
- BR-2 — addressed — slotreport.go:68-108 resolves through retainedPhysicalPath and takes the identity from Discover; TestShowThroughASymlinkedFleetReadsGitsIdentity fails on the reverted code (registration absent, host mismatched, branch elsewhere).
- BR-3 — addressed — slotobserve.go:447 uses rev-parse --show-toplevel == path; TestObserveGitfileDependencyIsReadable fails on the reverted code (broken/unreadable).
- BR-4 — addressed — README.md:383 and :541-551 now document --show repo:N, the resource states and readings, the plan line, and a slot with no thread; this matches slotreport/couchcmd rendering.
- BR-5 — addressed — Revision (o) amends (j): the walk stops at a non-layer dependency, the consequence is bounded, and Task 1.3's claim is retracted.
- BR-6 — addressed — slotplan.go:235 stops on a held lock only when setup is not present; the test "a held weave lock stops only pending setup work" checks both sides.
- BR-7 — addressed — StopReasonAgentUnknown is new and the test "an unknown agent holds with its own reason" pins it.
- BR-8 — not-addressed — The code now returns slotErr, but no test drives show of repo:N for a missing slot and asserts the "does not exist (existing: …)" message.

### Raised

- **BR-9** [Minor] `error-surface-preserved` --show still drops slot-resolution errors whenever thread resolution succeeds, or the physical path probe fails
  This is the 2nd finding in family error-surface-preserved; do not fix only this site. Rule: every error on --show's slot-resolution path (retainedPhysicalPath when the error is not not-exist, Discover, WorkspaceReferencePath) either fails the command or appears on the ShowResult as a slot-report error line. It never reduces the result to "not a slot". Today slotreport.go:73 returns (false, nil) on a path error, and operationdispatch.go:165-170 reads slotErr only when the thread lookup also failed. Make slotOfShowReference's error always reach the result (e.g. a SlotReport carrying PlanError) and add one test per error source.

## Open findings

- **BR-1** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
- **BR-8** [Minor] `error-surface-preserved` slotOfShowReference drops WorkspaceReferencePath's error, so --show repo:N of a missing slot shows the generic not-found message
- **BR-9** [Minor] `error-surface-preserved` --show still drops slot-resolution errors whenever thread resolution succeeds, or the physical path probe fails
