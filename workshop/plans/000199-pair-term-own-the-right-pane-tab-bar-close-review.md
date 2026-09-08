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
