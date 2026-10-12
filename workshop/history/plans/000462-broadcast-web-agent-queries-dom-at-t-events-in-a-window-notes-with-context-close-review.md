# Boundary Review — pair#462 (whole-issue close)

| field | value |
|-------|-------|
| issue | 462 — broadcast-web: agent queries — DOM at t, events in a window, notes with context |
| repo | pair |
| issue file | workshop/issues/000462-broadcast-web-agent-queries-dom-at-t-events-in-a-window-notes-with-context.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a5e1c8626b7f2557391fa933007c9da6e95e9c5..d310e59c250e4317a83e36d6d5848a93ffeb7092 |
| command | sdlc close --issue 462 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:38:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff read; now running the package tests and spot-checking edge cases.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff delivers what #462 asked for. `Querier` is a real `query.Surface` built on the reference reducer, with one pass for `Notes`. The text renderers are pure and kept within line budgets. The CLI is a closed form wired through `couchcmd` without starting the Couch runtime. All five fixtures (`recording.Fixtures`) have golden files for `dom`, `events` and `notes`, and the scripted agent check gets its expectations from the fixtures' records, not from the query code. README, atlas, the skill text and usage are all updated. `go test` on `./cmd/internal/webcast/...` and `couchcmd` passes.

One thing should be fixed first. `Select` reads the notes model's anchor grammar (`>>>`, and `>` across tree boundaries) more loosely than the model means it. I checked this: 6 anchors are ambiguous across the stress and blog fixtures. One blog anchor matches 6 elements. The atlas still says "never matched loosely" and that "a note's selector pastes into `--sel`".

`artifactpath` fails in this checkout, but the failures are about `runtimebundle` assets that aren't generated here (`pattern assets/runtime/files: no matching files found`, plus stale nvim-review entries). They don't touch this diff, and the #462 manifest lines are fine.

### 1. Strengths
- **One pass for notes:** `querier.go` `Notes` keeps each note's `*recording.NoteView` pointer, so a later `note-state` gives the final state. `TestNotesCarryTheirFinalState` pins this, and it is O(records) as the plan says.
- **Targets named as they were:** `narrator` describes each record against the state *before* applying it (`events.go`), so a removed node still has its name. `TestEventLinesNameRemovedNodes` covers it.
- **An independent oracle for the agent check:** `agent_test.go` builds its expectations from `Reduce` and the raw records, not from the renderers. A recorded subagent run is also in the Log.
- **The couch tail goes to the query CLI untouched:** `TestParseCLIBroadcastWebQuery` shows that words that look like layout flags after the flag are passed through.
- **Unsupported selector syntax is refused with the grammar** instead of being half-matched (`, `, `:hover`, `~`). I checked this.

### 2. Critical findings
None.

### 3. Important findings
- **`cmd/internal/webcast/query/selector.go:146-156` and `:72-85`: the notes model's anchors aren't matched with the model's meaning (ARCH-DRY, correctness).**
  - `>>>` is read as a plain descendant step. Because `ParentElement` goes up through shadow hosts and frame owners, `>` also matches a host to its shadow-root children.
  - `notes/selector.go:12-23` says anchors are "made and matched in this file only, so the two can't drift". There, `>>>` means a top-level child of the host's inner tree and `>` stays within one tree. The query package now has a second, looser implementation.
  - Probe results:
    - `html:nth-of-type(1) > body:nth-of-type(1) > astro-dev-toolbar:nth-of-type(1) >>> style:nth-of-type(1)` matches 6 elements on blog.
    - `html:nth-of-type(1)` matches 2 on stress (the iframe's html too).
  - **Fix:** route anchor-grammar selectors through the notes matcher (export `notes.MatchSelector` or similar). Or make `>>>` a real combinator: the parent is the host's Shadow/Content/Frame root. Make `>` require `n.Parent.Y == YEl`.
- **`selector_test.go:125`: the test can't see this bug.** `TestModelSelectorsSelectTheirElement` checks `slices.Contains(got, n)`. It should assert `len(got) == 1` wherever the model's own matcher finds exactly one element.

### 4. Minor findings
- **ARCH-DRY:** `Querier.Events` (`querier.go`) is the same loop as `FakeSurface.Events`. `answer`'s events text path checks `a.to < a.from` again instead of reusing `q.Events`' error. The "the recording doesn't read: record %d (…)" message is written twice (`querier.go`, `events.go`).
- **Real and fake disagree on note state.** `FakeSurface.Notes` reports a note's state at the note's time; `Querier` reports the final state. `RunSurfaceContract` doesn't pin either, so the plan's "they can't drift on meaning" isn't fully true. Add a resolved-later note to the contract.
- **Matches can be dropped without a trailing line.** In `DOMAt`, if one match uses up the budget exactly, the remaining matches are skipped with no "… more" line. Only the header's "N match(es)" hints at it.
- **`events --json` ignores `--max-lines`,** even though usage offers the flag for `events`.
- **Raw page text reaches stdout.** Page text lines (`render.go` `node` for YText, `onlyText`) print recorded text without quoting. A recorded page could put terminal control sequences in front of an agent or terminal (ARCH-SECURE, low impact).
- **`>>>` treats a deep step after a child step as fine.** `a > >>> b` errors, but `a >>> > b`-style combinations aren't tested.
- **Folding differs between `events` and `notes`.** In `events`, a driver's typing interleaves `input … by X` with framework `attr value` records that have no `by`, so the keys differ and the typing doesn't fold. `notes` folds it, because only user actions go in. Not seen in the fixtures.
- **Presentation fields in the contract struct.** `NoteView` gained `Header` and `BeforeLines`, which are text. A cleaner split would leave them to `NoteText` (ARCH-PURE taste).
- **#457's `UserAction` changed locally.** It now counts `seg`/`stop`. Only the query package uses it, so this is acceptable, but the issue asks that seam changes go to the TL. The Log says they did not.

### 5. Test coverage notes
- Covered: golden files for all three queries on every fixture, the contract on both surfaces, the CLI's usage and failure exit codes, the selector table, and render budgets.
- Gaps:
  - Selector ambiguity (above).
  - `--json` output is only checked for decoding; no golden file.
  - No test for `Nav`/`Selection` as the last action before a note: the agent check's `switch` skips them, so a regression there would compare against an earlier action.

### 6. Architectural notes
- **ARCH-DRY:** flag (selector matching implemented twice, above).
- **ARCH-PURE:** pass. The renderers and narrator are pure; `Main` is a thin IO shell around `ReadDir`.
- **ARCH-PURPOSE:** pass. All three queries work, the real #461 model is wired in, and the agent check is scripted and recorded.
- **ARCH-MOCK:** pass. There's no external dependency; tests run on fixtures.
- **ARCH-CONSTRAINTS:** pass with a note. Budgets are enforced, except `events --json`. Each `dom`/`events` call re-reduces from the start, which is fine for a one-shot CLI. The plan's seek index is the right follow-up for long recordings.
- **ARCH-SECURE:** pass with a Minor. Recording input is validated by `ReadDir`/the reducer and errors exit 1 naming the record; the only gap is unquoted page text.
- **ARCH-ORDER:** pass. No state is kept between invocations, and the narrator is a fold over ordered records. Final note state comes through pointer aliasing, and a test pins it.
- **ARCH-FUNERAL:** pass. Nothing durable is created; output goes to stdout only.

### 7. Plan revision recommendations
- Add a Revisions entry: "`Select` matches the notes model's anchors (`>>>`, `:nth-of-type`) with the model's own tree-scoped meaning (delegating to `notes`), not as descendant steps", once the fix lands.
- Correct the Querier DRY rationale ("can't drift on meaning"), or extend `RunSurfaceContract` to pin a note's final state.

```findings
findings:
  - id: new
    severity: Important
    family: duplicated-selector-semantics
    title: |
      query.Select matches notes-model anchors (>>>, >) more loosely than notes defines them; anchors are ambiguous
    detail: |
      selector.go reads >>> as a plain descendant, and > crosses shadow and frame boundaries via ParentElement. notes/selector.go's matcher is tree-scoped. Probe: a blog astro-dev-toolbar >>> style:nth-of-type(1) anchor matches 6 elements; html:nth-of-type(1) matches 2 on stress. Delegate to an exported notes matcher, or make >>> and > tree-aware.
  - id: new
    severity: Important
    family: test-asserts-containment-not-identity
    title: |
      TestModelSelectorsSelectTheirElement checks slices.Contains, so ambiguous anchors pass
    detail: |
      Assert exactly one match wherever the model's own matcher resolves the anchor uniquely.
  - id: new
    severity: Minor
    family: fake-real-contract-gap
    title: |
      FakeSurface reports note state at note time, Querier the final state; RunSurfaceContract pins neither
  - id: new
    severity: Minor
    family: duplicated-logic
    title: |
      Querier.Events duplicates FakeSurface.Events; the window check and the reducer-error format are repeated in answer() and events.go
  - id: new
    severity: Minor
    family: output-budget-gaps
    title: |
      events --json ignores --max-lines; DOMAt drops further matches with no trailing line when one page fills the budget exactly
  - id: new
    severity: Minor
    family: untrusted-text-to-terminal
    title: |
      Recorded page text is printed unquoted (render.go text lines), so control sequences pass through to stdout
  - id: new
    severity: Minor
    family: seam-change-unreported
    title: |
      UserAction (#457 seam) widened to seg/stop, and NoteView gained presentation fields, without the TL report the issue asks for
```

---

## Re-review — 2026-10-12T02:46:49-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 462 — broadcast-web: agent queries — DOM at t, events in a window, notes with context |
| repo | pair |
| issue file | workshop/issues/000462-broadcast-web-agent-queries-dom-at-t-events-in-a-window-notes-with-context.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a5e1c8626b7f2557391fa933007c9da6e95e9c5..8b97c68adfa3696782eb9366ea3c70072263aeaf |
| command | sdlc close --issue 462 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:46:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now look at the round-1 fix commit.

Mutation checks on the three round-1 fixes all go red, and I found a new panic in the fake on an empty recording. Next I'm checking that the docs still match the new selector rules.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All seven round-1 findings are fixed, and three of the fixes are proven by tests. I put each of the three main changes back the way it was and the tests failed: the `notes.Match` hand-off failed `TestModelSelectorsSelectTheirElement`, the shadow/frame scoping failed `TestSelect` and `TestModelSelectorsSelectTheirElement`, and the control-character scrubber failed `TestMainScrubsControlCharacters`. I also ran a probe across all five fixtures and five sample times. It took every note selector generated at one time and selected it at another, and never found a case where `notes.Match` matched nothing and the CSS fallback then matched something else. The webcast, notes, query and couchcmd test packages pass. `go vet` is clean, and gofmt is clean on every file in the diff. `artifactpath` fails, but only because the checkout lacks the gitignored `cmd/internal/runtimebundle/assets/` build output; this diff doesn't touch it. Two things to fix before ship. First, the round-1 change made `atlas/webcast-format.md` wrong: it still says selectors search through shadow roots. Second, the BR-3 fix brought a new panic into `FakeSurface.Notes` when a recording is empty.

1. **Strengths**
   - `query.Select` now hands note selectors to `notes.Match`, the notes package's own reader (`cmd/internal/webcast/query/selector.go:42`), so their grammar is defined in one place. CSS combinators now stay inside one tree (`matches`, `treeParent`, `treeOf`).
   - `TestModelSelectorsSelectTheirElement` now checks against `Structural.Resolve`. Where `Resolve` finds exactly one element, the test requires exactly that element, so an ambiguous match can't pass. It also checks the fake model's paths and the round-1 probe cases.
   - Control characters are scrubbed in one place, the output writer (`cli.go` `scrubber`), so every text path is covered. It also catches C1 controls and DEL.
   - The contract test now requires a note's final state (`contract.go:72`), and the fake and the real surface agree on it.
   - `--json` combined with `--max-lines` is now refused, and the text output adds a "… N more matches" line when matches outrun the budget.

2. **Critical:** none.

3. **Important**
   - **The atlas still describes the old selector rules** (`atlas/webcast-format.md:384-388`). It says selectors are "searched through shadow roots, template contents and iframe documents." After round 1, a space or `>` stays inside one tree, matching starts in the top document, and only `>>>` crosses into another tree. So `.badge` no longer finds content inside a shadow root (see the `TestSelect` case `{".badge", ""}`). An agent reading the atlas will expect matches it won't get.
     - Fix: say what `SelectorGrammar` says, and point to that constant rather than restating it by hand.

4. **Minor**
   - **`FakeSurface.Notes` panics on an empty recording** (`fake.go:54`, `f.Records[len(f.Records)-1]`). I reproduced it: the fake panics with index out of range [-1], while `Querier.Notes` returns nil.
     - This is the 2nd finding in family `fake-real-contract-gap`. The rule that covers both: any input the real surface accepts, the fake must answer the same way, and `RunSurfaceContract` is the list of those inputs. Add the empty recording (and any other edge case the real surface handles) to the contract, not just a guard in `fake.go`.
   - **A doc comment ended up on the wrong function** (`notes/selector.go:77-83`). The `matchSelector` comment now sits above `Match`'s, so godoc for `Match` begins "matchSelector is every element…", and `matchSelector` has no doc comment.

5. **Test coverage notes**
   - Regression tests exist and go red for BR-1, BR-2, BR-6 and BR-3 (the contract assertion).
   - BR-5 has `TestDOMSaysWhenMatchesAreCut` and two argv-refusal cases.
   - Nothing covers empty or minimal recordings in the contract; that is the gap the fake panic shows.

6. **Architecture**
   - ARCH-DRY: **pass.** `checkWindow` and `unreadable` are shared, and the selector grammar is delegated to notes. The two `Events` loops are still separate on purpose: sharing them would make the fake repeat the real implementation and weaken the contract test.
   - ARCH-PURE: **pass.** The Querier and rendering are pure functions over the records. `Main` is a thin layer that parses arguments and renders, and the scrubber sits at the IO edge.
   - ARCH-PURPOSE: **pass.** The fixes cover the whole class: both the combinator scoping and the note-selector grammar, not just the two probe cases.
   - ARCH-MOCK: **flagged** (the Minor above). The fake panics where the real surface doesn't. There is no external binary dependency.
   - ARCH-CONSTRAINTS: **pass.** Text output has budgets. JSON is deliberately returned in full, and the CLI now refuses a budget with it. `Notes` makes a single pass over the records.
   - ARCH-SECURE: **pass.** Recorded text is scrubbed before it reaches the terminal, and errors quote user input with `%q`.
   - ARCH-ORDER: **pass.** The component holds no state between calls: each query replays the reducer from the start, and the fold keys include `by`.
   - ARCH-FUNERAL: **pass.** It creates nothing durable: it is a read-only CLI over an existing recording directory.

7. **Plan revisions to add**
   - Add a "## Revisions" entry for close round 1: `Select` delegates note selectors to `notes.Match`, and CSS combinators are scoped to one tree. This replaces the plan's Core-concepts claim that `Select` "searches documents, shadow roots, template contents and iframe documents."
   - Note there that `--json` refuses `--max-lines`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Select delegates to notes.Match and scopes CSS combinators to one tree. Reverting either one turns TestModelSelectorsSelectTheirElement or TestSelect red; the cross-time probe on all fixtures found no CSS-fallback mismatch.
  - id: BR-2
    disposition: addressed
    note: |
      The test now requires exactly one match wherever Structural.Resolve resolves uniquely; turning the notes.Match hand-off off makes it fail.
  - id: BR-3
    disposition: addressed
    note: |
      The contract recording now resolves n1 after the note and RunSurfaceContract asserts NoteResolved; the fake was changed to match.
  - id: BR-4
    disposition: addressed
    note: |
      checkWindow and unreadable are shared helpers. The separate Events loops are a deliberate choice to keep the fake independent of the real implementation.
  - id: BR-5
    disposition: addressed
    note: |
      --json with --max-lines is refused (cli_test cases), and the trailing more-matches line is pinned by TestDOMSaysWhenMatchesAreCut.
  - id: BR-6
    disposition: addressed
    note: |
      The scrubber on stdout covers every text path; turning it off makes TestMainScrubsControlCharacters fail.
  - id: BR-7
    disposition: addressed
    note: |
      Prose-only: the issue Log records the seam additions and the report to the TL. The report itself is the operator's word and was not checked further.
findings:
  - id: new
    severity: Important
    family: doc-restates-changed-semantics
    title: |
      The atlas still says selectors search through shadow roots, frames and templates; after round 1 only >>> crosses trees
    detail: |
      atlas/webcast-format.md:384-388 contradicts the tree-scoped matching (TestSelect pins .badge and "#card span" as matching nothing). Restate SelectorGrammar, or point to it, instead of keeping a hand-written copy.
  - id: new
    severity: Minor
    family: fake-real-contract-gap
    title: |
      FakeSurface.Notes panics on an empty recording (fake.go:54 indexes Records[len-1]); Querier.Notes returns nil
    detail: |
      Brought in by the BR-3 fix and reproduced. This is the 2nd finding in this family. Rule: any input the real surface accepts, the fake must answer the same way, and RunSurfaceContract is the list of those inputs, so add the empty recording there rather than only guarding the fake.
  - id: new
    severity: Minor
    family: misplaced-doc-comment
    title: |
      The matchSelector doc comment now sits above Match in notes/selector.go, so godoc for Match starts "matchSelector is"
```

---

## Re-review — 2026-10-12T02:50:49-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 462 — broadcast-web: agent queries — DOM at t, events in a window, notes with context |
| repo | pair |
| issue file | workshop/issues/000462-broadcast-web-agent-queries-dom-at-t-events-in-a-window-notes-with-context.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a5e1c8626b7f2557391fa933007c9da6e95e9c5..9104b656d7c7646abeef5d67929baf6454a695a2 |
| command | sdlc close --issue 462 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T02:50:49-07:00 |
| verdict | SHIP |

## Review

Focus on the last round's fix commit first.

I'm wrapping up. All three open findings check out as fixed, and I'm writing the verdict now.

```verdict
verdict: SHIP
confidence: high
```

Round 2 fixes all three open findings, and I found nothing new. BR-9 is fixed through its rule, not just at the one site: `RunSurfaceContract` now feeds an empty recording to both surfaces. I removed the fake's guard and the contract test failed (`TestFakeSurfaceMeetsTheSurfaceContract` panics), then put it back. The atlas no longer keeps its own copy of the selector grammar. It describes the per-tree reading and points to `query.SelectorGrammar`, which is the same text the usage message and every refusal print. The `Match` doc comment is back in place. `go test` passes for `./cmd/internal/webcast/...`. I could not run `couchcmd` tests in this checkout: its setup fails because the generated `runtimebundle/assets/` directory is gitignored. That is a property of the environment, not of this diff.

1. **Strengths**
   - `cmd/internal/webcast/query/contract.go:83-94`: the empty-recording checks sit in the shared contract, so the fake and the `Querier` are held to the same answer for that input.
   - `cmd/internal/webcast/query/selector.go:12-15`: the grammar is stated once, and `cli.go:25` and both refusal paths read that one constant.
   - `selector.go:42`: a note's anchor goes through `notes.Match`, so the anchor grammar is defined only once (from round 1, still sound).
   - `atlas/webcast-format.md:384-388` now matches what the code does: tree-scoped combinators, with `>>>` as the only way into another tree.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** none new.

5. **Test coverage notes**
   - Removing the BR-9 fix makes the contract test fail, so it is regression evidence.
   - The `query` and `notes` packages pass.
   - `couchcmd` (which holds `broadcast_list_test.go` and the CLI dispatch for `query.Flag`) could not be built here because the generated assets are missing. The operator should run it in a full checkout before merging.

6. **Architectural notes for upcoming work**
   - **ARCH-DRY:** pass. The grammar lives in one constant and anchors are matched in one place.
   - **ARCH-PURE:** pass. Selecting, rendering and folding events are pure functions over `recording.State`, and only `Main` does IO.
   - **ARCH-PURPOSE:** pass. The atlas, usage text and refusals all derive from `SelectorGrammar`. README and SKILL.md don't restate the grammar.
   - **ARCH-MOCK:** pass. `FakeSurface` is a stateful fake built on the reference reducer, and both surfaces run the same contract. It still accepts only `#id` selectors, but that is documented and the contract only uses `#id`.
   - **ARCH-CONSTRAINTS:** pass. Line, attribute and text budgets are enforced, and round 1 pinned them.
   - **ARCH-SECURE:** pass. Recorded text is quoted before it reaches the terminal (BR-6), and a selector it can't parse is refused, not matched loosely.
   - **ARCH-ORDER:** pass. This is a read-only query layer and keeps no state between calls. Each answer is reduced fresh from the records.
   - **ARCH-FUNERAL:** pass. It writes nothing durable except the golden testdata, and that changes only through `-update`.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      atlas/webcast-format.md:384-388 now describes tree-scoped CSS with >>> as the only crossing and names query.SelectorGrammar (selector.go:12-15) as the single statement, which cli.go:25 and both refusal errors print.
  - id: BR-9
    disposition: addressed
    note: |
      fake.go:54 guards the empty case, and contract.go:83-94 adds the empty recording to RunSurfaceContract for both surfaces; removing the guard makes TestFakeSurfaceMeetsTheSurfaceContract panic (verified, then restored).
  - id: BR-10
    disposition: addressed
    note: |
      The matchSelector comment now sits directly above matchSelector (notes/selector.go:103-104), and Match's godoc starts "Match is".
```
