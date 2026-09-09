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
