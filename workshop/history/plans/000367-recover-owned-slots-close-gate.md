---
gate: boundary-review
issue: 367
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-04T02:11:59-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 2.4/2.5 enumerate individual test cases in prose instead of one strategy line per risky function
          detail: |-
            The poll loop over transport faults (table-driven fault injection) and the --resume/--reboot argv parser (fuzz or generated malformed argv) should each be one strategy line; the long case lists will be rewritten as code and go stale.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: plan-enumerates-test-cases
          round: 1
        - id: BR-2
          severity: Minor
          title: The live sdlc conformance test can be skipped at every close; nothing makes it run on a schedule
          detail: |-
            Chunk 3 step 4 allows accepting the skip. Record at each milestone close whether it ran unsandboxed, so drift between FakeFleetSDLC and real sdlc is detected (ARCH-MOCK).
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: live-conformance-cadence-unstated
          round: 1
        - id: BR-3
          severity: Minor
          title: A remote resume is recognized by Operation/Attempt/ContinuationID shape rather than an explicit origin
          detail: |-
            Task 2.3 clears reattach-failure for any resume with Attempt 0 and no ContinuationID. A future origin with that shape would match silently. The plan pins both sides with a test; recheck at the close review (ARCH-ORDER).
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: origin-recognized-by-field-shape
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-04T02:11:59-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Important
          title: Dependency-member claims are judged against the host branch, yielding an automatic restore-workspace for a peer-repo claim
          detail: Host pair:1 resting and clean plus dependency claim ariadne#000290 (the live golden shape) gives restore-workspace, Automatic true, steps resume + ask-agent-restore telling the agent to check out ariadne's issue branch (scratch-confirmed). A lost host claim plus any dependency claim similarly becomes conflict:claim-branch-mismatch instead of claim-likely-lost. Judge each claim's activity against its own member's branch.
          family: claim-judged-against-wrong-member
          round: 2
        - id: BR-5
          severity: Important
          title: A dangling claim on a present slot's missing dependency is dropped and the row reads idle
          detail: 'slotEvidenceOf reads s.dangling only when s.slot is nil, so a claim on worktree/pair-slot1/ariadne (dependency gone) vanishes from pair:1; scratch-confirmed: class idle, no claims, ignored 0. Always fold dangling refs into the slot''s claims.'
          family: unread-evidence-shown-as-absence
          round: 2
        - id: BR-6
          severity: Minor
          title: recoverCandidates swallows EnumerateSlotCandidates errors, so slot rows vanish without a recorded reason
          detail: recoverplan_source.go:163-168; append the error to the fleet observation's Error.
          family: unread-evidence-shown-as-absence
          round: 2
        - id: BR-7
          severity: Minor
          title: Candidate Missing is set from any stat error, so a permission error reads as directory-missing
          family: missing-conflates-unreadable
          round: 2
        - id: BR-8
          severity: Minor
          title: issueTerminalStatuses restates ariadne issue.cue's terminal set (ARCH-DRY)
          detail: It matches today; prefer sdlc emitting terminality on FleetIssue so Couch derives it.
          family: restated-vocabulary
          round: 2
        - id: BR-9
          severity: Minor
          title: consistent() excludes unread claim quality with claims present, which slotEvidenceOf can produce via a dependency read
          family: totality-domain-excludes-reachable-evidence
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-04T02:43:14-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Tasks 2.4/2.5 are M2 plan text and are unchanged; still Minor, revisit when M2 starts.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Issue Log line 214 records the live conformance run, sandboxed and unsandboxed, at M1; repeat at the M2 close.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: Task 2.3 is M2 scope; nothing in this window to recheck.
          round: 3
        - id: BR-4
          disposition: addressed
          note: judgeMemberClaim plus DepClaims; reverting to host-judged dependency claims fails 4 fixtures and TestClaimsAttachToTheirSlotPath (reviewer-run).
          round: 3
        - id: BR-5
          disposition: addressed
          note: Dangling claims always fold into their slot; reverting the dependency fold fails the present-slot-missing-dependency fixture (reviewer-run).
          round: 3
        - id: BR-6
          disposition: addressed
          note: Enumeration errors are appended to the fleet's Error; TestRecoverPlanCandidateErrorsAreRecordedNotMissing asserts "foreign slot path".
          round: 3
        - id: BR-7
          disposition: addressed
          note: Only fs.ErrNotExist is Missing; the permission-denied primary stays present, pinned by the same test.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: Deferred with a comment at issueTerminalStatuses pending sdlc emitting terminality; Minor.
          round: 3
        - id: BR-9
          disposition: addressed
          note: consistent() now excludes host claims only for absent and unsupported quality.
          round: 3
      findings:
        - id: BR-10
          severity: Important
          title: A landed host beside an active dependency claim is held as unidentified-work with a false reason
          detail: 'dependencyClaimDecision treats the host as at rest only on BranchResting, while rule 7 (landed) also treats a clean done-issue branch as at rest. Scratch-confirmed: pair:1 clean on done 000016-x plus ariadne#000290 on its own branch gives unidentified-work, "work with no claim or issue branch". Extract one hostAtRest predicate for rules 7/9 and dependencyClaimDecision, and add the fixture (ARCH-DRY).'
          family: host-at-rest-predicate-restated
          round: 3
        - id: BR-11
          severity: Important
          title: atlas/couch.md does not describe per-member claim judgment (DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency)
          detail: The landed rule got its atlas line in the same window, but d5052e19's new conflict fact, note, JSON field and the holding checkout in the restore message did not.
          family: docs-lag-new-vocabulary
          round: 3
        - id: BR-12
          severity: Minor
          title: The restore-workspace reason says "ask the slot's agent to restore it; no restore request" when the slot is dirty
          detail: recoverReason's new base text no longer names the claim and contradicts the resting-branch-dirty suffix.
          family: row-text-contradicts-decision
          round: 3
        - id: BR-13
          severity: Minor
          title: 'A host claim and a dependency claim both resting: the restore names only the host''s, and the dependency claim gets no note'
          detail: restoreWorkspaceDecision does not apply depNotes, so the second resting claim appears only in claims.dependency.
          family: evidence-dropped-from-next
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-04T03:00:48-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Tasks 2.4/2.5 are M2 scope and the plan text is unchanged; Minor, revisit at M2.
          round: 4
        - id: BR-3
          disposition: not-addressed
          note: Task 2.3 (remote resume origin) is M2 and not implemented in this window; recheck at M2 close.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: Explicitly deferred with a pointer comment at recoverplan.go:687; acceptable as Minor pending sdlc emitting terminality.
          round: 4
        - id: BR-10
          disposition: addressed
          note: hostAtRest shared by idle/landed/dependencyClaimDecision; landed-host fixture plus a totality invariant; making it resting-only fails both tests.
          round: 4
        - id: BR-11
          disposition: addressed
          note: atlas/couch.md now covers judgeMemberClaim, foldMemberClaims, DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency and the 3-arg RestoreWorkspaceMessage; all match the code.
          round: 4
        - id: BR-12
          disposition: addressed
          note: recoverReason derives the ask/refusal suffix from steps and notes; assertReasonMatchesDecision fails when the ask text is made unconditional.
          round: 4
        - id: BR-13
          disposition: addressed
          note: restoreWorkspaceDecision(e, depNotes(e)) on the host restore; the both-resting fixture fails when the notes are dropped.
          round: 4
      findings:
        - id: BR-14
          severity: Important
          title: hostAtRest and conflict:issue-terminal read git facts merged across checkouts, so dependency work counts as the host's
          detail: '2nd in family. Rule: any fact used to judge a member is that member''s own (claims were fixed in BR-4; dirty/unlanded/operation were not). Reproduced in scratch: resting host + ariadne#000290 claimed on 000290-y with ahead=2 gives unidentified-work "the host carries work no claim or issue branch names"; the same with dirty=2 gives agrees; a landed host with either gives conflict:issue-terminal. Fix: keep the host''s own facts for hostAtRest/issue-terminal/idle/landed, use the union only for the evidence list, add host-vs-dependency dimensions to the totality domain, and add member-level SetAhead/SetDirty/SetOperation to the fake plus fixtures.'
          family: claim-judged-against-wrong-member
          round: 4
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-04T03:22:04-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Tasks 2.4/2.5 are M2 scope; still Minor, carry to the M2 boundary.
          round: 5
        - id: BR-3
          disposition: not-addressed
          note: Task 2.3 remote-resume origin is M2 scope; recheck at M2 close.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: issueTerminalStatuses still restates issue.cue (comment marks it deferred); Minor.
          round: 5
        - id: BR-14
          disposition: addressed
          note: slotEvidenceOf fills host dimensions from member 0 only; judgeDependency/foldDependencies per member; union only in Evidence; SetMember* fixtures at recoverplan_test.go:292-333 plus TestDependencyFactsStayOnTheDependency pin the producer.
          round: 5
      findings:
        - id: BR-15
          severity: Minor
          title: Rule A's reboot guard treats only dependency operation and unknown verdict as slot-level, so dependency dirt or unread git allows reboot silently
          detail: A DepActive member with an unread dirty or ahead count allows reboot where the same host state holds reboot-unsafe-git, and dependency dirt no longer adds inspect-uncommitted-first. Fold slot-level dirt and git-unknown across members for withRuleA as DepOperation already is.
          family: slot-level-guard-reads-subset-of-member-facts
          round: 5
        - id: BR-16
          severity: Minor
          title: restoreWorkspaceDecision appends to extra twice, so the second append overwrites the first through a shared backing array
          detail: Correct today only because depNotes returns fresh len-1 slices; build notes explicitly instead.
          family: slice-alias-append
          round: 5
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-10-04T05:01:30-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: 'Overtaken by implementation: the poll loop and argv parser landed as table-driven tests, one strategy per row (messages_test.go:246, cli_test.go:152); the plan prose is now history.'
          round: 6
        - id: BR-3
          disposition: addressed
          note: The completion now carries an explicit origin (operationCompletion.remote); remoteOperationAddress returns false for nil remote, so a non-remote origin of the same shape clears nothing; TestRemoteResumeRecognitionIsPinnedBothSides pins both producer sides. Residual cleanup noted as Minor in prose.
          round: 6
      findings:
        - id: BR-17
          severity: Minor
          title: atlas/couch.md says every caller failure is unavailable; a no-identity request is invalid-request
          detail: 'This is the 2nd finding in family docs-lag-new-vocabulary. Rule: atlas text describing response codes should be checked against the plan''s latest Revisions entry for the same flow; here revision (d) and handleSlotOperation both say invalid-request.'
          family: docs-lag-new-vocabulary
          round: 6
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 7
      timestamp: "2026-10-05T10:45:56-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: Still restated at recoverplan.go:753; deferral is recorded in code (sdlc should emit terminality). Minor, non-blocking.
          round: 7
        - id: BR-15
          disposition: addressed
          note: memberTree computes per-member tree regardless of claim verdict; slotGitUnknown/slotDirty fold DepTree; tests at recoverplan_test.go:410,420 and invariant sweep :626-635.
          round: 7
        - id: BR-16
          disposition: addressed
          note: restoreWorkspaceDecision uses slices.Concat and withRuleA clones notes (recoverplan.go:679-697); aliasing impossible by construction.
          round: 7
        - id: BR-17
          disposition: addressed
          note: atlas/couch.md:403-405 now states a no-identity request is invalid-request and only a failed caller check is unavailable, matching revision (d).
          round: 7
      findings:
        - id: BR-18
          severity: Minor
          title: ThreadStore.withLock routes read-only stores to the waiting withPreviewLock, so a nested read-only caller would wait under a held lock
          detail: threadstore.go:157. da19d16e fixed the known nested sites by caller class; the read-only withLock path picks the wait implicitly. Consider making the wait an explicit parameter there.
          family: nested-read-waits-under-held-lock
          round: 7
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#367 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-04T02:11:59-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-enumerates-test-cases` Tasks 2.4/2.5 enumerate individual test cases in prose instead of one strategy line per risky function
  The poll loop over transport faults (table-driven fault injection) and the --resume/--reboot argv parser (fuzz or generated malformed argv) should each be one strategy line; the long case lists will be rewritten as code and go stale.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `live-conformance-cadence-unstated` The live sdlc conformance test can be skipped at every close; nothing makes it run on a schedule
  Chunk 3 step 4 allows accepting the skip. Record at each milestone close whether it ran unsandboxed, so drift between FakeFleetSDLC and real sdlc is detected (ARCH-MOCK).
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `origin-recognized-by-field-shape` A remote resume is recognized by Operation/Attempt/ContinuationID shape rather than an explicit origin
  Task 2.3 clears reattach-failure for any resume with Attempt 0 and no ContinuationID. A future origin with that shape would match silently. The plan pins both sides with a test; recheck at the close review (ARCH-ORDER).
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-10-04T02:11:59-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Important] `claim-judged-against-wrong-member` Dependency-member claims are judged against the host branch, yielding an automatic restore-workspace for a peer-repo claim
  Host pair:1 resting and clean plus dependency claim ariadne#000290 (the live golden shape) gives restore-workspace, Automatic true, steps resume + ask-agent-restore telling the agent to check out ariadne's issue branch (scratch-confirmed). A lost host claim plus any dependency claim similarly becomes conflict:claim-branch-mismatch instead of claim-likely-lost. Judge each claim's activity against its own member's branch.
- **BR-5** [Important] `unread-evidence-shown-as-absence` A dangling claim on a present slot's missing dependency is dropped and the row reads idle
  slotEvidenceOf reads s.dangling only when s.slot is nil, so a claim on worktree/pair-slot1/ariadne (dependency gone) vanishes from pair:1; scratch-confirmed: class idle, no claims, ignored 0. Always fold dangling refs into the slot's claims.
- **BR-6** [Minor] `unread-evidence-shown-as-absence` recoverCandidates swallows EnumerateSlotCandidates errors, so slot rows vanish without a recorded reason
  recoverplan_source.go:163-168; append the error to the fleet observation's Error.
- **BR-7** [Minor] `missing-conflates-unreadable` Candidate Missing is set from any stat error, so a permission error reads as directory-missing
- **BR-8** [Minor] `restated-vocabulary` issueTerminalStatuses restates ariadne issue.cue's terminal set (ARCH-DRY)
  It matches today; prefer sdlc emitting terminality on FleetIssue so Couch derives it.
- **BR-9** [Minor] `totality-domain-excludes-reachable-evidence` consistent() excludes unread claim quality with claims present, which slotEvidenceOf can produce via a dependency read

## Round 3 — 2026-10-04T02:43:14-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Tasks 2.4/2.5 are M2 plan text and are unchanged; still Minor, revisit when M2 starts.
- BR-2 — addressed — Issue Log line 214 records the live conformance run, sandboxed and unsandboxed, at M1; repeat at the M2 close.
- BR-3 — not-addressed — Task 2.3 is M2 scope; nothing in this window to recheck.
- BR-4 — addressed — judgeMemberClaim plus DepClaims; reverting to host-judged dependency claims fails 4 fixtures and TestClaimsAttachToTheirSlotPath (reviewer-run).
- BR-5 — addressed — Dangling claims always fold into their slot; reverting the dependency fold fails the present-slot-missing-dependency fixture (reviewer-run).
- BR-6 — addressed — Enumeration errors are appended to the fleet's Error; TestRecoverPlanCandidateErrorsAreRecordedNotMissing asserts "foreign slot path".
- BR-7 — addressed — Only fs.ErrNotExist is Missing; the permission-denied primary stays present, pinned by the same test.
- BR-8 — not-addressed — Deferred with a comment at issueTerminalStatuses pending sdlc emitting terminality; Minor.
- BR-9 — addressed — consistent() now excludes host claims only for absent and unsupported quality.

### Raised

- **BR-10** [Important] `host-at-rest-predicate-restated` A landed host beside an active dependency claim is held as unidentified-work with a false reason
  dependencyClaimDecision treats the host as at rest only on BranchResting, while rule 7 (landed) also treats a clean done-issue branch as at rest. Scratch-confirmed: pair:1 clean on done 000016-x plus ariadne#000290 on its own branch gives unidentified-work, "work with no claim or issue branch". Extract one hostAtRest predicate for rules 7/9 and dependencyClaimDecision, and add the fixture (ARCH-DRY).
- **BR-11** [Important] `docs-lag-new-vocabulary` atlas/couch.md does not describe per-member claim judgment (DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency)
  The landed rule got its atlas line in the same window, but d5052e19's new conflict fact, note, JSON field and the holding checkout in the restore message did not.
- **BR-12** [Minor] `row-text-contradicts-decision` The restore-workspace reason says "ask the slot's agent to restore it; no restore request" when the slot is dirty
  recoverReason's new base text no longer names the claim and contradicts the resting-branch-dirty suffix.
- **BR-13** [Minor] `evidence-dropped-from-next` A host claim and a dependency claim both resting: the restore names only the host's, and the dependency claim gets no note
  restoreWorkspaceDecision does not apply depNotes, so the second resting claim appears only in claims.dependency.

## Round 4 — 2026-10-04T03:00:48-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Tasks 2.4/2.5 are M2 scope and the plan text is unchanged; Minor, revisit at M2.
- BR-3 — not-addressed — Task 2.3 (remote resume origin) is M2 and not implemented in this window; recheck at M2 close.
- BR-8 — not-addressed — Explicitly deferred with a pointer comment at recoverplan.go:687; acceptable as Minor pending sdlc emitting terminality.
- BR-10 — addressed — hostAtRest shared by idle/landed/dependencyClaimDecision; landed-host fixture plus a totality invariant; making it resting-only fails both tests.
- BR-11 — addressed — atlas/couch.md now covers judgeMemberClaim, foldMemberClaims, DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency and the 3-arg RestoreWorkspaceMessage; all match the code.
- BR-12 — addressed — recoverReason derives the ask/refusal suffix from steps and notes; assertReasonMatchesDecision fails when the ask text is made unconditional.
- BR-13 — addressed — restoreWorkspaceDecision(e, depNotes(e)) on the host restore; the both-resting fixture fails when the notes are dropped.

### Raised

- **BR-14** [Important] `claim-judged-against-wrong-member` hostAtRest and conflict:issue-terminal read git facts merged across checkouts, so dependency work counts as the host's
  2nd in family. Rule: any fact used to judge a member is that member's own (claims were fixed in BR-4; dirty/unlanded/operation were not). Reproduced in scratch: resting host + ariadne#000290 claimed on 000290-y with ahead=2 gives unidentified-work "the host carries work no claim or issue branch names"; the same with dirty=2 gives agrees; a landed host with either gives conflict:issue-terminal. Fix: keep the host's own facts for hostAtRest/issue-terminal/idle/landed, use the union only for the evidence list, add host-vs-dependency dimensions to the totality domain, and add member-level SetAhead/SetDirty/SetOperation to the fake plus fixtures.

## Round 5 — 2026-10-04T03:22:04-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Tasks 2.4/2.5 are M2 scope; still Minor, carry to the M2 boundary.
- BR-3 — not-addressed — Task 2.3 remote-resume origin is M2 scope; recheck at M2 close.
- BR-8 — not-addressed — issueTerminalStatuses still restates issue.cue (comment marks it deferred); Minor.
- BR-14 — addressed — slotEvidenceOf fills host dimensions from member 0 only; judgeDependency/foldDependencies per member; union only in Evidence; SetMember* fixtures at recoverplan_test.go:292-333 plus TestDependencyFactsStayOnTheDependency pin the producer.

### Raised

- **BR-15** [Minor] `slot-level-guard-reads-subset-of-member-facts` Rule A's reboot guard treats only dependency operation and unknown verdict as slot-level, so dependency dirt or unread git allows reboot silently
  A DepActive member with an unread dirty or ahead count allows reboot where the same host state holds reboot-unsafe-git, and dependency dirt no longer adds inspect-uncommitted-first. Fold slot-level dirt and git-unknown across members for withRuleA as DepOperation already is.
- **BR-16** [Minor] `slice-alias-append` restoreWorkspaceDecision appends to extra twice, so the second append overwrites the first through a shared backing array
  Correct today only because depNotes returns fresh len-1 slices; build notes explicitly instead.

## Round 6 — 2026-10-04T05:01:30-07:00 (claude) — passed

### Disposed

- BR-1 — withdrawn — Overtaken by implementation: the poll loop and argv parser landed as table-driven tests, one strategy per row (messages_test.go:246, cli_test.go:152); the plan prose is now history.
- BR-3 — addressed — The completion now carries an explicit origin (operationCompletion.remote); remoteOperationAddress returns false for nil remote, so a non-remote origin of the same shape clears nothing; TestRemoteResumeRecognitionIsPinnedBothSides pins both producer sides. Residual cleanup noted as Minor in prose.

### Raised

- **BR-17** [Minor] `docs-lag-new-vocabulary` atlas/couch.md says every caller failure is unavailable; a no-identity request is invalid-request
  This is the 2nd finding in family docs-lag-new-vocabulary. Rule: atlas text describing response codes should be checked against the plan's latest Revisions entry for the same flow; here revision (d) and handleSlotOperation both say invalid-request.

## Round 7 — 2026-10-05T10:45:56-07:00 (claude) — passed

### Disposed

- BR-8 — not-addressed — Still restated at recoverplan.go:753; deferral is recorded in code (sdlc should emit terminality). Minor, non-blocking.
- BR-15 — addressed — memberTree computes per-member tree regardless of claim verdict; slotGitUnknown/slotDirty fold DepTree; tests at recoverplan_test.go:410,420 and invariant sweep :626-635.
- BR-16 — addressed — restoreWorkspaceDecision uses slices.Concat and withRuleA clones notes (recoverplan.go:679-697); aliasing impossible by construction.
- BR-17 — addressed — atlas/couch.md:403-405 now states a no-identity request is invalid-request and only a failed caller check is unavailable, matching revision (d).

### Raised

- **BR-18** [Minor] `nested-read-waits-under-held-lock` ThreadStore.withLock routes read-only stores to the waiting withPreviewLock, so a nested read-only caller would wait under a held lock
  threadstore.go:157. da19d16e fixed the known nested sites by caller class; the read-only withLock path picks the wait implicitly. Consider making the wait an explicit parameter there.

## Open findings

- **BR-8** [Minor] `restated-vocabulary` issueTerminalStatuses restates ariadne issue.cue's terminal set (ARCH-DRY)
- **BR-18** [Minor] `nested-read-waits-under-held-lock` ThreadStore.withLock routes read-only stores to the waiting withPreviewLock, so a nested read-only caller would wait under a held lock
