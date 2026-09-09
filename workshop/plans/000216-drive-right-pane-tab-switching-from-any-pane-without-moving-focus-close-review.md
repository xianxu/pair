# Boundary Review — pair#216 (whole-issue close)

| field | value |
|-------|-------|
| issue | 216 — drive right-pane tab switching from any pane, without moving focus |
| repo | pair |
| issue file | workshop/issues/000216-drive-right-pane-tab-switching-from-any-pane-without-moving-focus.md |
| boundary | whole-issue close |
| milestone | — |
| window | d15201957318de0b4a7172db0798cf176bc80d82..af1d98df34a53ae2467c4e2bba790de7d0bee22e |
| command | sdlc close --issue 216 |
| reviewer | claude |
| timestamp | 2026-09-09T12:04:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The mechanism is the right one and it is built the way the plan says: `Alt+Shift+←/→` become globals that the focused pane handles *itself* (`GlobalBinding.HandledInPane` withholds `DraftLuaFunction`, which the two Go executors branch on first), the right pane calls the same `mux.previousTab()/nextTab()` that `Alt+←/→` already call, and the other two panes deliver the *existing* `Alt+Left/Right` bytes into the terminal's pty via `zellij action write --pane-id` — so there is still exactly one implementation of tab switching. I verified `zellij 0.44.3`'s `action write [OPTIONS] [BYTES]...` with `--pane-id` against the live binary, ran the new tests green, and confirmed by `-overlay` mutation that `TestChordMaxFollowsEveryEncodedChord` genuinely reddens when the sentinel is moved above the new chords — the PQ-3 fix is pinned by a failing test, not decorative. What keeps this from SHIP is documentation, not code: `README.md:131` still tells the reader that `Shift+Alt+←/→` jump to region boundaries — the `nav_boundary` feature this diff **deleted** — and no README row describes the chord's new meaning, so the repo's own narrative keybinding table now contradicts the binary. Also note the `- [ ] Manual` plan item is genuinely unticked; the close gate's plan-check will fire and should be answered by running the live gesture rather than waived.

### 1. Strengths

- **The delivery seam is the elegant choice.** `workbenchshortcut.DeliverChordArgs` (`shortcut.go:684`) keeps the wire format in the package that owns it and returns argv, so `layoutcmd` does the IO and the byte knowledge is never restated. `pair term` needs zero changes to accept a written chord — the property the Spec called out.
- **PQ-3 was swept as a class, not an instance.** Both hand-maintained bounds moved to `chordMax` (`shortcut_test.go:474`, `run_test.go:1200`), and a grep of the tree finds no surviving `chord <= Chord…` bound anywhere. `TestChordMaxFollowsEveryEncodedChord` asserts the sentinel against the *encoding table* rather than the current last chord's name, which is what makes the guard survive the next append (ARCH-PURPOSE).
- **The picker was reused rather than re-derived.** `SwitchRightTerminalTab` (`layoutcmd.go:101`) calls the unexported `pickRightTerminal` in place, so `Alt+k` and `Alt+Shift+arrow` cannot disagree about which split half they mean — and it did so *without* the export the plan authorised (ARCH-DRY).
- **The right pane's "no delivery" claim is actually pinned.** The `pumpStdin` harness asserts `rt.ops` exactly (`run_test.go:212`), so the four new rows at `run_test.go:175-179` prove the in-place path spawns no zellij call — including two split-arrival rows that exercise the `held` buffer in both directions.
- **`switchTerminalTab` as a var** (`wrap.go:1648`) makes the *wiring* testable, not just the delivery. A missing `case` in `executeWorkbenchDecision` would otherwise have left every test green while the agent pane's chord did nothing.

### 2. Critical findings

None.

### 3. Important findings

**`README.md:131` — documents the deleted feature; no row for the new one.**
The row still reads `**Shift+Alt+←** / **Shift+Alt+→** | nvim (normal/insert) | Jump to the next region boundary: oldest-history, newest-history, *, front-of-queue, back-of-queue.` That behaviour (`nav_boundary`) was deleted in this same diff. The README preamble (`README.md:104-106`) explicitly frames this table as the hand-maintained *narrative* restatement of the chord surface — which makes it the one consumer of `globalBindings` that does not derive, and it was not swept (ARCH-PURPOSE shadow-sweep). Fix: delete line 131 and add a row next to `README.md:122` (`Alt+← / Alt+→ | layout 3 terminal`), e.g. `| **Shift+Alt+←** / **Shift+Alt+→** | any pane | Switch the right terminal's tabs from wherever focus is, without moving focus — the draft keeps the cursor. |`. The atlas half of the docs gate *was* done (`atlas/architecture.md:715`).

**`shortcut.go:370-372` — only the `;4` spelling; the `;9`-family analog is missing.**
`ChordAltLeft` deliberately carries three spellings (`\x1b[1;3D`, `\x1b[1;9D`, `\x1b[3D`) because modifier `9` is the *meta* encoding of the same key (added defensively in `e6eee5a3`). The shift analog of `1;9` is `1;10`, and neither `\x1b[1;10D` nor `\x1b[1;10C` is in the table. The failure mode is the awkward one: on a terminal that reports meta-style modifiers the chord still works from the **draft** (nvim resolves `<S-M-Left>` itself) but is silently dead in the agent pane and the right pane — a partial failure of the "from any pane" Done-when that the operator's single-terminal manual test cannot rule out for other terminals. Fix: two more rows; `TestNoChordSequenceIsAProperPrefixOfAnother` already covers the shadowing risk.

### 4. Minor findings

- `shortcut.go:694` — `for _, b := range encodings[0]` ranges a *string*, so `b` is a `rune`, not a byte; `strconv.Itoa(int(b))` is correct only because every encoding is ASCII. `for _, b := range []byte(encodings[0])` says what is meant and stays right if a non-ASCII byte ever lands in the table.
- `nvim/init.lua:2446` — `pos_rank` is now orphaned: its only caller was the deleted `nav_boundary`, and the sole remaining mention is the comment at `init.lua:3402`. No lua linter runs in `make test`, so nothing will flag it.
- `keyhelp/catalog.go:83-87` — the comment says "the per-row context column is what distinguishes them from the terminal-only rows above". `pair keys` renders no context column; what actually distinguishes them is the `Help` wording ("from any pane, without moving focus"), which does the job. Related: the group heading is `Terminal tabs (in the right terminal)` while these two work from anywhere — acceptable, since the row text corrects it, but the comment's justification is not the mechanism that holds.
- ARCH-DRY: the *Action → delivered chord* mapping (`prev ≡ ChordAltLeft`, `next ≡ ChordAltRight`) is restated at three call sites — `termcmd/run.go:197-200`, `wrapcmd/wrap.go:1674-1679`, `layoutcmd.go:130-136` — plus implicitly in the `handleTerminalChord` case grouping. A two-line `workbenchshortcut.TabChordFor(action)` would make it one fact.
- `nvim/init.lua:3409-3410` copy-pastes the `PAIR_HOME .. '/bin/pair'` fallback idiom for a fifth time (also at 755, 767, 950). Pre-existing repetition; a `pair_bin()` local would end it.

### 5. Test coverage notes

- **Mutation claim verified, not taken on faith.** I rebuilt `shortcut.go` with `chordMax` moved above `ChordAltShiftLeft` and ran the suite through `go test -overlay`: `TestChordMaxFollowsEveryEncodedChord` fails with both new chords named. The Log's sweep table is accurate on the row that mattered most.
- All new tests pass. The package-level `FAIL`s I saw in `termcmd`/`wrapcmd` are the environment's pty restriction (`ptychild: … operation not permitted`) on `TestHarnessTTYCapture*`, `TestTerminalMux*`, `TestSIGUSR2ReExecs…` — unrelated to this diff, and consistent with the repo's known sandbox behaviour.
- Coverage matches the risk surface: pure argv (`shortcut_test.go:493`), decision-from-all-three-roles with an explicit `DraftLuaFunction == ""` assertion (`:545`), the picker preference under a two-half split (`layoutcmd_test.go:196`), the inert no-terminal case (`:222`), CLI argument rows including `too many` (`:236`), the agent-pane wiring (`keymap_registry_test.go:36`), the draft's Lua route with `focus = false` (`workbench_route_test.lua:33`), and split-arrival scanning.
- One residual gap, by design rather than oversight: the byte encoding `\x1b[1;4D`/`\x1b[1;4C` is *derived* from the shift+alt family, never measured. That is exactly what the unticked `- [ ] Manual` step buys, and it should be run rather than waived.

### 6. Architectural notes

- **ARCH-DRY** — flagged (Minor): action→chord mapping restated 3×; `bin/pair` idiom 5×. Otherwise a good pass — the picker, the mux method, and the byte table each have one home.
- **ARCH-PURE** — pass. `DeliverChordArgs` is deterministic, IO-free and table-tested; `SwitchRightTerminalTab` is thin glue over an injected `Runtime`.
- **ARCH-PURPOSE** — flagged (Important, README). The three executors were all delivered — no "follow-up" swallowed the point — and the PQ-3 class was enumerated and swept in the same round. The one consumer that still hand-restates the model is `README.md`.
- **ARCH-MOCK** — pass. zellij stays behind the `Runtime` seam; `fakeRuntime` records the exact argv, and production and test flow share that boundary. There is no automated conformance check for `write --pane-id`; I verified it manually against `zellij 0.44.3` this round. Worth noting the repo has no scheduled zellij-CLI conformance job at all — a standing gap, not one this diff introduced.
- **ARCH-CONSTRAINTS** — pass with a note. The agent pane blocks its stdin translate loop on two synchronous `zellij action` spawns per press (`wrap.go:1675`), roughly ~100ms. That matches the sibling `ActionFocusRightTerminal` case and the issue explicitly accepted a subprocess per deliberate gesture. Critically, the **draft** — the pane the requirement is about — is async (`jobstart{detach = true}`), so typing genuinely is not interrupted.
- **ARCH-SECURE** — pass. argv arrays, never a shell string; `paneID == ""` rejected in `DeliverChordArgs`; sidecar read failures degrade to `""`/`nil` and cost the picker a preference rather than propagating.
- **ARCH-ORDER** — pass with one unmodelled interleaving. The scanner's `held` state is exercised with split-arrival rows both directions; stale-pane-id and rapid-double-press are stated with their chosen policy (ignore / no rollback). Unmodelled: `(rename session active) × (delivered chord)`. While `rename != nil`, `pumpStdin` routes bytes to `applyRename` before chord scanning (`run.go:428-433`), so an `Alt+Shift+←` sent from the draft while the right pane sits in the `Alt+r` rename editor feeds ESC into that editor. Identical to pressing `Alt+←` locally — but the remote presser cannot see that a rename is in progress, which is what makes it a new event rather than the same one. Worth one sentence in the issue's ordering paragraph; not worth code.

### 7. Plan revision recommendations

None for the mechanism — the 2026-09-09 "implementation" Revisions entry already records all three deviations (CLI verb moved to `layoutcmd`, PQ-4 resolved without exporting the picker, no `config.kdl` change, measured) and the code matches each. If the README finding is taken, add a short entry noting that the plan's `## Plan` "Atlas + `pair keys`" step under-scoped the docs sweep: `README.md` is the third hand-maintained restatement of the chord surface and belongs in that step alongside `atlas/` and `Help:`, so the next chord change sweeps all three.

```findings
findings:
  - id: new
    severity: Important
    family: docs-restate-chord-surface
    title: |
      README.md:131 still documents the deleted nav_boundary as Shift+Alt+arrows, and no row describes the new tab switching
    detail: |
      This diff removed nav_boundary (init.lua keymaps + both keyhelp catalog rows), but
      README.md:131 still reads "Shift+Alt+left / Shift+Alt+right | nvim (normal/insert) |
      Jump to the next region boundary...". The README preamble at 104-106 frames that table
      as the hand-maintained narrative restatement of the chord surface, so it is the one
      consumer of globalBindings that does not derive and it was not swept (ARCH-PURPOSE).
      Delete line 131 and add a row beside README.md:122 describing the from-any-pane,
      focus-preserving tab switch. The atlas half of the docs gate was done correctly.
  - id: new
    severity: Important
    family: chord-encoding-family-coverage
    title: |
      Only the modifier-4 spelling is registered; the meta-family \x1b[1;10D / \x1b[1;10C analog is missing
    detail: |
      shortcut.go:370-372 adds \x1b[1;4D / \x1b[1;4C only. ChordAltLeft deliberately carries
      three spellings including the meta form \x1b[1;9D (added in e6eee5a3); the shift analog
      of modifier 9 is 10, so \x1b[1;10D / \x1b[1;10C belong in the same table. On a terminal
      that reports meta-style modifiers the chord still works from the draft (nvim resolves
      <S-M-Left> itself) but is silently dead in the agent pane and the right pane — a partial
      failure of the "from any pane" Done-when that a single-terminal manual test cannot rule
      out. Two rows; TestNoChordSequenceIsAProperPrefixOfAnother already guards shadowing.
  - id: new
    severity: Minor
    family: bytes-vs-runes
    title: |
      DeliverChordArgs ranges a string, yielding runes rather than bytes
    detail: |
      shortcut.go:694 does `for _, b := range encodings[0]`, so b is a rune and
      strconv.Itoa(int(b)) emits a code point. Correct today only because every chord
      encoding is ASCII. `range []byte(encodings[0])` states the intent and stays correct
      if a non-ASCII byte ever enters the table.
  - id: new
    severity: Minor
    family: dead-code-after-removal
    title: |
      pos_rank is orphaned by the nav_boundary deletion
    detail: |
      nvim/init.lua:2446 defines pos_rank, whose only caller was the deleted nav_boundary.
      The sole remaining mention in the tree is the comment at init.lua:3402. No lua linter
      runs in make test, so nothing flags it. The plan's deletion step named
      nav_boundary/ordered_landmarks but not this one.
  - id: new
    severity: Minor
    family: comment-claims-unrendered-surface
    title: |
      catalog.go comment credits a "context column" that pair keys does not render
    detail: |
      keyhelp/catalog.go:83-87 justifies grouping the two globals under the terminal-tab
      heading by saying "the per-row context column is what distinguishes them". Running
      `pair keys` shows no context column — the distinction actually comes from the Help
      wording ("from any pane, without moving focus"), which does hold. The group heading
      "Terminal tabs (in the right terminal)" is likewise corrected only by the row text.
  - id: new
    severity: Minor
    family: action-to-chord-mapping-restated
    title: |
      The Action to delivered-chord mapping is restated at three call sites
    detail: |
      prev == ChordAltLeft / next == ChordAltRight appears in termcmd/run.go:197-200,
      wrapcmd/wrap.go:1674-1679 and layoutcmd.go:130-136 (plus implicitly in the
      handleTerminalChord case grouping). A two-line workbenchshortcut.TabChordFor(action)
      would make it one fact, matching the diff's own "one implementation of tab switching"
      framing (ARCH-DRY).
```

---

## Re-review — 2026-09-09T12:19:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 216 — drive right-pane tab switching from any pane, without moving focus |
| repo | pair |
| issue file | workshop/issues/000216-drive-right-pane-tab-switching-from-any-pane-without-moving-focus.md |
| boundary | whole-issue close |
| milestone | — |
| window | d15201957318de0b4a7172db0798cf176bc80d82..813a3e17c344caed13c06fd32d13538c8b870850 |
| command | sdlc close --issue 216 |
| reviewer | claude |
| timestamp | 2026-09-09T12:19:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is real and well-built: `Alt+Shift+←/→` decide identically from all three roles, deliver as `zellij action write --pane-id` bytes so `pair term` keeps exactly one tab-switch implementation, and the draft's Lua path is `detach = true` so focus genuinely never moves. The `chordMax` sentinel + `TestChordMaxFollowsEveryEncodedChord` is a proper class fix for PQ-3, and `TestAgentPaneDeliversTabChordsToTheRightTerminal` pins the wiring, not just the delivery. What holds it back from SHIP: I measured that **deleting all four meta-family encodings added for BR-2 leaves the entire suite green** (`\x1b[1;10D`, `\x1b[1;10C`, `\x1b[1;9A`, `\x1b[1;9B` appear nowhere but the source table), and that **deleting the whole `runDecision` tab case at `termcmd/run.go:197-202` also leaves it green**. Two orphans of the `nav_boundary` deletion also survive BR-4's fix, one of them in `atlas/`.

## 1. Strengths

- **`workbenchshortcut/shortcut.go:697-717` — `TabChordFor` genuinely collapsed BR-6.** All three executors (`termcmd/run.go:198`, `wrapcmd/wrap.go:1675`, `layoutcmd.go:151`) route through it, and it is reddened by both the layoutcmd argv test and the wrapcmd wiring test. This is the class fix, not the instance.
- **`shortcut.go:56 chordMax` + `shortcut_test.go:572` — the sentinel is asserted against the encoding table, not against the name of the current last chord.** That is precisely the mistake the old `<= ChordAltShiftEnter` bounds made, and the test would catch its recurrence.
- **`wrapcmd/wrap.go:1648-1652` — the `switchTerminalTab` var, with the comment explaining *why* the wiring (not only the delivery) needs a seam.** Correctly diagnoses the "green suite, dead chord" failure shape and defends against it.
- **`termcmd/run_test.go:176-180` — split-arrival rows at two different chunk boundaries** (`\x1b[1;4` + `D` and `\x1b[1;` + `4C`). This is the one real ordering seam in the diff and it is exercised at more than one interleaving (ARCH-ORDER).
- **`pair keys` renders correctly and BR-5's corrected comment is now true** — I confirmed `Context.String()` has zero callers and `Context` only propagates into `sections.go:68`. The Help wording, not a column, is what distinguishes the rows.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `nvim/init.lua:2441-2445` and `atlas/architecture.md:955` are still orphans of the `nav_boundary` deletion (family `dead-code-after-removal`, 2nd finding).**
This is the 2nd finding in family `dead-code-after-removal`. Round 1 fixed the instance BR-4 named (`pos_rank` — verified gone). **Do not fix these two sites one at a time.** The rule that covers all of them: *a deletion is complete only when the deleted thing's full reference set is swept — bodies, callers, comments that describe the behaviour, and `atlas/` — via an explicit enumeration run and recorded in the `## Log`, not by chasing the sites a reviewer happens to name.* Compiler and `make test` see none of these (no Lua linter, no atlas linter), which is exactly why the enumeration has to be written rather than felt.
Measured prevalence for this one deletion: the enumeration `grep -rn 'nav_boundary\|ordered_landmarks\|pos_rank\|Boundary-jump\|landmark' --exclude-dir=.git --exclude-dir=workshop` yields 2 stale sites beyond the removed bodies — `nvim/init.lua:2441-2445`, a 5-line comment still describing "Shift+Alt+←/→ steps between exactly three landmarks … the coarse jump to the far end / back to draft" (now a false description of a live chord, plus a stray double blank line at 2446-2447), and `atlas/architecture.md:955`, which lists `nav_boundary` (Shift+Alt jumps) among the helpers a reader should go look at. The remaining hits (`catalog.go:91`, `init.lua:3396`, `atlas:715`) are deliberate historical references and are fine.

**I-2 — `cmd/internal/termcmd/run.go:197-202`: the `runDecision` tab case has zero coverage, and I verified deleting it changes nothing.**
`handleTerminalChord` returns `true` for `ChordAltShiftLeft/Right` at `run.go:522-531`, so the stdin pump never reaches `runDecision` for these chords (`run.go:452`). Its only reachable caller is `pair term --test-shortcut`, which is what `tests/term-pane-shortcuts-test.sh` drives — the repo's stateful `zellij` fake, and the harness that covers every other terminal chord. That script was not extended, and no Go test references `ActionTerminalPrevTab`/`NextTab` outside `shortcut_test.go:550`. Failure scenario: I removed the entire case in a scratch checkout of `813a3e17` and `go test ./cmd/internal/termcmd/ ./cmd/internal/layoutcmd/ ./cmd/internal/workbenchshortcut/` reported only the pre-existing `ptychild: operation not permitted` sandbox class — nothing else went red. Either add the `Alt+Shift+Left` / `Alt+Shift+Right` rows to `term-pane-shortcuts-test.sh` (asserting `write --pane-id 4 27 91 49 59 51 68`), or delete the case and say in the Log that the pane path is the pump's. The `wrapcmd` comment at `wrap.go:1648` states this exact risk; it just wasn't applied to `termcmd`.

**I-3 — `cmd/internal/layoutcmd/layoutcmd.go:103-117` re-types `FocusRightTerminal`'s resolution preamble verbatim (ARCH-DRY).**
`SwitchRightTerminalTab` duplicates lines 38-53 step for step — `ListPanesJSON` → `LastTerminalPaneID` degrade-to-`""` → `TerminalPaneIDs` degrade-to-`nil` → `pickRightTerminal` — including a reworded copy of the graceful-degradation comment. The diff's own justification ("it reuses `pickRightTerminal` … so `Alt+k` and `Alt+Shift+arrow` cannot land on different halves") is only half-delivered: the picker is shared but its *inputs* are re-derived, so a future fourth signal (or a change to the degradation policy) gets added at one site and missed at the other. Fix: extract `resolveRightTerminal(rt Runtime) (zellijpane.Pane, bool, error)` and have both call it; `FocusRightTerminal` keeps its `move-focus right` fallback on `!ok`, `SwitchRightTerminalTab` keeps its inert `return nil`.

## 4. Minor findings

- `atlas/architecture.md:715` — "so `Alt+k` and `Alt+Shift+arrow` can never disagree about which split half they mean" is true only for the delivery path; from inside a split half the pump switches *that* half regardless of the recorded one. Scope the sentence to the draft/agent path. (2nd in family `docs-restate-chord-surface`; Minor, non-blocking — the rule is I-1's: hand-maintained restatements need the same sweep the derived surfaces get for free.)
- `nvim/init.lua:3402-3404` — the `local home = vim.env.PAIR_HOME or ''` / `local pair = … or 'pair'` pair is now copy-pasted a 5th time (also 755, 767, 892, 950). Pre-existing pattern; a `pair_bin()` helper would end it.
- `_G.PairTermPrevTab` / `PairTermNextTab` bodies at `init.lua:3407-3408` are untested — `workbench_route_test.lua` pins the key→name mapping, nothing pins the `jobstart` argv.
- Rendered `pair keys` now mixes `Shift+Alt+←` and `Alt+Shift+d` / `Alt+Shift+⏎` inside one section; README likewise carries both spellings for the physical `Alt+Shift+d` chord (lines 122 and 124). Pre-existing, but the new rows add to it.

## 5. Test coverage notes

- **Measured green baseline:** `workbenchshortcut`, `layoutcmd`, `keyhelp` fully green; `termcmd` and `wrapcmd` green for `-run 'Chord|PumpStdin|Documented|Catalog|Classified'`. The remaining `termcmd`/`wrapcmd` failures in this environment are all `ptychild: start …: operation not permitted`, the documented sandbox class, not this diff.
- **Two measured holes** (I-2 above and BR-2 below) — in both cases I deleted the code in a scratch checkout and the suite stayed green.
- Well covered: `DeliverChordArgs` as a complete argv vector for both chords plus both `!ok` paths; `Decide` across all three roles asserting no `DraftLuaFunction`; the inert no-terminal-pane case; `RunSwitchTerminalTab` argument rows including `too many`; the generated `workbench_actions.lua` mirror; the prefix-shadowing property over `chordSequences`.
- The Plan's last item (**Manual**) is deliberately unticked, so `sdlc close` will trip the plan-unchecked gate. That needs either the live gesture or an explicit `--no-plan-check` with the reason in `--verified`.

## 6. Architectural notes

- **ARCH-DRY** — flag, see I-3. Pass on `TabChordFor` and on the single mux tab-switch implementation.
- **ARCH-PURE** — pass. `DeliverChordArgs`/`TabChordFor` are pure and table-tested; `workbenchshortcut` keeps its no-IO invariant; all IO sits behind the injected `layoutcmd.Runtime`.
- **ARCH-PURPOSE** — flag on the class axis. All three panes deliver and focus is preserved, so the issue's purpose is met. But BR-2's class ("both modifier families for every arrow chord") was fixed by hand at four sites with no enumeration, and I-1 is the nav_boundary deletion's class left half-swept. Both are "fixed the instance, not the class".
- **ARCH-MOCK** — pass with a note. `zellij` is faked behind `layoutcmd.Runtime` in Go and behind a fake `zellij` binary in `tests/term-pane-shortcuts-test.sh`, and production flow shares that seam. But `zellij action write --pane-id` is a *new* dependency surface whose only verification is reading `--help`; the shell fake — the closest thing here to a conformance harness — never exercises it (I-2).
- **ARCH-CONSTRAINTS** — pass. Two synchronous subprocesses per press on the agent pane's stdin pump exactly matches the existing `ActionFocusRightTerminal` envelope (`wrap.go:1687`), and the issue explicitly accepts a subprocess per deliberate gesture. The draft path is `detach = true`.
- **ARCH-SECURE** — pass. `paneID` originates in zellij's own JSON as an integer field and reaches `exec.Command` as an argv element, never a shell string; the Lua side uses a `jobstart` argv list. No credentials. `ListPanesJSON` errors propagate; sidecar reads degrade deliberately and visibly (the chord goes inert rather than fabricating a pane id).
- **ARCH-ORDER** — pass. No new state carried between events. Fire-and-forget and double-press semantics are written down in both the Plan and `layoutcmd.go:96-101`. The chunk-boundary seam is the one real interleaving and it is tested at two split points.

## 7. Plan revision recommendations

None on the code side — the three deviations are already recorded in `## Revisions` and each is accurate against the diff (I verified the CLI verb, the un-exported picker reuse, and the absence of a `config.kdl` change). Add a `## Revisions` entry only if I-2 is resolved by *deleting* the `runDecision` case, since the Plan's "Three executors" step names the agent pane, the right pane, and the draft — it does not commit to a fourth out-of-pane path in `termcmd`, so a deletion is a clarification worth recording rather than a scope change.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:131 nav_boundary row deleted; new "any pane / without moving focus" row at README.md:123.
  - id: BR-2
    disposition: not-addressed
    note: |
      Rows added but nothing pins them; deleting all four leaves the suite green (measured).
  - id: BR-3
    disposition: withdrawn
    note: |
      ChordEncodings returns [][]byte (shortcut.go:393), so the loop variable is already a byte.
  - id: BR-4
    disposition: addressed
    note: |
      pos_rank is gone tree-wide; sibling orphans of the same deletion raised separately.
  - id: BR-5
    disposition: addressed
    note: |
      Verified Context.String() has zero callers; the corrected comment is accurate.
  - id: BR-6
    disposition: addressed
    note: |
      TabChordFor owns the mapping; all three executors route through it and two tests redden on it.
findings:
  - id: new
    severity: Important
    family: chord-encoding-family-coverage
    title: |
      The BR-2 meta-family rows are unpinned — deleting all four leaves the full suite green
    detail: |
      This is the 2nd finding in family `chord-encoding-family-coverage`. Round 1
      fixed instances (\x1b[1;10D, \x1b[1;10C, \x1b[1;9A, \x1b[1;9B added at
      shortcut.go:367-368,378-379). Do NOT re-add or re-check those rows. The rule
      that covers all of them: for every arrow chord encoded as \x1b[1;<m><letter>,
      the meta sibling \x1b[1;<m+6><letter> must also be registered (alt 3 -> meta 9,
      shift+alt 4 -> shift+meta 10). That rule is mechanically checkable over
      chordSequences and is what should be written, not four more table rows.
      Measured: the four new encodings appear nowhere in the tree but the source
      table; removing all four from a scratch checkout of 813a3e17 left
      `go test ./cmd/internal/workbenchshortcut/ ./cmd/internal/layoutcmd/` green and
      `-run 'Chord|PumpStdin|Documented'` on termcmd green. TestDecodeAltArrowChords
      (shortcut_test.go:320-334) still lists only the four original rows, and
      TestDecodeGlobalChord:258-259 only the bit-2 Up/Down forms.
  - id: new
    severity: Important
    family: dead-code-after-removal
    title: |
      Two nav_boundary orphans survive BR-4 — a stale init.lua comment and an atlas helper list
    detail: |
      This is the 2nd finding in family `dead-code-after-removal`. Round 1 fixed the
      instance BR-4 named (pos_rank — verified gone tree-wide). Do NOT fix these two
      sites individually. The rule: a deletion is complete only when the deleted
      thing's full reference set is swept — bodies, callers, comments that describe
      the behaviour, and atlas/ — via an explicit enumeration that is run and recorded
      in the Log, not by chasing the sites a reviewer names. No compiler and no
      `make test` step sees any of these.
      Measured prevalence for this deletion: nvim/init.lua:2441-2445 is a comment block
      still describing "Shift+Alt+arrows steps between exactly three landmarks … the
      coarse jump to the far end / back to draft", now a false description of a live
      chord (plus a stray double blank line at 2446-2447); atlas/architecture.md:955
      still lists "`nav_boundary` (Shift+Alt jumps)" among the nvim helpers a reader
      should go read. The other three hits (catalog.go:91, init.lua:3396, atlas:715)
      are deliberate historical references and are correct.
  - id: new
    severity: Important
    family: untested-executor-branch
    title: |
      termcmd/run.go:197-202 has no test — deleting the whole case leaves the suite green
    detail: |
      handleTerminalChord returns true for ChordAltShiftLeft/Right (run.go:522-531), so
      the stdin pump never reaches runDecision for these chords (run.go:452). The case's
      only reachable caller is `pair term --test-shortcut`, which is what
      tests/term-pane-shortcuts-test.sh drives — the repo's fake-zellij harness that
      covers every other terminal chord, and which was not extended. No Go test
      references ActionTerminalPrevTab/NextTab outside shortcut_test.go:550.
      Measured: removing the entire case from a scratch checkout of 813a3e17 left
      `go test ./cmd/internal/termcmd/ ./cmd/internal/layoutcmd/
      ./cmd/internal/workbenchshortcut/` reporting only the pre-existing
      `ptychild: operation not permitted` sandbox failures. Fix: add the two rows to
      term-pane-shortcuts-test.sh asserting `write --pane-id 4 27 91 49 59 51 68` /
      `… 67`, or delete the case because the pump owns this pane. wrap.go:1648 states
      this exact risk for the agent pane; it just wasn't applied here.
  - id: new
    severity: Important
    family: resolution-sequence-restated
    title: |
      SwitchRightTerminalTab re-types FocusRightTerminal's four-step pane-resolution preamble
    detail: |
      layoutcmd.go:103-117 duplicates layoutcmd.go:38-53 step for step — ListPanesJSON,
      LastTerminalPaneID degrading to "", TerminalPaneIDs degrading to nil,
      pickRightTerminal — including a reworded copy of the graceful-degradation
      comment. The diff's stated ARCH-DRY guarantee ("reuses pickRightTerminal … so
      Alt+k and Alt+Shift+arrow cannot land on different halves") is only half
      delivered: the picker is shared, its inputs are re-derived. Adding a fourth
      preference signal, or changing the degradation policy, will land at one site and
      miss the other. Extract resolveRightTerminal(rt) (zellijpane.Pane, bool, error);
      FocusRightTerminal keeps its `move-focus right` fallback on !ok, and
      SwitchRightTerminalTab keeps its inert `return nil`.
  - id: new
    severity: Minor
    family: docs-restate-chord-surface
    title: |
      atlas:715 "can never disagree about which split half" holds only for the delivery path
    detail: |
      This is the 2nd finding in family `docs-restate-chord-surface` and is Minor, so it
      does not block. Same rule as the I-1 sweep: hand-maintained restatements need the
      enumeration that derived surfaces get for free. From inside a split half the pump
      switches THAT half (run.go:522), regardless of the recorded last-terminal id, so
      Alt+k and Alt+Shift+arrow can point at different halves. Scope the sentence to the
      draft/agent delivery path.
  - id: new
    severity: Minor
    family: dead-code-after-removal
    title: |
      init.lua:3402-3404 copies the PAIR_HOME/bin/pair resolution idiom a fifth time
    detail: |
      Also at 755, 767, 892, 950. Pre-existing pattern, not introduced by this issue; a
      pair_bin() helper would end it.
  - id: new
    severity: Minor
    family: untested-executor-branch
    title: |
      _G.PairTermPrevTab / PairTermNextTab bodies are untested
    detail: |
      workbench_route_test.lua pins the key -> function-name mapping; nothing pins the
      jobstart argv at init.lua:3407-3408.
```

---

## Re-review — 2026-09-09T12:43:20-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 216 — drive right-pane tab switching from any pane, without moving focus |
| repo | pair |
| issue file | workshop/issues/000216-drive-right-pane-tab-switching-from-any-pane-without-moving-focus.md |
| boundary | whole-issue close |
| milestone | — |
| window | d15201957318de0b4a7172db0798cf176bc80d82..af1b51b72cdd2a4edd2f8524a9a708dd647a1a23 |
| command | sdlc close --issue 216 |
| reviewer | claude |
| timestamp | 2026-09-09T12:43:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's fixes are real this time: I verified each by mutation rather than by reading the Log. The BR-7 rule guard reddens (`"\x1b[1;4D" (21) has no meta sibling "\x1b[1;10D"`) when the four meta rows are deleted; the BR-9 harness rows redden when `runDecision`'s tab case is removed; the `nav_boundary` reference set re-greps clean tree-wide with only the deliberate historical mentions surviving; `resolveRightTerminal` is a real extraction both callers use; `pair keys` renders both new rows from `GlobalBinding.Help`. I also ran the draft's actual production path end-to-end against a fake zellij (`pair layout switch-terminal-tab prev` → `write --pane-id 4 27 91 49 59 51 68`) and it works. What keeps this off SHIP is one measured gap in the same family that has now produced three findings: the **draft** pane — the pane the issue exists for — is the only one of the three whose delivery chain has no test crossing its seams. Renaming the `Dispatch` case string leaves `go test ./...` with a byte-identical failure set to the control, and the chord goes silently inert because nvim's detached `jobstart` discards the loud stderr the dispatcher does produce.

## 1. Strengths

- **`TestEveryArrowChordRegistersBothModifierFamilies` (`cmd/internal/workbenchshortcut/shortcut_test.go:597`) is a genuine rule guard, not four more rows.** Measured: deleting `\x1b[1;10D` / `\x1b[1;10C` from a scratch checkout reddens it with a message naming the missing sibling. This is exactly what BR-7 asked for and it will catch the next arrow chord.
- **`resolveRightTerminal` (`cmd/internal/layoutcmd/layoutcmd.go:60`) is the right extraction.** Both callers now feed the picker the same signals, so the graceful-degradation policy and the recorded-half preference have one home. `FocusRightTerminal` keeps its `move-focus right` fallback and `SwitchRightTerminalTab` its inert `return nil` — the two callers differ only where they should.
- **`HandledInPane` (`shortcut.go:136`, consumed at `shortcut.go:318`) expresses "self-handled global" without inventing a new decision type**, and `TestTabChordsAreHandledInPaneFromEveryRole` pins the load-bearing part: `DraftLuaFunction` stays empty from all three roles, which is what stops tab switching depending on nvim being alive.
- **Split-arrival coverage (`termcmd/run_test.go:175-180`) pins the `held` buffer** for the new sequences in both `\x1b[1;4` + `D` and `\x1b[1;` + `4C` shapes — the failure mode where a chord straddling a read reads as a dead key.
- **The `nav_boundary` sweep's CHANGELOG carve-out** (`workshop/lessons.md`) is the right refinement of BR-8's rule: current-state docs get swept, dated records don't, because rewriting them falsifies history.

## 2. Critical findings

None.

## 3. Important findings

**The draft pane's delivery chain has no test crossing either of its two seams** — `tests/term-pane-shortcuts-test.sh:99`, `cmd/internal/dispatcher/dispatcher.go:203`.

The chain is: `<S-M-Left>` → `_G.PairTermPrevTab` (`nvim/init.lua:3405`) → `switch_terminal_tab_command` → `jobstart` → `pair layout switch-terminal-tab prev` → `Dispatch` case → `RunSwitchTerminalTab`. Each *half* is tested; neither *join* is. Measured: I renamed the `Dispatch` case to `"layout switch-terminal-tab-TYPO"` in a scratch checkout of `af1b51b7` and `go test ./...` produced a failure set identical to the control (36 pre-existing pty/exec `operation not permitted` failures, zero new). At runtime the dispatcher does fail closed — `pair-go: layout switch-terminal-tab has no buffered route wired`, exit 2 — but `jobstart(..., {detach = true})` has no `on_stderr`, so the operator sees a dead key.

The rows that *look* like they cover this don't. `term-pane-shortcuts-test.sh:99-115` is commented "Driven from the DRAFT's focus, since the whole point is that they work where the operator is typing" — but `handleChord` short-circuits at `DecideGlobal` before reading any focus, so `write_panes draft` is inert there. Measured: rewriting all three rows to `write_panes terminal` leaves them PASSing identically. What those rows actually exercise is `termcmd`'s `runDecision` case, which the pump short-circuits in production (`run.go:522` returns true first) and which no production path reaches at all — `--test-shortcut` is its only caller tree-wide.

Fix sketch: turn `Dispatch`'s buffered switch into a `map[string]handler` so `TestEveryImplementedFamilyIsRoutable` can iterate `Families()` and assert membership without executing anything, and add a Go test asserting the subcommand string `workbench_route.switch_terminal_tab_command` emits is a name `Families()` declares (the repo already reads `nvim/*.lua` from Go in `keyhelp`, so the cross-language read is idiomatic here).

## 4. Minor findings

- **`shortcut_test.go:613`** — the family rule is one-directional. `if modifier >= 9 { continue }` means a chord registered *only* in the meta spelling passes the guard with its bit-2 sibling missing; the comment asserts "its bit-2 sibling is the other row" rather than checking it. Separately, the rule's premise is unmeasured: `zellij/config.kdl:37` sets `support_kitty_keyboard_protocol true`, and under KKP bit 8 is **super**, not xterm's meta — so `\x1b[1;9A/B` and `\x1b[1;10D/C` also bind Cmd+↑/↓ to the layout ladder and Cmd+Shift+←/→ to tab switching in every byte-decoding pane. (`\x1b[1;9D/C` has been in the table since `e6eee5a3` with no reported misfire, so practical risk is low — but the diff promoted an unmeasured spelling to a tree-wide enforced rule.)
- **`tests/term-pane-shortcuts-test.sh:115`** — the "moves no focus" row is fully subsumed by the two rows above it, which assert `"$(actions)"` equals the single `write …` line exactly. It passes vacuously: under the delete-the-`runDecision`-case mutation, with `actions` empty, it still PASSes.
- **`nvim/scrollback.lua:273`** — BR-12's `pair_bin()` swept `init.lua`'s four sites but not the idiom's remaining member, which additionally lacks the empty-`PAIR_HOME` fallback (`vim.env.PAIR_HOME .. '/bin/pair'` errors on nil concat).
- **ARCH-CONSTRAINTS, `nvim/init.lua:3399`** — from the draft each press is 1 detached `jobstart` + 1 `pair` process + 2 `zellij` subprocesses (`list-panes`, then `write`), with no debounce. The Estimate's frequency note reasons about a *deliberate single press*; a held arrow key at auto-repeat is the case it doesn't cover, and `detach = true` means the extent is unbounded.

## 5. Test coverage notes

Coverage now stands: right pane ✅ (`run_test.go:173-180`, in-place `mux` calls, no zellij action), agent pane ✅ (`keymap_registry_test.go:36`, the `switchTerminalTab` var seam pins the *wiring*, which is the part that would otherwise be silently missing), draft pane ⚠️ (both halves unit-tested, neither seam crossed — §3). `TestChordMaxFollowsEveryEncodedChord` is the right shape for the PQ-3 class: it asserts the sentinel against the encoding table rather than against the current last chord's name. All 36 failures in `go test ./...` at head are pty/exec `operation not permitted` in this environment; none touch this diff's packages' logic.

## 6. Architectural notes

ARCH-DRY **pass** (`TabChordFor` + `resolveRightTerminal` are the two consolidations that mattered; one residual noted in §4). ARCH-PURE **pass** — `DeliverChordArgs`, `TabChordFor` and `switch_terminal_tab_command` are pure and table-tested; IO stays at `RunZellijAction`/`jobstart`. ARCH-PURPOSE **pass with one flag** — the shadow-sweep of the chord surface is complete (keyhelp derives from `GlobalBinding.Help`, `workbench_actions.lua` is generated, README/atlas swept, CHANGELOG deliberately preserved); the flag is that the pane named in the issue's Problem statement is the one whose chain isn't joined by a test. ARCH-MOCK **pass** — the zellij seam is the `Runtime` interface plus the fake-zellij shell harness, and `write --pane-id` was already a depended-on surface (`draftroute/route.go:103`), so no new unfaked dependency enters. ARCH-CONSTRAINTS **flag** (§4). ARCH-SECURE **pass** — the pane id comes from zellij's own JSON and is passed as a distinct argv element with no shell; malformed JSON degrades through `zellijpane.Parse` → picker `!ok` → inert chord, which is visible-as-nothing rather than fabricated. ARCH-ORDER **pass** — the `held` buffer has split-arrival rows, the resolve→write race is stated and lands inert, and the double-press interleaving is idempotent per press.

For upcoming work: `switchTerminalTab` as a package var (`wrap.go:1649`) is the first injection seam of its kind in `wrapcmd`; if a second appears, group them rather than accumulating package-level vars.

## 7. Plan revision recommendations

None — the issue's `## Plan` and `## Revisions` match the code, including the three recorded deviations. One process note for the closing agent, not a plan defect: the final `- [ ] Manual` item is still unticked by design (operator-pending live gesture), so the close will hit the plan-unchecked gate.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Both meta rows registered; superseded by BR-7, which is also now addressed.
  - id: BR-7
    disposition: addressed
    note: |
      Verified by mutation — deleting the meta rows reddens TestEveryArrowChordRegistersBothModifierFamilies.
  - id: BR-8
    disposition: addressed
    note: |
      Re-ran the enumeration at head; only deliberate historical references remain.
  - id: BR-9
    disposition: addressed
    note: |
      Verified by mutation — removing runDecision's case fails two harness rows; see the new finding on what those rows actually cover.
  - id: BR-10
    disposition: addressed
    note: |
      resolveRightTerminal extracted at layoutcmd.go:60; both callers use it.
  - id: BR-11
    disposition: addressed
    note: |
      atlas:715 now scopes the guarantee to the delivery path and states the in-split-half exception.
  - id: BR-12
    disposition: addressed
    note: |
      pair_bin() ends the idiom in init.lua; one member outside that file survives, raised below.
  - id: BR-13
    disposition: addressed
    note: |
      switch_terminal_tab_command is pure and pinned; its caller remains unpinned, folded into the new Important finding.
findings:
  - id: new
    severity: Important
    family: untested-executor-branch
    title: |
      The draft pane's chord chain crosses two seams and no test crosses either; renaming the Dispatch case leaves go test ./... identical to control
    detail: |
      This is the 3rd finding in family `untested-executor-branch`. Earlier rounds fixed
      instances (the agent pane's executeWorkbenchDecision case, then termcmd's runDecision
      case). Do NOT fix this instance by adding one more draft-specific test. The rule that
      covers all three: a chord is delivered by a CHAIN, and every process- or
      language-boundary in that chain needs a test that crosses it, not two tests that stop
      on either side. Mechanically checkable form: Dispatch's buffered switch becomes a
      map[string]handler so a table-driven test over Families() can assert every
      Status:"implemented" entry is routable, and a Go test asserts the subcommand string
      nvim/workbench_route.lua emits is a name Families() declares.
      Measured: renaming `case "layout switch-terminal-tab"` (dispatcher.go:203) to a typo in
      a scratch checkout of af1b51b7 left `go test ./...` with a failure set byte-identical to
      the unmutated control (36 pre-existing pty "operation not permitted" failures, zero new).
      Runtime behaviour of the mutant: exit 2, `pair-go: layout switch-terminal-tab has no
      buffered route wired` on stderr -- loud at a shell, invisible through
      jobstart(detach=true), which has no on_stderr. Also measured: the rows at
      tests/term-pane-shortcuts-test.sh:99-115 do not cover this path despite their comment
      ("Driven from the DRAFT's focus"). handleChord short-circuits at DecideGlobal before
      reading pane focus, so rewriting all three to `write_panes terminal` leaves them
      PASSing identically; what they exercise is termcmd's runDecision case, whose only
      caller tree-wide is the `--test-shortcut` seam.
  - id: new
    severity: Minor
    family: chord-encoding-family-coverage
    title: |
      The meta-family guard is one-directional, and its bit-8=Meta premise is Super under the kitty protocol zellij enables
    detail: |
      This is the 3rd finding in family `chord-encoding-family-coverage`. Earlier rounds fixed
      instances (four table rows), then replaced them with the rule. Do NOT patch the two
      spellings this names. The rule needs two corrections: (a) shortcut_test.go:613 skips
      `modifier >= 9`, so a chord registered ONLY in the meta spelling passes with its bit-2
      sibling missing -- the sibling relation should be enforced in both directions; (b) the
      premise "bit 8 (meta) gives 9" is xterm's convention, but zellij/config.kdl:37 sets
      support_kitty_keyboard_protocol true, and under KKP bit 8 is SUPER. So the enforced
      rule now also binds Cmd+arrow to the layout ladder and Cmd+Shift+arrow to tab
      switching in every byte-decoding pane. \x1b[1;9D/C has been in the table since
      e6eee5a3 with no reported misfire, so the practical risk is low -- but an encoding
      should be measured on the terminal in play before a test enforces it tree-wide, which
      is the same "derived is not measured" caveat the issue Log applied to modifier 4 and
      not to 9/10.
  - id: new
    severity: Minor
    family: untested-executor-branch
    title: |
      tests/term-pane-shortcuts-test.sh:115 asserts nothing the two rows above it do not already prove, and passes vacuously
    detail: |
      The two rows above assert `"$(actions)"` equals exactly the single `write --pane-id 4 ...`
      line, which already proves no focus action was emitted. Measured: under the
      delete-the-runDecision-case mutation, `actions` is empty and the grep -c row still
      PASSes. Either drop it or make it assert something the exact-match rows cannot.
  - id: new
    severity: Minor
    family: dead-code-after-removal
    title: |
      The pair_bin() consolidation swept init.lua but not the idiom's member in nvim/scrollback.lua:273
    detail: |
      This is the 4th finding in family `dead-code-after-removal`. The family rule already
      states it: a consolidation is complete only when the full instance set is enumerated
      and the enumeration recorded, not when the sites a reviewer named are fixed. BR-12
      enumerated five sites, all inside init.lua; the grep that would have found the sixth
      is `grep -rn "bin/pair'" nvim/`. scrollback.lua:273 additionally lacks the
      empty-PAIR_HOME fallback the helper has, so it errors on nil concat rather than
      falling back to `pair` on PATH.
  - id: new
    severity: Minor
    family: unbounded-keystroke-fanout
    title: |
      Held Alt+Shift+arrow in the draft spawns 3 detached processes per auto-repeat with no debounce
    detail: |
      nvim/init.lua:3399 fires jobstart(detach = true) per press; each spawned `pair layout
      switch-terminal-tab` runs `zellij action list-panes` then `zellij action write`. The
      Estimate's frequency note reasons about a deliberate single press, which is the right
      call for one press; key auto-repeat is the case it does not cover, and `detach = true`
      means the extent is not lexically bounded (ARCH-CONSTRAINTS / ARCH-ORDER extent).
      Note for future, not a gate issue -- Alt+k from the draft has the same shape today.
```
