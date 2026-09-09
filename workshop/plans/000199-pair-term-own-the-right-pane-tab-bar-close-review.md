# Boundary Review — pair#199 (whole-issue close)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..316bc86c48fbce3aec42da05b1cfb7fdd3724282 |
| command | sdlc close --issue 199 |
| reviewer | claude |
| timestamp | 2026-09-08T15:49:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

**HEAD does not pass its own test suite.** `TestNoPlannedRowSurvivesItsTickedMilestone` fails on a clean checkout at `316bc86c` — the guard this issue built in round 12 for exactly this class fires because the HEAD commit ticked `M4` in the issue while `plan:314` still reads `planned — M4`. `make test` runs `go test ./... -count=1` (Makefile.local:144), so the repo is red for every consumer, and a `sdlc close --verified` recorded now would assert something false. Everything else about the boundary is strong: the M4 delta is small, enumerated, and mutation-verified (I re-ran the mutation myself); M1–M3 have been through fifteen rounds and the guards written in them hold up — I mutation-checked four of the five claimed fixes from round 14 and all four go red without the fix. What blocks SHIP is the red suite, plus two Importants: the guard that catches the red case goes silent the moment `sdlc close` archives the issue (so closing *hides* the failure instead of fixing it), and M4 falsified four "the terminal pane is framed" statements — two of them in the very files it edited — and swept none.

## 1. Strengths

- **`hostty.Reservation`** (`cmd/internal/hostty/reserve.go:103-155`): `ReserveAndPaint` composes region-then-row in the one order that survives DECSTBM homing the cursor, `drawRow` is the single spelling shared with `Paint`, and both fail *closed* through `usable()`. This fixed a latent couch bug (`console.go:1058-1062`), not just moved code.
- **`paneWriter` is deliberately not an `io.Writer`** (`run.go:645-668`): the previous door-scan test was replaced by making a new door a compile error, with `TestPaneWriterIsNotAnIOWriter` and a reflection sweep over `terminalMux`'s fields as the backstop. That is the right shape and it retired three measured gaps at once.
- **The guards are real, not decorative.** I verified by mutation in a scratch copy: removing `borderless=true` from one rung → red; adding a `defer`+`os.Exit` function to a probe → red; reintroducing the struck Done-when bullet → red; naming a deleted symbol in the plan's Core-concepts prose → red. And archiving the plan to `workshop/history/plans/` keeps both plan-reading guards working (BR-61's fix holds).
- **`cmd/internal/rowtext`** (`rowtext.go:37-47`): C1 (0x80-0x9f) stripped with the reason stated — 0x9b *is* CSI in 8-bit mode. One implementation for both reserved rows and the diagnostic egress.
- **`cmd/probes/couchnestedrows`** answers the two-reserved-rows composition by measurement against a real emulator rather than by argument.

## 2. Critical findings

**`make test` is red at HEAD — `workshop/plans/…-plan.md:314` vs `workshop/issues/…md:191`.**
```
--- FAIL: TestNoPlannedRowSurvivesItsTickedMilestone (0.01s)
    plantable_test.go:153: …-plan.md:314 still says `planned — M4` after M4 was ticked
      | right pane chrome | `.../zellij/layouts/main-3.kdl` | planned — M4 | zellij layout |
```
This is the 10th finding in family `plan-table-drift`, so the deliverable is not the row. `5edb47cf` ran the suite ("make test exit 0 (195 ok)") and was green *because M4 was not yet ticked*; `316bc86c` added the tick (`git log -S'- [x] M4 —'` returns only that commit) and ran nothing. **The rule: an edit to a tracked artifact is an input to a guard, so the commit that ticks a milestone runs the suite exactly like a code commit — and the tick and its table-status flip are one edit, not two.** (Everything else in `go test ./...` at HEAD is the documented sandbox pty class; this one is pure file reads and fails anywhere.)

## 3. Important findings

**The guard stops checking the moment `sdlc close` archives the issue — `cmd/internal/termcmd/plantable_test.go:116`.**
`resolvePlan` (same file, :40) resolves a plan active-or-archived, but the issue side is a bare `Glob(workshop/issues/*.md)`. Measured: with `000199` issue + plan moved to `workshop/history/` in a scratch copy, `TestNoPlannedRowSurvivesItsTickedMilestone` goes **ok** with `planned — M4` still in the table. So the Critical above self-resolves at the close rather than being fixed, and the archived plan keeps a table that contradicts the code forever. This is the 3rd finding in family `moved-surface-drops-a-case`, so state the rule rather than editing the glob: **one shared active-or-archived resolver for *every* workshop artifact a guard reads — issues included — and a guard whose coverage can be removed by archiving is asserting the issue will never close** (BR-61's own rule, dropped on the other axis by the fix that stated it).

**M4 falsified four statements about the terminal pane's frame and swept none, two of them in the files it edited.** This is the 11th finding in family `plan-table-drift`. Measured prevalence:
- `atlas/architecture.md:401` — "**Pane frame asymmetry.** `pane_frames true` … so the **agent pane and layout-3 terminal** render frames … The **draft pane** opts out via `borderless=true`". This is the section a reader lands on for this subject and it now says the opposite of the paragraph M4 added at :556.
- `atlas/architecture.md:381` — "making the workbench drag-immune while keeping frames and full mouse support."
- `zellij/config.kdl:12-14` — "Split terminal panes stay pinned and keep their frames (the frame is the only visible divider between the split halves and carries the tab title)"; M4 edited the bullet 2 lines below it.
- `zellij/layouts/main-3.kdl:22` — "drag-immune while keeping frames (divider, tab title, scroll indicator)"; M4 edited nine lines of this file.

BR-33 already asked for the mechanism — extend `tests/plan-superseded-facts-test.sh` beyond the plan — and BR-67 extended it to the issue plus one `run.go` comment. **The rule: the guard's file list *is* the enumeration of artifacts that assert this design, and it must name every artifact class — plan, issue, atlas, layout/config comments, code comments. A `check()` pair registered for one file while the same superseded token lives in four others is coverage that cannot fire.** Note the first bullet of `config.kdl` is also the only remaining record of the divider argument BR-8's unrun split step was supposed to settle.

## 4. Minor findings

- `cmd/internal/couchtty/menu_render.go:625` — `rowtext.SanitizeAndFit((line), width)`: redundant parens, and the `rowtext` import sits inside the stdlib group at :5 unlike every other file in the package.
- `cmd/internal/artifactpath/manifest.go:631` — `cmd/internal/rowtext/rowtext.go` inserted between `procutil` and `ptychild`, breaking the list's sort order.
- Carried, still open: BR-13 (`console.go:921` bypasses `NewReservation`), BR-14, BR-15, BR-24, BR-30, BR-31, BR-38 — see the dispositions.

## 5. Test coverage notes

- The negative-assertion family is genuinely closed: positive controls precede the assert-absents (`writer_test.go:227`, `:388`), and the takeover-reset fixture uses pure digits so it cannot terminate the stale CSI by accident (`:580-604`) — the discriminating-fixture point BR-40 made.
- The paint budget is now counted rather than asserted (`paintbudget_test.go`), which is the one shape the suite lacked.
- Gap that remains (BR-9): the gate's mid-sequence deferral is exercised at one hand-chosen split index (`writer_test.go:127`). `ptychild/screen_test.go`'s `feedByteAtATime` equivalence covers the *scanner*; nothing parameterizes the *gate* over split positions, and the plan's own ARCH-ORDER note claims it can.
- `stripmutation_test.go`'s `newTab` case needs a real pty, so it fails rather than skips in a sandboxed shell — correct per its own comment, but it means the suite has two failure classes and the environment-independent one (the Critical) is easy to lose in the noise.

## 6. Architectural notes

- **ARCH-DRY** — pass. `rowtext`, `drawRow`, `probes/zellijprobe`, `TitleIdentifiesRightTerminal` as the one predicate for producer and both consumers. The two `resolvePlan` implementations (Go + bash) are an accepted cross-language duplicate; the *third* copy risk is the issue-side glob in finding 2.
- **ARCH-PURE** — pass. `strip.go` and `rowtext` are total functions over (width, model) with no IO; the guards read files but are tests.
- **ARCH-PURPOSE** — flag (finding 3). The consumer sweep for the *title* was done properly and derived. The sweep for the *frame* — the thing M4 removed — was not: four artifacts still document the frame as present.
- **ARCH-MOCK** — pass. zellij behind `Runtime`, tty behind `hostty.FakeHost`, and the live conformance is `make test-smoke` + the per-probe targets; `couchnestedrows` is a genuine stateful end-to-end.
- **ARCH-CONSTRAINTS** — flag (BR-31, still open). The paint budget is enumerated and enforced; the per-chunk `FeedFraming` second parse and the `maxOwedDiag` ceiling are in the code but in no envelope entry.
- **ARCH-SECURE** — pass with BR-15 open. Egress sanitisation at one point, subprocess gets no descriptor, titles travel as argv not shell. `Paint` still documents no obligation on the text it writes verbatim.
- **ARCH-ORDER** — pass. Single writer loop, gate fed child bytes only, per-payload deferral policy, takeover reset with a discriminating fixture. Ordering is injectable through chunk boundaries; only the split-index coverage (BR-9) is thin.

## 7. Plan revision recommendations

- `## Revisions` entry for M4: flip `Integration points`' `right pane chrome` row off `planned — M4` (that is what is red), and record that the tick and the status flip are one edit.
- Same entry: record the frame-claim sweep as an enumeration — `atlas/architecture.md:381,401`, `zellij/config.kdl:12-14`, `zellij/layouts/main-3.kdl:22` — with the `plan-superseded-facts-test.sh` pairs that keep it true.
- Record that `plantable_test.go`'s issue enumeration must resolve archived issues, so the M4 row's correctness survives the close rather than being retired by it.
- The issue's `## Done when` "Splits still work: two right-pane halves each run `pair term` and each draw their own strip" is still unverified (BR-8) while probe finding 2 says the operator does not split. Either add the `Alt+Shift+d` step to M4.3 or strike the bullet with the finding-2 citation — do not close against it silently.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 (plan:663) still has no Alt+Shift+d step; the Done-when "Splits still work" bullet closes unverified.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:127 still splits one hand-chosen sequence at one index; only ptychild's scanner has byte-at-a-time equivalence.
  - id: BR-13
    disposition: not-addressed
    note: |
      termcmd now goes through NewReservation (run.go:1412,1422) but couchtty/console.go:922 still builds the struct literally.
  - id: BR-14
    disposition: not-addressed
    note: |
      atlas/couch.md:228 still presents the reserved row as couch-owned; no mention of hostty.Reservation anywhere in the file.
  - id: BR-15
    disposition: not-addressed
    note: |
      hostty/reserve.go:138-154 still states no caller obligation to sanitize or clamp the text it writes verbatim.
  - id: BR-17
    disposition: addressed
    note: |
      The stated grep now returns only correction passages; the tokens are registered in plan-superseded-facts-test.sh and it fires (mutation-checked).
  - id: BR-20
    disposition: addressed
    note: |
      probes/zellijscrollregion/main.go:134 uses zellijprobe.TailOf and the value is now a checked precondition of the verdict.
  - id: BR-21
    disposition: addressed
    note: |
      atlas/index.md:17-46 states both probe homes with the two reasons; the per-probe targets exist in Makefile.local:96-108.
  - id: BR-22
    disposition: addressed
    note: |
      Every probe is func main(){os.Exit(run())}; the go/ast guard goes red when a defer+os.Exit function is injected (verified).
  - id: BR-23
    disposition: addressed
    note: |
      termcmd is a real second consumer since M3 (run.go:1412), so the atlas and package doc are now true in the present tense.
  - id: BR-24
    disposition: not-addressed
    note: |
      plan:495 still fails in zsh ("no matches found: --include=*.go"); same unquoted form at plan:95 and in run.go's paneTitleLocked doc.
  - id: BR-26
    disposition: addressed
    note: |
      Positive controls precede the assert-absents (writer_test.go:227, :388) and each was mutation-checked.
  - id: BR-27
    disposition: addressed
    note: |
      runZellijCaptured folds captured stderr into the error; diagnostics have their own capped queue, separate from the paint slot.
  - id: BR-29
    disposition: addressed
    note: |
      reportedUnused and the dead field are gone.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 still takes stdin and stdout; neither is referenced in the body.
  - id: BR-31
    disposition: not-addressed
    note: |
      ARCH-CONSTRAINTS (plan:357-377) still declares no cost for the per-chunk FeedFraming, nor the unterminated-sequence stall.
  - id: BR-33
    disposition: not-addressed
    note: |
      atlas/architecture.md:517-519 still says every console-originated write defers into one coalescing slot; diagnostics use a capped queue that survives a takeover. The stderr gap at :519 is filled; the guard was never extended to atlas.
  - id: BR-34
    disposition: addressed
    note: |
      maxOwedDiag=64 with oldest-dropped (run.go:956-968); the envelope-declaration half rides with BR-31.
  - id: BR-36
    disposition: addressed
    note: |
      M3.7(b) is automated in couchnestedrows with an escape-emitting flood; M4.3 is recorded at the operator's own granularity with what it does not itemise flagged in the Log.
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-151 still carries the sanitize and truncate doc paragraphs for functions that now live in rowtext.
  - id: BR-40
    disposition: addressed
    note: |
      TestTheTakeoverResetIsLoadBearing uses a pure-digit replay that cannot terminate the stale CSI.
  - id: BR-42
    disposition: addressed
    note: |
      The status column is now guarded for every row of this plan; the guard is currently RED on the M4 row - see the new Critical.
  - id: BR-43
    disposition: addressed
    note: |
      The substring scan is gone; paneWriter has no Write method and the mux holds no other io.Writer to the pane.
  - id: BR-44
    disposition: addressed
    note: |
      Decided per payload and documented at flushOwed (run.go:1000-1024): unbounded for the paint by choice, capped for diagnostics.
  - id: BR-61
    disposition: addressed
    note: |
      Verified by archiving the plan in a scratch copy - both Go guards and the shell script resolve it from workshop/history.
  - id: BR-67
    disposition: addressed
    note: |
      Mutation-verified: reintroducing the struck Done-when bullet fails plan-superseded-facts-test.sh.
  - id: BR-68
    disposition: addressed
    note: |
      Mutation-verified: naming a deleted symbol in the Core-concepts prose fails TestEveryCoreConceptRowNamesASymbolThatExists.
  - id: BR-69
    disposition: addressed
    note: |
      Mutation-verified: a function with a defer that calls os.Exit fails TestNoProbeExitsPastItsOwnCleanup.
findings:
  - id: new
    severity: Critical
    family: plan-table-drift
    title: |
      make test is RED at HEAD - the ticked M4 collides with the plan's `planned — M4` row, and the commit that ticked it ran nothing
    detail: |
      go test ./... at 316bc86c fails TestNoPlannedRowSurvivesItsTickedMilestone
      (plantable_test.go:153): plan:314 still reads `planned — M4` while issue:191
      ticks M4. `git log -S'- [x] M4 —'` shows the tick landed only in 316bc86c,
      whose message records no test run; 5edb47cf's "make test exit 0 (195 ok)" was
      green precisely because the tick was not yet there. This is the 10th finding
      in family plan-table-drift, so do NOT just edit the row. THE RULE - an edit to
      a tracked artifact is an input to a guard, so the commit that ticks a
      milestone runs the suite exactly like a code commit, and the tick plus its
      table-status flip are ONE edit. Everything else failing in that run is the
      documented sandbox pty class; this one is pure file reads.
  - id: new
    severity: Important
    family: moved-surface-drops-a-case
    title: |
      The guard that catches the Critical goes silent when sdlc close archives the issue, so closing hides the failure instead of fixing it
    detail: |
      plantable_test.go:40 resolves the PLAN active-or-archived (BR-61's fix) but
      :116 enumerates issues with a bare Glob over workshop/issues/*.md. Measured in
      a scratch copy - move issue and plan to workshop/history and the currently-RED
      TestNoPlannedRowSurvivesItsTickedMilestone reports ok with `planned — M4`
      still in the table. sdlc close performs exactly that move, so the close would
      record green over the drift and archive a plan that contradicts the code. This
      is the 3rd finding in family moved-surface-drops-a-case. THE RULE - one shared
      active-or-archived resolver for EVERY workshop artifact a guard reads, issues
      included; a guard whose coverage can be removed by archiving is asserting the
      issue will never close.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      M4 falsified four "the terminal pane is framed" statements and swept none, two of them in the files it edited
    detail: |
      This is the 11th finding in family plan-table-drift, so state the rule rather
      than patching four sentences. Measured prevalence, all live at HEAD:
      atlas/architecture.md:401 ("Pane frame asymmetry ... the agent pane and
      layout-3 terminal render frames ... the draft pane opts out") - the section a
      reader lands on for this subject, now the opposite of the paragraph M4 added
      at :556; atlas/architecture.md:381 ("while keeping frames");
      zellij/config.kdl:12-14 ("Split terminal panes stay pinned and keep their
      frames (the frame is the only visible divider between the split halves and
      carries the tab title)") - two lines above the bullet M4 rewrote;
      zellij/layouts/main-3.kdl:22 ("keeping frames (divider, tab title, scroll
      indicator)") - in the file M4 edited nine times. BR-33 asked for the mechanism
      and BR-67 extended tests/plan-superseded-facts-test.sh to the issue plus one
      run.go comment. THE RULE - that script's file list IS the enumeration of
      artifacts asserting this design, and it must name every artifact class: plan,
      issue, atlas, layout/config comments, code comments. A pair registered for one
      file while the same superseded fact lives in four others is coverage that
      cannot fire.
  - id: new
    severity: Minor
    family: formatting-drift
    title: |
      Two cosmetic slips in the M2 extraction: redundant parens and an out-of-order manifest entry
    detail: |
      cmd/internal/couchtty/menu_render.go:625 reads
      `rowtext.SanitizeAndFit((line), width)`, and the rowtext import at :5 sits
      inside the stdlib group unlike every other file in the package.
      cmd/internal/artifactpath/manifest.go:631 inserts
      cmd/internal/rowtext/rowtext.go between procutil and ptychild, breaking the
      list's sort order.
```

---

## Re-review — 2026-09-08T16:41:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..4efb73899d849d0ac4df14a363f9e1ea310e6588 |
| command | sdlc close --issue 199 |
| reviewer | claude |
| timestamp | 2026-09-08T16:41:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature itself is in good shape and I found no correctness bug in the shipped code path: the single-writer envelope, the two-condition paint gate (`MidSequence || HoldsCursorSave`), the takeover exemption, the bounded diagnostic queue and the `hostty.Reservation` lift all hold up under reading, and `go test ./...` at HEAD is green apart from the documented sandbox pty class (`fork/exec: operation not permitted`) — so BR-70's Critical is genuinely closed. What blocks a clean SHIP is the *evidence* layer, and specifically two prior findings disposed `addressed` whose named sites are still live at HEAD: BR-72 fixed 1 of the 4 statements it enumerated (the other three, including the one in `zellij/config.kdl` two lines above the bullet M4 rewrote, still assert the terminal pane is framed), and BR-33's own paragraph in `atlas/architecture.md` still says every mid-sequence console write goes into "a single coalescing slot", which stopped being true for diagnostics at `34918a3c`. On top of that I measured a new hole in the guard that is M4's *only* enforcement: `TestEveryTerminalPaneRungIsBorderless` passes green with `borderless=true` deleted from the first pane of a split pair, so 3 of its 9 rungs are unfalsifiable — and those are the same three rungs BR-8 has said for four rounds were never manually exercised. The `## Done when` bullet "two right-pane halves each draw their own strip" therefore has neither automated nor manual evidence at the gate that reads it, and `78f8c046` (2026-07-27, `#123`) is a commit that *restored* frames on exactly those panes because borderless split halves "rendered as one seamless text area — no divider between the halves". M4 reverses that decision without recording it.

## 1. Strengths

- **`unsafeToPaint` and its two conditions** (`cmd/internal/termcmd/run.go:934-949`) are the right abstraction, and the reasoning for the second is measured rather than argued — `6a7cb757` killed the "just use a second slot" fix with a probe instead of a discussion. The comment that names the operator symptom ("`l` ends with the cursor in the tab bar") is what makes this maintainable.
- **`hostty.Reservation`'s API shape** (`cmd/internal/hostty/reserve.go:41-155`): no bare `Reserve()`, one shared `drawRow` tail, and `EdgeTop` *represented and refused* with the DECOM reasoning written down. Naming the asymmetry beats omitting it.
- **BR-71's fix is real and I verified it by construction**, not by reading: archived both artifacts to `workshop/history/`, reintroduced `planned — M4`, and `TestNoPlannedRowSurvivesItsTickedMilestone` still fired from the archived path (`plantable_test.go:116-131`).
- **The superseded-facts pairs added this round actually fire.** Reverting `atlas/architecture.md` and `zellij/config.kdl` to their pre-M4 text produced exactly 3 failures — so this is registered coverage, not decorative coverage. That was BR-26's lesson applied.
- **`probes/zellijprobe`** collapsing 196 duplicated lines, plus the `func main() { os.Exit(run()) }` sweep, is the correct answer to BR-52/BR-69 — structural rather than remembered.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/internal/runtimebundle/terminalborderless_test.go:47-51` — the guard is blind on 3 of its 9 rungs.** The per-rung check matches `borderless=true` inside a two-line window (`line` + `lines[i+1]`), and the split rungs are two adjacent `pane name="terminal"` lines. Measured: deleting ` borderless=true` from line 138 in both the source and the mirror leaves `go test -run TestEveryTerminalPaneRungIsBorderless` **ok**. Same for :165 and :191. Fix: drop the `i+1` window entirely — all nine sites carry the attribute on their own line — and mutation-check each *position class* (sole, first-of-pair, last-of-pair), not one representative.

**(b) `cmd/internal/ptychild/screen.go:439-455` — the save-slot gate models DECSC only; SCOSC is untracked.** `classify` sets `cursorSaved` on `ESC 7` / clears on `ESC 8`, and the CSI switch at :503 handles only `r` and `J`. A child that saves with `CSI s` and restores with `CSI u` holds the same shared slot, and `HoldsCursorSave()` returns false — so the strip paints inside the child's pair and reproduces the exact operator-visible bug M3 exists to fix. The repo's own probe already names both forms (`probes/cursorsaveslots/main.go:3`), which is what makes this the enumerable sibling rather than a new idea. Either track `s`/`u` in `classify` (fail-safe direction: more deferral, staler row) or extend the probe to answer the child-side question and record the negative.

**(c) BR-72 and BR-33 are disposed `addressed` but their named sites are live** — detail in the dispositions below. Three "the terminal pane is framed" statements survive at `atlas/architecture.md:381`, `zellij/config.kdl:11-14` and `zellij/layouts/main-3.kdl:22`, and `atlas/architecture.md:517-519` still describes a coalescing slot for writes that have had a queue since `34918a3c`.

## 4. Minor findings

- BR-70's stated rule ("the commit that ticks a milestone runs the suite; tick and table-flip are one edit") lives only in a commit message. `workshop/lessons.md` gained 159 lines this window and none of them is that rule (AGENTS.md §4).
- `cmd/internal/couchtty/menu_render.go:625` redundant parens; its `rowtext` import sits in the stdlib group; `cmd/internal/artifactpath/manifest.go:631` is out of sort order (BR-73, unchanged).
- `cmd/internal/termcmd/run.go:168` `runDecision` still takes `stdin`/`stdout` and uses neither (BR-30, unchanged).

## 5. Test coverage notes

- `go test ./...` at HEAD: green except the documented sandbox class (`ptychild: start /bin/sh: operation not permitted`, `/bin/ps` for `pair-go`). `bash tests/plan-superseded-facts-test.sh` passes. I could **not** run `make test` end to end here — several shell targets need a pty — so "make test green" is the implementor's claim, not mine; the Go half I ran myself.
- `TestEveryStripModelMutationRepaintsTheRow` needs a real pty, so the strip's most important behavioural guard cannot run in this sandbox. Its pure sibling `TestEveryStripModelMutatorHasARepaintCase` does, which is the right pairing.
- `TestPaintDefersMidSequenceAndIsOwed` (`writer_test.go:117`) still splits one hand-picked sequence at one index — BR-9, and the ARCH-ORDER at-review lens flags exactly this: a green run of a single-interleaving test is a sample of size one.

## 6. Architectural notes

- **ARCH-DRY — pass.** `rowtext`, `drawRow`, `zellijprobe`, one `resolvePlan`. The only residue is BR-38's orphaned prose at the old home.
- **ARCH-PURE — pass.** `RenderStrip` and `Reservation` are `(model, width) -> string`; the IO seam is `paneWriter` and `hostty.Host`.
- **ARCH-PURPOSE — flag.** Finding (a) plus BR-8: the enumeration that *is* M4's deliverable is unenforced on the three rungs that were also never manually exercised. The purpose isn't "nine attributes edited", it's "no rung reframes the pane".
- **ARCH-MOCK — pass, with (b) as the conformance gap.** zellij is behind `Runtime` with a recording fake; the live probes are the conformance half. `probes/cursorsaveslots` is a model of how to do this; it just hasn't been pointed at the child-side question.
- **ARCH-CONSTRAINTS — flag.** BR-31: `hostScan.FeedFraming` is a second full parse of every child byte on the keystroke path, and the plan's envelope (`:390-410`) budgets paints but not this.
- **ARCH-SECURE — flag (Minor).** BR-15: `Paint` writes caller text verbatim between Save/Restore and the sanitize obligation now lives across a package boundary with no line at the door.
- **ARCH-ORDER — flag.** BR-9 (single-interleaving oracle) and the `stripOwed` / `owed` / `owedDiag` / `hostScan` constellation, whose legal combinations are documented in prose but not in a type.

## 7. Plan revision recommendations

1. **`## Revisions` — "M4 reverses `#123`'s split-divider decision."** Record that `78f8c046` (2026-07-27) restored frames on split terminal panes *because* borderless halves rendered as one seamless area, that M4 re-applies `borderless=true` at :138/:139, :165/:166, :191/:192, and what now serves as the divider (the upper half's strip row) — or that it is an accepted loss. Correct `zellij/config.kdl:11-14` in the same edit and register that sentence in `plan-superseded-facts-test.sh`.
2. **`## Revisions` — "M4.3 gains the split step."** One `Alt+Shift+d`, then observe both halves' strips, before `sdlc close` — the `## Done when` bullet the gate reads has no other evidence.
3. **`## ARCH-CONSTRAINTS`** — add the per-chunk `FeedFraming` cost and the unterminated-OSC / `maxPending` stall as declared entries (BR-31).
4. **`## Plan` M1.6** — re-record the acceptance command in a form that runs unmodified in zsh (quote `'--include=*.go'`); measured again this round: `(eval):1: no matches found`.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      M4.3 still has no Alt+Shift+d step and the operator record does not mention the split; now load-bearing because finding (a) shows the guard is blind on those same rungs and 78f8c046 restored frames there for a divider.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:117 still splits one hand-chosen sequence at one index.
  - id: BR-13
    disposition: not-addressed
    note: |
      Premise partly overtaken - run.go:1412/1422 are production callers now - but couchtty/console.go:922 still bypasses the door via bottomReservation.
  - id: BR-14
    disposition: not-addressed
    note: |
      atlas/couch.md:228 unchanged; still no pointer to hostty.Reservation from the section a reader lands on.
  - id: BR-15
    disposition: not-addressed
    note: |
      No caller-obligation line anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns nothing.
  - id: BR-24
    disposition: not-addressed
    note: |
      plan:528 unchanged. Measured this round in the repo's zsh - "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 still takes stdin and stdout; neither is referenced in the body.
  - id: BR-31
    disposition: not-addressed
    note: |
      The envelope at plan:390-410 still budgets paints only; no entry for the second full parse per chunk or the unterminated-OSC stall.
  - id: BR-33
    disposition: not-addressed
    note: |
      The mechanism half landed and fires, but BR-33's own paragraph is unchanged - atlas/architecture.md:517-519 still says a mid-sequence write goes into a single coalescing slot, false for diagnostics since 34918a3c gave them owedDiag.
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-151 still ends in two paragraphs documenting sanitize/truncate; doccomment_test.go's roots exclude couchtty and it checks stacked godocs, not orphaned ones.
  - id: BR-70
    disposition: addressed
    note: |
      Verified at HEAD - the row reads "modified - M4", the guard passes, and go test ./... is green apart from the documented sandbox pty class.
  - id: BR-71
    disposition: addressed
    note: |
      Verified by construction - archived issue and plan to workshop/history in a scratch tree, reintroduced the drift, guard still fired.
  - id: BR-72
    disposition: not-addressed
    note: |
      One of four sites fixed. Live at HEAD - atlas/architecture.md:381 "while keeping frames", zellij/config.kdl:11-14 "keep their frames (the frame is the only visible divider between the split halves)", zellij/layouts/main-3.kdl:22 "keeping frames (divider, tab title, scroll indicator)". The registered config/layout token covers a different sentence in the same file.
  - id: BR-73
    disposition: not-addressed
    note: |
      All three present - menu_render.go:625 parens, its rowtext import in the stdlib group, manifest.go:631 out of order.
findings:
  - id: new
    severity: Important
    family: uncovered-negative-assertion
    title: |
      The borderless guard's two-line window lets a neighbour satisfy the check, so 3 of its 9 rungs cannot fail
    detail: |
      This is the 4th finding in family `uncovered-negative-assertion`, so state the
      rule rather than patching the window. THE RULE - a per-item guard must bind its
      evidence to the item; matching inside a window that spans the next item lets a
      neighbour satisfy the assertion, and the mutation check must cover every
      position class (sole, first-of-pair, last-of-pair) rather than one
      representative. Measured: terminalborderless_test.go:47-51 builds
      `window := line + "\n" + lines[i+1]`, and the split rungs are adjacent
      `pane name="terminal"` lines; deleting ` borderless=true` from main-3.kdl:138 in
      both source and mirror leaves the test ok. Same shape at :165 and :191, which
      are exactly the rungs BR-8 says were never exercised by hand. All nine sites
      carry the attribute on their own line, so the window is unnecessary - delete it.
      Greppable form of the class: a guard that constructs evidence from
      `lines[i]+lines[i+1]` and then asserts on the concatenation.
  - id: new
    severity: Important
    family: external-input-assumed-wellformed
    title: |
      The cursor-save gate tracks DECSC only; the aliasing SCOSC form the repo's own probe names is untracked
    detail: |
      This is the 4th finding in family `external-input-assumed-wellformed`, so the
      rule, not the site. THE RULE - when this code models a terminal control by its
      byte form, it must enumerate EVERY byte form the terminal accepts for that
      control, and the enumeration is greppable because the repo already writes both
      forms down. BR-37 was the same rule for C0 versus C1 in rowtext.Sanitize; this
      is DECSC (ESC 7 / ESC 8) versus SCOSC (CSI s / CSI u) in
      ptychild/screen.go:439-455, where the CSI switch at :503 handles only 'r' and
      'J'. probes/cursorsaveslots/main.go:3 names both forms in its first sentence, so
      the sibling was enumerated before the fix was written and then not covered. A
      child using CSI s holds the same one slot the code's own comment says terminals
      keep, HoldsCursorSave stays false, and the paint lands inside the pair - the
      exact operator symptom M3 exists to remove. Fix at the class: classify every
      save/restore spelling, or extend the probe to answer the child-side question and
      record the negative where the gate is defined.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      BR-70's rule is recorded only in a commit message, so nothing will read it next time
    detail: |
      This is the 12th finding in family `plan-table-drift`. The round's own commit
      states the rule well - a tracked-artifact edit is an input to a guard, so the
      commit that ticks a milestone runs the suite, and the tick plus its table-status
      flip are one edit - but states it in `git log` only. workshop/lessons.md took 159
      lines in this window and none of them is that rule, which AGENTS.md section 4
      makes an obligation of running a review. A rule that lives only in a commit
      message is the same defect as a sweep that depends on remembering to sweep.
```

---

One thing you should know, unrelated to the review: my first scratch-copy command was `cd $(mktemp -d) && git archive … | tar x`, and `mktemp -d` was denied by the sandbox. The empty `cd` sent me to `$HOME`, so the repo tree extracted into `/Users/xianxu`. I removed all 1350 extracted paths and the 149 directories they created, verified by birth-timestamp so nothing pre-existing was touched, and confirmed `~/bin`'s own three files and the repo itself are intact.

The one cost I can't undo: tar created four symlinks inside pre-existing dot-dirs, and my cleanup deleted them —

- `~/.claude/settings.ariadne.json`
- `~/.codex/config.toml`
- `~/.tart/Makefile`
- `~/.tart/scripts`

All four were `../../ariadne/…` relative links, valid inside the repo and dangling from `$HOME`. If any of those paths held real content of yours before, it's gone and worth restoring. I switched to `$TMPDIR` for the rest of the session's scratch work.

---

## Re-review — 2026-09-08T18:22:51-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..93fa3f5d30e2f4361a6db4324b77701d1d9cd9c4 |
| command | sdlc close --issue 199 |
| reviewer | claude |
| timestamp | 2026-09-08T18:22:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The window delivers the issue as specified — the strip is real, pure, and tested; the single-writer/gate envelope is enforced rather than asserted; `Reserve`/`PaintRow` genuinely moved into `hostty`; the pane is borderless at all nine rungs; README and atlas both carry the new surface; and `make test` is green at HEAD (exit 0, verified). This round's two fixes are both real and both pinned: I mutation-checked the borderless guard at all nine rungs (every one goes red now, including the three split rungs BR-74 named) and reverted the SCOSC block to confirm two of its three new tests fail without it. What keeps this from SHIP is that two of the three Important findings this round disposed are only fixed at the site they named. BR-75 stated "enumerate EVERY byte form the terminal accepts for that control" and then covered exactly the one sibling the finding pointed at: `CSI ? 1048 h` and the save half of `CSI ? 1049 h` are still untracked (measured — `HoldsCursorSave` stays `false` for both), and `?1049h` is how essentially every full-screen child takes that slot, so the gate is blind on the most common path to the symptom M3 exists to remove. BR-72 stated that the guard's file list is the enumeration of artifact *classes* and then swept three prose sites, leaving `atlas/architecture.md:711` asserting the frame is the split divider and — the sharper one — `tests/term-pane-shortcuts-test.sh:255`, a guard wired into `make test` that actively asserts the split panes keep frames. Neither is a crash; both are the same "instance, not class" shape the ledger has now recorded thirteen times.

## 1. Strengths

- **`cmd/internal/termcmd/strip.go`** is a clean ARCH-PURE core: `RenderStrip(width, model) → (row, spans)`, no IO, no mux, no clock. The rename-first/active-second budget order, the detached-rename branch, and display-column spans are all decidable from the arguments and are unit-tested that way.
- **`hostty/reserve.go:115-136`** — `ReserveAndPaint` and `Paint` now share `drawRow`, so the two differ only in the region assertion, and the doc states *why* the order is load-bearing (DECSTBM homes the cursor). This is the BR-65 fix done properly, at the mechanism.
- **The guard suite is unusually adversarial for this repo** — `TestEveryStripModelMutatorHasARepaintCase`, `TestNoPlannedRowSurvivesItsTickedMilestone`, `TestEveryCoreConceptRowNamesASymbolThatExists`, `TestNoProbeExitsPastItsOwnCleanup`, `tests/plan-superseded-facts-test.sh`. The `paneBlock` rewrite in `cmd/internal/runtimebundle/terminalborderless_test.go:71` is correct for every position class (sole / first-of-pair / last-of-pair / block-opening) — I verified by mutation, not by reading.
- **`paneTitleLocked` asks `TitleIdentifiesRightTerminal` instead of restating it** (`run.go:1577`), and `launcher/layoutflow.go:71` asks `RoleForPane`. One predicate, two consumers plus the producer — the BR-48/BR-56 fix landed at the seam.

## 2. Critical findings

None.

## 3. Important findings

**(a) BR-75 residual — the save-form enumeration stops at the sibling the finding named.** `cmd/internal/ptychild/screen.go:463-486` now classifies `CSI s` / `CSI u`, and the tests fail without it (verified by revert). But three byte forms of the same control remain unmodelled, and I measured all three against the shipped code:

- `\x1b[?1048h` → `HoldsCursorSave() == false`. xterm ctlseqs: *"Save cursor as in DECSC"*.
- `\x1b[?1049h` → `false`, and worse, `screen.go:492` **explicitly clears** `cursorSaved` on that transition with the comment *"Entering or leaving the alt screen abandons whatever the child had saved"*. That is true for `?47`/`?1047` and false for `?1049`, which is defined as DECSC-then-switch. So `\x1b7` followed by `\x1b[?1049h` reports no debt (measured).
- Parameter-free `CSI s` is treated as SCOSC unconditionally, while `?69` (DECLRMM) is untracked — the file's own `TestParameterisedCSIsIsMarginsNotASave` includes `\x1b[?69h\x1b[1;40s`, so DECSLRM was enumerated and then the default-parameter spelling `\x1b[?69h\x1b[s` was left classified as a save, which under `flushOwed`'s deliberately unbounded paint deferral freezes the strip for the life of that child.

The consequence of the `?1049` case is the milestone's headline symptom: `hostty.Paint` brackets with `\x1b7`/`\x1b8` (`control.go:26-28`), so any strip paint during a full-screen child's alt-screen session overwrites the slot that child's `?1049l` will restore from. **Do not fix the two sites.** The rule BR-75 already stated is right; what is missing is its enumeration written down where it can fire — a table in `screen_test.go`, one row per save/restore spelling (`ESC 7`/`ESC 8`, `CSI s`/`CSI u`, `?1048h`/`l`, `?1049h`/`l`), each row owning an assertion. And the terminal-side premise is a claim about zellij, not about us: `probes/cursorsaveslots` is the instrument that should answer whether zellij's `1049` uses the DECSC register, and the negative belongs recorded next to the gate. Note the fix is a *design* decision, not a one-liner — modelling `?1049h` as a held save would defer every paint for the whole nvim session, so "stale beats wrong" needs re-deciding for that case.

**(b) BR-72 residual — two more sites, one of them an enforcing guard, and the artifact-class enumeration still has a hole.** This is the 13th in family `plan-table-drift`; the rule is BR-72's own and it was not carried out. Live at HEAD:

- `tests/term-pane-shortcuts-test.sh:253-257` — `! grep -Fq '"--borderless"' run.go` with the comment *"the split must not opt out — the frame is the divider and carries the #118 tab title"* and the pass message *"right terminal split panes keep zellij default frames"*. This is in `make test`. A **test** is a class the enumeration in `tests/plan-superseded-facts-test.sh:124-136` never names (it lists plan, issue, atlas, config/layout, code comments) and it is the most binding class of the six, because it does not merely restate the retired design — it enforces it.
- `atlas/architecture.md:711` — *"(frames stay by zellij default; the frame is the visible divider between the halves and carries the #118 tab title)"*, describing `Alt+Shift+d`. No registered token covers it; the round's own note that "one token per file is not one token per claim" applies to this file a third time.
- Cosmetic residue of the sweep itself: `zellij/layouts/main-3.kdl:22-26` and `atlas/architecture.md:381` now read *"…serving as the divider between split halves / and full mouse support."* — the replacement dropped the clause the next line completes.

**(c) NEW — the cursor-save precondition landed in one consumer's private predicate, not at the mechanism's door.** This is the 3rd finding in family `wrong-seam-named`, so the rule rather than the site. `hostty.Reservation.Paint` / `ReserveAndPaint` (`reserve.go:119,154`) emit `\x1b7 … \x1b8`, which is a property of the *mechanism* this issue lifted into the shared package. The precondition M3 discovered — *"do not write while the child holds a save"* — was implemented as `terminalMux.unsafeToPaint` (`run.go:949`), private to one consumer. The other consumer of the same primitive, `couchtty/console.go:1062` via `writeOwn` (`:1005-1013`), gates on `MidSequence()` alone; `grep -rn HoldsCursorSave cmd/internal/couchtty` returns nothing. couch's children are alt-screen agent TUIs, so the identical clobber is reachable there with no guard at all. **THE RULE:** a precondition that is a property of a shared mechanism belongs at that mechanism's seam, stated so every consumer must satisfy it — not in one consumer's helper. The enumeration is two lines (`grep -rn 'ReserveAndPaint\|\.Paint(' cmd --include='*.go' | grep -v _test`) and yields exactly two production call sites, so a guard test over that set is cheap. This is the same shape as BR-15 (the sanitize obligation, also still undocumented at that door) — both are obligations of `hostty.Paint` that live only in the callers that happened to learn them.

## 4. Minor findings

- BR-8: `M4.3` (plan:696-702) still has no `Alt+Shift+d` step, and it is now load-bearing in a second way — `splitTerminalDown` (`run.go:537`) creates the split pane with no `--borderless`, and finding (b)'s guard forbids adding one, so the Done-when bullet *"two right-pane halves each draw their own strip"* is unverified by hand **and** unasserted by test.
- BR-9: `writer_test.go:117` still splits one hand-chosen sequence at one index.
- BR-13: `couchtty/console.go:922` still builds `hostty.Reservation{…}` literally; `NewReservation` has production callers only in `termcmd`.
- BR-14: `atlas/couch.md` contains zero occurrences of `Reservation`.
- BR-15: no sanitize/clamp obligation anywhere in `cmd/internal/hostty` (grep for `sanitiz|rowtext` returns one unrelated hit).
- BR-24: plan:528 still writes `--include=*.go` unquoted; re-measured this round in the repo's zsh — `(eval):1: no matches found: --include=*.go`. Same shape at `run.go:1541`.
- BR-30: `run.go:168` still takes `stdin`, `stdout` — and `panes` — none of which the body references.
- BR-31: the envelope (plan ARCH-CONSTRAINTS) still budgets paints and subprocesses only; no entry for the second full parse of every child chunk (`hostScan.FeedFraming`, `run.go:847`) or the unterminated-OSC stall.
- BR-38: `couchtty/reserve.go:143-152` still ends in the doc paragraphs for `sanitize` and `truncate`, which now live in `rowtext`.
- BR-73: all three present — `menu_render.go:625` `SanitizeAndFit((line), width)`, the `rowtext` import inside the stdlib group at `:5`, and `manifest.go:631` out of sort order.
- BR-76: `workshop/lessons.md` gained two 2026-09-08 entries close to BR-70's rule but not it; the "a tracked-artifact edit is an input to a guard, so the commit that ticks a milestone runs the suite" rule is still only in `git log`.

## 5. Test coverage notes

- `make test` exits 0 at HEAD (full run, unsandboxed). All `go test ./...` failures I saw under the sandbox are the documented `operation not permitted` pty/`ps` class and disappear without it.
- Mutation evidence I generated myself: nine-of-nine borderless rungs go red under `paneBlock`; reverting the SCOSC block reds `TestTheCSISpellingOfSaveRestoreHoldsTheSameSlot` and `TestTheTwoSpellingsSettleEachOther`. Both claimed fixes are genuinely pinned.
- The gap that matters: there is no test that asserts the *negative* for an unmodelled save form. `TestParameterisedCSIsIsMarginsNotASave` asserts one negative for DECSLRM; nothing asserts what the model should say about `?1048`/`?1049`, which is why the enumeration could be declared complete while two spellings were missing.

## 6. Architectural notes

- **ARCH-DRY — flag.** Finding (c): one behavioural rule for a shared primitive, implemented in one of its two consumers. Also BR-13 (a validating door with a bypass in production).
- **ARCH-PURE — pass.** `strip.go` and `rowtext` are pure and tested without IO; the mux is the thin seam; `FakeHost`/fake child carry the boundary.
- **ARCH-PURPOSE — flag.** BR-75 and BR-72 are both "the instance the finding named, not the class the finding stated." The ledger now shows `plan-table-drift` ×13 and `external-input-assumed-wellformed` ×5, which per the principle is the ledger reporting that the enumeration was never written rather than that the reviewers are thorough.
- **ARCH-MOCK — pass, with a note.** `probes/cursorsaveslots`, `probes/zellijscrollregion`, `cmd/probes/couchnestedrows` are real live-conformance checks behind `make test-smoke`/named targets, and `zellijprobe` deduplicated the harness. The note is that the model's newest claims (which spellings share zellij's slot) are the exact thing that instrument exists to settle and were not put to it.
- **ARCH-CONSTRAINTS — flag.** BR-31: the per-chunk second parse is undeclared, and a long unterminated OSC holds `MidSequence` up to `maxPending`, stalling any owed paint. `maxOwedDiag` is bounded; the paint slot's unbounded deferral is now *decided* rather than defaulted, which is right — but finding (a) widens the set of children that can hold it open indefinitely.
- **ARCH-SECURE — flag (BR-15).** `rowtext` at the single egress is the correct shape and the C1 gap is closed; the residual is that `hostty.Paint` writes caller text verbatim between `SaveCursor`/`RestoreCursor` with no stated obligation, one package away from the consumer whose input is operator-supplied tab names.
- **ARCH-ORDER — flag (BR-9).** The states×events table exists and the takeover/defer/flush transitions are tested, including the fresh-supersedes-owed inversion. The oracle gap stands: one hand-chosen split index is a sample of size one, and the plan's own note says any index is legal.

## 7. Plan revision recommendations

- **`## Revisions` — the artifact-class enumeration is six classes, not five.** Record that `tests/*.sh` guards are an artifact class asserting the design, name `tests/term-pane-shortcuts-test.sh:253-257` as the instance found at the close boundary, and register the pair.
- **`## Revisions` — the save-form enumeration.** Record the four spellings, which of them the code models today, which are unmeasured against zellij, and what `probes/cursorsaveslots` is expected to answer. If `?1049` is carried as a follow-up rather than fixed here, say so explicitly with the trade-off (gating on it would freeze the strip for a whole alt-screen session), so it is a decision rather than an omission.
- **ARCH-CONSTRAINTS section** — add the two undeclared costs BR-31 names (second full parse per chunk; unterminated-OSC stall bound) with a budget and a bounded behaviour when exceeded.
- **M4.3** — add the `Alt+Shift+d` step, and state whether the runtime-created split half is borderless by swap-layout re-tiling or by nothing; the plan currently claims nine sites are the whole enumeration while a tenth pane is created at runtime.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      plan:696-702 still has no Alt+Shift+d step; and splitTerminalDown (run.go:537) creates the split pane with no --borderless while tests/term-pane-shortcuts-test.sh:255 forbids one, so "each half draws its own strip" is unverified by hand and unasserted by test.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:117 still splits one hand-chosen sequence at one index; no parameterisation over indices.
  - id: BR-13
    disposition: not-addressed
    note: |
      couchtty/console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.
  - id: BR-14
    disposition: not-addressed
    note: |
      grep for "Reservation" in atlas/couch.md returns zero hits; the reserved-row section still reads as couch-owned.
  - id: BR-15
    disposition: not-addressed
    note: |
      No caller obligation stated anywhere in cmd/internal/hostty; grep for sanitiz/rowtext in that package returns one unrelated line about row clamping.
  - id: BR-24
    disposition: not-addressed
    note: |
      plan:528 unchanged; re-measured in the repo's zsh this round -- "(eval):1: no matches found: --include=*.go", exit 1, pipeline never runs. Same shape at run.go:1541.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 still takes stdin and stdout -- and panes -- none of which the body references.
  - id: BR-31
    disposition: not-addressed
    note: |
      The envelope still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming second parse or the unterminated-OSC stall.
  - id: BR-33
    disposition: addressed
    note: |
      atlas:518-523 now scopes coalescing to a paint and the takeover drop to the paint; the ban token was the literal prior text and is registered at plan-superseded-facts-test.sh:156. The atlas still does not state that diagnostics queue and survive a takeover -- worth one sentence, not a re-raise.
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-152 still ends in the two doc paragraphs for sanitize and truncate, which now live in rowtext.
  - id: BR-72
    disposition: not-addressed
    note: |
      Two more live sites, and the class hole is the point: tests/term-pane-shortcuts-test.sh:253-257 is a make-test guard ENFORCING "split panes keep zellij default frames", a sixth artifact class the enumeration never names; atlas/architecture.md:711 still calls the frame the visible divider between split halves. The sweep also left main-3.kdl:22-26 and atlas:381 mid-sentence.
  - id: BR-73
    disposition: not-addressed
    note: |
      All three present at HEAD -- menu_render.go:625 redundant parens, its rowtext import inside the stdlib group at :5, manifest.go:631 out of sort order.
  - id: BR-74
    disposition: addressed
    note: |
      Mutation-verified independently: deleting borderless=true from each of the nine rungs in source and mirror reds the guard in all nine cases, including sole, first-of-pair, last-of-pair and block-opening positions.
  - id: BR-75
    disposition: not-addressed
    note: |
      The named site is fixed and pinned (revert-verified). The RULE is not: measured at HEAD, "\x1b[?1048h" and "\x1b[?1049h" both leave HoldsCursorSave false -- and screen.go:492 actively clears the flag on the alt-screen transition, false for 1049 which is defined as DECSC-then-switch and is how nearly every full-screen child takes the slot. Parameter-free CSI s is also read as SCOSC with ?69/DECLRMM untracked. Write the enumeration as a table with one assertion per spelling, and let probes/cursorsaveslots answer the zellij-side premise.
  - id: BR-76
    disposition: not-addressed
    note: |
      lessons.md's two new 2026-09-08 entries are adjacent rules (mutation-check the guard; stating a class is not closing it) but not BR-70's; that rule is still only in git log.
findings:
  - id: new
    severity: Important
    family: wrong-seam-named
    title: |
      The cursor-save precondition landed in termcmd's private predicate, not at the shared Reservation door, so couch paints unguarded
    detail: |
      This is the 3rd finding in family `wrong-seam-named`, so the rule, not the
      site. hostty.Reservation.Paint / ReserveAndPaint (reserve.go:119,154) emit
      \x1b7 ... \x1b8 -- a property of the MECHANISM this issue lifted into the
      shared package. The precondition M3 measured for it ("never write while the
      child holds a save") was implemented as terminalMux.unsafeToPaint
      (run.go:949), private to one consumer. The other consumer of the same
      primitive, couchtty/console.go:1062 via writeOwn (:1005-1013), gates on
      MidSequence() alone; grep -rn HoldsCursorSave cmd/internal/couchtty returns
      nothing, and couch's children are alt-screen agent TUIs, so the identical
      clobber is reachable there with no guard. THE RULE - a precondition that is
      a property of a shared mechanism belongs at that mechanism's seam, stated so
      every consumer must satisfy it, not in whichever consumer happened to
      discover it. The enumeration is one grep
      (`ReserveAndPaint\|\.Paint(` over cmd, minus tests) and yields exactly two
      production call sites, so a guard over that set is cheap. Same door as
      BR-15's undocumented sanitize obligation - both are obligations of
      hostty.Paint recorded only in the callers that learned them (ARCH-DRY).
```

---

## Re-review — 2026-09-08T18:46:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..f233b29d0cb99a646a8f324ae41340b0e35d0bac |
| command | sdlc close --issue 199 |
| reviewer | claude |
| timestamp | 2026-09-08T18:46:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The window's doc/guard work is genuinely finished this round — BR-72's sixth artifact class (a `make test` guard encoding the superseded design) is registered, the four "the terminal is framed" statements are swept, and I verified the new guard pair fires and the script exits 1 when the old text returns. BR-75's save-slot enumeration is now a real table with a mutation-verified test. What blocks SHIP is the code that landed with them: the BR-77 "shared door" swap replaced `MidSequence()` at four of five production sites in `couchtty/console.go` and left the fifth — the loop that *drains* the deferral — on the old predicate, which I reproduced as a deterministic infinite busy-loop on couch's single writer goroutine (console frozen, 100% CPU); and the new `?1049h` arm makes `SafeToPaint()` false for the entire lifetime of any alt-screen child, which I reproduced as the tab strip never painting again after `\x1b[?1049h` in `termcmd` and couch's row never painting in `couchtty`. Both are regressions against the immediately preceding commit (verified by reverting each change in a scratch copy), and the premise the second rests on — that `?1049h` takes the *same* slot our paint clobbers — is asserted from the spec while `probes/cursorsaveslots`, the instrument this repo built for exactly that question, was not extended to ask it.

## 1. Strengths

- `cmd/internal/ptychild/screen.go:513-541` — the save-slot enumeration is a table with one arm per spelling and a comment that says *why* a missing arm is silent. Mutation-verified: collapsing `1049` back into the `1047/47` arm reds `TestEverySaveSlotSpellingIsAccountedFor` on three sub-cases plus `TestAltScreenWithoutTheSaveHalfDoesNotReleaseTheSlot`. This is the shape BR-75 asked for.
- `tests/plan-superseded-facts-test.sh:137-165` — the artifact-class enumeration is now written down (plan / issue / atlas / config+layout / code comments / **test guard**) and each class carries a registered pair. Verified: restoring `keep zellij default frames` into `tests/term-pane-shortcuts-test.sh` fails the script with exit 1; HEAD is green.
- `tests/term-pane-shortcuts-test.sh:253-267` — the right call on an inverted-meaning check: keep the assertion, restate the reason, one declaration in the layout rather than two. That is the non-obvious answer.
- `cmd/internal/hostty/reserve.go:113-135` — `drawRow` as the shared tail of both painters, with the "they were two copies and the test compared only a substring" rationale, is the right consolidation (ARCH-DRY).
- `cmd/internal/termcmd/run.go:828-880` — "the gate models the terminal, so feed it exactly what the terminal is shown" is a clean invariant, and putting the row-dirty trigger *inside* the active branch for the same reason is consistent rather than incidental.

## 2. Critical findings

**C1 — `cmd/internal/couchtty/console.go:1202`: the drain loop spins forever when a save is taken mid-drain.**
`flushDeferredNotifications` now enters on `!SafeToPaint()` (`:1188`) and `onChunk` defers on `!SafeToPaint()` (`:1111`), but the loop's continue-condition at `:1202` is still `MidSequence()`. When a save is taken by bytes written *during* the drain, the popped chunk re-defers, is re-appended, and the loop pops it again — forever. Reproduced deterministically in a scratch copy (drain never returned within 3s; the same test returns immediately at `f233b29d^`):

```
c1 "\x1b[31"                          -> mid-sequence
c1 <envelope>+"\x1b7"                 -> deferred #1
c2 <envelope>                         -> deferred #2
c1 "m"                                -> safe -> drain -> #1 writes \x1b7 -> #2 re-defers -> SPIN
```

The Run goroutine is couch's only writer, so this is a full console freeze at 100% CPU, not a stalled queue. Fix: `:1202` must be the *same expression* as the defer condition (`!c.hostScan.SafeToPaint()`), and the loop must not re-pop a chunk it cannot make progress on. `grep -rn "MidSequence()" cmd --include=*.go | grep -v _test.go` returns exactly this one production site outside `ptychild` — the enumeration is one grep. (ARCH-ORDER: the defer predicate and its drain predicate are one pair; splitting them makes an illegal state reachable and unwritten.)

**C2 — `cmd/internal/ptychild/screen.go:527-531`: `?1049h` holds the gate closed for the child's entire lifetime, in both consumers.**
`SafeToPaint()` (`:181`) is false while `cursorSaved` is set, and `?1049h` now sets it until `?1049l`. Measured in a scratch copy:

- `termcmd`: feed `\x1b[?1049h`, then five `paintStrip()` + output rounds — no paint reaches the pane. Reverting only the `1049` arm makes the same test paint. So `less`, `vim`, `man`, `htop` in a `pair term` tab freeze the strip for the whole session, and `Alt+r` becomes blind typing again — the exact symptom M3/M4 exist to remove.
- `couchtty`: after `\x1b[?1049h`, `repaint()` writes nothing at all.

`flushOwed`'s own doc (`cmd/internal/termcmd/run.go:995-1024`) justifies unbounded deferral by "the debt is paid by ... the child's own DECRC" — under `?1049h` there is no DECRC until the child exits, so the justification no longer covers the case the code just created. The load-bearing premise (that `?1049h`'s save is the *same* slot a console `\x1b7`/`\x1b8` clobbers — false in xterm, where each screen buffer has its own saved cursor) is asserted from the spec; `probes/cursorsaveslots` exists precisely to answer that under zellij and was not extended, which is the half of BR-75 that asked for the instrument. Fix: extend the probe to answer "does a DECSC on the alt screen clobber the `?1049h` save under zellij", then either narrow the gate (`!(cursorSaved && !altScreen)`) or give the paint a release the child cannot withhold. Do not ship the freeze on an unmeasured premise.

## 3. Important findings

**I1 — `cmd/internal/ptychild/screen.go:181`: `SafeToPaint` is the one new exported shared-package symbol in this window with no plan row, no atlas sentence, and no test that names it.**
Enumerated from the diff — 16 exported declarations added to `cmd/internal/{hostty,ptychild,rowtext,workbenchshortcut}`; 15 appear in the plan's Core-concepts material, `SafeToPaint` appears 0 times. `grep -rn SafeToPaint` outside the three production files returns nothing, so the door BR-77 asked for is unpinned in every artifact class. `atlas/architecture.md:604-615` still names `HoldsCursorSave` as "the bit that says so" and does not record that couch now asks the same door. The Core-concepts guard is table→code only (`plantable_test.go:189`, direction gap attributed to pair#188), so nothing can catch this direction. This is the 13th finding in family `plan-table-drift`; the rule: the artifact enumeration BR-72 built for *superseded* facts needs its mirror for *added* shared surface — table row + atlas sentence + a test naming the symbol — and that enumeration is producible from the diff (new exported decls under `cmd/internal/`), which is what makes it a guard rather than a habit.

## 4. Minor findings

- `atlas/architecture.md:517-521` — the BR-33 sweep left a line-wrap artifact ("queueing would draw a / burst of stale rows"); same family as the still-open BR-73.
- The save-slot table omits `?47l` (it is covered only by the sibling test), so the table and the tests enumerate slightly different sets.
- `cmd/internal/ptychild/screen.go:238` — the framing scanner recognises only `0x1b`-introduced sequences; 8-bit C1 `CSI` (`0x9b`) is invisible to both `MidSequence` and `classify`. Pre-existing, out of this window's scope, worth a line where the enumeration is written.

## 5. Test coverage notes

- The couch half of BR-77's fix is unpinned: reverting all four `SafeToPaint` call sites in `console.go` to `MidSequence()` leaves `go test ./cmd/internal/couchtty/` green (only the sandbox-blocked `TestNotificationPTYConformance` fails, as it does at HEAD). Per the claimed-fix rule, that fix has no failing test behind it.
- Nothing exercises the deferral machinery with a *save* rather than a mid-sequence chunk on the couch side; `TestCrossActorNotificationDeferral` covers only the mid-sequence path, which is why C1 ships green.
- No test covers "child on the alt screen" against either reserved row, which is why C2 ships green. Both C1 and C2 were reproducible in under 30 lines using the existing `notificationConsole`/`stripMux` harnesses — the harnesses are good, the cases are missing.
- `go test ./...` at HEAD in a scratch copy fails only the documented sandbox-pty and no-`.git` classes. Both shell guards (`plan-superseded-facts-test.sh`, `term-pane-shortcuts-test.sh`) pass at HEAD and were mutation-checked.

## 6. Architectural notes

- **ARCH-DRY** — pass on the mechanism (`drawRow`, `rowtext`, one `SafeToPaint`), flag on the *obligation*: `hostty.Reservation.Paint` is the thing that emits `\x1b7 … \x1b8`, and neither the sanitize obligation (BR-15) nor the cursor-save precondition is stated at that door; both live in the callers that learned them.
- **ARCH-PURE** — pass. `Screen`, `Reservation`, `rowtext`, `RenderStrip` are pure and tested without IO; the zellij seam stays behind `Runtime`.
- **ARCH-PURPOSE** — flag (I1). The shadow-sweep of *consumers* passes: both production callers derive from one predicate. The sweep of *added surface* does not.
- **ARCH-MOCK** — flag (C2). The repo's live instrument for terminal-semantics premises exists and was not asked the question the code now depends on.
- **ARCH-CONSTRAINTS** — flag. BR-31 is still open (the second full parse per chunk is undeclared), and C2 adds a second undeclared envelope violation: a paint deferral whose release the child can withhold for its whole lifetime.
- **ARCH-SECURE** — pass on input handling (`rowtext.Sanitize` at the egress, replay fed to the gate), open on BR-15's undocumented caller obligation.
- **ARCH-ORDER** — flag (C1). `cursorSaved` is now one bit standing for two different slots (the pre-alt-screen save and an in-alt DECSC), and the drain loop's transition condition is a different expression from the defer condition it drains. Both are "legal combinations left unwritten"; the fix is one predicate, asked in both places.

## 7. Plan revision recommendations

- `## Revisions` entry for the save-slot model: `?1049h` now TAKES the slot (reversing the prior "an alt-screen transition abandons the save"), what that costs (no paint for the child's alt-screen lifetime), and what measurement settles it. Nothing in the plan, issue, or atlas currently records this decision — it exists only in a code comment and a commit message.
- Core-concepts table: add a row for `ptychild.Screen.SafeToPaint` at `cmd/internal/ptychild/screen.go`, status `new`, and state in the bullets that it is the shared paint door both consumers ask.
- ARCH-CONSTRAINTS section: add the deferral-release entry (what clears the gate, and the bound when the child does not clear it), alongside the still-open BR-31 per-chunk parse entry.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      plan:696-702 still has no Alt+Shift+d step; the split rungs remain hand-unverified.
  - id: BR-9
    disposition: not-addressed
    note: |
      writer_test.go:117 still splits one hand-chosen sequence at one index.
  - id: BR-13
    disposition: not-addressed
    note: |
      console.go:922 still returns hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom} directly.
  - id: BR-14
    disposition: not-addressed
    note: |
      grep Reservation in atlas/couch.md returns nothing; hostty appears only at :213 and :812.
  - id: BR-15
    disposition: not-addressed
    note: |
      No sanitize/clamp obligation in cmd/internal/hostty; the same door now also omits the cursor-save precondition.
  - id: BR-24
    disposition: not-addressed
    note: |
      plan:528 unchanged; the unquoted --include=*.go still aborts in the repo's zsh.
  - id: BR-30
    disposition: not-addressed
    note: |
      run.go:168 still takes panes, stdin and stdout; none is referenced in the body.
  - id: BR-31
    disposition: not-addressed
    note: |
      plan:390-410 still budgets paints and subprocesses only; no entry for the per-chunk FeedFraming or the stall.
  - id: BR-38
    disposition: not-addressed
    note: |
      couchtty/reserve.go:143-152 still ends in the sanitize and truncate doc paragraphs for functions now in rowtext.
  - id: BR-72
    disposition: addressed
    note: |
      Sixth class registered and the sweep is complete; verified the new GUARD pair fires and the script exits 1.
  - id: BR-73
    disposition: not-addressed
    note: |
      All three present; plus a third instance from this round's sweep, the line-wrap artifact at atlas:517-521.
  - id: BR-75
    disposition: addressed
    note: |
      Enumeration is a table and mutation-verified; its 1049 arm creates the new Critical, raised separately.
  - id: BR-76
    disposition: not-addressed
    note: |
      lessons.md untouched this round; BR-70's rule still lives only in git log.
  - id: BR-77
    disposition: not-addressed
    note: |
      Reverting all four couch call sites to MidSequence leaves the suite green, the obligation is still unstated at hostty.Paint, and the swap as landed spins.
findings:
  - id: new
    severity: Critical
    family: moved-surface-drops-a-case
    title: |
      The predicate swap reached four of five sites; the drain loop kept the old one and spins forever
    detail: |
      couchtty/console.go:1202 still continues on MidSequence() while :1111 defers on
      !SafeToPaint(), so a save taken during the drain makes the loop re-pop a chunk it
      cannot progress on. Reproduced deterministically (drain never returns in 3s; the same
      test returns instantly at f233b29d^). couch's Run goroutine is its only writer, so this
      is a frozen console at 100% CPU. THE RULE - a defer condition and the loop that drains
      it are ONE pair and must be the same expression; after replacing a gate predicate,
      grep the old name (one production site remains outside ptychild).
  - id: new
    severity: Critical
    family: deferred-work-lacks-own-trigger
    title: |
      The new 1049 arm holds the paint gate closed for the child's whole alt-screen lifetime, in both consumers
    detail: |
      screen.go:527-531 makes cursorSaved true from ?1049h until ?1049l, and SafeToPaint
      (:181) is false throughout. Measured - in termcmd the strip never paints again after
      \x1b[?1049h (five paint rounds, nothing reaches the pane; reverting only that arm
      paints), and in couchtty repaint writes nothing. So less/vim/man freeze the strip and
      make Alt+r blind typing, the symptom M3 exists to remove. flushOwed's justification for
      unbounded deferral ("paid by the child's own DECRC") no longer covers this case, and
      the premise that ?1049h takes the SAME slot our paint clobbers is asserted from the
      spec while probes/cursorsaveslots, built for exactly this question, was not extended.
      This is the 3rd finding in family deferred-work-lacks-own-trigger. THE RULE - a defer
      condition is only admissible with a release the deferrer can observe will arrive;
      widening it to a condition the child controls indefinitely requires either a narrower
      condition or a bounded release, and the measurement that settles which.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      SafeToPaint is the one new exported shared-package symbol with no plan row, no atlas sentence and no test
    detail: |
      This is the 13th finding in family plan-table-drift, so the rule, not the site.
      Measured - the window adds 16 exported declarations under cmd/internal/{hostty,
      ptychild,rowtext,workbenchshortcut}; 15 appear in the plan's Core-concepts material and
      SafeToPaint appears zero times, in the plan, in the atlas, and in any test.
      atlas/architecture.md:604-615 still names HoldsCursorSave as the bit that says so. The
      Core-concepts guard is table-to-code only, so this direction cannot fire. THE RULE -
      the artifact enumeration BR-72 wrote for SUPERSEDED facts needs its mirror for ADDED
      shared surface (table row, atlas sentence, a test naming the symbol), and that
      enumeration is producible from the diff's new exported declarations rather than from
      memory.
```
