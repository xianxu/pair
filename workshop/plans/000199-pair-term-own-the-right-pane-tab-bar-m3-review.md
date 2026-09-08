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
