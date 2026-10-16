# Boundary Review — pair#607 (whole-issue close)

| field | value |
|-------|-------|
| issue | 607 — Reboot of a legacy slot leaves the old wrapper tree running |
| repo | pair |
| issue file | workshop/issues/000607-reboot-of-a-legacy-slot-leaves-the-old-wrapper-tree-running.md |
| boundary | whole-issue close |
| milestone | — |
| window | a6dcbbcb83e1b054b2abf46adc5d8d65e914cf50..2a756c8d00e99a177b23c1d29c3676cff445c53e |
| command | sdlc close --issue 607 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T02:13:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking that process-identity formats match across the registry, the process table and ProcOps, then running the tests.

Identity formats line up. Running the targeted tests, then reverting the fix to confirm they go red.

Fix regressions confirmed red. Now checking whether the tab-bar change dropped a case the old code handled on purpose.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The fix works. A Fresh reboot now adds the wrapper named in the message registry, that wrapper's descendants from one process-table snapshot, and the slot's own zellij session to the stop set, even when terminal attribution fails (`recovery_slot_tree.go:23`). Each process is checked by exact pid and start identity, sent SIGTERM, and awaited until it exits; the reboot refuses if any exit can't be confirmed. I checked that the identity formats agree. `procutil.Table` on darwin reports `sec.usec`, the same as `darwinIdentity`, and `Binding.Start` is already compared against `Proc.Identity` in `live_restart_probe.go:108`. `DeleteSession` only returns success once the session is observed absent, which covers the spec's "confirmed absent". The confirmation digest covers the added stops, so a tree that changes between preview and confirm makes the token fail. Targeted tests and vet pass. I reverted the `withSlotTree` call: the legacy-reboot and refusal tests went red. I restored the base `console_presentation.go`: the tab-chip test went red. Nothing blocks the boundary. Two cheap gaps remain: a wrapper whose liveness can't be observed is silently treated as no wrapper, and the shared-session case is only tested at the helper level.

1. **Strengths**
   - `slotWrapperTree` (`recovery_slot_tree.go:118`) is pure and works on one snapshot. It returns nothing if the table doesn't show the exact registered wrapper, and it skips zombies.
   - The added stops are de-duplicated against the stops attribution already proved, by pid and identity. The new steps wrap the existing Check and Retire rather than replacing them, so a proven reboot still runs its own retirement first.
   - It reuses `awaitExactProcessExit`, `observeExactProcess` and the existing `QuiesceNamed` seam (ARCH-DRY).
   - The real behaviour tests fail when the fix is removed (checked by reverting it here).
   - The atlas gained both rules (`atlas/couch.md:98`, `:1240`). The new source is classified in the artifact manifest.

2. **Critical:** none.

3. **Important**
   - `recovery_slot_tree.go:28`: if the registry names a wrapper but its liveness comes back `Unknown`, the code returns the retirement unchanged. The reboot then goes ahead and may leave a live tree behind, which is the original bug. That collapses uncertainty into absence (ARCH-ORDER). Fix sketch: on `Unknown`, refuse with "slot wrapper state cannot be observed; nothing was stopped", or at least add a note, and add a test.
   - `recovery_slot_tree_test.go:116`: the shared-session test calls only `otherLiveSessionHolder`. No reboot-level test shows that a shared session is left running, the "left running" note is produced, and the wrapper tree is still stopped. "Done when" names this case.

4. **Minor**
   - `console_presentation.go:45`: right after a reboot the new pane has a new address. If the inventory still lists the old address for the same worktree, the new pane is skipped and the old pane keeps the chip until the next refresh. The base code deliberately gave lagging panes a chip with the slot target (#307), and no test covers this ordering.
   - `slotWrapperTree` silently skips children with an empty identity. They survive with no note.
   - If attribution's own retirement already quiesces the session, the confirmation also lists `zellij session …`. The second quiesce is harmless because `!Present` returns nil.
   - The issue's `## Plan` section is an empty checkbox.

5. **Test coverage:** the legacy-stop, unconfirmed-refusal and one-chip tests fail without the fix, and the no-wrapper test pins the unchanged path. The process-table fake is a static slice separate from `FakeProcOps` state, so the two fakes could disagree. That's acceptable at this scale. Missing:
   - Unknown wrapper liveness.
   - Shared session at reboot level.
   - A process-table read error.
   - A wrapper identity mismatch between the registry check and the table snapshot.

6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. The tree walk is pure, and IO comes in through `SlotWrapper`, `ProcessTable` and `Proc`.
   - **ARCH-PURPOSE:** pass. The stop set covers every part of the spec.
   - **ARCH-MOCK:** pass, with the static-table caveat above.
   - **ARCH-CONSTRAINTS:** pass. One table read per reboot, on an operator action. Exit waits run one after another, but every signal is sent first, so the waits mostly overlap.
   - **ARCH-SECURE:** pass. The trust model is the same as live restart: the binding is checked by peer credentials and start identity.
   - **ARCH-ORDER:** flagged. The Unknown-to-absent collapse above. The digest pinning between preview and confirm is good.
   - **ARCH-FUNERAL:** pass. Nothing durable is created; this change removes leftovers.
   - **Upcoming work:** a child that daemonizes and re-parents away from the wrapper won't appear in the tree walk.

7. **Plan revisions:** fill `## Plan` with the delivered items: `withSlotTree` with the registry wrapper, descendants and own-session kill; the one-chip-per-slot tab rule; and the tests. Also record that a wrapper whose liveness can't be observed is now refused, if that fix is adopted.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: uncertainty-collapsed-into-absence
    title: |
      withSlotTree treats a registered wrapper with Unknown liveness as no wrapper and reboots leaving the tree untouched
    detail: |
      recovery_slot_tree.go:28 returns the retirement unchanged when observeExactProcessOrUnknown is not Live, so an unobservable (possibly live) registered wrapper reproduces the #607 orphan silently (ARCH-ORDER). Refuse or at least note on Unknown, and add a test.
  - id: new
    severity: Important
    kind: defect
    family: done-when-case-untested-end-to-end
    title: |
      Shared-session reboot path is only tested at the helper level
    detail: |
      TestSlotSessionSharedWithAnotherLiveSlotIsLeft calls otherLiveSessionHolder directly; no reboot test asserts that a shared session is left, the left-running note is emitted, and the wrapper tree is still stopped, although Done-when names this case.
  - id: new
    severity: Minor
    kind: defect
    family: snapshot-lag-presentation
    title: |
      Tab bar hides a newly attached reboot pane while the inventory still lists the old address for that worktree
    detail: |
      console_presentation.go:45 skips any pane whose address is not in inventory but whose tree is a slot. Right after a reboot this can drop the active pane's chip until the next refresh; the base code deliberately kept such lagging panes (#307), and no test covers this ordering.
  - id: new
    severity: Minor
    kind: defect
    family: silent-skip-in-stop-set
    title: |
      slotWrapperTree silently skips descendants with an empty identity
    detail: |
      Such children survive the reboot and no note says so; consider adding a note to the result.
  - id: new
    severity: Minor
    kind: defect
    family: plan-traceability
    title: |
      Issue Plan section is an empty checkbox
    detail: |
      The delivered items are recorded only in the Log; fill in the Plan so close review can trace it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"05ec9152-fb8d-4051-832c-60d1e4e85195","repository":"github.com/xianxu/pair","issue_id":"000607","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4b4849597082ea12","reviewed_head":"2a756c8d00e99a177b23c1d29c3676cff445c53e","slot":"pair:2","requested_at":"2026-10-16T09:09:10.023688Z"},"phase":"finished","pid":5239,"started_at":"2026-10-16T09:09:10.186362Z","finished_at":"2026-10-16T09:13:20.694377Z","session_id":"0cff751c-94d1-4486-a668-9485dc1a39fd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.037354375s
head: "2a756c8d00e99a177b23c1d29c3676cff445c53e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T02:23:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 607 — Reboot of a legacy slot leaves the old wrapper tree running |
| repo | pair |
| issue file | workshop/issues/000607-reboot-of-a-legacy-slot-leaves-the-old-wrapper-tree-running.md |
| boundary | whole-issue close |
| milestone | — |
| window | a6dcbbcb83e1b054b2abf46adc5d8d65e914cf50..c8e2c29a75356cb65202fd8b0f1afcc52f4b262f |
| command | sdlc close --issue 607 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T02:23:08-07:00 |
| verdict | SHIP |

## Review

Checking that the identity format matches, then running tests and reverting the fixes.

Now reverting BR-1's fix and BR-2's reachability to confirm the tests go red.

```verdict
verdict: SHIP
confidence: high
```

The change fixes the #607 orphan. A Fresh reboot now adds the slot's registered wrapper, that wrapper's descendants and the slot's own zellij session to the stop set. Every process in the set is matched by exact pid and identity, appears in the confirmation, and goes into the confirmation digest (`recoverySelectionDigest` hashes `Stops`, `recovery_prepared.go:21`). Both Important findings from round 1 are fixed, and I checked each by reverting it in this checkout. With the BR-1 Unknown-refusal reverted, `TestRebootRefusesWhenRegisteredWrapperIsUnobservable` fails. With the BR-2 shared-holder branch disabled, `TestRebootLeavesSessionSharedWithAnotherLiveSlot` fails ("shared session listed to stop"). The targeted tests and `go vet` pass on head. The only gap left is Minor: the note added for BR-4 has no test.

1. **Strengths**
   - The added stops reuse the existing retirement shape. They are dedup'd against the stops terminal attribution already proved, `Check` and `Retire` are wrapped rather than replaced, and the digest suffix forces a re-confirm (`recovery_slot_tree.go:279-356`).
   - An unobservable process refuses at both offer time and confirm time (`:263-270`, `:323`). Uncertainty is no longer treated as absence.
   - The pid/identity lookup is nil-safe (`messageService.LivenessForThread` handles a nil receiver) and follows the identity pattern the live-restart probe already uses (`live_restart_probe.go:108`).
   - The shared-session test runs through the real reboot. It checks the stop list, the notice, the pid deaths and that the shared session was not quiesced (`recoverplan_acceptance_test.go:128-208`).
2. **Critical:** none.
3. **Important:** none new.
4. **Minor:**
   - BR-4 is still open. The note is in place, but no test asserts it. Also, `slotWrapperTree` hits `continue` before walking into an unidentified child. That child's own descendants are never visited, so they are neither stopped nor counted (`recovery_slot_tree.go:388-391`).
5. **Test coverage:** All Done-when cases are covered at the reboot level: legacy slot, no wrapper, unconfirmed exit, shared session and unobservable wrapper. No test covers a child spawned after the snapshot, which is acceptable.
6. **Architecture:**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. `slotWrapperTree` is pure, and the IO comes in through `SlotWrapper` and `ProcessTable`.
   - ARCH-PURPOSE: pass.
   - ARCH-MOCK: pass. Tests use `FakeProcOps` and an injected table.
   - ARCH-CONSTRAINTS: pass. It takes one table snapshot per reboot.
   - ARCH-SECURE: pass. Only exact-identity signals are sent.
   - ARCH-ORDER: pass. Unknown refuses, Dead is skipped, and Check re-observes before acting.
   - ARCH-FUNERAL: pass. Nothing durable is created.
7. **Plan revisions:** none. The Plan now matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Unknown now refuses (recovery_slot_tree.go:266-269); reverting it turns TestRebootRefusesWhenRegisteredWrapperIsUnobservable red.
  - id: BR-2
    disposition: addressed
    note: |
      TestRebootLeavesSessionSharedWithAnotherLiveSlot runs the full reboot; disabling the holder branch fails it.
  - id: BR-3
    disposition: withdrawn
    note: |
      TL-directed design (chip follows the switcher's current row), recorded in atlas/couch.md "One chip per slot (#607)"; non-slot panes keep the lag fallback.
  - id: BR-4
    disposition: not-addressed
    note: |
      Note added (recovery_slot_tree.go:290-292) but no test asserts it; an unidentified child's subtree is also neither walked nor counted.
  - id: BR-5
    disposition: addressed
    note: |
      Plan now lists the three delivered items with file locations.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3212c9c3-a425-4039-8c58-9a090eab3f76","repository":"github.com/xianxu/pair","issue_id":"000607","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0e5c0d0116166dc2","reviewed_head":"c8e2c29a75356cb65202fd8b0f1afcc52f4b262f","slot":"pair:2","requested_at":"2026-10-16T09:19:34.684939Z"},"phase":"finished","pid":57249,"started_at":"2026-10-16T09:19:34.924742Z","finished_at":"2026-10-16T09:22:52.846698Z","session_id":"c98c23ee-d3fd-491d-923d-32246f7e31f6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.227393583s
head: "c8e2c29a75356cb65202fd8b0f1afcc52f4b262f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
