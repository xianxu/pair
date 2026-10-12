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
