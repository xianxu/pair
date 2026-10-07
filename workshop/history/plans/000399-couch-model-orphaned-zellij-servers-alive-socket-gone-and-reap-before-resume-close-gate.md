---
gate: boundary-review
issue: 399
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-06T22:27:27-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
          detail: (carried from plan-quality PQ-5, deferred to the boundary review)
          family: test-strategy-not-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-06T22:27:27-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Startup orphan refusal says "kill PID; its agent goes with it", which the issue shows is false
          detail: couch.go:500 tells the operator to kill only the server. On 2026-10-06 wrap ignored SIGTERM and title helpers were left with PPID 1. Tell them to stop the whole process tree (or point to reap) instead.
          family: operator-advice-contradicts-evidence
          round: 2
        - id: BR-3
          severity: Minor
          title: sessionOwnerWord and AgentRunning do not handle the new orphaned values
          detail: launch_existing.go:466 prints "unknown" for SessionOwnerOrphaned; slotobserve.go:18 returns not-known for AgentOrphaned, though an orphan is known to be running.
          family: vocabulary-consumer-missing-member
          round: 2
        - id: BR-4
          severity: Minor
          title: ScopeHoldsOrphanedThread also requires Orphan != nil, so a nil-server orphan row reads as debris
          family: derived-field-guard-redundant
          round: 2
        - id: BR-5
          severity: Minor
          title: Ranking of orphaned vs unusable-unknown agent evidence is written in two places (slotreport rank map, recoverplan loop)
          family: single-ranking-source
          round: 2
        - id: BR-6
          severity: Minor
          title: An orphan plus a new live server for the same name reads Present, hiding the orphan
          family: orphan-shadowed-by-live-same-name
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-06T22:40:36-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: FuzzParseServerProcesses with a one-line Strategy comment, session_servers_test.go:28-46.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: couch.go:1281 pkill -P reaches only direct children, and the KILL follow-up runs after kill <server>, so SIGTERM-ignoring children have already moved to PID 1 and it matches nothing. Same failure as 2026-10-06; the test checks strings only.
          round: 3
        - id: BR-3
          disposition: addressed
          note: sessionOwnerWord and AgentRunning handle the orphan (TestVocabularyConsumersKnowTheOrphan); other SessionOwner switches fail closed by default.
          round: 3
        - id: BR-4
          disposition: addressed
          note: startup.go:125 keys on the reason only; TestAnOrphanRowWithoutItsServerStillBlocksStartup.
          round: 3
        - id: BR-5
          disposition: addressed
          note: One agentRank at recoverplan.go:1481 used by slotreport and slotEvidenceOf.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Contested outranks list-sessions live (sessionevidence.go:121, TestAContestedServerNameIsNeverPresent); Probe treats len!=1 as ambiguous.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: ServerState is three booleans (Orphaned/Unresolved/Contested) with unwritten legal combinations
          detail: Contested implies Unresolved and Orphaned+Unresolved can be represented; a tagged enum (Reachable, Orphaned, UnknownSocket, Contested) would rule the bad combinations out (ARCH-ORDER/ARCH-SECURE).
          family: boolean-constellation-not-enum
          round: 3
        - id: BR-8
          severity: Minor
          title: A server still starting (in ps, socket not yet bound) reads as orphaned on a single snapshot
          detail: Harmless in M1, where it only causes a refusal; M2 reap must require two observations or a minimum process age and re-check the identity before signalling, or it can kill a session that is starting (ARCH-ORDER).
          family: point-observation-as-settled-state
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-06T23:00:42-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: couch.go orphanStartRefusal now lists the tree, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident fails if the order changes or the old advice returns.
          round: 4
        - id: BR-7
          disposition: addressed
          note: ServerState carries one ServerVerdict enum (zero value Unresolved); every consumer uses hasServer or a verdict comparison; the table test pins all four verdicts.
          round: 4
        - id: BR-8
          disposition: addressed
          note: Harmless in M1 (it only refuses); the plan's 2026-10-07 Revisions requires two snapshots or a minimum process age, an identity re-read before each signal, and a just-started-server fixture in M2.
          round: 4
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-10-07T00:20:01-07:00"
      agent: claude
      findings:
        - id: BR-9
          severity: Important
          title: 'reduceOperationResult has no success arm for reap/recover: the confirmation frame lingers, then an error notice appears'
          detail: 'Reproduced with a scratch test: a confirmed recover succeeds and the frame stays at depth 3; the next inventory drops it with an error-level notice "thread action is no longer applicable" (menu.go:1703-1724). This is the 2nd finding in this family. The rule to fix: every per-operation switch in couchtty/couchcmd derives from an Operations() declaration or is guarded by a sweep test over Operations() that fails on a missing arm. About 12 hand-maintained restatements of the slot-op set exist (cli.go, messages.go, message_service.go, protocol.go, run.go, slot_operation.go, menu_slot.go, menu.go).'
          family: vocabulary-consumer-missing-member
          round: 5
        - id: BR-10
          severity: Important
          title: OSOrphanReaper skips the tag's title-poller/nvim pidfile reapers that the plan's ARCH-ORDER listed
          detail: pair title is started detached by the launcher (osruntime.go:382), not under the zellij server, so the tree kill never reaches it. That is the PPID-1 residue the Done-when forbids and that M3 step 4 checks for. Call KillTitlePoller and ReapNvim for the tag after the tree kill, or revise the plan; also correct the atlas wording.
          family: declared-order-step-dropped
          round: 5
        - id: BR-11
          severity: Important
          title: orphanStartRefusal still prints manual ps/kill steps despite the M1-review revision requiring advice derived from the reap mechanism
          detail: couch.go:1279-1300. The sibling refusal at couch.go:494 already gives a working switcher gesture ("run couch in another repository, select it, Tab → reboot"). The orphan case should name Tab → recover (or couch --recover <ref> --confirm), or a plan revision should explain why manual steps must stay.
          family: advice-restates-mechanism
          round: 5
        - id: BR-12
          severity: Important
          title: README does not document couch --reap / couch --recover or the switcher recover action
          detail: README.md:388-389 and :440 list --resume/--reboot only; usageWith and the atlas were updated in this range, but the README was not.
          family: docs-surface-missing
          round: 5
        - id: BR-13
          severity: Minor
          title: Plan says reap is reached only through recover, but the switcher lists reap as its own entry too
          detail: menuRowActions returns recover plus ActorActions (["reap"]). The Log records this as deliberate; add a Revisions entry so the plan matches the code.
          family: plan-drift-unrevised
          round: 5
        - id: BR-14
          severity: Minor
          title: ActorActions doc comment still says "the two actor operations, resume and reboot"
          family: stale-doc-comment
          round: 5
        - id: BR-15
          severity: Minor
          title: OSProcessTable.Snapshot reads ppid from ps and start identity from a later sysctl, so a pid recycled between the two reads is planned under the wrong parent
          detail: 'This is the 2nd finding in this family; it is very unlikely in practice. The rule: a process fact used to authorize a signal comes from one atomic read. Here that means reading ppid and start time from the same kinfo_proc.'
          family: point-observation-as-settled-state
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-10-07T00:47:45-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: menu.go default arm closes any confirmation on success; TestReapSuccessClosesItsConfirmation drives the real key path and fails without the arm.
          round: 6
        - id: BR-10
          disposition: addressed
          note: OSOrphanReaper calls ReapTagHelpers (title pidfile + nvim) after the tree; TestReapTagHelpersEndsTheHelpersOutsideTheTree. The atlas and the code comment's reason are raised as a new Minor.
          round: 6
        - id: BR-11
          disposition: addressed
          note: couch.go orphanStartRefusal names Tab → recover and the inspect step, with no kill recipe; TestOrphanRefusalNamesTheReapMechanism asserts this.
          round: 6
        - id: BR-12
          disposition: addressed
          note: README.md 390-391 (CLI), 420-428 (orphaned threads), 863-867 (Tab → recover) match the code.
          round: 6
        - id: BR-13
          disposition: not-addressed
          note: No Revisions entry added; plan line 472 still says reap is not its own menu entry.
          round: 6
        - id: BR-14
          disposition: not-addressed
          note: actor_actions.go:47 unchanged.
          round: 6
        - id: BR-15
          disposition: not-addressed
          note: session_reap.go Snapshot still takes ppid from ps and identity from a later read.
          round: 6
      findings:
        - id: BR-16
          severity: Minor
          title: Atlas omits reap's helper step, and the code and lessons say the poller is outside the tree because of Setsid, which is false for Couch threads
          detail: 'This is the 2nd finding in family stale-doc-comment. The rule: when a mechanism changes, every prose description of it (atlas, code comment, lessons) changes in the same commit. Here: atlas/couch.md''s OSOrphanReaper sentence lacks the helper step and still lists pair title among the server''s children. lifecycle.go:515 and lessons.md say the poller is outside the tree because it is spawned with Setsid, but sidecarProcessAttributes returns nil for Couch-launched Pair. The real reason is that the launcher, not the zellij server, is its parent.'
          family: stale-doc-comment
          round: 6
        - id: BR-17
          severity: Minor
          title: editorPathsOf duplicates the quit path's inline editor-path literal and no test checks they stay equal
          detail: lifecycle.go:196-199 vs 297-302 (ARCH-DRY). The inventory check is why the literal stays inline; a test asserting launcherCleanupOps.editorPaths equals editorPathsOf(paths) would keep the two from drifting.
          family: duplicate-derivation
          round: 6
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 7
      timestamp: "2026-10-07T12:38:06-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: Plan revision "2026-10-07 — reap is its own switcher entry too (M2)" (17124a86) records [recover, reap].
          round: 7
        - id: BR-14
          disposition: addressed
          note: actor_actions.go ActorActions comment now names resume, reboot and reap, and the live-orphan exception.
          round: 7
        - id: BR-15
          disposition: not-addressed
          note: session_reap.go OSProcessTable.Snapshot still takes ppid from ps and start identity from a later per-pid sysctl.
          round: 7
        - id: BR-16
          disposition: not-addressed
          note: atlas/couch.md about line 2197 still skips ReapTagHelpers and lists pair title as a child; lifecycle.go:516 and lessons.md:591 still credit Setsid.
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: The only test (session_reap_test.go:222) compares against editorPathsOf itself; nothing compares it with the quit path's inline literal at lifecycle.go:196.
          round: 7
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#399 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T22:27:27-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-strategy-not-enumeration` Test cases are spelled out as prose and code; use one strategy line per risky function (fuzz ParseServerProcesses over ps output)
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-10-06T22:27:27-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `operator-advice-contradicts-evidence` Startup orphan refusal says "kill PID; its agent goes with it", which the issue shows is false
  couch.go:500 tells the operator to kill only the server. On 2026-10-06 wrap ignored SIGTERM and title helpers were left with PPID 1. Tell them to stop the whole process tree (or point to reap) instead.
- **BR-3** [Minor] `vocabulary-consumer-missing-member` sessionOwnerWord and AgentRunning do not handle the new orphaned values
  launch_existing.go:466 prints "unknown" for SessionOwnerOrphaned; slotobserve.go:18 returns not-known for AgentOrphaned, though an orphan is known to be running.
- **BR-4** [Minor] `derived-field-guard-redundant` ScopeHoldsOrphanedThread also requires Orphan != nil, so a nil-server orphan row reads as debris
- **BR-5** [Minor] `single-ranking-source` Ranking of orphaned vs unusable-unknown agent evidence is written in two places (slotreport rank map, recoverplan loop)
- **BR-6** [Minor] `orphan-shadowed-by-live-same-name` An orphan plus a new live server for the same name reads Present, hiding the orphan

## Round 3 — 2026-10-06T22:40:36-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — FuzzParseServerProcesses with a one-line Strategy comment, session_servers_test.go:28-46.
- BR-2 — not-addressed — couch.go:1281 pkill -P reaches only direct children, and the KILL follow-up runs after kill <server>, so SIGTERM-ignoring children have already moved to PID 1 and it matches nothing. Same failure as 2026-10-06; the test checks strings only.
- BR-3 — addressed — sessionOwnerWord and AgentRunning handle the orphan (TestVocabularyConsumersKnowTheOrphan); other SessionOwner switches fail closed by default.
- BR-4 — addressed — startup.go:125 keys on the reason only; TestAnOrphanRowWithoutItsServerStillBlocksStartup.
- BR-5 — addressed — One agentRank at recoverplan.go:1481 used by slotreport and slotEvidenceOf.
- BR-6 — addressed — Contested outranks list-sessions live (sessionevidence.go:121, TestAContestedServerNameIsNeverPresent); Probe treats len!=1 as ambiguous.

### Raised

- **BR-7** [Minor] `boolean-constellation-not-enum` ServerState is three booleans (Orphaned/Unresolved/Contested) with unwritten legal combinations
  Contested implies Unresolved and Orphaned+Unresolved can be represented; a tagged enum (Reachable, Orphaned, UnknownSocket, Contested) would rule the bad combinations out (ARCH-ORDER/ARCH-SECURE).
- **BR-8** [Minor] `point-observation-as-settled-state` A server still starting (in ps, socket not yet bound) reads as orphaned on a single snapshot
  Harmless in M1, where it only causes a refusal; M2 reap must require two observations or a minimum process age and re-check the identity before signalling, or it can kill a session that is starting (ARCH-ORDER).

## Round 4 — 2026-10-06T23:00:42-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — couch.go orphanStartRefusal now lists the tree, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident fails if the order changes or the old advice returns.
- BR-7 — addressed — ServerState carries one ServerVerdict enum (zero value Unresolved); every consumer uses hasServer or a verdict comparison; the table test pins all four verdicts.
- BR-8 — addressed — Harmless in M1 (it only refuses); the plan's 2026-10-07 Revisions requires two snapshots or a minimum process age, an identity re-read before each signal, and a just-started-server fixture in M2.

## Round 5 — 2026-10-07T00:20:01-07:00 (claude) — BLOCKED

### Raised

- **BR-9** [Important] `vocabulary-consumer-missing-member` reduceOperationResult has no success arm for reap/recover: the confirmation frame lingers, then an error notice appears
  Reproduced with a scratch test: a confirmed recover succeeds and the frame stays at depth 3; the next inventory drops it with an error-level notice "thread action is no longer applicable" (menu.go:1703-1724). This is the 2nd finding in this family. The rule to fix: every per-operation switch in couchtty/couchcmd derives from an Operations() declaration or is guarded by a sweep test over Operations() that fails on a missing arm. About 12 hand-maintained restatements of the slot-op set exist (cli.go, messages.go, message_service.go, protocol.go, run.go, slot_operation.go, menu_slot.go, menu.go).
- **BR-10** [Important] `declared-order-step-dropped` OSOrphanReaper skips the tag's title-poller/nvim pidfile reapers that the plan's ARCH-ORDER listed
  pair title is started detached by the launcher (osruntime.go:382), not under the zellij server, so the tree kill never reaches it. That is the PPID-1 residue the Done-when forbids and that M3 step 4 checks for. Call KillTitlePoller and ReapNvim for the tag after the tree kill, or revise the plan; also correct the atlas wording.
- **BR-11** [Important] `advice-restates-mechanism` orphanStartRefusal still prints manual ps/kill steps despite the M1-review revision requiring advice derived from the reap mechanism
  couch.go:1279-1300. The sibling refusal at couch.go:494 already gives a working switcher gesture ("run couch in another repository, select it, Tab → reboot"). The orphan case should name Tab → recover (or couch --recover <ref> --confirm), or a plan revision should explain why manual steps must stay.
- **BR-12** [Important] `docs-surface-missing` README does not document couch --reap / couch --recover or the switcher recover action
  README.md:388-389 and :440 list --resume/--reboot only; usageWith and the atlas were updated in this range, but the README was not.
- **BR-13** [Minor] `plan-drift-unrevised` Plan says reap is reached only through recover, but the switcher lists reap as its own entry too
  menuRowActions returns recover plus ActorActions (["reap"]). The Log records this as deliberate; add a Revisions entry so the plan matches the code.
- **BR-14** [Minor] `stale-doc-comment` ActorActions doc comment still says "the two actor operations, resume and reboot"
- **BR-15** [Minor] `point-observation-as-settled-state` OSProcessTable.Snapshot reads ppid from ps and start identity from a later sysctl, so a pid recycled between the two reads is planned under the wrong parent
  This is the 2nd finding in this family; it is very unlikely in practice. The rule: a process fact used to authorize a signal comes from one atomic read. Here that means reading ppid and start time from the same kinfo_proc.

## Round 6 — 2026-10-07T00:47:45-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — menu.go default arm closes any confirmation on success; TestReapSuccessClosesItsConfirmation drives the real key path and fails without the arm.
- BR-10 — addressed — OSOrphanReaper calls ReapTagHelpers (title pidfile + nvim) after the tree; TestReapTagHelpersEndsTheHelpersOutsideTheTree. The atlas and the code comment's reason are raised as a new Minor.
- BR-11 — addressed — couch.go orphanStartRefusal names Tab → recover and the inspect step, with no kill recipe; TestOrphanRefusalNamesTheReapMechanism asserts this.
- BR-12 — addressed — README.md 390-391 (CLI), 420-428 (orphaned threads), 863-867 (Tab → recover) match the code.
- BR-13 — not-addressed — No Revisions entry added; plan line 472 still says reap is not its own menu entry.
- BR-14 — not-addressed — actor_actions.go:47 unchanged.
- BR-15 — not-addressed — session_reap.go Snapshot still takes ppid from ps and identity from a later read.

### Raised

- **BR-16** [Minor] `stale-doc-comment` Atlas omits reap's helper step, and the code and lessons say the poller is outside the tree because of Setsid, which is false for Couch threads
  This is the 2nd finding in family stale-doc-comment. The rule: when a mechanism changes, every prose description of it (atlas, code comment, lessons) changes in the same commit. Here: atlas/couch.md's OSOrphanReaper sentence lacks the helper step and still lists pair title among the server's children. lifecycle.go:515 and lessons.md say the poller is outside the tree because it is spawned with Setsid, but sidecarProcessAttributes returns nil for Couch-launched Pair. The real reason is that the launcher, not the zellij server, is its parent.
- **BR-17** [Minor] `duplicate-derivation` editorPathsOf duplicates the quit path's inline editor-path literal and no test checks they stay equal
  lifecycle.go:196-199 vs 297-302 (ARCH-DRY). The inventory check is why the literal stays inline; a test asserting launcherCleanupOps.editorPaths equals editorPathsOf(paths) would keep the two from drifting.

## Round 7 — 2026-10-07T12:38:06-07:00 (claude) — passed

### Disposed

- BR-13 — addressed — Plan revision "2026-10-07 — reap is its own switcher entry too (M2)" (17124a86) records [recover, reap].
- BR-14 — addressed — actor_actions.go ActorActions comment now names resume, reboot and reap, and the live-orphan exception.
- BR-15 — not-addressed — session_reap.go OSProcessTable.Snapshot still takes ppid from ps and start identity from a later per-pid sysctl.
- BR-16 — not-addressed — atlas/couch.md about line 2197 still skips ReapTagHelpers and lists pair title as a child; lifecycle.go:516 and lessons.md:591 still credit Setsid.
- BR-17 — not-addressed — The only test (session_reap_test.go:222) compares against editorPathsOf itself; nothing compares it with the quit path's inline literal at lifecycle.go:196.

## Open findings

- **BR-15** [Minor] `point-observation-as-settled-state` OSProcessTable.Snapshot reads ppid from ps and start identity from a later sysctl, so a pid recycled between the two reads is planned under the wrong parent
- **BR-16** [Minor] `stale-doc-comment` Atlas omits reap's helper step, and the code and lessons say the poller is outside the tree because of Setsid, which is false for Couch threads
- **BR-17** [Minor] `duplicate-derivation` editorPathsOf duplicates the quit path's inline editor-path literal and no test checks they stay equal
