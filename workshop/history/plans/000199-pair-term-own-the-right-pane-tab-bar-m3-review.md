# Boundary Review — pair#199 (milestone M3)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 244a72a5594155bfda2d980f0e84c591648a76a8..b6c9fc684a223bc66f0d9a9f988372fc0f09ba91 |
| command | sdlc milestone-close --issue 199 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-08T13:28:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The strip lands as a genuinely pure renderer over a shared, now-correct row primitive, and the three defects the milestone found by measurement (cursor homing in `SetRegion`, colour-inheriting erase, the shared cursor-save slot) are all fixed at the shared primitive with tests that go red when reverted — I mutation-checked `unsafeToPaint`, `ReserveAndPaint`, and the rename-first width budget, and all three fail without their fix. What blocks a clean SHIP is that the milestone's paint-trigger sweep stopped at the three rename methods: `removeTab`'s `preserveRename` branch mutates the strip model and repaints nothing (reproduced: **zero bytes** written after a tab exits mid-rename), and the two pending-paint slots flush in the wrong order so a stale row can land after a fresh one (also reproduced). Alongside that, `renamePaneTitleLocked` was left unswept by the title degradation — it still packs the tab set and loses the classifier prefix that M3.6 established as load-bearing — and M3.7(b), the one manual step that reaches the defer-and-owe path under load, is ticked but was never run with an escape-emitting generator. `go vet`, `go build` and `-race` are clean; every test failure in the tree is the documented sandbox-pty class.

## 1. Strengths

- **`hostty.ReserveAndPaint` is the right consolidation** (`cmd/internal/hostty/reserve.go:105`). Two latent couch bugs since `#146` — DECSTBM homing the cursor before the save, and an erase inheriting the child's SGR — are fixed once, in the primitive, with `TestSaveComesBeforeTheRegionChangeThatHomesTheCursor` and `TestTheRowResetsColourBeforeErasingAndDrawing` pinning the *order*, not just the presence, of each byte. Folding `ResetSGR` into `HomeAndClear` rather than asking each caller to remember it is exactly the right shape.
- **`RenderStrip` is genuinely PURE** (`cmd/internal/termcmd/strip.go:78`). `(width, model) → (body, spans)`, no clock, no IO, no mux; every hard case (wide runes, escape injection, out-of-range `Active`/`Rename.Tab`, degenerate widths, empty tab set) is a unit test. `stripBytes` is a thin seam that takes the lock, snapshots, and renders. ARCH-PURE passes cleanly.
- **`RenameEditor.Field`** (`cmd/internal/termcmd/rename.go:68`) — one caret composer for two surfaces, with the cursor clamp done once. That is the correct answer to "the strip and the title disagreed about the caret", not a second copy.
- **`cmd/probes/couchnestedrows` is a serious instrument.** It reads ROWS off a terminal emulator rather than raw bytes, waits on the strip rather than sleeping a guessed interval, refuses to emit a verdict when its session never came up, uses `--new-session-with-layout` for a deterministic session name, and strips OSC before the model after the emulator itself was caught leaking non-ASCII OSC payload onto the screen. The `lessons.md` entry drawn from that ("a probe's first surprising result is a claim about the instrument") is the most valuable artifact in this window.
- **`ptychild.Screen.HoldsCursorSave` is modelled, not guessed** (`screen.go:156`): one slot not a stack, cleared by RIS and both alt-screen transitions, with `TestASecondSaveDoesNotDeepenTheDebt` explicitly checking the state *after* the second DECSC so a toggling implementation cannot pass. I additionally split every representative sequence at every byte index and fed the halves as separate chunks — the framer is correct across chunk boundaries.

## 2. Critical findings

None.

## 3. Important findings

### I1 — A tab exiting during a rename leaves the strip listing the closed tab (`run.go:1296`)

**This is the 2nd finding in family `exit-path-drops-cleanup`.** Do not fix this instance alone.

`removeTab` takes `if !preserveRename { m.applyTakeover(activeSnapshot) }`. With a rename open, the branch is skipped — and `applyTakeover` is the *only* thing that repaints the strip on that path. Reproduced:

```
bytes written after the tab exited: ""
```

The renderer was written with this exact event in mind — `StripModel.Active` and `RenameField.Tab` both carry comments about "a background tab exiting reindexes the slice while a rename is open" — so the *representation* was defended and the *transition* was missed. Staleness is bounded by the rename's end (`finishRename` repaints), but while it lasts the strip lies about which tabs exist, and after M4 the strip is the only surface that says so.

**The rule that covers the family:** every site that mutates the strip model owes a repaint, and the set of those sites must be enumerated by a test rather than swept by hand. Concretely: a table-driven test over `{newTab, switchRelative, closeActive, removeTab (both branches), beginRename, refreshRename, finishRename, inheritSize, rowDirty}` that drives each site and asserts a repaint carrying the post-mutation model reached the recorder. That test fails today on exactly one row, and it is the instrument that stops the next one. The three rename methods were fixed in this window because a probe happened to catch them end to end; an enumeration is what makes that not luck.

### I2 — `flushOwed` can write a stale row after a fresher one (`run.go:866-872`)

Family: `superseded-write-not-dropped` (new).

```go
if chunk.rowDirty { m.stripOwed = true }
if m.stripOwed && !m.unsafeToPaint() { m.stripOwed = false; m.paintStripInline() }
m.flushOwed()
```

Two independent slots hold a pending paint — `m.owed` (the coalescing slot) and `m.stripOwed` (the row-dirty debt) — and they are drained in the wrong order. The inline paint renders the *current* model and writes it; `flushOwed` then writes whatever `m.owed` was holding, which is older. Reproduced (last write is the stale one):

```
"…\x1b[2Kone [built]\x1b[0m\x1b8\x1b7\x1b[1;23r\x1b[24;1H\x1b[0m\x1b[2Kone [two]\x1b[0m\x1b8"
```

This directly contradicts `writeOwn`'s own stated invariant: *"the row is a rendering of current state, so the freshest paint is the only one worth landing"*. It is reachable in production through I1's path — `removeTab` mutates the model on the writer goroutine and neither posts a paint nor clears `m.owed`.

**Fix sketch:** have the fresh paint supersede the slot — set `m.owed = nil` when `paintStripInline` writes directly — or call `m.flushOwed()` *before* the inline repaint. Prefer the former; it states the invariant where the invariant lives.

### I3 — M3.7(b) is ticked but the escape-carrying flood was never run

**This is the 4th finding in family `acceptance-command-does-not-hold`.** Do not fix this instance.

M3.7 is `[x]`. Its (b) clause requires flooding a tab with output that *contains escapes* (`ls --color=always`, `tput setaf`) and switching tabs, because that is the only way to request a paint while the child's stream is mid-sequence. The plan's own 2026-09-07 revision records that specifying `yes` was a defect for exactly this reason. Nothing in the Log records (b) being run: the smoke pass predates the strip's current gate, the Ctrl-C investigation used `yes aaa`, and `couchnestedrows` floods with `seq 1 400` — neither emits an escape. So the defer-and-owe path has never been exercised outside unit tests, in the milestone that *widened* the defer condition to include `HoldsCursorSave`.

**The rule:** an acceptance step is ticked only against recorded evidence that the step as written was executed — the command run, the output seen. Three prior findings in this family were "the command does not establish what the step claims"; this one is "the step was never run at all", which is the same rule with the evidence set to empty. The mechanical form: each `M*.*` manual step gets a Log line quoting its command and what was observed, and the milestone-close refuses a ticked manual step with no such line. Until that exists, un-tick M3.7 and either run (b) or move it to M4.3 with the deferral written down.

### I4 — The degraded title's consumer assertion covers one of the two derived consumers, and the sibling producer was never swept (`run.go:1510`, `run.go:1533`, `strip_test.go:242`)

**This is the 3rd finding in family `consumer-set-not-derived`.** Do not fix this instance.

Two measured instances of one rule:

1. **`ClassifyLiveLayout` was never asserted.** M3.6 says *"Assert against the DERIVED consumer set … `RoleForPane` **and `ClassifyLiveLayout`** fed the degraded title"*. `TestTheDegradedTitleStillClassifiesThePane` asserts only `RoleForPane`. Measured:

   ```
   title="terminal 1"      -> ("layout2", true)     # degraded, no command
   title="[terminal 1] work" -> ("layout3", true)   # packed, no command
   ```

   `layoutflow.go:62` matches `Title == "terminal"` or `HasPrefix(Title, "[terminal")`. The degraded title matches neither, so the title-only arm went from "matches when tab 1 is active" to "never matches". In the shipped call path `ProbeLiveLayout` passes `--command`, so the fallback masks it — but the sibling case was treated as a real defect for `RoleForPane`, and `layoutflow_test.go:143` still fixtures `Title: "[terminal 1]"`, a form the code can no longer emit.

2. **`renamePaneTitleLocked` was not degraded at all.** It still packs the whole tab set *and* drops the `terminal ` prefix that `paneTitleLocked` calls load-bearing. Measured:

   ```
   rename title = "[rename: work│] other"
   RoleForPane(no command) = PaneRoleOther
   ```

   So for the duration of every rename, the pane loses the classification that routes global shortcuts — the exact defect M3.6 fixed, at the producer next door. (`RoleForPaneWith`'s registry overlay cushions it, which is why this is Important and not Critical.)

**The rule:** where a milestone changes a value a derived consumer set reads, the test is a table over `{every producer} × {every consumer}`, generated from the recorded derivation — not one hand-picked pair. The plan already records both greps; the test should walk them. That single table catches both instances above and any third producer added later.

### I5 — The gate condition was widened to one that can persist, and M3's assigned flush deadline was not delivered (`run.go:981`, `run.go:939`)

**This is the 2nd finding in family `deferred-work-lacks-own-trigger`.** Do not fix this instance.

`flushOwed`'s comment reads: *"KNOWN GAP, deferred to M3 with the strip … M3 owes it a flush deadline."* No deadline exists (`grep` for a timer in `run.go` finds only `RenameTimer`). Meanwhile `unsafeToPaint` now defers on `HoldsCursorSave` as well as `MidSequence` — and where mid-sequence is inherently transient (a pty chunk boundary), a held save can persist for as long as a child chooses not to restore. `owedDiag` is an unbounded `append`, so a child holding a save while `zellij action` failures keep arriving grows a queue with no ceiling and no drain (ARCH-CONSTRAINTS: unbounded buffering behind a condition with no timeout).

`TestAHeldSaveLeavesTheRowStaleRatherThanCorruptingTheChild` establishes "stale beats wrong" for the *paint*, which is a correct and deliberate call. It says nothing about the *diagnostic*, whose own comment insists every one must land.

**The rule:** a write deferred on a condition the deferrer does not control needs a trigger the deferrer *does* control — a deadline, a bound, or a written argument that unbounded deferral is correct for that payload class. Applied here: state it per payload. Paint → unbounded deferral is correct, and say so where the KNOWN GAP comment currently promises a deadline. Diagnostic → cap the queue or flush it on a deadline, because "failure reported as nothing" is the shape `#208` spent fourteen rounds on.

### I6 — A `zellij` subprocess per rename keystroke, on the keystroke path, now that the strip carries the field (`run.go:390`, `run.go:1173`)

**This is the 4th finding in family `envelope-claim-unenforced`.** Do not fix this instance.

The plan's ARCH-CONSTRAINTS declares: *"the degraded title keeps one spawn on tab switch only, **not on every render**."* `refreshRename` is called once per rename event from the stdin pump and calls `setPaneTitle` → `RunZellijAction("rename-pane", …)` → a process fork per keystroke, on the interaction path the plan names as the one that matters. M3 now also posts a strip paint on the same path. The issue's own Problem statement opens with "A subprocess per title change" as cost #1; the strip was supposed to retire it, and during a rename the pane is by definition focused, so the unfocused-pane label the title exists for is not being read.

**The rule:** an ARCH-CONSTRAINTS budget is a claim that must be enforced by something a change can trip over, not a sentence in the plan. Three prior findings in this family are the same shape. Either the budget gets a test (count `fakeRuntime` ops across a scripted rename and assert ≤ 1), or the budget line is rewritten to state what the code actually does. Since the strip now renders the field, the cheapest true fix is to drop the per-keystroke title update entirely and refresh the title once on rename *commit*.

### I7 — `probes/cursorsaveslots` can force-delete a zellij session it did not create, from `make test-smoke`

Family: `test-can-touch-real-state` (new). ARCH-SECURE, at-review: *tests able to write real user or system state*.

`main.go` diffs `zellij list-sessions` before/after, picks an arbitrary new name (`for name := range after { … break }` over a map), and `defer`s `zellij delete-session <name> --force`. If any other session appears in that 8-second window — the operator launching a workbench, a parallel probe — the probe force-deletes it. This probe sits in `probes/`, which `make test-smoke` runs **wholesale**, so a routine target now carries that blast radius.

The fix was discovered inside this same milestone and not applied here: `cmd/probes/couchnestedrows/main.go:112` uses `--new-session-with-layout` with a deterministic `couchnestedrows-<pid>` name, which is precisely the flag that defeats the "`--session` with `--layout` means attach" problem the diff-the-list approach exists to work around. Port it; failing that, refuse to proceed when more than one new session appears rather than picking one arbitrarily.

### I8 — 196 identical lines between the two `probes/` zellij harnesses (ARCH-DRY)

Family: `copy-instead-of-extract` (new).

`probes/cursorsaveslots/main.go` (287 lines) and `probes/zellijscrollregion/main.go` (281 lines) share **196 identical lines**: `sessionSet`, `syncBuffer`, `tailOf`, `writeLayout`, the `ZELLIJ*` scrub, the pty reader goroutine, `lastCursorRowBefore`, and the session-discovery/cleanup dance. `cmd/probes/couchnestedrows` repeats 83 of them, `termctrlc` 49, `termrows` 45. Nothing blocks extraction here — unlike BR-7's unexported-in-another-package case, a plain `probes/zellijprobe/` package is importable by every `probes/*` main. The concrete cost is already visible: I7 is a bug that exists in the copy and was fixed only in the original's successor, which is what duplication of a *harness* buys you.

## 4. Minor findings

- `run.go:1508` — the `paneTitleLocked` doc comment contradicts itself inside one block: *"the packed title matched shortcut.go's arm NEVER"* twenty lines above *"The packed form happened to keep classifying only because it began with the FIRST tab's default name"*. The plan's 2026-09-08 revision records the first claim as measured-false; the code kept it.
- `run.go:1368-1370` — `childSizeLocked` carries two stacked doc comments, the first of which (*"`pair term` gives its children the whole terminal"*) is now the opposite of what the function does.
- `ptychild/screen.go:141-156` — `HoldsCursorSave`'s doc was inserted into the middle of `TakeRowDirty`'s, so godoc now attaches the whole row-dirty rationale to `HoldsCursorSave` and `TakeRowDirty` has no comment at all.
- Plan `Core concepts` (`plan.md:225-228`) — `TabChip`/`StripModel`/`RenderedStrip`/`RenderStrip` still say `planned — M3` after M3; `RenameField`, `TabSpan`, `hostty.ResetSGR`, `ptychild.Screen.HoldsCursorSave` and the `stripOwed` debt are new entities absent from the table. **6th finding in `plan-table-drift`** — the covering rule is that a hand-maintained table restating the tree drifts every milestone; either derive the table (grep the stated paths for the stated symbols and fail on a mismatch) or stop asserting `status` in it.
- `layoutflow_test.go:143` — the `layout3 tiled terminals after split` fixture asserts `Title: "[terminal 1]"`, a title form no producer emits after this window.
- `README.md:15` — the right pane now loses a row to the strip (`probes/termsmoke` changed `40 100` → `39 100`) and gains a visible tab bar carrying the rename field. Neither is documented. M4 is the natural place, but it should be listed there rather than left to be noticed.

## 5. Test coverage notes

- Mutation-verified by me, all red without their fix: dropping `HoldsCursorSave` from `unsafeToPaint` (3 tests), `ReserveAndPaint`→`Paint` in `stripBytes` (1 test), removing the rename-first width placement (1 test). The claims in the Log about these hold.
- Two behaviours have reproducers but no test in the tree: I1 (tab exit during rename → no repaint) and I2 (stale owed paint after a fresh one). Both scripts are in this review and drop straight into `strip_test.go`.
- `TestPaintDefersMidSequenceAndIsOwed` still splits one hand-chosen sequence at one index (BR-9, still open). The new save-gate tests inherit the same shape. I checked the underlying framer by splitting five representative sequences at *every* index and comparing against the whole-chunk feed — it is correct — but the suite still samples one interleaving and reports it as coverage.
- `probes/cursorsaveslots` exits 2 on `PROBE-INCONCLUSIVE`, and `make test-smoke`'s loop is `go run ./$$p || exit 1`, so a precondition failure fails the smoke suite rather than reporting `n/a`. Worth confirming that is intended, since `#208`'s rule distinguishes the two.

## 6. Architectural notes

Working the markers explicitly:

- **ARCH-DRY** — *flag* (I8: 196 duplicated lines across probe harnesses). Passes strongly in production code: `rowtext` reused at both text ingress points, `ReserveAndPaint` consolidating a two-call sequence both consumers got wrong, `RenameEditor.Field` composing the caret once for two surfaces.
- **ARCH-PURE** — *pass*. `RenderStrip` is a pure function over `(width, StripModel)`; `strip.go` imports only `strings`, `rowtext`, `textwidth`. Every strip unit test runs without a pty. `stripBytes` is a correctly thin seam.
- **ARCH-PURPOSE** — *flag* (I3, I4, I6). The shadow-sweep on the title change finds one producer (`renamePaneTitleLocked`) and one consumer (`ClassifyLiveLayout`) still holding the retired model, and the subprocess-per-title-change that the issue's Problem opens with survives on the rename path.
- **ARCH-MOCK** — *pass with a note*. zellij stays behind `Runtime`; the tty behind the writer recorder; production and test share the boundary. The live conformance layer is genuinely strong (`test-smoke`, `test-couch-nested-rows`, `test-term-ctrlc`, `test-term-rows`). Note for later: `fakeRuntime` is a call *recorder*, not a stateful fake — it cannot model that `rename-pane` changes what a subsequent `list-panes` returns, which is precisely the coupling I4 turns on. Not this milestone's debt, but it is why I4 could not have been caught by the existing double.
- **ARCH-CONSTRAINTS** — *flag* (I5 unbounded `owedDiag` behind an indefinitely-holdable condition, I6 declared spawn budget unenforced). The row-dirty-as-debt change is the right call and correctly argued from couch's policy rather than copied from its mechanism.
- **ARCH-SECURE** — *flag* (I7, a routine target able to destroy a real zellij session). Untrusted text is handled well: operator tab names and the rename field both go through `rowtext` at the single renderer, with C1 controls covered; no credentials in scope.
- **ARCH-ORDER** — *flag* (I1 a transition with no effect, I2 an unwind that lands the wrong effect, BR-9's single-interleaving oracle still open). The `(state, event)` table in the plan is good and the re-`Reserve`-before-repaint rule is correctly identified as the most-likely-wrong thing and correctly implemented; what is missing is that the table's "child exits → repaint" row has an unlisted branch.

For M4: I1's enumeration test is the thing to land *before* the frame comes off, because after M4 the strip is the only surface that reports tab state — a stale strip stops being cosmetic. I4's producer sweep likewise, since M4 removes the frame that currently displays the title.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000199-…-plan.md`:

1. **`Core concepts` status column is stale.** Flip the four `strip.go` rows from `planned — M3` to `new`, and add the entities M3 actually introduced: `RenameField`, `TabSpan` (`termcmd/strip.go`, PURE, new), `RenameEditor.Field` (`termcmd/rename.go`, PURE, new), `hostty.ResetSGR` + `Reservation.ReserveAndPaint` (PURE, new), `ptychild.Screen.HoldsCursorSave` + `cursorSaved` (PURE, new). Record that the table drifted for the sixth time and what will be done about the class.
2. **The Integration-points table needs the debt.** `strip repaint trigger` is listed as `ptychild.OutputBatch.RowDirty`; the delivered mechanism is *row-dirty records a debt, paid by the first chunk that leaves the stream safe*, and the safety condition is now two-part (`MidSequence || HoldsCursorSave`). Add a `paint debt (stripOwed)` row and correct the `paint gate` row's wrapped entity.
3. **M3.7 is ticked ahead of its evidence.** Un-tick it, or split (b) into its own unticked step with the load generator named and a Log line reserved for the observation. Record that `couchnestedrows` floods with `seq 1 400`, which emits no escapes, so it does not stand in for (b).
4. **ARCH-CONSTRAINTS: correct the spawn budget.** *"one spawn on tab switch only, not on every render"* is false while `refreshRename` calls `setPaneTitle` per keystroke. Either state the intended fix (title refreshed on commit only) or state the actual cost.
5. **`flushOwed`'s KNOWN GAP.** M2 assigned the flush deadline to M3; M3 did not deliver it and widened the defer condition. Record the decision per payload class — unbounded deferral is correct for the paint ("stale beats wrong", already tested), and say what happens to a deferred diagnostic.

The issue file needs no revision on the struck scroll-position Done-when — that is correctly recorded at `## Revisions` (2026-09-06), and leaving the original bullet in place is the append-don't-overwrite convention working as intended.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3's acceptance still has no Alt+Shift+d step, so :138/:139, :165/:166, :191/:192 remain unverified.
  - id: BR-9
    disposition: not-addressed
    note: |
      TestPaintDefersMidSequenceAndIsOwed is unchanged and still splits one hand-chosen sequence at one index.
findings:
  - id: new
    severity: Important
    family: exit-path-drops-cleanup
    title: |
      A tab exiting during a rename repaints nothing, so the strip keeps listing the closed tab
    detail: |
      run.go:1296 -- removeTab's `if !preserveRename { m.applyTakeover(...) }` skips the only
      thing that repaints the strip on that path. Reproduced: zero bytes written after tab 1's
      child EOFs while a rename is open. 2nd in this family: do NOT patch removeTab alone.
      The rule is that every site mutating the strip model owes a repaint, and the site set is
      enumerated by a test -- a table over {newTab, switchRelative, closeActive, removeTab (both
      branches), beginRename, refreshRename, finishRename, inheritSize, rowDirty} that drives
      each and asserts a repaint carrying the post-mutation model. It fails on exactly one row today.
  - id: new
    severity: Important
    family: superseded-write-not-dropped
    title: |
      flushOwed writes the coalesced older row after the fresh inline repaint already landed
    detail: |
      run.go:866-872 -- the row-dirty debt repaints inline with the current model, then flushOwed
      writes m.owed, which is older. Reproduced: the recorder's last write is "one [two]" after
      "one [built]". This contradicts writeOwn's own stated invariant that the freshest paint is
      the only one worth landing. Reachable via removeTab's preserveRename branch, which mutates
      the model on the writer goroutine without posting a paint or clearing m.owed. Fix: clear
      m.owed when paintStripInline writes directly, or flushOwed before the inline repaint.
  - id: new
    severity: Important
    family: acceptance-command-does-not-hold
    title: |
      M3.7 is ticked but (b) -- the escape-carrying flood that reaches the defer-and-owe path -- was never run
    detail: |
      4th in this family, so do NOT just run the step. Every load generator in the record emits no
      escapes: the smoke pass predates the current gate, the Ctrl-C probe used `yes aaa`, and
      couchnestedrows floods with `seq 1 400`. So the defer-and-owe path has no manual evidence in
      the milestone that WIDENED the defer condition to include HoldsCursorSave. The rule: a manual
      acceptance step is ticked only against a Log line quoting the command run and what was seen,
      and milestone-close refuses a ticked manual step with no such line. Until then un-tick M3.7.
  - id: new
    severity: Important
    family: consumer-set-not-derived
    title: |
      The degraded title asserts one of two derived consumers, and the sibling producer was never swept
    detail: |
      3rd in this family. Two measured instances of one rule. (1) M3.6 required asserting
      ClassifyLiveLayout as well as RoleForPane; only RoleForPane is asserted, and measured,
      Title="terminal 1" with no command now classifies layout3 as layout2 where the packed
      "[terminal 1] work" classified layout3 -- layoutflow_test.go:143 still fixtures the retired
      "[terminal 1]" form. (2) renamePaneTitleLocked (run.go:1533) was not degraded at all: it still
      packs the tab set and drops the load-bearing prefix, measured as
      "[rename: work-bar] other" -> PaneRoleOther. The rule: where a milestone changes a value a
      derived consumer set reads, the test is a table over every producer x every consumer,
      generated from the derivation the plan already records -- not one hand-picked pair.
  - id: new
    severity: Important
    family: deferred-work-lacks-own-trigger
    title: |
      The defer condition was widened to one that can persist indefinitely, and M3's assigned flush deadline was not delivered
    detail: |
      2nd in this family. run.go:981 states "M3 owes it a flush deadline"; no deadline exists.
      Meanwhile unsafeToPaint (run.go:939) now also defers on HoldsCursorSave, which unlike
      MidSequence can be held for as long as a child chooses, and owedDiag is an unbounded append
      with no cap. The rule: a write deferred on a condition the deferrer does not control needs a
      trigger the deferrer does control -- a deadline, a bound, or a written argument that unbounded
      deferral is correct for that payload class. State it per class: unbounded is right for the
      paint (stale beats wrong, already tested); it is not right for a diagnostic.
  - id: new
    severity: Important
    family: envelope-claim-unenforced
    title: |
      A zellij subprocess forks per rename keystroke, against the plan's declared "not on every render" budget
    detail: |
      4th in this family. refreshRename (run.go:1173), called once per rename event from the stdin
      pump (run.go:390), calls setPaneTitle -> RunZellijAction -> a process fork, on the interaction
      path ARCH-CONSTRAINTS names as the one that matters, and M3 now adds a strip paint alongside
      it. The issue's Problem opens with "a subprocess per title change" as cost #1. The rule: an
      ARCH-CONSTRAINTS budget must be enforced by something a change trips over, not asserted in
      prose -- count fakeRuntime ops across a scripted rename and assert the bound, or rewrite the
      budget to state the real cost. Cheapest true fix: refresh the title on commit only, since the
      strip now renders the field and the pane is focused throughout a rename.
  - id: new
    severity: Important
    family: test-can-touch-real-state
    title: |
      probes/cursorsaveslots force-deletes a zellij session it did not create, from make test-smoke
    detail: |
      It diffs `zellij list-sessions` before/after, picks an arbitrary new name by ranging a map,
      and defers `zellij delete-session <name> --force`. Any session appearing in that 8-second
      window -- the operator's own workbench -- is destroyed. It lives in probes/, which test-smoke
      runs wholesale. The fix was found in this same milestone and not applied: couchnestedrows
      (main.go:112) uses --new-session-with-layout with a deterministic couchnestedrows-<pid> name,
      which is exactly the flag the session-diff dance exists to work around.
  - id: new
    severity: Important
    family: copy-instead-of-extract
    title: |
      196 identical lines duplicated between the two probes/ zellij harnesses
    detail: |
      probes/cursorsaveslots/main.go (287 lines) and probes/zellijscrollregion/main.go (281 lines)
      share 196 identical lines: sessionSet, syncBuffer, tailOf, writeLayout, the ZELLIJ* scrub, the
      pty reader goroutine, lastCursorRowBefore, and the session discovery/cleanup. couchnestedrows
      repeats 83, termctrlc 49, termrows 45. Nothing blocks extraction -- a plain probes/zellijprobe/
      package is importable by every probes/* main, unlike BR-7's unexported-in-another-package case.
      The cost is already realised: the finding above is a bug present in the copy and fixed only in
      the copy's successor.
  - id: new
    severity: Minor
    family: doc-states-planned-as-current
    title: |
      Three superseded doc comments left standing beside their corrections
    detail: |
      2nd in this family, so state the rule rather than editing three sites: when a comment is
      superseded, DELETE it -- do not prepend the correction, and never leave two doc comments on
      one declaration. Sites: run.go:1508 (paneTitleLocked claims the packed title matched
      shortcut.go's arm NEVER, twenty lines above the paragraph saying it did, and the plan records
      the first as measured-false); run.go:1368-1370 (childSizeLocked has two stacked doc comments,
      the first now the opposite of the behaviour); ptychild/screen.go:141-156 (HoldsCursorSave's
      doc was inserted mid-comment, so godoc attaches TakeRowDirty's whole rationale to it and
      TakeRowDirty has none). A vet-style check for two comment blocks on one declaration catches
      the class.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      The Core concepts table still says "planned — M3" and omits five entities M3 introduced
    detail: |
      6th in this family. plan.md:225-228 still marks TabChip/StripModel/RenderedStrip/RenderStrip as
      planned; RenameField, TabSpan, RenameEditor.Field, hostty.ResetSGR/ReserveAndPaint and
      Screen.HoldsCursorSave are absent, as is the stripOwed debt from the integration table. The
      rule, since instance-fixing has not held five times: a hand-maintained table restating the
      tree drifts every milestone. Either derive it -- a test that greps each row's stated path for
      its stated symbol and fails on a status that disagrees with the diff -- or drop the status
      column so the table stops making a claim nothing checks.
  - id: new
    severity: Minor
    family: user-facing-change-undocumented
    title: |
      README does not mention that the right pane loses a row to the strip or gains a tab bar
    detail: |
      README.md:15 describes the pane's tabs but not the strip. probes/termsmoke changed its
      assertion from "40 100" to "39 100": the child is now one row shorter, which is observable to
      anything the operator runs in the pane. M4 is the natural place to land both, but it should be
      listed there rather than left to be noticed.
```

---

## Re-review — 2026-09-08T14:02:38-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 244a72a5594155bfda2d980f0e84c591648a76a8..61e58395914ebb2c7c1f6c07ebcd6062d0f3d358 |
| command | sdlc milestone-close --issue 199 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-08T14:02:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Eight of the eleven open findings are genuinely closed, and several are closed *structurally* rather than at the site — `renamePaneTitleLocked` deleted outright (one title producer, zero subprocesses per keystroke), `probes/zellijprobe` making "a probe can only destroy a session it named" unrepresentable, and a go/ast pass that makes a new strip-model mutator announce itself. I mutation-verified BR-45, BR-46 and BR-48's consumer half by reverting each in a scratch copy and watching the named test go red. What keeps this off SHIP is that two of the fixes stopped one level short of the class they claim, and both are measured, not argued: `paneTitleLocked` still decides "does this name already classify?" with a *restatement* of `RoleForPane`'s predicate rather than by asking it, so a tab renamed `terminals` produces a title that classifies as `PaneRoleOther` — the exact operator-visible consequence BR-48 described; and a **background** tab's `batch.RowDirty` repaints the strip and re-asserts DECSTBM over the active child's margins, against the plan's own declared paint budget, for a screen the terminal never saw. BR-53 is not addressed: the superseded sentence was left standing in `paneTitleLocked`'s godoc with a correction appended twenty lines below, which is precisely what that finding's rule forbids. Everything else — full `go test ./cmd/... ./probes/...` — is green apart from the documented pty-sandbox class.

## 1. Strengths

- **The largest fix is a deletion.** `renamePaneTitleLocked` (was `run.go:1533`) is gone rather than patched, which closes BR-48 and BR-50 with one change: one title producer, and `TestARenameCostsExactlyOneZellijSubprocess` (`run_test.go:461`) drives three keystrokes and asserts `rt.ops == "rename-pane terminal work"` — a budget with a test instead of a sentence in a plan.
- **BR-45's class fix needs both halves and says so.** `stripmutation_test.go:174` (go/ast over `run.go`, `m.tabs`/`m.active`/`m.rename`) plus `:147` (a table driving each mutator). I reverted `removeTab`'s `paintStripInline()` and only the table row went red — the static half really would have passed the defect, exactly as the file's own comment claims.
- **`probes/zellijprobe` makes the safety property structural.** `Start` names the session, `Close` deletes *that* name; there is no path to `delete-session` a name the probe did not choose. I swept the tree: every `delete-session` under `probes/` and `cmd/probes/` now targets a self-chosen name (`zellijpark`'s constant, `couchnestedrows-<pid>`). 287→172 and 281→165 lines, as claimed.
- **`hostty.ReserveAndPaint` (`reserve.go:117`) fixes a latent couch bug at the shared primitive**, with the ordering argument (DECSTBM homes the cursor, so save must precede it) written where the next reader will hit it.
- **Atlas and README both updated in-range** — the nested-reservation measurement, the two-condition gate, the row-dirty debt, and the "your pane is one row shorter" note a reader running `stty size` will want.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/internal/termcmd/run.go:1557` — the producer restates the consumer's predicate instead of asking it.**
`paneTitleLocked` prefixes with `terminal ` only when `!strings.HasPrefix(name, "terminal")`. That is an approximation of `RoleForPane`'s `title == "terminal" || HasPrefix(title, "terminal ")`. Measured against the real consumer:

```
tab "terminalwork" -> title "terminalwork"  -> PaneRoleOther
tab "terminal-2"   -> title "terminal-2"    -> PaneRoleOther
tab "terminals"    -> title "terminals"     -> PaneRoleOther
```

So a tab renamed `terminals` costs the pane its global shortcuts whenever `TerminalCommand == ""` — the `zellijpane.go:79-84` case the plan names as the whole reason the prefix is load-bearing. `TestEveryPaneTitleProducerSatisfiesEveryConsumer` cannot see it: it enumerates *consumers* but its three producer rows are hand-picked names. Fix: export the predicate from `workbenchshortcut` and have `paneTitleLocked` prefix iff the bare name does not already classify, and generate the table's producer rows over the predicate's boundary cases.

**(b) `cmd/internal/termcmd/run.go:850-871` — a background tab's `rowDirty` repaints the strip.**
`if chunk.rowDirty { m.stripOwed = true }` sits *outside* `if m.isActive(chunk.id)`. Measured with the repo's own harness (background tab id 1, active id 2):

```
m.output <- ptyChunk{id: 1, data: []byte("\x1b[2J"), rowDirty: true}
→ "\x1b7\x1b[1;23r\x1b[24;1H\x1b[0m\x1b[2Kone [two]\x1b[0m\x1b8"
```

A tab the terminal never saw drove a full re-`Reserve` + repaint. Two costs: the plan's ARCH-CONSTRAINTS budget ("a repaint happens on tab change, resize, and `batch.RowDirty`; **not** per output chunk") is violated by any background child that erases, on the keystroke path; and re-asserting `\x1b[1;23r` clobbers the *active* child's own margins for no reason. This is M2's BR-35 lesson ("the gate models the terminal — feed it exactly what the terminal is shown") applied to the repaint trigger rather than the gate.

**(c) `workshop/plans/…-plan.md:239` — the Core concepts table declares `ResetSGR` at `cmd/internal/hostty/reserve.go`; it is defined at `cmd/internal/hostty/control.go:41`.**
Same table, `right pane chrome` carries status `modified` while `main-3.kdl` has no terminal-pane `borderless=true` yet (M4). Not Critical — the symbol exists and is exported from the declared package, so no consumer is misled about behaviour; what is unchecked is the table's own claim. `TestNoPlannedRowSurvivesItsTickedMilestone` cannot catch either, because it only regexes `planned — Mx`, and `couchtty`'s contract filters #199's rows to `cmd/internal/couchtty/` paths — so no test reads a single one of M3's twelve new rows.

## 4. Minor findings

- `run.go:1536` / `run.go:1551` — BR-53's named site is unchanged in substance: the godoc still asserts "the packed title matched shortcut.go's arm NEVER" and the body's parenthetical says it was measured false. `run.go:1566-1579` (`redrawTab`) carries three stacked doc paragraphs, two of them contradicting opening sentences; the proposed vet-style guard for two comment blocks on one declaration was not added.
- `strip.go` / `run.go:1428` — when the renamed tab itself exits, `RenameField.Tab` resolves to `-1` and the in-progress field vanishes from the row. The deleted `renamePaneTitleLocked` had an explicit `if !found { append("[rename: …]") }` branch for exactly this; the move to the strip dropped it. Measured: after the renamed tab's EOF the row is `[one]`.
- `stripmutation_test.go:53` — `stripRepaintCase.needsPty` is set at one site and read at zero (`mustNewTab`'s `t.Skipf` does the work). `:325-326` — `var _ = io.Discard` / `var _ ptychild.Size` keep two otherwise-unused imports alive.
- `strip.go:141` — `rowtext.Fit` truncating a `[rename: …]` label drops the closing bracket; cosmetic only.

## 5. Test coverage notes

The new suite is strong where it counts: `strip_test.go` drives ≥2 tabs everywhere (the standing BR-35 rule), pins spans as display columns with a non-ASCII fixture, and covers escape injection from both the tab name and the rename field. `TestNoPaintLandsInsideTheChildsCursorSave` and `TestARowDirtyBatchDefersWhileTheChildHoldsASave` pin the widened gate deterministically. Gaps: nothing counts paints, which is why (b) is invisible; the producer × consumer table's producer axis is three fixtures rather than a corpus, which is why (a) is invisible; and `TestEveryStripModelMutationRepaintsTheRow/newTab` silently `Skip`s without a pty while two sibling tests hard-fail in the same environment.

## 6. Architectural notes

ARCH-DRY **pass** (196 duplicated probe lines extracted; `ResetSGR` folded into `HomeAndClear` so both callers get it; `ClassifyLiveLayout` derives from `RoleForPane`). ARCH-PURE **pass** — `RenderStrip` is a pure `(width, model) → (row, spans)` and its hard cases are unit tests. ARCH-PURPOSE **flag (a)** — the single-source sweep reached the consumers but not the producer's own copy of the predicate. ARCH-MOCK **pass** — zellij behind `Runtime`/`fakeRuntime`, the tty behind the recorder, live conformance in `make test-smoke` and `make test-couch-nested-rows`. ARCH-CONSTRAINTS **flag (b)** — one declared budget is now enforced by a test, the other is violated in the tree. ARCH-SECURE **pass** — operator text sanitized at the row for both surfaces, and BR-51's blast-radius hole closed structurally. ARCH-ORDER **flag** — `applyTakeover`'s reset ordering and the `owed`/`stripOwed` interaction are now coherent and pinned, but `TestPaintDefersMidSequenceAndIsOwed` still samples one interleaving at one hand-chosen split index (BR-9).

For M4: the `main-3.kdl` nine-site enumeration test is the right shape; carry (a) with you, because taking the frame off removes the last non-strip surface that could hint the pane lost its role.

## 7. Plan revision recommendations

- `## Revisions` entry correcting the Core concepts table: `ResetSGR` lives in `cmd/internal/hostty/control.go`, not `reserve.go`; and either give `right pane chrome` a `planned — M4` status (which the new guard then polices) or say what `modified` means before its milestone ships.
- `## Revisions` entry against ARCH-CONSTRAINTS: the "no paint on the common path" budget is currently false for background-tab output; state the corrected rule (row-dirty is a signal about the *active* tab only) and name the test that will enforce it.
- M4.3 still has no `Alt+Shift+d` step (BR-8) and `TestPaintDefersMidSequenceAndIsOwed` is still one split index (BR-9); both are recorded as deferred to M4 in the 2026-09-08 revision, which is fine — leaving them named there is the ask.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 still has no Alt+Shift+d step; the plan records it as deferred to M4.
  - id: BR-9
    disposition: not-addressed
    note: |
      TestPaintDefersMidSequenceAndIsOwed is byte-identical and still splits one sequence at one index.
  - id: BR-45
    disposition: addressed
    note: |
      Mutation-verified: removing removeTab's paintStripInline turns the rename-branch table row red.
  - id: BR-46
    disposition: addressed
    note: |
      Mutation-verified: removing writeOwn's `m.owed = nil` makes the stale row land after the fresh one.
  - id: BR-47
    disposition: addressed
    note: |
      Automated in couchnestedrows with an SGR-per-line flood plus tab switches; could not re-run here (needs zellij + a real pty).
  - id: BR-48
    disposition: addressed
    note: |
      Second producer deleted, ClassifyLiveLayout derives from RoleForPane, table mutation-verified; see the new finding for the producer's own restatement of the predicate.
  - id: BR-49
    disposition: addressed
    note: |
      Decided per payload; owedDiag capped at maxOwedDiag with a test, unbounded paint deferral argued and pinned.
  - id: BR-50
    disposition: addressed
    note: |
      TestARenameCostsExactlyOneZellijSubprocess asserts exactly one recorded op across three keystrokes.
  - id: BR-51
    disposition: addressed
    note: |
      zellijprobe.Start names and Close deletes that name; swept the tree, every delete-session under probes/ and cmd/probes/ targets a self-chosen name.
  - id: BR-52
    disposition: addressed
    note: |
      Verified by line count -- cursorsaveslots 172, zellijscrollregion 165, shared harness 219.
  - id: BR-53
    disposition: not-addressed
    note: |
      paneTitleLocked's godoc still carries the measured-false claim with a correction appended below it, and redrawTab still stacks three doc paragraphs; no class guard added.
  - id: BR-54
    disposition: addressed
    note: |
      Table flipped and extended and the planned-Mx rot guarded; residual noted in the new plan-table-drift finding.
  - id: BR-55
    disposition: addressed
    note: |
      README now documents the strip and the row the pane gives up.
findings:
  - id: new
    severity: Important
    family: consumer-set-not-derived
    title: |
      paneTitleLocked restates RoleForPane's predicate instead of asking it, so a tab named `terminals` classifies as PaneRoleOther
    detail: |
      This is the 4th finding in family `consumer-set-not-derived`, so state the rule rather than
      patching the prefix. Measured against the real consumer at run.go:1557 --
      tab `terminalwork` -> title `terminalwork` -> PaneRoleOther; same for `terminal-2` and
      `terminals`. The guard is `HasPrefix(name, "terminal")`, an approximation of
      `title == "terminal" || HasPrefix(title, "terminal ")`, so the pane loses the classification
      that routes global shortcuts whenever TerminalCommand is unavailable -- the exact consequence
      BR-48 described, one level upstream of where BR-48 was fixed.
      THE RULE: a producer must never restate the predicate its consumer applies; it ASKS the
      consumer. Export the predicate from workbenchshortcut and have paneTitleLocked add the prefix
      iff the bare name does not already classify -- then the disagreement is unrepresentable.
      And the producer x consumer table needs its PRODUCER axis derived too: three hand-picked names
      cannot see a boundary case, which is why this survived a table written specifically for BR-48.
  - id: new
    severity: Important
    family: envelope-claim-unenforced
    title: |
      A background tab's rowDirty batch repaints the strip and re-asserts DECSTBM over the active child's margins
    detail: |
      This is the 5th finding in family `envelope-claim-unenforced`, so state the rule rather than
      moving the one `if`. run.go:850 sets `m.stripOwed = true` outside the `m.isActive(chunk.id)`
      guard, so a tab whose bytes never reached the terminal drives a full re-Reserve plus repaint.
      Measured with the repo's own harness: `ptyChunk{id: 1, data: "\x1b[2J", rowDirty: true}` on the
      BACKGROUND tab wrote `\x1b7\x1b[1;23r\x1b[24;1H\x1b[0m\x1b[2Kone [two]\x1b[0m\x1b8`. That
      violates the plan's declared budget ("a repaint happens on tab change, resize, and
      batch.RowDirty; NOT per output chunk") on the keystroke path, and clobbers the ACTIVE child's
      own scroll region for a screen the terminal never saw -- M2's BR-35 lesson (the gate models the
      terminal, so feed it exactly what the terminal is shown) applied to the repaint trigger rather
      than to the gate.
      THE RULE: every budget bullet in the plan's ARCH-CONSTRAINTS block is an enumeration, and each
      entry owes a test that trips when it is exceeded. Two of the four have one now
      (TestARenameCostsExactlyOneZellijSubprocess, TestOnlyOneGoroutineWritesTheHost); the paint-
      frequency budget is the one with no test, and it is the one the tree violates. Count paints
      across a scripted background-tab flood and assert the bound, rather than asserting the bound in
      prose next to a mechanism that does not honour it.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      The Core concepts table declares ResetSGR at hostty/reserve.go; it is defined at hostty/control.go:41
    detail: |
      This is the 7th finding in family `plan-table-drift`. Do NOT fix the row -- the rule is what is
      missing. plan.md:239 states `ResetSGR / ReserveAndPaint | cmd/internal/hostty/reserve.go`;
      ReserveAndPaint is there, ResetSGR is at control.go:41. Same table, `right pane chrome` carries
      status `modified` while main-3.kdl still has no terminal-pane borderless=true (M4 unshipped).
      Rated Important, not Critical: the symbol exists and is exported from the declared package, so
      no consumer is misled about behaviour -- what is unchecked is the table's own claim.
      THE RULE, and it is the half of BR-54 that was not delivered: the new guard
      (TestNoPlannedRowSurvivesItsTickedMilestone) only regexes `planned - Mx` after Mx is ticked,
      and couchtty's contract filters #199's rows to `cmd/internal/couchtty/` paths -- so NO test
      reads a single one of M3's twelve new rows. Either grep each row's stated path for its stated
      symbol (which is what BR-54 asked for and what would have caught this in the same commit that
      wrote it), or drop the path and status columns so the table stops making claims nothing checks.
  - id: new
    severity: Minor
    family: moved-surface-drops-a-case
    title: |
      When the renamed tab itself exits, the in-progress rename field vanishes from the row
    detail: |
      Measured: with a rename open on the active tab, driving removeTab on that tab's id leaves the
      row as `[one]` -- no field at all -- while the stdin pump goes on routing keystrokes into the
      editor. stripModelLocked (run.go:1428) resolves the missing tab to `Tab: -1` and the renderer
      treats that as "mark nothing". The deleted renamePaneTitleLocked had an explicit
      `if !found { append("[rename: "+field+"]") }` branch for exactly this case; the move from the
      title to the strip did not carry it. The commit is a no-op afterwards, so the cost is brief
      blind typing rather than data loss -- but it is a branch lost in a move that was presented as a
      move.
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      stripRepaintCase.needsPty is set at one site and read at zero, and two imports are kept alive by blank vars
    detail: |
      This is the 2nd finding in family `dead-test-scaffolding`, so state the rule: a test-harness
      field or declaration that no assertion reads is scaffolding that reads as protection.
      stripmutation_test.go:53 declares `needsPty`, :80 sets it, nothing reads it -- mustNewTab's
      t.Skipf does the work. :325-326 carry `var _ = io.Discard` and `var _ ptychild.Size`, which
      exist only to keep two otherwise-unused imports compiling. Delete the field and the imports;
      a struct field with no reader is the same shape as the "field set at zero call sites" the
      claimed-fix check exists to catch.
```

---

## Re-review — 2026-09-08T14:26:24-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 244a72a5594155bfda2d980f0e84c591648a76a8..07f8160c107f8f9aff828181bb05f59c5c94c65d |
| command | sdlc milestone-close --issue 199 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-08T14:26:24-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 delivers the strip, and the round-13 fixes hold up under adversarial checking: I reverted BR-56, BR-57 and BR-59 in a compiler overlay and each turned its test red, so those are genuinely pinned rather than plausibly-fixed. `make test` is green apart from the documented pty/fork-exec sandbox class (`TestTerminalMuxNewTab*`, `stripmutation` `newTab`, `ptychild/child_test.go`, `hostty` OS-host conformance), and `-race` is clean. No Critical: I found no correctness defect in the strip renderer, the two-condition gate, the debt machinery or the degraded title. What blocks a clean SHIP is three things the round's *class* fixes stopped short of — a guard that hard-fails the whole suite the moment this plan is archived at close, and two enumerations (the ARCH-CONSTRAINTS budget, the ARCH-ORDER transitions) where I measured that deleting a live behaviour leaves the suite green.

**1. Strengths**

- `hostty/reserve.go:117` `ReserveAndPaint` fixes the DECSTBM-homes-the-cursor bug at the *shared* primitive, and `TestSaveComesBeforeTheRegionChangeThatHomesTheCursor` asserts the **order** rather than the presence of the bytes — the only assertion shape that could have caught it. Folding `ResetSGR` into `HomeAndClear` (`control.go:41`) rather than asking two callers to remember it is the right altitude.
- `ptychild/screen.go:161` `HoldsCursorSave` is one bit, not a stack, with the alt-screen/RIS abandonment cases tested. I additionally checked a case the suite does not: `\x1b` at the end of one chunk and `7` at the start of the next is still observed as a save — the framer handles it, so the gate does not go blind on a pty read boundary.
- `strip_test.go:301` `TestEveryPaneTitleProducerSatisfiesEveryConsumer` generating its producer axis over the predicate's *boundary cases* is the fix that actually retires BR-48/BR-56. Mutation-verified: restoring `HasPrefix(name, "terminal")` fails on `terminals`, `terminal-2` and `terminalwork` against **both** consumers.
- `stripmutation_test.go:299` `TestOnlyTheActiveTabsOutputDirtiesTheRow` asserts the region is *not* re-asserted as well as that no row was drawn — a spurious repaint stamps on the active child's margins, and the test says so. Mutation-verified red.
- `probes/zellijprobe` makes "a probe can only delete a session it created" structural (`Start` names it, `Close` deletes that name), and `cmd/probes/couchnestedrows:396` independently honours the same property with a pid-derived name and a temp `PAIR_DATA_DIR`.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/internal/termcmd/plantable_test.go:133`** — `TestEveryCoreConceptRowNamesASymbolThatExists` hardcodes `workshop/plans/000199-…-plan.md` and `t.Fatal(err)`s if it cannot read it. `sdlc close` archives plans to `workshop/history/plans/` (39 already there), so `make test` breaks for the whole repo at M4.5, the very next gate. The repo already resolves this — `couchtty/core_concepts_contract_test.go:272` `findConceptPlans` looks in `workshop/plans` then walks `workshop/history`. 2nd in `moved-surface-drops-a-case`: the new guard is a deliberate re-implementation ("NOT a copy of that machinery") that dropped a case the original had. The rule, not the path: one shared plan resolver, or every artifact-reading guard states active-or-archived.
- **`cmd/internal/termcmd/run.go:863` and `run.go:1387`** — 6th in `envelope-claim-unenforced`. BR-57's rule ("every budget bullet is an enumeration, and each entry owes a test that trips when it is exceeded") was stated but only the background-tab entry was delivered. Measured, both with the whole `termcmd` suite: changing `if chunk.rowDirty` to `if true` — a repaint on **every** active-tab output chunk, the literal violation of ARCH-CONSTRAINTS bullet 1 — is green; deleting `m.paintStripInline()` from `inheritSize` — the ARCH-ORDER row "host resize → re-`Reserve`, re-`Paint`" — is green. (For the enumeration's sake: replacing `restoreTerminal`'s `ResetRegion` write, ARCH-ORDER's "`Release` on every exit path", is green too; that one predates the window.) The bound is a *count over a stream*, and no test counts one.
- **`cmd/internal/termcmd/stripmutation_test.go:199` and `doccomment_test.go:86`** — 5th in `acceptance-command-does-not-hold`. Two guards written this round to close classes both check less than they claim. `parser.ParseFile(fset, "run.go", …)` enforces "every mutator announces itself" over one of the package's five non-test files. `declDoc` returns nothing when a `GenDecl` has more than one Spec, so `hostty/control.go:19-80` — a single grouped `const (…)` block, and the file where M3 rewrote `ResetSGR` and `HomeAndClear`'s doc comments — is entirely unread by the guard written for that exact class.

**4. Minor findings**

- `cmd/internal/termcmd/rename.go:64` — `Field`'s doc justifies itself by "there are now two surfaces … the tab strip and the degraded pane title"; the title's copy was deleted in this same window and `Field()` has one production caller (`run.go:1441`).
- `cmd/internal/hostty/reserve.go:117,147` — `ReserveAndPaint` and `Paint` duplicate the save/move/reset/clear/text/reset/restore body; `TestReserveAndPaintAgreesWithItsParts` only checks the region substring, so a change to one will not propagate. `Reservation.Reserve()` now has zero production callers.
- `cmd/internal/workbenchshortcut/shortcut.go:198` — `TitleIdentifiesRightTerminal` is new exported surface in a shared package with no Core-concepts row and no atlas mention; nothing checks the code→table direction.

**5. Test coverage notes**

Behavioural coverage of the new strip is strong where it was attacked (rename steps, detached field, narrow-width priority, gate deferral, takeover, row-dirty debt, supersession) and I confirmed four of them fail on revert. The gap is uniformly on the *negative* side: the suite asks "did a repaint happen" almost everywhere and "did one happen that should not have" in exactly one place. The two green mutations in Important 2 are both negative assertions nothing makes. Separately, `TestEveryStripModelMutationRepaintsTheRow/newTab` is the only case in the new table that needs a real pty, so in a sandboxed run that row measures nothing — the hard-fail is the right call, but it means the `newTab` repaint is unverified wherever fork/exec is denied.

**6. Architectural notes**

- ARCH-DRY — flag (Minor, `reserve.go:117/147`). Otherwise good: `rowtext` is the one row-safety strip, `TitleIdentifiesRightTerminal` is the one title predicate, `zellijprobe` retired 196 duplicated lines.
- ARCH-PURE — pass. `RenderStrip` is a pure `(width, model) → (row, spans)` tested with hand-built models; `stripBytes`/`paintStrip*` are the thin seam.
- ARCH-PURPOSE — pass for M3. The deletion of `renamePaneTitleLocked` is the class fix, not the instance, and M4 (the frame coming off) is still the declared deliverable rather than a follow-up.
- ARCH-MOCK — pass. zellij behind `Runtime`, the tty behind `writerRecorder`; `couchnestedrows` and `termsmoke` are the live conformance checks, and `termsmoke`'s `40 100 → 39 100` edit is that conformance actually catching the change.
- ARCH-CONSTRAINTS — flag (Important 2). The declared keystroke-path budget is not enforced.
- ARCH-SECURE — pass. Operator text reaches the row only through `rowtext`; diagnostics sanitize at the single egress; the probe's session deletion can no longer touch a session it did not create.
- ARCH-ORDER — flag (Important 2, resize row). The state carried between events is small and lives on one goroutine; `-race` is clean. The pre-existing single-interleaving weakness in `TestPaintDefersMidSequenceAndIsOwed` is BR-9, still open.

**7. Plan revision recommendations**

- A `## Revisions` entry recording that the ARCH-CONSTRAINTS budget block and the ARCH-ORDER transition table are now **enumerations with an owed test each**, naming which entries have one (subprocess budget, single-writer, row-dirty trigger, defer-and-owe) and which do not (paint frequency per chunk, resize, release-on-exit) — so the gap is a tracked list rather than a prose claim.
- The Core concepts table needs no row edits (the new guard passes), but it is missing `unsafeToPaint`, `maxOwedDiag` and `workbenchshortcut.TitleIdentifiesRightTerminal`; add them or record in the same Revisions entry that code→table completeness is pair#188's.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 still has no Alt+Shift+d step; the plan's own revision defers it to M4.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:117 still splits one hand-chosen sequence at one index.
  - id: BR-53
    disposition: addressed
    note: |
      Three sites rewritten and TestNoDeclarationCarriesTwoStackedGodocs added; its scope is Important 3.
  - id: BR-56
    disposition: addressed
    note: |
      Mutation-verified: restoring HasPrefix(name,"terminal") fails on terminals/terminal-2/terminalwork for both consumers.
  - id: BR-57
    disposition: addressed
    note: |
      Instance fixed and mutation-verified; the rule it stated is only half delivered, raised anew below.
  - id: BR-58
    disposition: addressed
    note: |
      Row corrected to control.go and the guard checks a DEFINITION, not a mention.
  - id: BR-59
    disposition: addressed
    note: |
      Mutation-verified: forcing detached=false reddens TestARenameWhoseTabExitedStaysOnTheRow and a stripRepaintCase.
  - id: BR-60
    disposition: addressed
    note: |
      needsPty and both blank vars are gone; no `var _` remains in the package's tests.
findings:
  - id: new
    severity: Important
    family: moved-surface-drops-a-case
    title: |
      The Core-concepts guard hardcodes the ACTIVE plan path, so make test breaks the moment this plan is archived
    detail: |
      plantable_test.go:133 reads workshop/plans/000199-...-plan.md and t.Fatal(err)s on failure.
      sdlc close archives plans to workshop/history/plans/ (39 are already there), so the whole
      repo's suite goes red at M4.5, the next gate. This is the 2nd finding in family
      moved-surface-drops-a-case, so state the rule rather than editing the path: the repo
      ALREADY resolves a plan active-or-archived (couchtty/core_concepts_contract_test.go:272,
      findConceptPlans walks workshop/history when workshop/plans misses), and this guard is a
      deliberate re-implementation that dropped that case. THE RULE - a guard that reads a
      workshop artifact resolves it through one shared active-or-archived resolver; a guard that
      hardcodes a path under workshop/plans is asserting the issue will never close.
  - id: new
    severity: Important
    family: envelope-claim-unenforced
    title: |
      The paint-frequency budget and the resize transition still have no test that trips when exceeded
    detail: |
      This is the 6th finding in family envelope-claim-unenforced. BR-57 stated the rule -- every
      budget bullet is an enumeration and each entry owes a test that trips when exceeded -- and
      the round delivered the test for exactly the entry the finding named. Measured against the
      full termcmd suite with a compiler overlay - (a) run.go:863, changing `if chunk.rowDirty` to
      `if true`, i.e. a repaint on EVERY active-tab output chunk, which is the literal violation of
      ARCH-CONSTRAINTS bullet 1 ("a repaint happens on tab change, resize, and batch.RowDirty; NOT
      per output chunk"), leaves the suite green; (b) deleting m.paintStripInline() from
      inheritSize (run.go:1387), the ARCH-ORDER row "host resize -> re-Reserve, re-Paint", also
      green -- after which a resized pane keeps a scroll region computed from the OLD row count.
      For the enumeration's sake, replacing restoreTerminal's ResetRegion write (ARCH-ORDER's
      "Release on every exit path", pre-dating this window) is green too. THE RULE - write the
      enumeration itself as a table in the test file, one row per ARCH-CONSTRAINTS bullet and per
      ARCH-ORDER transition, each owning an assertion that fails when its bound is exceeded. The
      paint bound is a COUNT over a scripted chunk stream, not a boolean "did a paint happen":
      TestOnlyTheActiveTabsOutputDirtiesTheRow is the only test in the package that asks the
      negative question, and it asks it about one chunk.
  - id: new
    severity: Important
    family: acceptance-command-does-not-hold
    title: |
      Both new structural guards check less than they claim - one file of five, and no grouped declaration
    detail: |
      This is the 5th finding in family acceptance-command-does-not-hold, so state the rule rather
      than widening two globs. stripmutation_test.go:199 does parser.ParseFile(fset, "run.go", nil, 0)
      while its own doc says it "is what makes a method added later announce itself" -- the package
      has five non-test .go files, and a terminalMux method added in strip.go, rename.go or a new
      file assigns m.tabs/m.active/m.rename invisibly. doccomment_test.go:86 returns no name for a
      GenDecl with more than one Spec, so cmd/internal/hostty/control.go:19-80 -- ONE grouped
      const block, and the file where M3 rewrote ResetSGR's and HomeAndClear's doc comments -- is
      not inspected at all by the guard written for exactly that class; its `checked == 0` floor
      still passes because the file's two funcs are counted. The plan's own 2026-09-07 revision
      already records this shape ("it read run.go only, while M3 adds strip.go to the same
      package") as one of three gaps that made M2's scan insufficient. THE RULE - a structural
      guard enumerates the package's non-test files and descends into grouped declarations, or its
      doc names the narrower scope it actually has, so nobody reads it as covering the class.
  - id: new
    severity: Minor
    family: doc-states-planned-as-current
    title: |
      RenameEditor.Field's doc justifies itself by two surfaces; the second was deleted in the same window
    detail: |
      rename.go:64 says "One function, because there are now two surfaces -- the tab strip and the
      degraded pane title -- and a caret drawn two ways is a caret that disagrees with itself".
      renamePaneTitleLocked is gone and Field() has exactly one production caller (run.go:1441).
      3rd in family doc-states-planned-as-current. The rule cannot be mechanised the way the
      stacked-godoc one was -- TestNoDeclarationCarriesTwoStackedGodocs catches a doc RESTARTED,
      not a single opener whose stated fact died -- so record the prevalence instead: when a
      commit deletes a producer, every doc that cited it as a live reason is part of that commit.
  - id: new
    severity: Minor
    family: copy-instead-of-extract
    title: |
      ReserveAndPaint copies Paint's body rather than composing it, and Reserve() now has no production caller
    detail: |
      2nd in family copy-instead-of-extract. reserve.go:117 and :147 differ only by the SetRegion
      insertion; TestReserveAndPaintAgreesWithItsParts checks only that the region substring is
      present, so a change to Paint (a hide-cursor, a different erase) silently will not reach
      ReserveAndPaint. THE RULE - extract the shared tail (MoveTo + ResetSGR + ClearLine + text +
      ResetSGR) and let both spell only what differs. Separately, Reservation.Reserve() has zero
      production callers now that couch and termcmd both use ReserveAndPaint.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      A new exported symbol in a shared package has no Core-concepts row, and nothing checks that direction
    detail: |
      This is the 8th finding in family plan-table-drift. workbenchshortcut.TitleIdentifiesRightTerminal
      (shortcut.go:198) is new exported surface in a package two other components consume, with no
      row in the plan's table and no mention in atlas/architecture.md's paragraph about the
      predicate. The round-12 fix built the table->code direction
      (TestEveryCoreConceptRowNamesASymbolThatExists); nothing checks code->table. THE RULE - the
      table needs both directions, the machinery exists (couchtty's conceptInventory), and it is
      scoped to one package; widening it is pair#188. Until then, say so in the plan rather than
      leaving the table's completeness as an unstated claim.
```

---

## Re-review — 2026-09-08T14:46:06-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 244a72a5594155bfda2d980f0e84c591648a76a8..fc318eb9ef4d50eeba1e6c4e273364af3b6b7c13 |
| command | sdlc milestone-close --issue 199 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-08T14:46:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 13's three claimed fixes were checked by mutation rather than by reading the commit message, and two of the three hold up completely: BR-62's paint-budget enumeration goes red on all three mutations the previous round measured green, and BR-63's two structural guards now catch the cases they were missing (a mutator added to `strip.go`, a stacked godoc inside `hostty/control.go`'s grouped `const` block). BR-61 does **not** hold: the round stated the rule — "a guard that hardcodes a path under `workshop/plans` is asserting the issue will never close" — swept the two Go guards, and left `tests/plan-superseded-facts-test.sh:45`, which is wired into `make test` and hardcodes that exact path. I moved the plan to `workshop/history/plans/` in a scratch clone: the two fixed guards stay green and the shell script exits 1 with seven failures. That is BR-61's own failure mode, at BR-61's own gate (M4.5), in the third instance of the class the round claimed to close. Fixing it is one `resolvePlan`-shaped change to a shell script, so this is FIX-THEN-SHIP rather than REWORK — the M3 feature code itself is in good shape, `-race` clean, and everything that fails locally is the documented pty sandbox class.

## 1. Strengths

- **`paintbudget_test.go:29` turns three sentences into three executable bounds.** I re-ran the review's own mutations in a scratch clone: `if chunk.rowDirty` → `if true` (run.go:863) now fails two subtests; deleting `m.paintStripInline()` from `inheritSize` fails the resize subtest; replacing `restoreTerminal`'s `hostty.ResetRegion` fails the teardown subtest. The "a budget is a bound on the *negative* direction, so it needs tests that COUNT" framing is correct and is what was missing.
- **BR-63's fixes were verified against the case they had been missing, not just widened.** Appending `func (m *terminalMux) sneakyMutator() { m.active = 0 }` to `strip.go` (not `run.go`) fails `TestEveryStripModelMutatorHasARepaintCase`; stacking a second opener on `ClearLine` inside `control.go`'s single grouped `const (…)` fails `TestNoDeclarationCarriesTwoStackedGodocs`. The `scanned < 2` floor (stripmutation_test.go:246) is the right kind of self-check — it fails the guard rather than passing vacuously.
- **`hostty/reserve.go:135` `drawRow` + `TestReserveAndPaintIsPaintPlusTheRegion` (reserve_test.go:196)** is the correct answer to `copy-instead-of-extract`: the test asserts `ReserveAndPaint == Paint with the region spliced in`, an equality, not a substring presence. A future change to one painter cannot now miss the other.
- **`workbenchshortcut.TitleIdentifiesRightTerminal` (shortcut.go:199) with the producer×consumer table generated over the predicate's boundary cases** (strip_test.go:304) — `terminals`, `terminal-2`, `terminalwork`, `Terminal 1` — is the right corpus. Three hand-picked fixtures provably could not see BR-56; this one can.
- **`resolvePlan` (plantable_test.go:26)** is correct and I verified the archived branch by actually moving the plan: both Go guards stayed green.

## 2. Critical findings

None.

## 3. Important findings

**BR-61 is not addressed — `tests/plan-superseded-facts-test.sh:45`**

```sh
PLAN="workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md"
```

`check()` does `[ -f "$path" ] || { bad "$file does not exist"; return; }`, so with the plan archived the script prints seven `does not exist` failures plus a `grep: … No such file or directory` and exits 1. It is wired into `make test` (the plan's own M1 revision says so). Measured in a scratch clone at `fc318eb9` with the plan moved to `workshop/history/plans/`. Fix: give the script the same active-or-archived resolution the two Go guards got — a `resolve_plan()` that checks `workshop/plans/` then `find workshop/history -name "$1"` — and keep the `REV` computation reading whichever path it found. `cmd/internal/sessioninventory/concept_contract_test.go:30` is *not* an instance: it reads through `git show <pinned-commit>:<path>`, which is archive-proof by construction.

**New — `doc-states-planned-as-current` (4th in family): the issue's `## Done when` contradicts the issue's own `## Revisions`, in both directions**

`workshop/issues/000199-…md:170` still carries, verbatim and unmarked, the bullet that `:701` explicitly records as **Struck**:

> The strip displays scroll position for its own pane, sourced from `ptychild` …

and the bullet the same revision entry says was **"Added to `## Done when`"** — "the right pane in layout3 takes `borderless=true`" — is absent from the list (`grep -n borderless` finds it only in the Spec, the Plan and the Revisions). The Spec at `:113` also still reads "going borderless is a **follow-on**, not part of this issue", which the same entry inverted and M4 implements. So the artifact `sdlc close --verified` is checked against declares one deliverable that was retired and omits the one that replaced it.

Earlier rounds fixed instances of this family in Go doc comments. **Do not fix these three instances — state the rule.** The rule is already half-built: `tests/plan-superseded-facts-test.sh` exists as this family's class guard and is bounded to `$PLAN`, never reading the issue, which is the artifact the close gate reads. Extend it to take a list of `(file, token, replacement, max_line)` over *both* the plan and the issue body, and add the pairs this window creates. The same enumeration catches the code-side instance in this window: `cmd/internal/termcmd/run.go:1359` still says "inheritSize does not write the pane today, but it becomes a writer in M3" — M3 is the milestone closing here, and `paintbudget_test.go:71` now asserts the opposite.

**New — `plan-table-drift` (9th in family): the Core-concepts guard reads only the pipe rows, so the bullets under the table are unchecked, and one now names a method this commit deleted**

`workshop/plans/000199-…-plan.md:266-267`:

> `Reservation{Rows, Edge}` answers `ChildRows()`, `Reserve()`, `Release()`, `Paint(text)`.

`Reservation.Reserve()` was deleted in `fc318eb9` (reserve.go's own godoc now says "There is no bare `Reserve()`"). `TestEveryCoreConceptRowNamesASymbolThatExists` cannot see this: plantable_test.go:175 skips every line that does not start with `|`, and the descriptive bullets immediately beneath the table make exactly the same kind of claim about exactly the same symbols.

**Do not fix this instance — state the rule.** The guard reads the whole `## Core concepts` *section*'s backticked identifiers, not just its pipe rows, resolving each against the path its row declared; `goIdentifier` already filters out `\x1b[r`, `rename-pane` and `#200`, so the widening is mostly free. If a bullet legitimately names a symbol from another package, it should carry that package qualifier and the guard should skip unqualified misses only with a stated reason — not by not looking. (Note also: M1.1's illustrative snippets at `:455` and `:468` still call `r.Reserve()`; those are inside a milestone block that predates the deletion, so they are historical rather than live claims — the bullet is not.)

## 4. Minor findings

- **`exit-path-drops-cleanup` (3rd in family) — `probes/cursorsaveslots/main.go:72`, `probes/zellijscrollregion/main.go:89`, `cmd/probes/couchnestedrows/main.go:396`.** Each registers cleanup as a `defer` (`session.Close()`, `delete-session --force`, `os.RemoveAll(dataDir)`, `os.Remove(layout)`) and then leaves through `os.Exit` on 3–4 paths each — including the *most likely* one, `PROBE-INCONCLUSIVE: the session never appeared`. `os.Exit` skips defers, so a failed `make test-smoke` leaves a live `cursorsaveslots-<pid>` / `zellijscrollregion-<pid>` session and a temp data dir behind, and the name embeds the pid so nothing later reclaims it. BR-51 made "a probe can only destroy a session it made" structural; nothing made "a probe always destroys the session it made" structural. **State the rule rather than patching nine call sites:** a probe's `main` is a two-line `func main() { os.Exit(run()) }` and every diagnostic path *returns* a code, so cleanup is a `defer` in `run`. This is mechanisable with the go/ast machinery the package already uses — a guard over `probes/` and `cmd/probes/` that fails when a function containing a `defer` also calls `os.Exit`. Prevalence measured: 3 probes, 11 `os.Exit` sites downstream of a `defer`. (Shape is pre-existing — the pre-window `zellijscrollregion` had it too — but BR-51's fix re-landed it in new code.)
- `reserve_test.go:191` cites `(BR-64)` where it means BR-65; cosmetic.
- `zellijprobe.Start` names the session `<prefix>-<pid>`; a leaked session from the previous bullet with a recycled pid will make `--new-session-with-layout --session <name>` fail in a way the probe reports as a start error rather than as "clean this up". One sentence in `Start`'s doc would close the loop.

## 5. Test coverage notes

- `go test ./cmd/internal/termcmd/ -count=1 -race` is clean apart from the tests that need `fork/exec` (`TestTerminalMuxNewTab*`, `TestEveryStripModelMutationRepaintsTheRow/newTab`); same for `hostty` and `ptychild`. That is the documented sandbox class, not a regression — and `mustNewTab` failing rather than skipping (stripmutation_test.go:143) is the right call.
- `tests/plan-superseded-facts-test.sh` passes at HEAD and fails only under the archived-plan condition described above.
- Coverage shape is good where it matters: the strip's integration tests (`stripMux`, strip_test.go:144) run on a real writer loop with an injected recorder and no pty, so the row-dirty → re-`Reserve` → repaint ordering, the takeover repaint, the held-save deferral and the eight strip-model mutators are all deterministic.
- The remaining gap is BR-9's: `TestPaintDefersMidSequenceAndIsOwed` (writer_test.go:117) splits one sequence at one index. The plan's own ARCH-ORDER note says "a test can split an escape sequence at any index"; parameterising over every index of a small representative sequence set is the difference between a sample of size one and coverage.

## 6. Architectural notes

Working through each marker on this diff:

- **ARCH-DRY — pass.** `drawRow` collapses the two painters; `zellijprobe` removes 196 duplicated lines; `TitleIdentifiesRightTerminal` is one predicate with three askers; `rowtext` is the single row-safety strip. The only residual is documentation (the `plan-table-drift` finding).
- **ARCH-PURE — pass.** `RenderStrip` is `(width, StripModel) → RenderedStrip`, no clock, no mux, no terminal, and its tests need no double. The mux is the thin shell; `stripBytes` is the one place the pure renderer meets the reservation.
- **ARCH-PURPOSE — flag (BR-61).** This is the entry's exact failure mode, stated in its own words: the round named the class ("a guard that reads a workshop artifact resolves it active-or-archived"), fixed the two instances the finding pointed at, and never wrote the enumeration the class implies — `grep -rn "workshop/plans" cmd tests scripts`, which takes seconds and returns the third site. A family that repeats across rounds is the ledger reporting the enumeration was never written; this one repeated *inside* the round that claimed to close it.
- **ARCH-MOCK — pass.** zellij sits behind `Runtime`/`fakeRuntime`, the tty behind `writerRecorder`/`hostty.FakeHost`, and production and test share the `paneWriter` boundary. Live conformance is real and re-runnable (`make test-smoke`, `make test-couch-nested-rows`, 15/15 per the Log). The Minor above is the only blemish.
- **ARCH-CONSTRAINTS — pass, newly.** Each declared budget bullet now owns a counting assertion, verified red by mutation. The keystroke path is honestly cheap: a rename costs zero subprocesses and one posted paint per keystroke.
- **ARCH-SECURE — pass.** Untrusted input is enumerated and typed at the boundary: operator tab names and the live rename field both go through `rowtext.Sanitize`/`Fit` before reaching the row (`strip.go:170,185`), diagnostics sanitize at the single egress (`reportError`), and the child's byte stream is parsed by `ptychild.Screen` rather than trusted. No credentials. The probes only ever name and delete their own session.
- **ARCH-ORDER — pass with one open gap.** The `(state, event) → (state, effects)` table is real and each transition now has a test that fails when its behaviour is removed. `stripOwed` / `owed` / `owedDiag` are three pieces of pending state with a written ordering (`writeOwn` clearing `owed` states the invariant where it lives), and `unsafeToPaint` collapses two defer conditions into one predicate rather than a flag constellation. Extent is bounded — no goroutine outlives `Run`. The gap is BR-9: the interleaving space is describable but only one point in it is sampled.

**For M4.** BR-8 remains the one worth acting on before M4.3 is ticked: six of the nine `borderless=true` rungs only take effect after `Alt+Shift+d`, and losing the divider in a split is the precise failure that caused the 2026-07-27 revert. One keystroke in the acceptance step. Also note the M4.4 atlas/`config.kdl` correction is now partly pre-done — `atlas/architecture.md` already carries the nested-reservation and strip-ownership paragraphs — so M4.4 should be narrowed to `config.kdl`'s scroll-indicator rationale rather than re-asserting what this window already landed.

## 7. Plan revision recommendations

1. **`## Revisions` — "round 14: the class guard was scoped to one artifact, twice."** BR-61 disposed `not-addressed`: `tests/plan-superseded-facts-test.sh` still hardcodes the active plan path and breaks `make test` repo-wide at M4.5. Record the enumeration command (`grep -rn "workshop/plans" cmd tests scripts Makefile*`) alongside the fix, since the rule's whole point was that the site set is derivable.
2. **Core concepts, `Edge / Reservation` bullet (`:266-267`)** — drop `Reserve()` from the method list (it is deleted; `ReserveAndPaint` is what both consumers call), and record that `TestEveryCoreConceptRowNamesASymbolThatExists` is being widened from pipe-rows to the whole section, so the bullets stop being an unchecked claim.
3. **M4.4** — narrow to `zellij/config.kdl`'s scroll-indicator comment; note that the `atlas/architecture.md` half landed in M3.
4. **M4.3** — add the `Alt+Shift+d` split step (BR-8) and the "each half draws its own strip / the boundary between them is legible" observation, so the Done-when bullet about splits has a step that produces it.
5. **Issue file, not the plan:** `## Done when` needs the struck scroll-position bullet removed and the `borderless=true` bullet added, and the Spec's `:113` "going borderless is a follow-on" sentence rewritten — the 2026-09-06 revision already decided all three and none were applied.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 still has no Alt+Shift+d step; six of nine borderless rungs stay unverified.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:117 still splits one hand-chosen sequence at one index.
  - id: BR-61
    disposition: not-addressed
    note: |
      tests/plan-superseded-facts-test.sh:45 still hardcodes workshop/plans/000199-…-plan.md and is wired into make test; measured 7 failures with the plan archived.
  - id: BR-62
    disposition: addressed
    note: |
      All three mutations re-run in a scratch clone and all three now go red.
  - id: BR-63
    disposition: addressed
    note: |
      Verified by mutation: a mutator in strip.go and a stacked godoc in control.go's grouped const are both caught.
  - id: BR-64
    disposition: addressed
    note: |
      rename.go:64 now states one surface and explains the second was deleted.
  - id: BR-65
    disposition: addressed
    note: |
      drawRow extracted; TestReserveAndPaintIsPaintPlusTheRegion asserts equality, not substring presence; Reserve() deleted.
  - id: BR-66
    disposition: addressed
    note: |
      Table row and atlas paragraph added; the table now states it is checked table→code only, attributing the other direction to pair#188.
findings:
  - id: new
    severity: Important
    family: doc-states-planned-as-current
    title: |
      The issue's Done-when still carries a bullet its own Revisions marks struck, and omits the one that entry says replaced it
    detail: |
      workshop/issues/000199-…md:170 still lists the scroll-position deliverable that :701
      records as "Struck"; the borderless=true bullet the same entry says was "Added to
      Done when" is absent; and the Spec at :113 still calls borderless a follow-on that
      the same entry moved into scope. This is the artifact sdlc close verifies against.
      Fourth in family, so state the rule rather than editing three bullets: this family's
      class guard already exists as tests/plan-superseded-facts-test.sh and is bounded to
      $PLAN, never reading the issue. Widen it to take (file, token, replacement, max_line)
      over both artifacts. The same enumeration catches the code-side instance in this
      window, run.go:1359 ("inheritSize does not write the pane today, but it becomes a
      writer in M3"), which paintbudget_test.go:71 now asserts the opposite of.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      The Core-concepts guard reads only pipe rows, so the bullets under the table are unchecked and one names a method this commit deleted
    detail: |
      Plan :266-267 says Reservation "answers ChildRows(), Reserve(), Release(),
      Paint(text)"; Reservation.Reserve() was deleted in fc318eb9, and reserve.go's own
      godoc now says "There is no bare Reserve()". plantable_test.go:175 skips every line
      not starting with "|", so the descriptive bullets immediately beneath the table --
      which name the same symbols at the same declared paths -- are invisible to the guard
      written for exactly that claim. Ninth in family: do not edit the bullet. The guard
      reads the whole "## Core concepts" section's backticked identifiers; goIdentifier
      already filters the non-symbols (escapes, zellij action names, issue refs), so the
      widening is close to free.
  - id: new
    severity: Minor
    family: exit-path-drops-cleanup
    title: |
      Every probe registers teardown as a defer and then leaves through os.Exit, so a failed test-smoke leaks the session it created
    detail: |
      probes/cursorsaveslots/main.go:72 defers session.Close() and then os.Exit(2)s at :80,
      :124 and :150; probes/zellijscrollregion/main.go:89 has the same shape at :97/:135/:138;
      cmd/probes/couchnestedrows/main.go defers delete-session (:396), os.RemoveAll(dataDir)
      (:387) and os.Remove(layout) (:392) then calls inconclusive(), which os.Exit(1)s -- and
      the path most likely to fire is "the session never came up". os.Exit skips defers, so a
      live zellij session named <prefix>-<pid> plus a temp data dir survive each failure, from
      make test-smoke. BR-51 made "a probe can only destroy a session it made" structural;
      nothing made "a probe always destroys the session it made" structural. THE RULE - a
      probe's main is func main() { os.Exit(run()) } and every diagnostic path returns a code,
      so cleanup is a defer in run. Mechanisable with the go/ast machinery this package
      already uses: fail when a function containing a defer also calls os.Exit. Measured
      prevalence: 3 probes, 11 such exit sites.
```
