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
    - "n": 4
      timestamp: "2026-10-05T20:09:14-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Task 2.5 still enumerates its test cases as prose bullets (as do 1.4 and 3.1); Minor, non-blocking.
          round: 4
      findings:
        - id: BR-10
          severity: Important
          title: 'Broken host under a live agent: the advice says reboot, and reboot refuses with the same advice'
          detail: 'Outcome-based severity makes host-not-present blocking, so the first selectedSlot in rebootSlot (reboot.go:208) refuses before stopping the agent; reproduced in SlotWorld (AgentLive, RepairFixes=false, host broken/unreadable). Rule: advice must name an action that can succeed from the state that produced it. Let reboot''s first pass get past agent holds, or change the host hold text; fix the atlas and SKILL.md claim.'
          family: advice-names-reachable-action
          round: 4
        - id: BR-11
          severity: Important
          title: A refused SetAside leaves an empty pending saved-work entry that counts toward the 16-entry cap
          detail: slotsave.go creates the entry and pending manifest (line 91) before the setup-lock, agent and evidence checks (104/111/115). Each refusal leaves an entry, and 16 of them leave the slot saved-work-full until a one-year GC that is not yet built (M3). Run the refusable checks first, or remove the entry on refusal.
          family: failed-step-leaves-no-residue
          round: 4
        - id: BR-12
          severity: Minor
          title: setAside's agent-appeared and limit refusals are classified as a hand-off, not a hold
          detail: 'They are plain fmt.Errorf values that ClassifyConvergeError does not recognize, so the operator is told to ask the :0 agent instead of to reboot or clean up. This is the 2nd finding in this family: the rule is that every stop reason a converge step can return must be a typed error that ClassifyConvergeError maps to the same class PlanSlot would give it.'
          family: stop-reason-precision
          round: 4
        - id: BR-13
          severity: Minor
          title: couch --reconcile prints only the advice on a blocking failure, not the resources and plan
          detail: 'The outcome table promises "report, exit 1", and SlotReconcileError.Result carries the report. This is the 3rd finding in this family: the rule is that every failure path of a slot operation renders the observation it already holds next to the advice. Render Result.Observation/Plan on the error path.'
          family: error-surface-preserved
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-05T20:24:53-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Tasks 1.4, 2.5 and 3.1 still list test cases in prose. Minor, and does not block the gate.
          round: 5
        - id: BR-10
          disposition: addressed
          note: selectSlot(toleratesHolds) accepts only the live-agent hold, and the unknown-agent hold now advises looking again. TestRebootGetsPastAHoldItsAdviceNames fails when the tolerance is removed. Atlas and SKILL.md are updated.
          round: 5
        - id: BR-11
          disposition: addressed
          note: The limit, lock, agent and evidence checks now run before the entry and manifest are written (slotsave.go:93-111). assertNoSavedWork and the 16-entry count assertion pin it.
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: 'setAside still returns errAgentAppeared when the agent is unknown (running || !known), so ClassifyConvergeError reports the live-agent hold where PlanSlot gives the unknown-agent one. After BR-10, reboot''s first pass accepts that hold and the operator is told to reboot. The "no longer broken" refusal is still untyped and classified as a hand-off, though a rerun would converge. Rule: derive the converge-time hold reason from the same pure function PlanSlot''s setAside closure uses, and give the evidence-changed refusal a typed retryable error.'
          round: 5
        - id: BR-13
          disposition: addressed
          note: run.go renders SlotReconcileError.Result's observation and plan on the error path. TestReconcileCLIShowsTheReportOnABlockingFailure pins it.
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-10-05T21:06:31-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan prose for Tasks 1.4/2.5/3.1 unchanged in this window; still Minor, carried to close as Revision (u) states.
          round: 6
      findings:
        - id: BR-14
          severity: Important
          title: 'Printed restore command does not restore: mv into the recreated checkout nests the tree; missing on failure; Go-quoted not shell-quoted'
          detail: '2nd in family. slotsave.go:134 builds mv "<entry>/tree" "<path>" and provision.go:110 prints it after path was re-cloned/re-added, so mv yields path/tree (verified). The blocking path returns before SetAside is reported, and %q is Go quoting (ShellQuote exists). Rule: every printed command is executed in a test from the state that prints it. Make Restore move the recreated checkout aside then move the tree back via ShellQuote, run r.SetAside[0].Restore with sh -c in TestAcceptanceDirtySlot, and carry the set-aside list on failure too.'
          family: advice-names-reachable-action
          round: 6
        - id: BR-15
          severity: Important
          title: recoverReason has no case for slot-needs-:0 or reconcilable, so those rows read "no rule matched" and drop the reconciler's cause
          detail: '4th in family. Task 3.4 promised needs-zero with ReconcileAdvice. The DirectoryMissing text "repairing it is pair#387" is also stale. Rule: a SlotReport consumer carries the report''s known cause (stopFailure/ReconcileAdvice) into its text, and recoverReason is total over AllRecoverClasses, enforced by a test sweeping every class and rejecting the default text.'
          family: error-surface-preserved
          round: 6
        - id: BR-16
          severity: Important
          title: RecoverSlotClass holds a usable slot on any handoff stop, while SlotOutcome treats it as a warning
          detail: recoverplan.go:518 maps StopHandoff to needs-zero, but SlotOutcome blocks only when OutcomeSeverity is blocking. A resting branch checked out elsewhere with a working host loses its resume step (fixture "workspace needs :0" asserts this). It is the same decision made twice (ARCH-DRY, lessons BR-12). Derive needs-zero from SlotOutcome returning blocking and turn non-blocking handoff stops into a note. The Retried branch is unreachable in the report.
          family: report-agrees-with-caller-outcome
          round: 6
        - id: BR-17
          severity: Important
          title: 'Couch skill not updated despite Task 3.6 tick: still says "Nothing is ever deleted" and omits the new classes, hold and notes'
          detail: '2nd in family. cmd/internal/couchcmd/skills/couch/SKILL.md:125-127 contradicts retention collection and gives the recovering agent no procedure for slot-needs-:0/workspace-handoff, the reconcile step, or workspace-held/unknown. Rule: a diff adding a class/hold/note or changing a lifecycle statement sweeps every doc listing that vocabulary (README, atlas, skill) in the same window.'
          family: readme-tracks-user-surface
          round: 6
        - id: BR-18
          severity: Minor
          title: 'Task 3.4 ticked items without delivery or Revision: 150 ms/slot budget with 10 slots, row-advice sweep including reconcile'
          detail: '2nd in family. Rule: a ticked item that shipped differently gets a Revision line in the same commit that ticks it. The unmeasured budget is also an ARCH-CONSTRAINTS gap.'
          family: plan-tracks-implementation
          round: 6
        - id: BR-19
          severity: Minor
          title: Idle plus reconcilable replaces the whole decision, dropping notes computed earlier such as claims-stale
          detail: recoverplan.go:661 rebuilds recoverDecision; it should keep d.Notes.
          family: decision-preserves-notes
          round: 6
        - id: BR-20
          severity: Minor
          title: SavedWorkSince picks this run's entries by wall-clock window, not by the step's own entry
          detail: 'ARCH-ORDER: have setAside return its entry path in Executed. Separately, a hand-edited future saved_at is never collected (ARCH-SECURE).'
          family: identity-not-time-window
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-10-05T21:36:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Still carried to close by the plan's own note at line 1389; Minor, does not block.
          round: 7
        - id: BR-14
          disposition: addressed
          note: SavedWorkRestoreCommand is ShellQuoted and moves the path aside first; executed via sh -c in TestAcceptanceDirtySlot; failure path covered by TestAFailedRunStillNamesWhatItSetAside.
          round: 7
        - id: BR-15
          disposition: addressed
          note: Reconcilable and slot-needs-zero reason cases added, stale text fixed; coverage test rejects the default text per fixture class.
          round: 7
        - id: BR-16
          disposition: addressed
          note: RecoverSlotClass derives needs-zero via SlotOutcome; non-blocking handoff becomes the workspace-degraded note; TestRecoverSlotClassAgreesWithTheCallers sweeps the planner domain x agents.
          round: 7
        - id: BR-17
          disposition: addressed
          note: SKILL.md step 10 documents slot-needs-zero, workspace-handoff, reconcile and the three notes; deletion statement corrected; README and atlas also list workspace-degraded.
          round: 7
        - id: BR-18
          disposition: addressed
          note: TestAcceptanceReportBudgetTenSlots measures the budget; Revision (z) records that the reconcile row-advice item does not apply.
          round: 7
        - id: BR-19
          disposition: not-addressed
          note: Fix present at recoverplan.go:667, but reverting it leaves the Recover tests green; add a notes-preserved check for idle to reconcilable.
          round: 7
        - id: BR-20
          disposition: not-addressed
          note: Identity reporting addressed (setAsideDone; SavedWorkSince removed); the future saved_at guard has no test, as removing it keeps the suite green.
          round: 7
      findings:
        - id: BR-21
          severity: Minor
          title: A set-aside whose final manifest write fails after the rename is not reported
          detail: 'This is the 5th finding in family error-surface-preserved. Rule: report an effect when it happens, not when its bookkeeping completes. setAsideDone fires only after the complete-manifest write, so a tree already moved by rename goes unreported if that write fails. Fix: call setAsideDone right after the rename succeeds.'
          family: error-surface-preserved
          round: 7
      boundary: M3
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

## Round 4 — 2026-10-05T20:09:14-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Task 2.5 still enumerates its test cases as prose bullets (as do 1.4 and 3.1); Minor, non-blocking.

### Raised

- **BR-10** [Important] `advice-names-reachable-action` Broken host under a live agent: the advice says reboot, and reboot refuses with the same advice
  Outcome-based severity makes host-not-present blocking, so the first selectedSlot in rebootSlot (reboot.go:208) refuses before stopping the agent; reproduced in SlotWorld (AgentLive, RepairFixes=false, host broken/unreadable). Rule: advice must name an action that can succeed from the state that produced it. Let reboot's first pass get past agent holds, or change the host hold text; fix the atlas and SKILL.md claim.
- **BR-11** [Important] `failed-step-leaves-no-residue` A refused SetAside leaves an empty pending saved-work entry that counts toward the 16-entry cap
  slotsave.go creates the entry and pending manifest (line 91) before the setup-lock, agent and evidence checks (104/111/115). Each refusal leaves an entry, and 16 of them leave the slot saved-work-full until a one-year GC that is not yet built (M3). Run the refusable checks first, or remove the entry on refusal.
- **BR-12** [Minor] `stop-reason-precision` setAside's agent-appeared and limit refusals are classified as a hand-off, not a hold
  They are plain fmt.Errorf values that ClassifyConvergeError does not recognize, so the operator is told to ask the :0 agent instead of to reboot or clean up. This is the 2nd finding in this family: the rule is that every stop reason a converge step can return must be a typed error that ClassifyConvergeError maps to the same class PlanSlot would give it.
- **BR-13** [Minor] `error-surface-preserved` couch --reconcile prints only the advice on a blocking failure, not the resources and plan
  The outcome table promises "report, exit 1", and SlotReconcileError.Result carries the report. This is the 3rd finding in this family: the rule is that every failure path of a slot operation renders the observation it already holds next to the advice. Render Result.Observation/Plan on the error path.

## Round 5 — 2026-10-05T20:24:53-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Tasks 1.4, 2.5 and 3.1 still list test cases in prose. Minor, and does not block the gate.
- BR-10 — addressed — selectSlot(toleratesHolds) accepts only the live-agent hold, and the unknown-agent hold now advises looking again. TestRebootGetsPastAHoldItsAdviceNames fails when the tolerance is removed. Atlas and SKILL.md are updated.
- BR-11 — addressed — The limit, lock, agent and evidence checks now run before the entry and manifest are written (slotsave.go:93-111). assertNoSavedWork and the 16-entry count assertion pin it.
- BR-12 — not-addressed — setAside still returns errAgentAppeared when the agent is unknown (running || !known), so ClassifyConvergeError reports the live-agent hold where PlanSlot gives the unknown-agent one. After BR-10, reboot's first pass accepts that hold and the operator is told to reboot. The "no longer broken" refusal is still untyped and classified as a hand-off, though a rerun would converge. Rule: derive the converge-time hold reason from the same pure function PlanSlot's setAside closure uses, and give the evidence-changed refusal a typed retryable error.
- BR-13 — addressed — run.go renders SlotReconcileError.Result's observation and plan on the error path. TestReconcileCLIShowsTheReportOnABlockingFailure pins it.

## Round 6 — 2026-10-05T21:06:31-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan prose for Tasks 1.4/2.5/3.1 unchanged in this window; still Minor, carried to close as Revision (u) states.

### Raised

- **BR-14** [Important] `advice-names-reachable-action` Printed restore command does not restore: mv into the recreated checkout nests the tree; missing on failure; Go-quoted not shell-quoted
  2nd in family. slotsave.go:134 builds mv "<entry>/tree" "<path>" and provision.go:110 prints it after path was re-cloned/re-added, so mv yields path/tree (verified). The blocking path returns before SetAside is reported, and %q is Go quoting (ShellQuote exists). Rule: every printed command is executed in a test from the state that prints it. Make Restore move the recreated checkout aside then move the tree back via ShellQuote, run r.SetAside[0].Restore with sh -c in TestAcceptanceDirtySlot, and carry the set-aside list on failure too.
- **BR-15** [Important] `error-surface-preserved` recoverReason has no case for slot-needs-:0 or reconcilable, so those rows read "no rule matched" and drop the reconciler's cause
  4th in family. Task 3.4 promised needs-zero with ReconcileAdvice. The DirectoryMissing text "repairing it is pair#387" is also stale. Rule: a SlotReport consumer carries the report's known cause (stopFailure/ReconcileAdvice) into its text, and recoverReason is total over AllRecoverClasses, enforced by a test sweeping every class and rejecting the default text.
- **BR-16** [Important] `report-agrees-with-caller-outcome` RecoverSlotClass holds a usable slot on any handoff stop, while SlotOutcome treats it as a warning
  recoverplan.go:518 maps StopHandoff to needs-zero, but SlotOutcome blocks only when OutcomeSeverity is blocking. A resting branch checked out elsewhere with a working host loses its resume step (fixture "workspace needs :0" asserts this). It is the same decision made twice (ARCH-DRY, lessons BR-12). Derive needs-zero from SlotOutcome returning blocking and turn non-blocking handoff stops into a note. The Retried branch is unreachable in the report.
- **BR-17** [Important] `readme-tracks-user-surface` Couch skill not updated despite Task 3.6 tick: still says "Nothing is ever deleted" and omits the new classes, hold and notes
  2nd in family. cmd/internal/couchcmd/skills/couch/SKILL.md:125-127 contradicts retention collection and gives the recovering agent no procedure for slot-needs-:0/workspace-handoff, the reconcile step, or workspace-held/unknown. Rule: a diff adding a class/hold/note or changing a lifecycle statement sweeps every doc listing that vocabulary (README, atlas, skill) in the same window.
- **BR-18** [Minor] `plan-tracks-implementation` Task 3.4 ticked items without delivery or Revision: 150 ms/slot budget with 10 slots, row-advice sweep including reconcile
  2nd in family. Rule: a ticked item that shipped differently gets a Revision line in the same commit that ticks it. The unmeasured budget is also an ARCH-CONSTRAINTS gap.
- **BR-19** [Minor] `decision-preserves-notes` Idle plus reconcilable replaces the whole decision, dropping notes computed earlier such as claims-stale
  recoverplan.go:661 rebuilds recoverDecision; it should keep d.Notes.
- **BR-20** [Minor] `identity-not-time-window` SavedWorkSince picks this run's entries by wall-clock window, not by the step's own entry
  ARCH-ORDER: have setAside return its entry path in Executed. Separately, a hand-edited future saved_at is never collected (ARCH-SECURE).

## Round 7 — 2026-10-05T21:36:08-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Still carried to close by the plan's own note at line 1389; Minor, does not block.
- BR-14 — addressed — SavedWorkRestoreCommand is ShellQuoted and moves the path aside first; executed via sh -c in TestAcceptanceDirtySlot; failure path covered by TestAFailedRunStillNamesWhatItSetAside.
- BR-15 — addressed — Reconcilable and slot-needs-zero reason cases added, stale text fixed; coverage test rejects the default text per fixture class.
- BR-16 — addressed — RecoverSlotClass derives needs-zero via SlotOutcome; non-blocking handoff becomes the workspace-degraded note; TestRecoverSlotClassAgreesWithTheCallers sweeps the planner domain x agents.
- BR-17 — addressed — SKILL.md step 10 documents slot-needs-zero, workspace-handoff, reconcile and the three notes; deletion statement corrected; README and atlas also list workspace-degraded.
- BR-18 — addressed — TestAcceptanceReportBudgetTenSlots measures the budget; Revision (z) records that the reconcile row-advice item does not apply.
- BR-19 — not-addressed — Fix present at recoverplan.go:667, but reverting it leaves the Recover tests green; add a notes-preserved check for idle to reconcilable.
- BR-20 — not-addressed — Identity reporting addressed (setAsideDone; SavedWorkSince removed); the future saved_at guard has no test, as removing it keeps the suite green.

### Raised

- **BR-21** [Minor] `error-surface-preserved` A set-aside whose final manifest write fails after the rename is not reported
  This is the 5th finding in family error-surface-preserved. Rule: report an effect when it happens, not when its bookkeeping completes. setAsideDone fires only after the complete-manifest write, so a tree already moved by rename goes unreported if that write fails. Fix: call setAsideDone right after the rename succeeds.

## Open findings

- **BR-1** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
- **BR-8** [Minor] `error-surface-preserved` slotOfShowReference drops WorkspaceReferencePath's error, so --show repo:N of a missing slot shows the generic not-found message
- **BR-9** [Minor] `error-surface-preserved` --show still drops slot-resolution errors whenever thread resolution succeeds, or the physical path probe fails
- **BR-12** [Minor] `stop-reason-precision` setAside's agent-appeared and limit refusals are classified as a hand-off, not a hold
- **BR-19** [Minor] `decision-preserves-notes` Idle plus reconcilable replaces the whole decision, dropping notes computed earlier such as claims-stale
- **BR-20** [Minor] `identity-not-time-window` SavedWorkSince picks this run's entries by wall-clock window, not by the step's own entry
- **BR-21** [Minor] `error-surface-preserved` A set-aside whose final manifest write fails after the rename is not reported
