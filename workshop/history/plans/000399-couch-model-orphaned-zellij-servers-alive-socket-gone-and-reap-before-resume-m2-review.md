# Boundary Review — pair#399 (milestone M2)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8446547c1556b650435cc58a50d3fcc975f3b340..ac668b55b2acd10ffa8c5af400ef88ecacf77646 |
| command | sdlc milestone-close --issue 399 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T00:20:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 mostly delivers what the plan promised. `PlanReap` and `Reaper` are pure and use a stateful fake that models the 2026-10-06 tree. `Couch.Reap` checks the orphan twice before it signals. The report's `reap → resume` steps, `ConfirmByPlan`, and the menu, CLI and socket paths for `recover` all match the Spec. None of the findings below blocks the boundary on its own, but four Important gaps should be fixed first:

- A confirmed `recover` or `reap` that succeeds in the switcher leaves its confirmation frame open, and the next refresh replaces it with an error notice. I reproduced this with a scratch test, then deleted the test.
- The reap never ends the tag's detached `pair title` poller. The plan's ordering section explicitly said it would.
- The startup refusal still prints hand-written `kill` steps. The plan revision from the M1 review said it would render the plan's steps instead.
- The README doesn't document `--reap` or `--recover`.

There are no prior findings left to dispose.

**1. Strengths**
- `launcher/session_reap.go:53` `PlanReap`: pure, takes one snapshot, and signals deepest-first with the server last. `Reaper` re-reads each pid's start identity before every signal and refuses before any signal if the server changed. A process that survives SIGKILL is named, not waited on forever. The fake table models a wrap that ignores SIGTERM.
- `couchcore/reap.go:46`: the two-observation check (`reapConfirmInterval`, comparing the server identity) carries out the M1 review's "a single snapshot is provisional" requirement. `TestReapRequiresTheOrphanToHoldAcrossTwoObservations` covers a server still starting, a recycled server and an observation failure.
- `recover_action.go:162` `Recover`: re-derives the preview, refuses a stale one, and checks the row is still admitted before each step. A failure partway names the steps already done. This is good ordering discipline.
- `OperationConfirms` gaining an explicit `byPlan` answer makes every caller decide. The menu (`menu.go:850`), CLI parse, socket admission and frame reconciliation each handle it.
- The totality test was extended correctly: resume after reap is allowed, reboot on an orphan is "unsafe reboot", and `consistent()` ties `OfferReap` to `AgentOrphaned`.

**2. Critical findings**
None.

**3. Important findings**
- **`couchtty/menu.go:1703-1724` `reduceOperationResult`: the success switch has no case for `reap` or `recover`.**
  - Reproduced: a confirmed recover on an orphan succeeds. The confirmation frame stays (depth 3). On the next inventory, `reconcileMenuFrames` drops it with an error-level notice "thread action is no longer applicable", and the menu is left on the Tab frame instead of the root.
  - **This is the 2nd finding in family `vocabulary-consumer-missing-member`.** The same operation set is also restated by hand in `operationNeedsProjectionRefresh`, `menuOperationProgressText`, `actorOperation`, `slotOperations`, `cli.go` (twice), `messages.go`, `message_service.go`, `protocol.go` (twice) and `run.go` (twice).
  - The rule to fix: any per-operation decision in couchtty or couchcmd either comes from a property declared in `Operations()`, or is guarded by a sweep test that enumerates `Operations()` (at least every `RowAction` / `ExecuteLiveOwner` op) and fails when an operation has no explicit arm.
  - Add `reap` and `recover` to the restore-to-root arm under that test, not as a one-off edit.
- **`launcher/session_reap.go:192` `OSOrphanReaper`: the plan's ordering step "existing title-poller/nvim pidfile reapers for the tag" was dropped.**
  - `pair title` is started detached by the `pair` launcher (`osruntime.go:382`, `spawnDetached`), so it is never under the zellij server. The tree kill cannot reach it.
  - That is the PPID-1 `pair title` residue the Done-when forbids, and M3 step 4 checks for it with `ps`. The poller does exit on its own after N missed session checks, but until then a resume's new poller sees it and backs off.
  - Fix: call `KillTitlePoller(tag)` and `ReapNvim(tag)` after the tree kill (the tag must reach the reaper). Otherwise, record a plan revision explaining why self-exit is enough.
  - The atlas claim that `pair title` children were survivors of the server tree should also be corrected.
- **`couchcore/couch.go:1279-1300` `orphanStartRefusal` still prints manual `ps` and `kill -KILL` steps.**
  - The plan revision from the M1 review ("One source for the advice") made this an M2 requirement: render the reap mechanism, so the advice and the behaviour cannot drift.
  - The sibling refusal just above (`couch.go:494`) already gives a gesture that works: "run couch in another repository, select it, Tab → reboot".
  - The orphan equivalent is "Tab → recover", or `couch --recover <ref> --confirm` from a live slot. Otherwise, record why manual steps must remain.
- **The README has no entry for `couch --reap repo:N --confirm` or `couch --recover repo:N [--confirm]`.**
  - `README.md:388-389` and `:440` document `--resume` and `--reboot`; `usageWith` and the atlas were updated, but the README was not.
  - The switcher's new default Tab action, `recover`, is also undocumented there.

**4. Minor findings**
- `actor_actions.go:47`: the doc comment still says "the two actor operations, resume and reboot".
- Plan drift: the revision says reap is reached from the switcher "through recover rather than as its own menu entry", but `menuRowActions` shows both `recover` and `reap`. The Log records this as deliberate, so the plan needs a revision entry.
- `DeriveRecoverPreview`: falling back to `ActorActions` when the report gives no actor steps (idle/landed) means recover can run a confirmed reboot the report did not suggest. On a row whose report step is `reconcile` (a `:N` slot with its directory missing), it holds with "no-actor-action" instead of pointing at reconcile. Consider naming the report's non-actor step in that hold.
- Recover skips `ask-agent-restore` without telling the switcher user that the report has a follow-up step.
- `OSProcessTable.Snapshot` reads ppid from `ps` and start identity from a later sysctl. A pid recycled between the two reads gets the new process's identity under the old parent. This is very unlikely; reading ppid and start time from the same `kinfo_proc` would close it.
- The `SKILL.md` command table gained `--reap` but not `--recover`.

**5. Test coverage notes**
- The launcher reaper tests are strong: tree order, a recycled pid, a changed server, and a bounded wait that names survivors. The "server-first ordering" mutation check is recorded.
- The `Couch.Reap` fake reaper only flips presence to absent. That is acceptable at this level, since the real tree logic is covered in the launcher.
- No menu test drives recover or reap to a successful `MenuEventOperationResult`. That gap is why the frame bug above shipped. Add a test that asserts the root frame and no error notice after success and refresh.
- The second Done-when item ("the next report shows `parked` with `resume` only" after a reap) is covered only indirectly, through the generic parked → resume case.

**6. Architectural notes**
- ARCH-DRY: flagged. The slot-operation set is restated in about 12 places across couchtty, couchcmd and couchmessage; this is the root of the first Important finding.
- ARCH-PURE: pass. `PlanReap`, `DeriveRecoverPreview` and `classifyRecover` are pure, and IO sits behind `ProcessTable` and `OrphanReaper`.
- ARCH-PURPOSE: flagged. The title-poller step was dropped and the startup advice is not derived from the reap mechanism.
- ARCH-MOCK: pass. The stateful `ProcessTable` fake is behind the same seam production uses; the live conformance check is M3's acceptance run.
- ARCH-CONSTRAINTS: pass. Reap is rare and bounded (3 s, then 2 s). The per-row sysctl is cheap. `prepare-recover` runs `sdlc` off the UI thread.
- ARCH-SECURE: pass, with the minor ps-then-sysctl race noted above. The identity gate covers every signal.
- ARCH-ORDER: flagged. In couchcore, everything that can refuse runs before the first signal. But the menu's state after a successful operation is undefined for the new operations (first Important finding).
- ARCH-FUNERAL: Reap creates nothing durable. The residue it leaves behind is the detached title poller (second Important finding).

**7. Plan revision recommendations**
- A revision entry for the M2 menu: `reap` appears beside `recover` because `ActorActions` is one admission table.
- A revision entry for `OSOrphanReaper`: it runs the tag's title/nvim pidfile reapers after the tree kill, or the plan states why it doesn't.
- A revision entry for the startup refusal: it renders the switcher's recover gesture, or the plan records why manual steps remain while no Couch is live.

```findings
findings:
  - id: new
    severity: Important
    family: vocabulary-consumer-missing-member
    title: |
      reduceOperationResult has no success arm for reap/recover: the confirmation frame lingers, then an error notice appears
    detail: |
      Reproduced with a scratch test: a confirmed recover succeeds and the frame stays at depth 3; the next inventory drops it with an error-level notice "thread action is no longer applicable" (menu.go:1703-1724). This is the 2nd finding in this family. The rule to fix: every per-operation switch in couchtty/couchcmd derives from an Operations() declaration or is guarded by a sweep test over Operations() that fails on a missing arm. About 12 hand-maintained restatements of the slot-op set exist (cli.go, messages.go, message_service.go, protocol.go, run.go, slot_operation.go, menu_slot.go, menu.go).
  - id: new
    severity: Important
    family: declared-order-step-dropped
    title: |
      OSOrphanReaper skips the tag's title-poller/nvim pidfile reapers that the plan's ARCH-ORDER listed
    detail: |
      pair title is started detached by the launcher (osruntime.go:382), not under the zellij server, so the tree kill never reaches it. That is the PPID-1 residue the Done-when forbids and that M3 step 4 checks for. Call KillTitlePoller and ReapNvim for the tag after the tree kill, or revise the plan; also correct the atlas wording.
  - id: new
    severity: Important
    family: advice-restates-mechanism
    title: |
      orphanStartRefusal still prints manual ps/kill steps despite the M1-review revision requiring advice derived from the reap mechanism
    detail: |
      couch.go:1279-1300. The sibling refusal at couch.go:494 already gives a working switcher gesture ("run couch in another repository, select it, Tab → reboot"). The orphan case should name Tab → recover (or couch --recover <ref> --confirm), or a plan revision should explain why manual steps must stay.
  - id: new
    severity: Important
    family: docs-surface-missing
    title: |
      README does not document couch --reap / couch --recover or the switcher recover action
    detail: |
      README.md:388-389 and :440 list --resume/--reboot only; usageWith and the atlas were updated in this range, but the README was not.
  - id: new
    severity: Minor
    family: plan-drift-unrevised
    title: |
      Plan says reap is reached only through recover, but the switcher lists reap as its own entry too
    detail: |
      menuRowActions returns recover plus ActorActions (["reap"]). The Log records this as deliberate; add a Revisions entry so the plan matches the code.
  - id: new
    severity: Minor
    family: stale-doc-comment
    title: |
      ActorActions doc comment still says "the two actor operations, resume and reboot"
  - id: new
    severity: Minor
    family: point-observation-as-settled-state
    title: |
      OSProcessTable.Snapshot reads ppid from ps and start identity from a later sysctl, so a pid recycled between the two reads is planned under the wrong parent
    detail: |
      This is the 2nd finding in this family; it is very unlikely in practice. The rule: a process fact used to authorize a signal comes from one atomic read. Here that means reading ppid and start time from the same kinfo_proc.
```

---

## Re-review — 2026-10-07T00:47:45-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 8446547c1556b650435cc58a50d3fcc975f3b340..5c180442cec5c7634943ad286f7bc5887f7f1c9d |
| command | sdlc milestone-close --issue 399 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T00:47:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

**Verdict: SHIP.** All four Important findings from round 3 (BR-9 through BR-12) are fixed, and each behaviour fix has a test that fails without it.
- **BR-9:** `reduceOperationResult` gets a default arm. A confirmation frame now always closes once its operation succeeds, so reap and any future operation are covered.
- **BR-10:** `OSOrphanReaper` now ends the thread's title poller and editors through their pidfiles after it kills the server's tree.
- **BR-11:** startup's refusal names `Tab → recover` instead of a hand-written kill recipe.
- **BR-12:** the README documents `--reap`, `--recover`, orphaned threads and `Tab → recover`.

What remains is Minor. Three earlier Minors were not touched (BR-13 plan revision, BR-14 doc comment, BR-15 two-read snapshot). Two new ones: the atlas does not describe the new helper step and the code gives the wrong reason for it, and the editor paths are now built in two places.

The #399 tests pass in the sandbox: launcher `Reap|Orphan|PlanReap`, couchcore `Reap|Orphan|Recover|…`, couchtty `Reap|Recover|Menu|Action`. Running the full packages fails, but every failure I traced is environmental:
- **couchcmd:** `mkdir /tmp/...: operation not permitted`, which is the sandbox.
- **launcher, couchcore, couchtty:** pty and conformance tests. The memory notes record these as sandbox failures.
- **artifactpath:** `TestProductionArtifactReferencesAreExactlyClassified` is already a known failure. The only #399-touched file it names is `recoverplan.go`, and its `"agent"` constants are unchanged since before M2. None of the new files appear in its output.

1. **Strengths**
   - `menu.go:1725-1739`: the default arm states the rule ("a confirmation frame never outlives its operation's success") instead of adding one more operation to the list. `TestReapSuccessClosesItsConfirmation` drives the real key path through confirm and dispatch.
   - `lifecycle.go:521-551`: `ReapTagHelpers` takes a narrow `tagHelperReaper` interface, so it is tested with a recording fake. `OSOrphanReaper` refuses visibly when it has no data directory, rather than leaving the helpers running.
   - `couch.go:1286-1297`: the refusal now hands off to the tested reap mechanism instead of restating it, which closes off the `operator-advice-contradicts-evidence` family. The negative assertions (`kill `, `pkill`) guard against a regression.
   - The README additions at `README.md:390-391, 420-428, 863-867` match the code: the CLI forms, the reap-then-resume steps, and when `recover` asks for confirmation.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-14 (still open)**, `actor_actions.go:47`: the doc comment still says "the two actor operations, resume and reboot".
   - **BR-13 (still open)**: the plan's Revisions still say reap is "not its own menu entry", but `ActorActions` offers it.
   - **BR-15 (still open)**, `session_reap.go:156-180`: `ppid` still comes from `ps`, and the start identity from a later, separate read.
   - **Atlas and the poller explanation**, `atlas/couch.md:~2175`: the atlas describes `OSOrphanReaper` as tree, then zellij record. It omits the helper step, and it still lists `pair title` among the server's children.
     - `lifecycle.go:515` and `workshop/lessons.md` say the title poller is outside the tree because it is spawned with `Setsid`. For a Couch-launched thread, `sidecarProcessAttributes` returns nil, so there is no `Setsid`.
     - The actual reason is that the launcher, not the zellij server, is the poller's parent. The behaviour is correct either way; only the explanation is wrong.
   - **Two copies of the editor paths**, `lifecycle.go:196-199` and `lifecycle.go:297-302` (ARCH-DRY): `editorPathsOf` duplicates the quit path's inline literal. No test checks that the two stay equal.
   - **`reaper()` assertion**, `reap.go:90`: it asserts against an anonymous interface instead of the existing `PairLifecycleEnvironment`. If the controller ever gets wrapped, reap silently loses its data directory and refuses every time. That failure is visible, but it would be confusing.

5. **Test coverage**
   - No test covers a successful `recover` result in the reducer, whether it ends in reap→resume or in reboot with a new address. The explicit `"recover"` arm and the default arm handle these cases, but nothing pins them.
   - There is no conformance test for `Couch.reaper()` wiring the data directory in production.

6. **Architecture**
   - **ARCH-DRY: flag** (editor paths, above).
   - **ARCH-PURE: pass.** `PlanReap` and `ReapTagHelpers` are pure or injected; the OS code stays in `OS*` types.
   - **ARCH-PURPOSE: pass.** Helpers are now reaped, so no PPID-1 leftovers remain.
   - **ARCH-MOCK: pass.** There are fakes for the process table and the helper reaper. A live conformance test of `ps` and `sysctl` is still missing.
   - **ARCH-CONSTRAINTS: pass.** The term and kill waits are bounded.
   - **ARCH-SECURE: pass.** The pid identity is re-read before every signal (BR-15 is the remaining gap).
   - **ARCH-ORDER: pass.** Tree, then helpers, then quiesce. The second orphan check gates the reap.
   - **ARCH-FUNERAL: pass.** Reap ends the processes and the zellij session record.

   For upcoming work: the `vocabulary-consumer-missing-member` family (about 12 hand-written lists of slot operations) still has no sweep over `Operations()`. The default arm fixes the menu reducer only.

7. **Plan revisions needed**
   - Add a Revisions entry: "reap is also offered as its own `ActorActions` entry on orphaned rows; recover remains the default."
   - Add a Revisions entry: "`OSOrphanReaper` also reaps the tag's title poller and nvim pidfiles after the tree (BR-10)."

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      menu.go default arm closes any confirmation on success; TestReapSuccessClosesItsConfirmation drives the real key path and fails without the arm.
  - id: BR-10
    disposition: addressed
    note: |
      OSOrphanReaper calls ReapTagHelpers (title pidfile + nvim) after the tree; TestReapTagHelpersEndsTheHelpersOutsideTheTree. The atlas and the code comment's reason are raised as a new Minor.
  - id: BR-11
    disposition: addressed
    note: |
      couch.go orphanStartRefusal names Tab → recover and the inspect step, with no kill recipe; TestOrphanRefusalNamesTheReapMechanism asserts this.
  - id: BR-12
    disposition: addressed
    note: |
      README.md 390-391 (CLI), 420-428 (orphaned threads), 863-867 (Tab → recover) match the code.
  - id: BR-13
    disposition: not-addressed
    note: |
      No Revisions entry added; plan line 472 still says reap is not its own menu entry.
  - id: BR-14
    disposition: not-addressed
    note: |
      actor_actions.go:47 unchanged.
  - id: BR-15
    disposition: not-addressed
    note: |
      session_reap.go Snapshot still takes ppid from ps and identity from a later read.
findings:
  - id: new
    severity: Minor
    family: stale-doc-comment
    title: |
      Atlas omits reap's helper step, and the code and lessons say the poller is outside the tree because of Setsid, which is false for Couch threads
    detail: |
      This is the 2nd finding in family stale-doc-comment. The rule: when a mechanism changes, every prose description of it (atlas, code comment, lessons) changes in the same commit. Here: atlas/couch.md's OSOrphanReaper sentence lacks the helper step and still lists pair title among the server's children. lifecycle.go:515 and lessons.md say the poller is outside the tree because it is spawned with Setsid, but sidecarProcessAttributes returns nil for Couch-launched Pair. The real reason is that the launcher, not the zellij server, is its parent.
  - id: new
    severity: Minor
    family: duplicate-derivation
    title: |
      editorPathsOf duplicates the quit path's inline editor-path literal and no test checks they stay equal
    detail: |
      lifecycle.go:196-199 vs 297-302 (ARCH-DRY). The inventory check is why the literal stays inline; a test asserting launcherCleanupOps.editorPaths equals editorPathsOf(paths) would keep the two from drifting.
```
