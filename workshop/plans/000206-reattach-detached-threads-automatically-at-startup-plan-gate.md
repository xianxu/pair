---
gate: plan-quality
issue: 206
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-11T14:02:11-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Task 1 must count claims per address's effective (newest) binding across the merged index, not raw index lines
          detail: readSessionNameIndexes merges the legacy and scope indexes (session_index.go:206-235), AppendSessionNameIndex appends per create (createflow.go:504), and lookupSessionName is newest-wins (artifactcollision.go:139). Raw-line counting fails closed on any re-indexed thread, whose cwd row then reads session-gone, so startup creates a second thread in the tree. State the semantics and add pure rows for two lines for one address and a superseded stale line.
          family: fail-closed-rule-domain
          round: 1
        - id: PQ-2
          severity: Important
          title: The background flag has no channel from finishOperation to installObservedThreadActor through the declared attach operation
          detail: 'Attach goes finishOperation, fn(OperationCall attach) (console.go:1817), DispatchOperation against two declared args (ops.go:227-235), then ExecuteConsoleOperation (console.go:1955). Choose and state the channel (implicit arg on a declared op, or typed payload). Calling the installer directly would skip wireResolver''s AbortStarted (couchcmd/run.go:462-466), the #230 route-6 protection.'
          family: unstated-seam-change
          round: 1
        - id: PQ-3
          severity: Important
          title: Enter and click on the loading row pick their verb via enterOperationFor from the lagging inventory, so cell 9 adoption never fires
          detail: 'enterOperationFor (menu.go:605-610) returns resume only for Parked or Detached. In the stale, Busy and Live-unhosted states that Decision 9 names it returns switch, claimReattach is skipped, and the console refuses with not attached (console.go:1950). Route every reader of a pass-owned row through the pass: actionable gate, verb, Tab actions, render. Drive the adoption test through each lagging state.'
          family: authority-bypassed-by-reader
          round: 1
        - id: PQ-4
          severity: Important
          title: Attached is cleared only on an inventory showing X Live, so a pass-attached pane that exits first leaves the row stuck rendering live
          detail: Pane exit (onExit, console.go:853) before the next inventory means X never reads Live. Render and the actionable gate read the pass first, so the row lies and Enter acts on a dead thread. Clear on any inventory newer than the success's ProjectionAfterGeneration (console.go:1843-1845), whatever its state, and add the table row.
          family: interrupting-event-transition
          round: 1
        - id: PQ-5
          severity: Minor
          title: Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
          detail: 'Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 enumerate cases. Invariants to check: at most one attempt, root never queued, queue never grows, no emit while an operator op holds the slot, Queue, Attached and Failed disjoint. Hand-picked rows sample one interleaving each.'
          family: test-plan-enumerates-cases
          round: 1
        - id: PQ-6
          severity: Minor
          title: Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
          detail: DetachedSessions runs in ResumeContext and again in confirmStillDetached, each query bounded at 5 s (zellij.go:20). finishOperation's requestMenuRefresh (console.go:1884) re-asks every remaining candidate. Task 12 should measure it.
          family: envelope-omits-cost-source
          round: 1
        - id: PQ-7
          severity: Minor
          title: State whether couch resume in a terminal arms the pass; justify or drop rename in cell 14; give non-refusal failures a code; ask about the deviation before M2
          detail: couchcmd/run.go:213 routes couch resume through runConsole. Cell 14 contradicts Done-when bullet 2 for rename. resume.go:377 errors have an empty diagnostic code.
          family: unstated-decision
          round: 1
        - id: PQ-8
          severity: Minor
          title: The fake DetachedSessions bypasses ProjectDetachedSessions, so the Task 1 duplicate-name rule is invisible to fake-backed tests
          detail: artifactcollision_fake.go:87-116 is a map lookup. Route it through ProjectDetachedSessions so production and fake share the rule (ARCH-MOCK).
          family: fake-diverges-from-shared-rule
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-11T14:17:27-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Per-scope counts are summed (artifactcollision.go:286) and each scope read merges the shared legacy file (session_index.go:215-218), so a legacy-indexed thread counts once per asked scope; measured 2 of 77 couch threads here count 6 in a 6-scope ask and read session-gone. Count distinct addresses once over the union of files read; add a two-scope row with a legacy entry.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Implicit background arg on attach, read by ExecuteConsoleOperation, so the path still passes wireResolver's AbortStarted.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 7 Step 4 consults the pass before enterOperationFor for the loading, queued and failed rows; the Attached state is raised separately.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Generation-bound expiry is stated in the prose and cell 13; the stale struct and function name are in the new Minor finding.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: Property tests were added in Task 6 Step 0; the prose case lists in Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 remain.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Extent still says 10 s + 5 s; ResumeContext and confirmStillDetached each run several zellij queries with their own 5 s bound (zellij.go:120).
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: Decided in Decisions 9-11 and the Deviation section, but Task 10 arms inside runConsole (shared with couch resume), cell 14 still lists rename, and the only operator ask is Task 12, at the close.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: The fake builds bindings and claims and calls ProjectDetachedSessions; pinned by TestFakeDetachedSessionsAppliesTheDuplicateNameRule.
          round: 2
      findings:
        - id: PQ-9
          severity: Important
          title: An Attached row renders live from the pass, but Enter, click and the actionable gate still read the lagging inventory
          detail: 'This is the 2nd finding in family authority-bypassed-by-reader, so state the rule, not this instance. Task 7 Step 4 routes only Loading, Queue and Failed through the pass, and Task 9 renders Attached as live. Until the first inventory past the attach''s generation, X reads Detached, so Enter sends a second resume that DecideResume refuses as occupied (resume.go:89-90), or Busy/stale, so Enter refuses it as unusable. menuActionItems also returns its Busy branch before menuThreadActionable (menu.go:1120-1126). Rule: one pure per-address view of the pass, which render, reduceRootKey, MenuEventMouseSwitch, menuThreadActionable and menuActionItems each consult before the inventory. It maps {Queued, Loading, Failed, Attached} to {text, verb, actionable, Tab items}. Put that 4x4 table in the plan and drive the generated-sequence test''s Enter and click through it. Prevalence: 2 instances in 2 rounds.'
          family: authority-bypassed-by-reader
          round: 2
        - id: PQ-10
          severity: Minor
          title: Revisions changed the prose but not the code block, the task steps and the file lists that restate them
          detail: 'The ReattachPass struct still has Attached map[...]bool, commented "awaiting an inventory that shows them Live", and Task 6 Step 2 names clearAttachedWhenLive; both contradict cell 13. Task 2 Step 4 says the fake does not model the duplicate-name rule, which is false since PQ-8. Task 8 omits declaring background on attach in ops.go and bumping the attach arity of 2 (run_test.go:586). Core concepts also says PathHoldsUnreadableThread reads only cwd rows; it checks the whole scope (startup.go:112-117). That is harmless, because unreadable records never reach the part of the scan the filter controls. Rule: when a decision changes, grep every restatement (code blocks, tables, steps, file lists, test names) and change them in the same edit. PQ-7''s three sites belong to the same class.'
          family: decision-restated-not-swept
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-11T14:30:31-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: 'Right within one scope, wrong across scopes. DetachedSessions sums sessionNameClaims once per scoped read (artifactcollision.go:286-288). Every scoped read merges the legacy-global file (session_index.go:215-232), and sessionNameClaims keys each row by its own ScopeKey without filtering (artifactcollision.go:147-150). So a legacy-bound thread counts once per scope asked. With 2+ scopes the name reads as contested, the thread gets no observation and reads session-gone (detachedsessions.go:74, actionableinventory.go:318-319), and the pass never seeds it. This host has 56 legacy-only bindings; 32 are pinned for good by AssignSessionName (session_index.go:394-396). Tests use one scope (artifactcollision_zellij_test.go:93-94), and the fake counts its own map (artifactcollision_fake.go:128-133). Rule (fail-closed-rule-domain, 2nd instance): count each claimant once, at its effective binding, over the union of files the call reads. Replay the legacy file once, then each scope file, newest row wins per address, in one pure function the checker and the fake both call. Pin it with a two-scope case where one thread is bound only by a legacy row.'
          round: 3
        - id: PQ-5
          disposition: not-addressed
          note: Half done. Task 6 Step 0 adds the generated-sequence invariants. The prose case lists remain in Task 2 Step 4, 5.1, 7.1, 8.1 and 9.1. The generated alphabet also has no Tab or actions-menu event, which is where this round's crash lives.
          round: 3
        - id: PQ-6
          disposition: not-addressed
          note: 'Unchanged. Extent still names one 5 s zellij bound, but a warm resume makes several bounded queries: ResumeContext''s DetachedSessions, confirmStillDetached, and the launcher''s liveness snapshots. The 7 s pass figure doesn''t say it leaves out the per-completion refresh. Naming #229 as that cost''s owner is fine; Task 12 should record the refresh count and duration during the pass.'
          round: 3
        - id: PQ-7
          disposition: addressed
          note: Decisions 9, 10 and 11 and the deviation section now state all four answers. The task, table and struct sites that restate them were not updated; those are carried under PQ-10.
          round: 3
        - id: PQ-9
          disposition: addressed
          note: 'The table, passViewOf, the five readers routed through it, and Enter/click driven through the view in the generated test are what was asked. Two new findings follow: the reattaching row''s empty Tab items crash, and readers beyond the five still bypass the view.'
          round: 3
        - id: PQ-10
          disposition: not-addressed
          note: 'Partly updated (the Attached type, expireAttached, Task 8 Step 4, the PathHoldsUnreadableThread wording). Still stale: Task 2 Step 4 still says the fake does not model the duplicate-name rule. Cell 14 still lists rename, against Decision 10. Task 10 arms inside runConsole, but runConsole also serves couch resume (run.go:311-312, 333-335) and consoleFinisher carries no op name (run.go:216), against Decision 9. Task 12 asks about the deviation at close, and no step asks before M2. Decision 11''s error first line has no field (Failed holds only a code) and no Task 9 row. The view table gives Queued and Failed the Tab items resume, park, archive, but today''s non-live items are resume, archive, name, describe (menu.go:1142), park needs a live row (menu.go:565-570), and name is missing although Decision 10 reasons about renaming a queued row.'
          round: 3
      findings:
        - id: PQ-11
          severity: Important
          title: The pass view gives the reattaching row no Tab items, and reduceRootKey indexes items[0], so Tab on that row panics
          detail: 'reduceRootKey''s Tab branch does SelectedItem: items[0] with no guard (menu.go:480-482). Every branch of menuActionItems returns at least 2 items today (menu.go:1120-1142), and that caller relies on it. The table''s "none (an operation is in flight)" breaks that, and the panic is on the Run loop, so it takes couch down. Fix: give Loading the Busy items ("name", "describe"). Next to the table, state the post-condition each overridden reader must keep: a non-empty item list, and a verb enterOperationFor could return. Add Tab and actions-menu selection to Task 6 Step 0''s generated alphabet so a future empty list fails there.'
          family: override-breaks-caller-invariant
          round: 3
        - id: PQ-12
          severity: Minor
          title: Readers beyond the five still read inventory state directly, and a test over five named functions cannot catch a sixth
          detail: 'This is the 3rd finding in family authority-bypassed-by-reader, so it states the rule rather than fixing instances. Direct readers today: confirmation Enter checks !thread.Live() (menu.go:714) and throws away a park, detach or relaunch the view offered on an Attached row; the same check in reconcile (menu.go:1372); the leave confirmation''s live count (menu.go:1156); Escape (menu.go:492); age colouring (menu_render.go:442). Rule: apply the pass where rows are looked up, not in each reader. Overlay it in the lookup every reader already uses (findMenuThread, selectedMenuThread, visibleRootThreads). Then guard that no menu code reads state.Inventory except through that lookup; that guard is what makes "a sixth reader fails" true. Prevalence: 3 instances in 3 rounds (PQ-3, PQ-9, this one). Each stale read lasts at most one refresh (cell 13), hence Minor.'
          family: authority-bypassed-by-reader
          round: 3
        - id: PQ-13
          severity: Minor
          title: Task 7 says a background success sets ProjectionPending so the row is not read as stale, but no row reader checks ProjectionPending
          detail: ProjectionPending is read only by the notice line (menu_render.go:213-220), the row budget (menu_render.go:411) and its own clearing (menu.go:380-383). The Attached map is what keeps the row live (cell 13). As written, every background attach puts "refresh pending" on the operator's notice line for the whole pass. Either drop it or state that the notice text is intended.
          family: existing-behavior-misstated
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-11T14:43:43-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: effectiveBindings merges the union of reads, own-scope authoritative; claimsFromBindings counts distinct threads; fake shares it; the cross-scope legacy test exists.
          round: 4
        - id: PQ-5
          disposition: not-addressed
          note: Task 6 Step 0's generated invariants landed; Tasks 2.4, 5.1, 7.1, 8.1, 9.1 still enumerate cases in prose. Minor, carry.
          round: 4
        - id: PQ-6
          disposition: not-addressed
          note: 'Extent still says one 5 s query bound; a warm attempt runs DetachedSessions twice (resume.go:373, :467), two zellij calls each at 5 s, so worst case is about 30 s. Refresh fan-out delegated to #229, acceptable.'
          round: 4
        - id: PQ-10
          disposition: not-addressed
          note: 'Struct, expireAttached, attach arity and PathHoldsUnreadableThread swept; still stale: Task 2.4 fake sentence and row list vs startup_proof_test.go:147, unticked M1 tasks, Task 6.0 alphabet lacking Tab, and the Queued/Failed Tab items (menu.go:1135-1140 offers no park on non-live rows).'
          round: 4
        - id: PQ-11
          disposition: addressed
          note: Loading gets the Busy items, verified as name and describe at menu.go:1120; post-conditions stated against the items[0] index at menu.go:481.
          round: 4
        - id: PQ-12
          disposition: addressed
          note: Rule stated as overlay-in-lookup plus a source-parsing guard; feasible, since only the setters, clone, leave count, reconcile snapshot and LabelsFor read Inventory outside the two lookups.
          round: 4
        - id: PQ-13
          disposition: addressed
          note: Task 7 Step 1 now says a background success does not set ProjectionPending, with the notice-line reason.
          round: 4
      blocked: false
content_hash: bcc608b9601f96502a73f88a171126c0948e4367d331cb7bfca8a76c9d6374ad
---

# Gate ledger — pair#206 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T14:02:11-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `fail-closed-rule-domain` Task 1 must count claims per address's effective (newest) binding across the merged index, not raw index lines
  readSessionNameIndexes merges the legacy and scope indexes (session_index.go:206-235), AppendSessionNameIndex appends per create (createflow.go:504), and lookupSessionName is newest-wins (artifactcollision.go:139). Raw-line counting fails closed on any re-indexed thread, whose cwd row then reads session-gone, so startup creates a second thread in the tree. State the semantics and add pure rows for two lines for one address and a superseded stale line.
- **PQ-2** [Important] `unstated-seam-change` The background flag has no channel from finishOperation to installObservedThreadActor through the declared attach operation
  Attach goes finishOperation, fn(OperationCall attach) (console.go:1817), DispatchOperation against two declared args (ops.go:227-235), then ExecuteConsoleOperation (console.go:1955). Choose and state the channel (implicit arg on a declared op, or typed payload). Calling the installer directly would skip wireResolver's AbortStarted (couchcmd/run.go:462-466), the #230 route-6 protection.
- **PQ-3** [Important] `authority-bypassed-by-reader` Enter and click on the loading row pick their verb via enterOperationFor from the lagging inventory, so cell 9 adoption never fires
  enterOperationFor (menu.go:605-610) returns resume only for Parked or Detached. In the stale, Busy and Live-unhosted states that Decision 9 names it returns switch, claimReattach is skipped, and the console refuses with not attached (console.go:1950). Route every reader of a pass-owned row through the pass: actionable gate, verb, Tab actions, render. Drive the adoption test through each lagging state.
- **PQ-4** [Important] `interrupting-event-transition` Attached is cleared only on an inventory showing X Live, so a pass-attached pane that exits first leaves the row stuck rendering live
  Pane exit (onExit, console.go:853) before the next inventory means X never reads Live. Render and the actionable gate read the pass first, so the row lies and Enter acts on a dead thread. Clear on any inventory newer than the success's ProjectionAfterGeneration (console.go:1843-1845), whatever its state, and add the table row.
- **PQ-5** [Minor] `test-plan-enumerates-cases` Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
  Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 enumerate cases. Invariants to check: at most one attempt, root never queued, queue never grows, no emit while an operator op holds the slot, Queue, Attached and Failed disjoint. Hand-picked rows sample one interleaving each.
- **PQ-6** [Minor] `envelope-omits-cost-source` Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
  DetachedSessions runs in ResumeContext and again in confirmStillDetached, each query bounded at 5 s (zellij.go:20). finishOperation's requestMenuRefresh (console.go:1884) re-asks every remaining candidate. Task 12 should measure it.
- **PQ-7** [Minor] `unstated-decision` State whether couch resume in a terminal arms the pass; justify or drop rename in cell 14; give non-refusal failures a code; ask about the deviation before M2
  couchcmd/run.go:213 routes couch resume through runConsole. Cell 14 contradicts Done-when bullet 2 for rename. resume.go:377 errors have an empty diagnostic code.
- **PQ-8** [Minor] `fake-diverges-from-shared-rule` The fake DetachedSessions bypasses ProjectDetachedSessions, so the Task 1 duplicate-name rule is invisible to fake-backed tests
  artifactcollision_fake.go:87-116 is a map lookup. Route it through ProjectDetachedSessions so production and fake share the rule (ARCH-MOCK).

## Round 2 — 2026-09-11T14:17:27-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Per-scope counts are summed (artifactcollision.go:286) and each scope read merges the shared legacy file (session_index.go:215-218), so a legacy-indexed thread counts once per asked scope; measured 2 of 77 couch threads here count 6 in a 6-scope ask and read session-gone. Count distinct addresses once over the union of files read; add a two-scope row with a legacy entry.
- PQ-2 — addressed — Implicit background arg on attach, read by ExecuteConsoleOperation, so the path still passes wireResolver's AbortStarted.
- PQ-3 — addressed — Task 7 Step 4 consults the pass before enterOperationFor for the loading, queued and failed rows; the Attached state is raised separately.
- PQ-4 — addressed — Generation-bound expiry is stated in the prose and cell 13; the stale struct and function name are in the new Minor finding.
- PQ-5 — not-addressed — Property tests were added in Task 6 Step 0; the prose case lists in Tasks 2.4, 5.1, 7.1, 8.1 and 9.1 remain.
- PQ-6 — not-addressed — Extent still says 10 s + 5 s; ResumeContext and confirmStillDetached each run several zellij queries with their own 5 s bound (zellij.go:120).
- PQ-7 — not-addressed — Decided in Decisions 9-11 and the Deviation section, but Task 10 arms inside runConsole (shared with couch resume), cell 14 still lists rename, and the only operator ask is Task 12, at the close.
- PQ-8 — addressed — The fake builds bindings and claims and calls ProjectDetachedSessions; pinned by TestFakeDetachedSessionsAppliesTheDuplicateNameRule.

### Raised

- **PQ-9** [Important] `authority-bypassed-by-reader` An Attached row renders live from the pass, but Enter, click and the actionable gate still read the lagging inventory
  This is the 2nd finding in family authority-bypassed-by-reader, so state the rule, not this instance. Task 7 Step 4 routes only Loading, Queue and Failed through the pass, and Task 9 renders Attached as live. Until the first inventory past the attach's generation, X reads Detached, so Enter sends a second resume that DecideResume refuses as occupied (resume.go:89-90), or Busy/stale, so Enter refuses it as unusable. menuActionItems also returns its Busy branch before menuThreadActionable (menu.go:1120-1126). Rule: one pure per-address view of the pass, which render, reduceRootKey, MenuEventMouseSwitch, menuThreadActionable and menuActionItems each consult before the inventory. It maps {Queued, Loading, Failed, Attached} to {text, verb, actionable, Tab items}. Put that 4x4 table in the plan and drive the generated-sequence test's Enter and click through it. Prevalence: 2 instances in 2 rounds.
- **PQ-10** [Minor] `decision-restated-not-swept` Revisions changed the prose but not the code block, the task steps and the file lists that restate them
  The ReattachPass struct still has Attached map[...]bool, commented "awaiting an inventory that shows them Live", and Task 6 Step 2 names clearAttachedWhenLive; both contradict cell 13. Task 2 Step 4 says the fake does not model the duplicate-name rule, which is false since PQ-8. Task 8 omits declaring background on attach in ops.go and bumping the attach arity of 2 (run_test.go:586). Core concepts also says PathHoldsUnreadableThread reads only cwd rows; it checks the whole scope (startup.go:112-117). That is harmless, because unreadable records never reach the part of the scan the filter controls. Rule: when a decision changes, grep every restatement (code blocks, tables, steps, file lists, test names) and change them in the same edit. PQ-7's three sites belong to the same class.

## Round 3 — 2026-09-11T14:30:31-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Right within one scope, wrong across scopes. DetachedSessions sums sessionNameClaims once per scoped read (artifactcollision.go:286-288). Every scoped read merges the legacy-global file (session_index.go:215-232), and sessionNameClaims keys each row by its own ScopeKey without filtering (artifactcollision.go:147-150). So a legacy-bound thread counts once per scope asked. With 2+ scopes the name reads as contested, the thread gets no observation and reads session-gone (detachedsessions.go:74, actionableinventory.go:318-319), and the pass never seeds it. This host has 56 legacy-only bindings; 32 are pinned for good by AssignSessionName (session_index.go:394-396). Tests use one scope (artifactcollision_zellij_test.go:93-94), and the fake counts its own map (artifactcollision_fake.go:128-133). Rule (fail-closed-rule-domain, 2nd instance): count each claimant once, at its effective binding, over the union of files the call reads. Replay the legacy file once, then each scope file, newest row wins per address, in one pure function the checker and the fake both call. Pin it with a two-scope case where one thread is bound only by a legacy row.
- PQ-5 — not-addressed — Half done. Task 6 Step 0 adds the generated-sequence invariants. The prose case lists remain in Task 2 Step 4, 5.1, 7.1, 8.1 and 9.1. The generated alphabet also has no Tab or actions-menu event, which is where this round's crash lives.
- PQ-6 — not-addressed — Unchanged. Extent still names one 5 s zellij bound, but a warm resume makes several bounded queries: ResumeContext's DetachedSessions, confirmStillDetached, and the launcher's liveness snapshots. The 7 s pass figure doesn't say it leaves out the per-completion refresh. Naming #229 as that cost's owner is fine; Task 12 should record the refresh count and duration during the pass.
- PQ-7 — addressed — Decisions 9, 10 and 11 and the deviation section now state all four answers. The task, table and struct sites that restate them were not updated; those are carried under PQ-10.
- PQ-9 — addressed — The table, passViewOf, the five readers routed through it, and Enter/click driven through the view in the generated test are what was asked. Two new findings follow: the reattaching row's empty Tab items crash, and readers beyond the five still bypass the view.
- PQ-10 — not-addressed — Partly updated (the Attached type, expireAttached, Task 8 Step 4, the PathHoldsUnreadableThread wording). Still stale: Task 2 Step 4 still says the fake does not model the duplicate-name rule. Cell 14 still lists rename, against Decision 10. Task 10 arms inside runConsole, but runConsole also serves couch resume (run.go:311-312, 333-335) and consoleFinisher carries no op name (run.go:216), against Decision 9. Task 12 asks about the deviation at close, and no step asks before M2. Decision 11's error first line has no field (Failed holds only a code) and no Task 9 row. The view table gives Queued and Failed the Tab items resume, park, archive, but today's non-live items are resume, archive, name, describe (menu.go:1142), park needs a live row (menu.go:565-570), and name is missing although Decision 10 reasons about renaming a queued row.

### Raised

- **PQ-11** [Important] `override-breaks-caller-invariant` The pass view gives the reattaching row no Tab items, and reduceRootKey indexes items[0], so Tab on that row panics
  reduceRootKey's Tab branch does SelectedItem: items[0] with no guard (menu.go:480-482). Every branch of menuActionItems returns at least 2 items today (menu.go:1120-1142), and that caller relies on it. The table's "none (an operation is in flight)" breaks that, and the panic is on the Run loop, so it takes couch down. Fix: give Loading the Busy items ("name", "describe"). Next to the table, state the post-condition each overridden reader must keep: a non-empty item list, and a verb enterOperationFor could return. Add Tab and actions-menu selection to Task 6 Step 0's generated alphabet so a future empty list fails there.
- **PQ-12** [Minor] `authority-bypassed-by-reader` Readers beyond the five still read inventory state directly, and a test over five named functions cannot catch a sixth
  This is the 3rd finding in family authority-bypassed-by-reader, so it states the rule rather than fixing instances. Direct readers today: confirmation Enter checks !thread.Live() (menu.go:714) and throws away a park, detach or relaunch the view offered on an Attached row; the same check in reconcile (menu.go:1372); the leave confirmation's live count (menu.go:1156); Escape (menu.go:492); age colouring (menu_render.go:442). Rule: apply the pass where rows are looked up, not in each reader. Overlay it in the lookup every reader already uses (findMenuThread, selectedMenuThread, visibleRootThreads). Then guard that no menu code reads state.Inventory except through that lookup; that guard is what makes "a sixth reader fails" true. Prevalence: 3 instances in 3 rounds (PQ-3, PQ-9, this one). Each stale read lasts at most one refresh (cell 13), hence Minor.
- **PQ-13** [Minor] `existing-behavior-misstated` Task 7 says a background success sets ProjectionPending so the row is not read as stale, but no row reader checks ProjectionPending
  ProjectionPending is read only by the notice line (menu_render.go:213-220), the row budget (menu_render.go:411) and its own clearing (menu.go:380-383). The Attached map is what keeps the row live (cell 13). As written, every background attach puts "refresh pending" on the operator's notice line for the whole pass. Either drop it or state that the notice text is intended.

## Round 4 — 2026-09-11T14:43:43-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — effectiveBindings merges the union of reads, own-scope authoritative; claimsFromBindings counts distinct threads; fake shares it; the cross-scope legacy test exists.
- PQ-5 — not-addressed — Task 6 Step 0's generated invariants landed; Tasks 2.4, 5.1, 7.1, 8.1, 9.1 still enumerate cases in prose. Minor, carry.
- PQ-6 — not-addressed — Extent still says one 5 s query bound; a warm attempt runs DetachedSessions twice (resume.go:373, :467), two zellij calls each at 5 s, so worst case is about 30 s. Refresh fan-out delegated to #229, acceptable.
- PQ-10 — not-addressed — Struct, expireAttached, attach arity and PathHoldsUnreadableThread swept; still stale: Task 2.4 fake sentence and row list vs startup_proof_test.go:147, unticked M1 tasks, Task 6.0 alphabet lacking Tab, and the Queued/Failed Tab items (menu.go:1135-1140 offers no park on non-live rows).
- PQ-11 — addressed — Loading gets the Busy items, verified as name and describe at menu.go:1120; post-conditions stated against the items[0] index at menu.go:481.
- PQ-12 — addressed — Rule stated as overlay-in-lookup plus a source-parsing guard; feasible, since only the setters, clone, leave count, reconcile snapshot and LabelsFor read Inventory outside the two lookups.
- PQ-13 — addressed — Task 7 Step 1 now says a background success does not set ProjectionPending, with the notice-line reason.

## Open findings

- **PQ-5** [Minor] `test-plan-enumerates-cases` Compress the prose test-case lists into per-function strategy lines, and add generated event-sequence invariant tests for ReattachPass
- **PQ-6** [Minor] `envelope-omits-cost-source` Worst-case attempt bound and pass duration omit repeated 5 s zellij queries and the O(N squared) per-completion inventory refresh
- **PQ-10** [Minor] `decision-restated-not-swept` Revisions changed the prose but not the code block, the task steps and the file lists that restate them
