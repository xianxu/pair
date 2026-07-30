# Boundary Review — pair#132 (whole-issue close)

| field | value |
|-------|-------|
| issue | 132 — Alt+h keybind help is a circular dead end |
| repo | pair |
| issue file | workshop/issues/000132-alt-h-keybind-help-dead-end.md |
| boundary | whole-issue close |
| milestone | — |
| window | 277978cb6000d1c0fc54cae549ff2c04b5461bce..HEAD |
| command | sdlc close --issue 132 |
| reviewer | claude |
| timestamp | 2026-07-29T23:10:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of #132 is genuinely delivered and well-built: `Alt+h` now pages `pair keys`, wording is derived per-row from four named sources with no "whichever source has prose wins" fallback, and the bidirectional drift tests plus `TestEmbeddedSourcesMatchTree` install the anti-rot property that #99 M5c lacked. The parser work in particular is the strongest part of the diff — the three-way `KeymapScan` and argument-position rule close PQ-1 properly rather than cosmetically. What blocks SHIP is that the four-source enumeration turned out to be five: `Alt+←`/`Alt+→` (previous/next terminal tab) are handled in `termcmd`, are documented in README, and are missing from the "Terminal tabs" section — and no test can catch that, because the role-coverage test hardcodes its chord list instead of deriving it. Alongside that, the plan's Task 9 doc sweep was not completed: `atlas/architecture.md:407` still says `Alt+h` "pops a floating pane running `pair -h | less`", directly contradicting line 274 added by this same diff.

## 1. Strengths

- **The parser earns its complexity.** `ParseNvimKeymaps` takes the lhs by argument *position* (`parse.go:50`, `argAt` at `parse.go:101`) rather than "second quoted string", and the three-way split is pinned by `TestParseNvimKeymapsUnquotedLhsIsUnresolvedNotMisassigned` plus the named 3-site allowlist in `TestParseNvimKeymapsReconcilesAgainstRealFile` (`parse_test.go:161-175`). I verified against the tree: `nvim/init.lua` has 34 `vim.keymap.set(` calls and 34 `desc = 'pair: ` lines, with unquoted lhs at `:3873`, `:3878`, `:3932`. A *new* unquoted-lhs keymap fails the build rather than vanishing — which is exactly what a bare count could not do.
- **No wording fallback, and it's tested where it bites.** `descFor` (`sections.go:104-135`) errors instead of borrowing; `TestRoleLocalWordingComesFromRoleTableNotNvimNoOp` asserts the rendered help never contains "disabled in draft" and does contain "new terminal tab". The `Alt+k`-renders-twice test pins (key, context) identity. This is the PQ-2 fix done right.
- **The tree-vs-embedded asymmetry is handled correctly, not hand-waved.** Classification reads `../../../nvim/init.lua`; `TestEmbeddedSourcesMatchTree` (`drift_test.go:83-98`) is the single tie-back so a stale gitignored bundle fails by name. This is the PQ-3 fix, and the comment explaining *why* is accurate.
- **`textwidth` extracted, not copied** (`textwidth/textwidth.go`, `launcher/list.go:90-92`) — ARCH-DRY resolved by extraction with the caller delegating, and the wide-glyph test cases (`⏎` narrow, `📁` wide) pin the distinction that matters.
- **The exit-0 contract is real reasoning, not a comment.** `keyscmd.Run` returns 0 on source failure with a visible body (`keyscmd.go:44-49`) and `TestRunExitsZeroAndExplainsWhenSourcesFail` proves it through an injected failing reader — because `bin/pair-help:7` runs under `set -euo pipefail`.
- **PQ-4's deferred detail was delivered:** `TestParseZellijRunBindsAttributesRunToItsOwnBind` (`parse_test.go:120-131`) covers the Write-immediately-before-Run shape the round-2 disposition punted to this review.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `Alt+←` / `Alt+→` (terminal tab switching) are missing from the help, in the section named for them.** `cmd/internal/termcmd/run.go:484-489` handles `ChordAltLeft`/`ChordAltRight` as `previousTab()`/`nextTab()`, and `README.md:117` documents them ("layout 3 terminal | Switch local terminal tabs"). They appear in neither `roleBindings` (`workbenchshortcut/shortcut.go:154-161`) nor the catalog, so `pair keys` renders "Terminal tabs (in the right terminal)" with new/close/rename/split/jump-back/toggle-width and no way to change tabs. The design's "four sources, verified during design" enumeration missed that the terminal chord surface is split across two seams: `Decide`'s terminal branch (`shortcut.go:198-228`) and `termcmd`'s `handleTerminalChord` — and their sets differ (`handleTerminalChord` has AltLeft/AltRight but not AltR, which is special-cased at `run.go:408`). Fix: add `{Chord: ChordAltLeft, Role: PaneRoleRightTerminal, Help: "previous terminal tab"}` and the `ChordAltRight` twin to `roleBindings`, two `roleChordKey` cases returning `"Alt+←"`/`"Alt+→"`, and two `groupTerminal` catalog rows. The `(key, context)` identity rule already makes these safe against the existing `<M-Left>`/`<M-Right>` history rows.

**I2 — `TestRoleBindingsCoverTerminalSwitch` hardcodes what it claims to derive** (`workbenchshortcut/shortcut_test.go:448`). Its doc comment says "The role table must cover every chord `Decide` actually handles for the right terminal", but the body iterates a hand-written `handled := []Chord{...}` slice, so it only asserts that a hand-maintained list matches a hand-maintained table. The plan's Task 4b Step 1 specified deriving from `Decide`; the implementation substituted a literal. Adding a chord to either terminal seam fails nothing — which is how I1 slipped through. Fix: iterate the dense `Chord` iota (`ChordUnknown+1` .. `ChordAltShiftEnter`), require a `roleBindings` row for every chord where `Decide(ShortcutInput{Role: PaneRoleRightTerminal, Chord: c}).Disposition == DispositionHandle` and `DecideGlobal(c)` is false; and add the mirror test in `termcmd` asserting every chord `handleTerminalChord` returns `true` for appears in `workbenchshortcut.RoleBindings()`.

**I3 — the atlas now contradicts itself; the Task 9 doc sweep was not completed.** The plan's Task 9 Step 1 sweep covered `atlas/` and `zellij/`, but four stale claims survive:
- `atlas/architecture.md:407` — "`Alt+h` … pops a floating pane running `pair -h | less`" (directly contradicts `:274` added in this diff)
- `atlas/architecture.md:19` — "`bin/pair-help` # shell shim: `pair -h` in an ESC-to-quit pager"
- `zellij/config.kdl:159` — "Alt+h — pop up a large floating pane with `pair -h` in a pager" (this file ships in the runtime bundle)
- `atlas/go-migration-inventory.md:138` — "Displays `pair -h` through `less`"

`:407` also trips the issue's own Done-when ("No text anywhere claims `pair --help` shows keybindings unless it does"). All four are one-line repoints to `pair keys`.

**I4 — the `Alt+h` chain has no automated coverage end to end.** Two gaps:
- `dispatcher_test.go:16` lists the implemented set that must be present under a comment stating "The full implemented set MUST be present … if one of these were accidentally left `planned`, `pair changelog` would fall through to the launcher (start a session)". `"keys"` is not in that list, and no test exercises `Dispatch([]string{"keys"})`. A regression there makes `Alt+h` launch a session inside the floating pane. Fix: add `"keys"` to the list and one `Dispatch` routing assertion mirroring `TestDispatchContextReturnsHelperOutput`.
- Nothing mechanically checks that `bin/pair-help` invokes `pair keys`. `tests/pair-embedded-runtime-test.sh:47` only asserts the file is executable. This repo already has the pattern for exactly this — `contextcmd/panejson_kdl_test.go` runs the real shell line with stubbed binaries on `PATH` (ARCH-MOCK). This was PQ-6's second sub-point ("a mechanical check of the bundled shim plus `pair keys` output is cheap"), which the round-2 disposition marked "addressed" while addressing only the awk/pipefail/centering half.

**I5 — `README.md`'s CLI synopsis omits `pair keys`.** `launcher/help.go:21` now lists `pair keys` in `UsageText`, but the mirrored block at `README.md:262-281` (same rows: `pair list`, `pair rename`, `pair -h, --help`) was not updated, so README and `pair -h` have silently diverged on the very subcommand this issue adds. One line.

**I6 — the anti-rot guard depends on an unenforced string convention.** `descMarker = "desc = 'pair: "` (`parse.go:141`) is an exact literal, and `TestParseNvimKeymapsReconcilesAgainstRealFile` counts with the *same* literal (`parse_test.go:150`). A future keymap written `desc = "pair: …"` (double-quoted) or `desc='pair: …'` is invisible to the parser *and* to the reconciliation, so `TestEveryNvimKeymapIsClassified` cannot flag it — it never appears in `scan` at all. The load-bearing "cannot reach a release undocumented" claim (`drift_test.go:9-18`, `atlas/architecture.md:280`) is therefore conditional on formatting nobody checks. Fix: in the reconciliation test, count with a regexp (`desc\s*=\s*["']pair: `) and assert it equals the strict-marker count, so a deviant spelling fails loudly rather than disappearing.

## 4. Minor findings

- `TestEveryCatalogEntryStillExists` checks only `Catalog.Keys()` (the include list); a stale entry in `internal` (e.g. `<M-t>` after the draft no-op is removed) is never flagged — same reverse-drift, one list over.
- `keyscmd` uses a package-level mutable `var sources = func() …` as its test seam (`keyscmd.go:16`), where every sibling in the dispatcher table injects instead (`contextcmd.Run(args, EnvFromOS(), …)`, `agentcmd.RunRestart(rest, os.Getenv, OSRuntime{}, …)`). Passing a `keyhelp.SourceReader` to `Run` would match the convention and make the test parallel-safe.
- `keyscmd.Run` silently ignores unknown arguments — `pair keys --bogus` prints the help and exits 0. `--center=120` is also unsupported.
- `ParseZellijRunBinds` captures only the first key of a multi-key `bind "A" "B" { … }` (`parse.go:210-215`). Not reachable in today's `config.kdl` (all 20 binds are single-key), but a future two-key `Run` bind would be half-documented with no test failing.
- `isBarePrintableKey` (`parse.go:172`) lives in the non-test file but is referenced only from `parse_test.go`.
- `Ctrl+/` is listed under "Draft — compose and send" (`catalog.go:52`), but `init.lua:3801-3807` documents it as the transport `pair clip clipboard-to-pane` sends after a mouse selection. Its desc is honest; the grouping reads as a compose gesture.
- `README.md:127` says `Alt+q` pushes to the **front** of the queue (`+1`) while the now-authoritative derived wording says "queue current draft for later (back of queue)". Pre-existing, but the diff's new README paragraph ("that list is **derived** … so it cannot drift out of date") invites the reader straight into the contradiction. Worth resolving one way or the other.
- The durable plan `workshop/plans/000132-alt-h-keybind-help-dead-end-plan.md` has **zero** ticked checkboxes despite all nine tasks being complete (`grep '^- \[x\]'` → 0 matches). The issue's `## Plan` chunks are ticked; the plan artifact was not.

## 5. Test coverage notes

Coverage is strong where the design put its attention and thin exactly at the seams:

- **Well covered:** parser shape invariants (including a table-driven property over mode form × lhs form × whitespace), the reconciliation against the real `init.lua`, the wording-join rule, dual-context rendering, per-section display-width alignment, centering by columns not bytes, the exit-0-on-source-failure contract, and the bundle-vs-tree tie-back. The log's claim that the drift test was mutation-tested (adding an undocumented keymap → `TestEveryNvimKeymapIsClassified` fails) is consistent with the code as written.
- **Uncovered:** the dispatcher route for `keys` (I4), the `bin/pair-help` shim (I4), the terminal-chord surface owned by `termcmd` (I1/I2), and any deviation from the `desc = 'pair: ` literal (I6). Every one of these is a path where the *shipped* Alt+h experience breaks while the whole suite stays green — which is the failure mode this issue exists to eliminate.
- `Sections(DefaultSources())` in `sections_test.go` and `keyscmd_test.go` depends on the gitignored `assets/` having been generated. `make test` regenerates it transitively (`test-review: $(BIN_DIR)/pair` → `$(RUNTIMEBUNDLE_ASSETS)`), but a bare `go test ./cmd/internal/keyhelp/ ./cmd/internal/keyscmd/` on a fresh tree fails. Worth one sentence in the package doc so the next reader doesn't chase it.

## 6. Architectural notes

- **ARCH-DRY — pass, with one note.** `textwidth` is a real extraction with the original caller delegating, not a copy. `roleChordKey` (`catalog.go:143`) as a second Chord→string map is correct and correctly justified: `ChordName` is a wire format round-tripped by `ChordFromName`, so coupling display to it would let a rewording break routing. The note: after I1's fix, the terminal chord surface is described by `roleBindings` but *driven* by two separate switches (`Decide` and `handleTerminalChord`). Three places for one fact is the DRY pressure to watch; I2's derived test is the cheap way to hold it without a risky dispatch rewrite.
- **ARCH-PURE — pass.** Parsers, `Render`, `Center` and `Sections` are pure over injected input; `sources.go` is the only IO and is a 6-line adapter. `failingSources` in both `keyhelp` and `keyscmd` proves the seam is real rather than decorative. The drift tests read the filesystem by design (they are conformance checks against real assets), which is the right call, not a purity violation.
- **ARCH-PURPOSE — flag (I1).** The shadow-sweep: `pair keys`/Alt+h derives ✓; `PAIR_CHEATS` is a declared non-goal ✓; the Homebrew caveat is #131's, in another repo ✓; README's table is a declared hand-maintained restatement that now says the derived list is authoritative ✓. But the derived list itself is **incomplete** — the fifth source (`termcmd`) was never enumerated, so a user-facing terminal-tab binding is absent from the section named for it. Under this principle a missing consumer is a finding, and this one is inside the thing being delivered rather than beside it. Making README's table generated is genuinely a separate issue; getting `Alt+←/→` into the derived list is not.
- **ARCH-MOCK — flag (I4).** No external binary or service is consumed by the Go code, so the principle mostly doesn't bite. Where it does: `bin/pair-help` shells out to `pair`, `tput` and `less` outside any seam, and there is no test that can run that stack. The repo's own `panejson_kdl_test.go` shows the shape — extract the real command line from the shim, run it with stubs on `PATH`, assert the output is the keybinding list. Production flow and test flow currently do not share a boundary here.

## 7. Plan revision recommendations

1. **Core concepts — `UsageText` path.** The table states `cmd/internal/launcher/format.go`; the function is at `cmd/internal/launcher/help.go:21`. Footnote ¹ makes this non-blocking, but the row should name the real file so the table stops asserting a path that does not exist.
2. **Scope note — "four sources, not one" is wrong; there are five.** Add a `## Revisions` entry recording that terminal tab switching (`Alt+←`/`Alt+→`) is handled by `cmd/internal/termcmd/run.go:484-489`, not by `Decide`, so the terminal chord surface is the *union* of two switches. Name it explicitly so the next reader doesn't re-derive the same incomplete inventory.
3. **Task 4b Step 1 — record the substitution.** The plan's test derives handled-ness from `Decide`; the shipped test hardcodes six chords. Either restore the derivation (preferred — see I2) or revise the plan so it stops claiming a property the code doesn't have.
4. **Task 9 Step 1 — record that the sweep is incomplete.** Four `pair -h` claims survive in `atlas/` and `zellij/`. Once fixed, note in the issue `## Log` that the sweep was re-run and what it found, given the `lessons.md` entry this same branch added about verification commands needing verification.
5. **Plan gate ledger — PQ-6.** The round-2 disposition marked PQ-6 "addressed", but its shim-conformance sub-point was not delivered. Add a note so the ledger reflects what actually shipped rather than closing a finding the code doesn't satisfy.
