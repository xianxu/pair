# Boundary Review — pair#266 (whole-issue close)

| field | value |
|-------|-------|
| issue | 266 — muse Alt+Return from draft stays in composer; agent pane Return should be Send |
| repo | pair |
| issue file | workshop/issues/000266-muse-alt-return-from-draft-stays-in-composer-agent-pane-return-should-be-send.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6edff3386b9269d3327242898932477a75601af..8df58a84c23670c2503dd0439bd21d1008d92676 |
| command | sdlc close --issue 266 |
| reviewer | claude |
| timestamp | 2026-09-16T11:34:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The window delivers the issue's purpose: Muse draft `Alt+Return` submits (the 100 ms post-write settle is the measured, operator-confirmed fix), the agent-pane contract is now Return→Muse's native Shift+Return / Alt+Return→bare CR, the falsely-broad `Enter to select` picker marker is gone with a regression test, a live 1.3.0 fixture is checked in and replayed at every split, and README + atlas + the harness bring-up doc all follow the revised contract. The build is clean, `go vet` is clean, the focused Muse/translate/fixture/overlay/orientation suites are green, and `lua nvim/draft_send_test.lua` passes (the only red in `./cmd/internal/wrapcmd` is the pre-existing `/tmp` mkdir permission failure in `TestOrientationChildEnvironmentAndReadinessStatus`, environmental and unrelated). What keeps this from SHIP is a class of finding, not a broken build: two of the bytes pair now emits toward Muse are verified only against pair's own output expectations, never against Muse's interpretation of them — the restored in-paste submit emits `\r` *inside* an still-open bracketed-paste window (where a `?2004h` consumer reads it as pasted text, i.e. the exact symptom it exists to prevent), and `plainCR=\x1b[13;2u` silently depends on Muse keeping the Kitty keyboard protocol pushed. Alongside that, this window propagates Muse's speculative prompt-glyph set into a second decision point (orientation auto-submit) while removing a picker marker, for the one harness that has no captured declining state at all.

## 1. Strengths

- **The real fix is the measured one.** `nvim/draft_send.lua:48` settling after *every* body write, backed by the 18 ms out-of-order live trace, is the right root-cause fix (`ARCH-CONSTRAINTS`), and it is injected as a `settle` callback so `nvim/draft_send_test.lua:38` pins it with no sleep and no IO (`ARCH-PURE`).
- **The expectation is read from the profile, not restated.** `TestComposerReturnExpectationMatchesProfile` plus `ttyFixtureReturnExpectation` meant the `plainCR` change flowed to the fixture replay automatically — exactly the single-source shape that made this contract change cheap.
- **Live 1.3.0 capture checked in** (`testdata/tty/muse/1.3.0-R3057.1/`) and replayed at every byte split by `TestHarnessTTYFixtureConformance` (13 s, green). Capturing the drifted version rather than loosening a test around it is the right move.
- **The overlay-marker removal is correctly narrowed and pinned**: `"Press Enter to select"` survives, `overlay_test.go:139` pins that the bare hint no longer arms. Both strings were added speculatively in the same commit (`e4d15575`), and the removed one was always a superset of the kept one.
- **The issue's `## Revisions` ledger is exemplary** — three superseding contract changes appended, each with its disproof, rather than overwritten.

## 2. Critical findings

None.

## 3. Important findings

**I-1 `cmd/internal/wrapcmd/wrap.go:1960` — the in-paste submit is emitted inside the still-open paste window.** When the Alt chord lands before `bpEnd`, the branch emits `keymap.altCR` and deliberately leaves `inPaste` true, so the harness receives `\x1b[200~draft body\r\x1b[201~`. A consumer that requested bracketed paste (Muse 1.3.0 emits `\x1b[?2004h` — verified in the new fixture) buffers everything between the markers as literal text; a `\r` there is a newline in the pasted body, not an Enter key. That is the original symptom (`draft sits unsent`), now with a stray blank line. `TestMuseDraftAltEnterSubmissionInsidePaste` and `translate_test.go:165` both assert only the *wrapper's* output bytes, so neither can distinguish "submitted" from "pasted a newline", and the operator's live re-confirmation at `0a05b283` is evidence for the settle (which keeps this path from firing), not for this branch. There is a second-order consequence: `publishLifecycleObservation(ObservationUserSubmission)` fires regardless, so if the harness does not submit, pair opens a turn and the notification floor later raises a spurious "no agent output" alert — the hazard `returnDecision.submits` is documented to avoid. Fix sketch: emit `bpEnd` *before* `altCR`, mark the paste closed, and consume the trailing real `\x1b[201~` (carry a "paste already closed" bit across chunks), so the submit lands outside the window. If that ordering can't be made safe, the honest alternative is to keep the settle alone and drop the branch. Note this also changes the cost/benefit the `ARCH-SECURE` acceptance in the atlas rests on: the accepted misread ("a paste whose own payload contains that chord reads as a submit") was justified by a benefit that is currently unproven. Narrowing the scan to a chord that terminates the chunk (or is followed only by `bpEnd`) would shrink the misread surface either way. (`ARCH-ORDER`, `ARCH-MOCK`)

**I-2 `cmd/internal/wrapcmd/harness_tty.go:57` — `plainCR = "\x1b[13;2u"` has an unrecorded, unguarded precondition.** KKP `CSI 13;2u` is only parseable by an application that pushed progressive enhancement. Muse does (`\x1b[>3u` in the 1.3.0 fixture, `\x1b[>1u` in 0.1.0), so the mapping is sound *today* — but nothing in the code says so, and nothing detects the day it stops: `assertHarnessTTYLiveDecision` reads `want` from the same profile it is checking, so the live conformance check is tautological for this byte string. If Muse pops KKP (a subview, a future release), plain Return in the composer becomes a silently-swallowed escape — Enter does nothing at all, and `return-remap` telemetry still reports `fired`. Cheap fix: assert the captured composer prefix contains a KKP push before the profile's KKP-encoded `plainCR` is trusted (the bytes are already in the fixture), and add a one-line comment at the profile naming the dependency. (`ARCH-MOCK`, `ARCH-SECURE`)

**I-3 `cmd/internal/wrapcmd/orientation.go:205` — the speculative Muse glyph set is now also the orientation auto-submit gate, in a harness with no declining evidence.** `orientationPromptOK` accepts `●`, `▶`, `▸`, `>` — conventional *menu selection markers*; `codexComposerActive`'s own comment exists because Codex reuses its prompt glyph as a selection marker. Muse is listed in both `ttyFixtureNegativeGaps` and `ttyFixtureDiscriminationGaps` ("no captured declining state at all"), and this same window removes one of the text markers that backstops the shape heuristic. Separately, `!` is accepted as a Muse prompt while `orientationComposerActive` rejects `!` as *content* precisely to avoid submitting into a shell/non-coding mode — and `orientationPromptOK` rejects `!` for Claude for that reason. So `cc253e42` ("preserve orientation menu guards") leaves a hole in the guard it preserved: a Muse row painted `! …` passes. Fix sketch: drop `!` from the Muse set (and justify `●/▶/▸` with an observation or drop them too), and capture one Muse declining state — a `/`-menu screen is reachable without a tool call and would close the `ttyFixtureDiscriminationGaps` entry. (`ARCH-PURPOSE`)

**I-4 `cmd/internal/wrapcmd/orientation.go:205` + `composer_recognizers.go:187` — two hand-maintained copies of the Muse prompt-glyph set.** `a239c856` removed orientation's duplicate authority citing `ARCH-DRY`; `cc253e42` reintroduced it as a literal list. Nothing pins the two together, so adding a glyph in one place silently produces a state where the Return remap accepts a composer that orientation rejects — the exact divergence `a239c856` was written to kill. Fix: one package-level `musePromptGlyphs` (or expose the spec's `promptOK`), consumed by both; keep orientation's menu/non-coding guard layered on top. (`ARCH-DRY`)

**I-5 `cmd/internal/wrapcmd/wrap.go:1978` — the restored split-Alt holdback has no test.** The loop that extends `tail` to cover a partial `enterKKPAlt`/`enterLegacyAlt` is load-bearing: for a chunk ending `…\x1b[1` inside a paste, `bpEnd`'s partial is 2 bytes and the KKP partial is 3, so without the loop the `1` is forwarded and the chord is lost on the next read. No existing case reaches it — the in-paste partial case (`translate_test.go:82`) uses `\x1b[20`, where `bpEnd` already wins. Add `{startPase: true, in: "data\x1b[13;3", wantOut: "data", wantHold: "\x1b[13;3", wantPaste: true}`.

## 4. Minor findings

- `harness_tty_fixture_test.go:383-386` — the edited doc comment is now both stale and ungrammatical: "Codex, Muse and Agy all remap to LF" is false for Muse, and the sentence lost its predicate ("…whose plainCR is the profile's active-composer Return bytes." with no verb).
- `muse_draft_submit_test.go:50` — `TestMuseDraftAltEnterSubmissionInsidePaste` lost the doc comment explaining *why* the coalesced case exists; the rewrite left the trickiest test in the file unexplained.
- `wrap.go:1969` recomputes `indexOfSubseq(data[i:], bpEnd)` into `idx` when `endIdx` two lines up already holds it.
- `nvim/init.lua:734` — the comment still enumerates "claude: `\<Enter>`, codex/agy: \n" in the function this diff edited; Muse's new Shift+Return mapping is missing.
- `testdata/tty/muse/1.3.0-R3057.1/metadata.json` — `captured_at: 06:00:00Z` is hand-rounded; every other fixture records a real capture second, which is what makes the field usable as provenance.
- `muse_draft_submit_test.go:17,39` branches on the subtest *name string* rather than a table field; a rename silently flips the assertion to the wrong expectation.
- The 100 ms settle is a fixed sleep against an ordering problem with no stated upper bound (18 ms was the one observation) and no detector when it is exceeded — `draft_send.lua` already carries a phase state machine that could confirm delivery instead. (`ARCH-CONSTRAINTS`, `ARCH-ORDER`)

## 5. Test coverage notes

- Green here: `TestMuse*`, `TestTranslateChunk*`, `TestHarnessTTYFixtureConformance`, `TestOverlayDetectorByAgent`, `TestOrientation*` (except the pre-existing `/tmp` permission failure), `TestEmitPlainCR_*`, `lua nvim/draft_send_test.lua`.
- The short-body settle has a genuine failing-without-it test (`draft_send_test.lua:39`) — the strongest regression evidence in the window.
- Two behavior claims have no test that could fail if the behavior were wrong at the harness: the in-paste submit (I-1) and the KKP encoding (I-2). Both assert pair's output bytes against pair's own profile.
- The split-Alt holdback path is uncovered (I-5).
- No test pins the two Muse glyph lists against each other (I-4) — a two-line table test over `musePromptGlyphs` would close it.
- Muse still has zero captured declining states; the `ttyFixtureDiscriminationGaps` entry is honest but this window widened the positive gate's reach while narrowing a text backstop.

## 6. Architectural notes

- `ARCH-DRY` — **flag** (I-4): duplicated glyph authority. Elsewhere the diff is exemplary: the fixture expectation reads from the profile rather than restating it, and `keyhelp` derives from `init.lua`/`shortcut.go` so no second keybinding table needed updating.
- `ARCH-PURE` — pass. `translateChunk` stays a function of `(data, inPaste)`; `draft_send.lua` takes `action`/`settle` injected and is tested with neither zellij nor a clock.
- `ARCH-PURPOSE` — **flag** (I-3). Shadow-sweep of the "Muse Return contract" consumers is otherwise complete: profile, orientation, fixture expectation, `muse_return_test`, README, `atlas/architecture.md` (both the remap and conformance paragraphs), `atlas/how-to-bring-up-a-new-harness-cli.md`. The generated runtime bundle under `cmd/internal/runtimebundle/assets/` is gitignored and rebuilt — verified byte-identical to `nvim/`, so no stale shipped copy.
- `ARCH-MOCK` — **flag** (I-1, I-2). zellij is properly faked (`stateful_zellij` in the lua test); the harness is faked only on the *output* side. There is no double modeling Muse's *input* interpretation, which is why both unverified-byte findings can hide behind green tests. The live conformance check reads its expectation from the profile under test, so it cannot detect drift in what Muse accepts — only in what Muse paints.
- `ARCH-CONSTRAINTS` — **flag** (minor): a blocking 100 ms sleep now sits on every draft submit, budget basis = one measurement, no bounded behavior when exceeded.
- `ARCH-SECURE` — pass with a noted cost. The in-paste chord scan treats clipboard payload as a control event; the tradeoff is explicitly documented in `atlas/architecture.md`, and I am not re-litigating it — but I-1 argues the benefit side of that accepted trade may be zero as implemented.
- `ARCH-ORDER` — **flag** (I-1). The in-paste branch introduces an unmodeled state ("a submit was injected while the paste window is still open") whose legality at the consumer is the whole question. The `(inPaste, leftover)` pair is the right shape; the new transition is the one that isn't written down.
- `ARCH-FUNERAL` — pass. New durable artifacts are one versioned fixture directory (~3 KB, per-Muse-release, operator-curated, same family as the four existing ones) and two `workshop/lessons.md` entries. No new unbounded family, no writer whose per-event growth increased.

## 7. Plan revision recommendations

The plan and Spec are reconciled by the append-only `## Revisions` ledger, and the ledger correctly supersedes its own earlier entries. Two additions worth making rather than rewrites:

- A `## Revisions` entry recording **what the operator actually observed** at `0a05b283` for agent-pane plain Return under the new Shift+Return mapping (composer newline inserted, no submit) — the earlier smoke item was checked before `204bbe24` changed that mapping, so the current plan reads as if the smoke covered a contract that did not exist yet.
- A `## Revisions` entry stating whether the paste-coalesced interception was ever observed to *submit* in a live Muse session, or whether it remains an untriggered defense. The current entry asserts the coalescing can happen; it does not claim the emitted `\r` inside the window reaches Muse as a submit, and I-1 argues it does not.

```findings
findings:
  - id: new
    severity: Important
    family: boundary-semantics-unverified
    title: |
      In-paste submit emits CR inside the still-open bracketed-paste window
    detail: |
      wrap.go:1960 emits keymap.altCR before bpEnd and leaves inPaste true, so the
      harness receives \x1b[200~body\r\x1b[201~. Muse enables ?2004h (verified in the
      1.3.0 fixture), so a CR inside the window is pasted text, not an Enter key — the
      original "draft sits unsent" symptom, now with a stray newline. Both tests assert
      only the wrapper's output bytes, and the operator's live confirmation covers the
      100ms settle, which prevents this path from firing. ObservationUserSubmission
      fires regardless, so a non-submit opens a turn and later raises a spurious
      "no agent output" alert. Emit bpEnd before altCR and consume the trailing real
      close marker, or drop the branch and keep the settle alone (ARCH-ORDER, ARCH-MOCK).
  - id: new
    severity: Important
    family: boundary-semantics-unverified
    title: |
      Muse plainCR uses a KKP sequence whose precondition is unrecorded and unguarded
    detail: |
      harness_tty.go:57 sets plainCR to \x1b[13;2u, parseable only while Muse keeps
      Kitty progressive enhancement pushed. Muse 1.3.0 does push it (\x1b[>3u in the new
      fixture), so the mapping is correct today — but no comment records the dependency
      and assertHarnessTTYLiveDecision reads its expectation from the same profile, so
      the live check cannot detect the day Muse stops. Failure mode is silent: Enter does
      nothing and telemetry still reports fired. Assert a KKP push in the captured prefix
      before trusting a KKP-encoded plainCR (ARCH-MOCK, ARCH-SECURE).
  - id: new
    severity: Important
    family: positive-gate-needs-declining-evidence
    title: |
      Orientation now accepts menu-bullet glyphs and bang as Muse composer prompts
    detail: |
      orientation.go:205 extends the speculative Muse glyph set to the orientation
      auto-submit gate, accepting ●, ▶, ▸, > — conventional selection markers — for the
      one harness listed in both ttyFixtureNegativeGaps and ttyFixtureDiscriminationGaps,
      while this same window removes a Muse picker text marker. Accepting "!" also
      defeats the non-coding-mode guard the commit says it preserved: the same glyph is
      rejected as content, and rejected outright for Claude. Drop "!", justify or drop
      ●/▶/▸, and capture one Muse declining state (a slash menu needs no tool call)
      (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: duplicated-authority
    title: |
      Muse prompt-glyph set is hand-maintained in two places with nothing pinning them
    detail: |
      composer_recognizers.go:187 and orientation.go:205 each carry the same literal
      glyph list. a239c856 removed orientation's duplicate authority citing ARCH-DRY;
      cc253e42 reintroduced it. A glyph added to one produces a state where the Return
      remap accepts a composer orientation rejects. Extract one musePromptGlyphs source
      consumed by both, keeping orientation's menu guard layered on top (ARCH-DRY).
  - id: new
    severity: Important
    family: regression-evidence-missing
    title: |
      Restored split-Alt-partial holdback inside a paste has no test
    detail: |
      wrap.go:1978 extends the held-back tail to cover a partial Alt chord. It is
      load-bearing — for a chunk ending \x1b[1 inside a paste, bpEnd's partial is 2 bytes
      and the KKP partial is 3 — and no existing case reaches it; the in-paste partial
      test uses \x1b[20, where bpEnd already wins. Add a startPase case ending \x1b[13;3
      expecting that exact holdback.
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      Fixture expectation comment is stale and ungrammatical after the Muse change
    detail: |
      harness_tty_fixture_test.go:383-386 still says "Codex, Muse and Agy all remap to
      LF" (false for Muse) and the edited sentence lost its predicate. nvim/init.lua:734
      likewise enumerates claude/codex/agy newline sequences without Muse's new one, in
      the function this diff edited.
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      Coalesced-paste test lost the doc comment explaining why the case exists
    detail: |
      muse_draft_submit_test.go:50 — the rewrite dropped the rationale from the file's
      least self-evident test.
  - id: new
    severity: Minor
    family: duplicated-authority
    title: |
      bpEnd index recomputed one line after endIdx already holds it
    detail: |
      wrap.go:1969 recomputes indexOfSubseq(data[i:], bpEnd) into idx; endIdx at 1947 is
      the same value.
  - id: new
    severity: Minor
    family: fixture-provenance
    title: |
      New Muse fixture records a hand-rounded captured_at
    detail: |
      testdata/tty/muse/1.3.0-R3057.1/metadata.json uses 06:00:00Z; every other fixture
      records a real capture second, which is what makes the field usable as provenance.
  - id: new
    severity: Minor
    family: test-asserts-on-subtest-name
    title: |
      Muse draft submit test branches on the subtest name string
    detail: |
      muse_draft_submit_test.go:17 and :39 key both the fixture paint and the expected
      plain-Return bytes off name == "with composer"; a rename silently flips the
      assertion to the wrong expectation. Use a table field.
  - id: new
    severity: Minor
    family: timing-heuristic-without-bound
    title: |
      100ms settle is a fixed sleep against an ordering problem, with no detector
    detail: |
      nvim/draft_send.lua:48 now blocks every draft send for 100ms, budgeted from a
      single 18ms observation, with nothing reporting when delivery exceeds it.
      draft_send.lua already carries a phase state machine that could confirm delivery
      instead of sleeping (ARCH-CONSTRAINTS, ARCH-ORDER).
```
