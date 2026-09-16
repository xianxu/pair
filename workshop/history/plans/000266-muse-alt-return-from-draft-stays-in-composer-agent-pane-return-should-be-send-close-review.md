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

---

## Re-review — 2026-09-16T12:05:07-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 266 — muse Alt+Return from draft stays in composer; agent pane Return should be Send |
| repo | pair |
| issue file | workshop/issues/000266-muse-alt-return-from-draft-stays-in-composer-agent-pane-return-should-be-send.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6edff3386b9269d3327242898932477a75601af..41812c208d33b3265a8f686a415f8dc887ebe094 |
| command | sdlc close --issue 266 |
| reviewer | claude |
| timestamp | 2026-09-16T12:05:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eleven round-1 findings are genuinely disposed, and the evidence holds up under inspection rather than resting on commit prose: the in-paste Alt branch is gone from `wrap.go` (the paste loop now scans only for `bpEnd`), `musePromptGlyphs` is a single authority pinned by `TestMusePromptAuthorityIsShared`, the selection markers are excluded and pinned by `TestMuseComposerActive_RejectsSelectionMarkers`, and the KKP precondition BR-2 named is now *checked against the capture* in both the frozen replay and the live path. I verified the new `muse/1.3.0-R3233.1` fixture records a real capture second (`2026-09-16T18:44:17Z`) and that both its `.raw` files contain a `CSI > … u` push, so the precondition assert passes on real bytes rather than vacuously; `go vet` is clean, `lua nvim/draft_send_test.lua` passes, and the focused wrapcmd suite is green (the only failure, `TestOrientationChildEnvironmentAndReadinessStatus`, is the known `mkdir /tmp/pair255-notify-*: operation not permitted` environment trap, reproduced across unrelated tests in the same run). What keeps this from SHIP is one cheap gap: the in-paste Alt interception has now been added, removed, restored and removed again inside this one issue, and nothing in the test suite pins its absence — the next author has no red test standing between them and round four.

## 1. Strengths

- **`assertKittyKeyboardPrecondition` attacks the right oracle.** `harness_tty_fixture_test.go:400` is the rare assertion in this file that does *not* read its expectation from the profile — it checks the capture. That is exactly the gap BR-2 named, and wiring the same helper into `harness_tty_live_test.go:652` means the frozen replay and the live check share one precondition instead of two restatements.
- **`menu.raw` is real discrimination evidence, and it is load-bearing.** `ttyFixtureExpectation["muse"]["menu.raw"] = true` (`harness_tty_fixture_test.go:270`) puts the slash-menu capture through `replayHarnessTTYFixture` at every byte split, so the claim "the gate stays open on the menu and emits Shift+Return there" is a frozen regression, not a comment. The glob in `orientation_test.go:261` picks the new fixture up automatically, so orientation's refusal to auto-submit into it is checked for free.
- **`TestMuseComposerActive_RelaxedPrompt` iterates `musePromptGlyphs` itself** (`muse_draft_submit_test.go:104`) rather than a parallel literal list — a glyph admitted to the authority cannot arrive without coverage. Paired with `TestMusePromptAuthorityIsShared`, BR-4's drift is structurally closed, not just repaired.
- **Deleting the R3057.1 fixture rather than back-filling its `captured_at`** was the right call on BR-9: unverifiable provenance is worse than no fixture.
- **BR-11 was deferred honestly.** `workshop/issues/000269-*.md` states the actual defect (a budget from one measurement, paid by every send, with no detector) and names the phase machine as the seam — that is a follow-up issue, not a shelf.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — No test pins that an Alt+Enter chord inside a bracketed paste stays literal.** `cmd/internal/wrapcmd/translate_test.go:47`

> **This is the 2nd finding in family `regression-evidence-missing`.** Earlier rounds fixed instances. Do NOT fix this instance alone — state the rule that covers all of them, and fix that.

The rule: **a deliberate decision that the translator will *not* do something is a behavior, and it needs a named test asserting the non-behavior — the absence of code is not regression evidence.** Measured prevalence on this issue alone: the in-paste Alt branch was added (`0a05b283`'s ancestor), removed (`901074d3`), restored (`0a05b283`), and removed again (`41812c20`) — three flips, each justified by reasoning in a commit body, none of them blocked by a red test. The existing paste cases (`translate_test.go:47`, `:59`, `:169`) all pin plain `\r` inside a paste; none contains an ESC, so the exact byte sequence the branch existed to intercept has no coverage in either direction. Applying the rule here is one table row: `in: "\x1b[200~body\x1b\r\x1b[201~"`, `wantOut` identical, plus the KKP form `\x1b[13;3u` — named so its failure message says *why* the bytes must stay literal (Muse has `?2004h` on; a CR emitted inside the window is pasted text, not an Enter key, which is BR-1's finding in one sentence). The same rule should be applied to the other accepted tradeoff this round introduces — see M-3.

## 4. Minor findings

**M-2 — A comment cites a fixture directory this same window deleted.** `cmd/internal/wrapcmd/harness_tty.go:61`

> **This is the 3rd finding in family `stale-comment`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

The rule: **a comment must not restate a fact the tree can check; where it names a path, a version, or a registry entry, a test asserts the referent resolves.** Measured prevalence: I enumerated every `testdata/tty/<agent>/<version>` reference in non-`workshop/` source — 5 references, 4 resolve, 1 is dead, and the dead one is the newest (`harness_tty.go:61` points at `testdata/tty/muse/1.3.0-R3057.1`, removed later in this same range as part of BR-9). A second instance of the same rule sits in this diff: `harness_tty_fixture_test.go:156` says the slash menu and the `?` shortcut sheet "were both driven live … (see `harnessTTYDrivenScenarios`)", but `harness_tty_live_test.go:719` registers only the slash menu — the pointer that makes the acknowledgment reviewable does not resolve for half of what it claims. Both are instances of a checkable fact restated in prose. The enforcement already has a template in this file: `TestHarnessTTYFixtureConformance` fails when a gap acknowledgment outlives its gap; the same shape can walk the package's comments for `testdata/tty/...` and assert each directory exists, which retires the whole family rather than these two sites.

**M-3 — The KKP push guard is flags-agnostic, and the menu's Return behavior is asserted from reasoning rather than observation.** `cmd/internal/wrapcmd/harness_tty_fixture_test.go:390`, `:265`

> **This is the 3rd finding in family `boundary-semantics-unverified`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

The rule: **a claim about what the harness does with bytes we emit is either backed by a capture or a live drive, or it is recorded in the acknowledged-gap tables — never asserted in a comment as if settled.** The repo already owns the mechanism (`ttyFixtureNegativeGaps` / `ttyFixtureDiscriminationGaps` are reviewable and expire); the rule is to route unverified harness-reaction claims through it instead of into prose. Measured prevalence, two instances this round: (a) `kittyKeyboardPush = "\x1b\\[>[0-9;]*u"` matches `\x1b[>0u` — a push of flags 0 is KKP *disabled* — and is agnostic to whether bit 1 (disambiguate escape codes) is set, which is the specific bit that makes `CSI 13;2u` the encoding of Shift+Enter; the guard therefore establishes "Muse spoke KKP at us", not "Muse will parse this key". (b) `ttyFixtureExpectation` at `:265` asserts "Enter therefore inserts a newline instead of picking the highlighted command" for the slash menu — nobody pressed Return in that menu. Contrast the Agy entry three lines above, which states the same tradeoff as a *checked property* ("safe only because Agy inserts a newline on LF there"). The escape hatch here is sound by construction (Alt+Return → bare `\r` → exactly what Muse receives natively), so the consequence is bounded; the claim about the plain-Return half is the unverified part.

Below ledger threshold, noted only:

- `nvim/draft_send.lua:48-49` — two `if cmd.kind == 'write'` tests one line apart; collapse into the existing branch.
- `composer_recognizers.go:196` — the reflowed doc comment lost its wrap ("…remap; the admitted glyphs are musePromptGlyphs. The box shape (prompt row enclosed by two "─" rules) remains the"), a 100+ column line in a file that otherwise wraps at ~75.

## 5. Test coverage notes

- The behavior change at the centre of this round — Muse composer Return → `\x1b[13;2u` — is pinned in four independent places that would each go red on a profile edit: `harness_tty_test.go:30` (registry), `harness_tty_fixture_test.go:425` (profile-vs-documented cross-check), `muse_return_test.go:17` (literal fixture through `emitPlainCR`), and `muse_draft_submit_test.go:19` (table-driven, per BR-10). That is appropriate redundancy, not duplication — three are deliberate restatements acting as pins.
- `TestMuseComposerActive_RejectsSelectionMarkers` tests both the set membership *and* the recognizer verdict for each excluded glyph, so it survives a refactor that changes how the glyph reaches the recognizer.
- The `nvim/draft_send_test.lua` addition is the right shape (`short body settles exactly once`) — it would have gone red against the old `body:find('\n') or #body > 200` predicate, so it is real regression evidence for the settle broadening.
- Gap, per I-1: the removed in-paste path. Gap, per M-3(b): no capture or drive establishes what Muse does with `\x1b[13;2u` while the slash menu is open.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** BR-4's duplicate authority is collapsed into `musePromptGlyphs` and pinned. Worth knowing for the next harness: `orientationPromptOK` still hand-restates `claude: ❯`, `codex: ›`, `agy: >` separately from those harnesses' recognizers (`orientation.go:208`). Muse is now the only one where the two gates share a source. That is pre-existing and outside this window, but the fifth harness should join Muse's shape rather than the map's.
- **ARCH-PURE — pass.** `musePromptGlyphs`, `museComposerActive`, `orientationPromptOK` and `plainCRNeedsKittyKeyboard` are all pure over a snapshot or a byte slice; the only IO in their tests is fixture reads through the established seam.
- **ARCH-PURPOSE — pass.** The issue's stated purpose (draft Alt+Return submits; agent-pane Return follows the convention) is delivered, and the shadow-sweep of the Return contract's consumers is clean: `harnessTTYProfiles` is the source, `emitPlainCR` / `composerReturnBytes` / the live check all derive from it, and every hand-maintained restatement (README:120, `atlas/architecture.md:697,707`, `atlas/how-to-bring-up-a-new-harness-cli.md:29`, `nvim/init.lua:734`) was updated in this same range. I checked `cmd/internal/keyhelp` — it parses source, so it needs no edit. BR-11's deferral to #269 is a separable redesign of the delivery handshake, not the point of #266, so deferring it is legitimate.
- **ARCH-MOCK — pass, strengthened.** The frozen capture + driven scenario + live conformance triad is the stateful-fake seam, and this round added a Muse driven scenario, a menu capture, and the first assertion that reads the *capture* rather than the profile. The residual limit is in M-3: the fake models bytes we emit, never Muse's reaction to them.
- **ARCH-CONSTRAINTS — flagged, already tracked.** The 100 ms settle now sits on the keystroke path for *every* draft send including short ones, budgeted from a single 18 ms observation with no detector. This is BR-11 exactly; #269 owns it and states it correctly. Not re-raised.
- **ARCH-SECURE — pass, improved.** Removing the in-paste interception removes the case where arbitrary paste payload containing one chord was reinterpreted as a trusted out-of-band submit. The harness output parsed here (`terminalSnapshot`, fixture bytes) is handled totally — `musePromptGlyphs[c.Content]` on an unknown glyph is `false`, and an absent profile fails closed to bare CR via `decidePlainReturn`.
- **ARCH-ORDER — pass, strongest part of the diff.** The removal shrinks the state carried between events: there is no longer a reachable `(inPaste=true, submit emitted, ObservationUserSubmission published)` state, which is precisely the illegal combination BR-1 named. The `(state, event)` space of `translateChunk` is small and readable off the code, and `replayHarnessTTYFixture` exercises every chunk boundary rather than the one the author happened to get — a real ordering oracle, not a sample of size one.
- **ARCH-FUNERAL — pass, with an unwritten rule.** Fixture directories are a growing per-harness-version family, and this window demonstrated the removal path (R3057.1 deleted once superseded). But that "superseding generation" rule lives only in this issue's Log — `atlas/architecture.md`'s conformance paragraph documents capture and recapture and says nothing about pruning. Muse now carries `0.1.0-R708.1` and `1.3.0-R3233.1`; Codex carries two. One sentence in the conformance paragraph ("a version is retained only while a test names it; superseded captures are deleted, not archived") would make the practice checkable before the directory count makes it a question.

## 7. Plan revision recommendations

The issue's `## Revisions` log is thorough and the round-1 disposal entry is exemplary. Two statements in the *original* sections are now false and were never corrected by a revision — the Log records the truth but the Spec still asserts the old contract, so a reader who stops at `## Spec` is misled:

- **`## Spec`, bullet 4** still reads "relaxed to allow any prompt glyph in `{⟩,›,❯,>,!,●,▶,▸}`". BR-3 removed `!`, `●`, `▶`, `▸`; the admitted set is the chevron family `{⟩ › ❯ >}`. Add a `## Revisions` entry: *"2026-09-16 — admitted Muse prompt glyphs narrowed to the chevron family. `!`, `●`, `▶` and `▸` were speculative and are selection markers; `!` additionally contradicted orientation's non-coding-mode guard. Authority is now `musePromptGlyphs` (`composer_recognizers.go:116`), pinned by `TestMuseComposerActive_RejectsSelectionMarkers` and `TestMusePromptAuthorityIsShared` (#266 close BR-3, BR-4)."*
- **`## Spec` bullet 2 and `## Done when` bullet 2** still say bare Return "must insert a **newline** (`\n`)" and "`Return` inserts newline (`\n`)". The "Muse composer newline mapping clarified" revision supersedes this with `ESC [13;2u`, but the Done-when criterion is the thing a closer reads to decide the issue is satisfied, and as written it contradicts the shipped and verified behavior. Extend that revision (or add one) to restate the Done-when explicitly: *"Done-when bullet 2 now reads: with the Muse composer active, `Return` emits Muse's native Shift+Return (`ESC [13;2u`) and `Alt+Return` emits bare CR."*

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      In-paste branch removed; wrap.go:1942-1956 scans only bpEnd, no altCR or ObservationUserSubmission is reachable inside a paste.
  - id: BR-2
    disposition: addressed
    note: |
      assertKittyKeyboardPrecondition checks the capture (not the profile) in both the frozen replay and the live path; both muse fixtures carry a real CSI > u push.
  - id: BR-3
    disposition: addressed
    note: |
      Selection markers dropped and pinned by TestMuseComposerActive_RejectsSelectionMarkers; slash menu driven live and captured as muse/1.3.0-R3233.1/menu.raw.
  - id: BR-4
    disposition: addressed
    note: |
      One musePromptGlyphs authority read by both gates, pinned by TestMusePromptAuthorityIsShared.
  - id: BR-5
    disposition: addressed
    note: |
      The untested holdback is gone with the branch it served.
  - id: BR-6
    disposition: addressed
    note: |
      harness_tty_fixture_test.go:418 and nvim/init.lua:733 both corrected; a new instance appeared elsewhere, raised separately under the family rule.
  - id: BR-7
    disposition: addressed
    note: |
      Test removed with the branch; the surviving cases carry rationale comments.
  - id: BR-8
    disposition: addressed
    note: |
      Single indexOfSubseq call at wrap.go:1945.
  - id: BR-9
    disposition: addressed
    note: |
      R3057.1 deleted; muse/1.3.0-R3233.1/metadata.json records captured_at 2026-09-16T18:44:17Z, a real capture second.
  - id: BR-10
    disposition: addressed
    note: |
      muse_draft_submit_test.go:19 keys composer state and expected bytes off table fields.
  - id: BR-11
    disposition: addressed
    note: |
      Filed as pair#269 with the defect stated correctly (budget from one measurement, no detector) and the phase machine named as the seam; settle retained as the confirmed fix.
findings:
  - id: new
    severity: Important
    family: regression-evidence-missing
    title: |
      No test pins that an Alt+Enter chord inside a bracketed paste stays literal
    detail: |
      2nd in family. RULE: a deliberate decision that the translator will NOT do
      something is a behavior and needs a named test asserting the non-behavior;
      absence of code is not regression evidence. Prevalence on this issue alone:
      the in-paste Alt branch was added, removed, restored and removed again
      (901074d3, 0a05b283, 41812c20) with no red test blocking any flip. Existing
      paste cases (translate_test.go:47, :59, :169) all use plain CR; none
      contains an ESC, so the intercepted sequence is uncovered in either
      direction. Applying the rule here is one row per protocol form asserting
      "\x1b[200~body\x1b\r\x1b[201~" and the KKP variant forward verbatim, named
      so the failure says why (Muse has ?2004h on; a CR inside the window is
      pasted text, not an Enter key).
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      Comment cites a fixture directory this same window deleted
    detail: |
      3rd in family. RULE: a comment must not restate a fact the tree can check;
      where it names a path, version or registry entry, a test asserts the
      referent resolves. Measured: 5 testdata/tty/<agent>/<version> references in
      non-workshop source, 4 resolve, 1 dead — harness_tty.go:61 points at
      testdata/tty/muse/1.3.0-R3057.1, removed later in this same range. Second
      instance of the same rule: harness_tty_fixture_test.go:156 claims the slash
      menu and the `?` sheet were both driven "see harnessTTYDrivenScenarios",
      but harness_tty_live_test.go:719 registers only the slash menu. The
      enforcement template already exists in this file (the gap-acknowledgment
      expiry check); walking package comments for testdata/tty paths retires the
      family rather than these two sites.
  - id: new
    severity: Minor
    family: boundary-semantics-unverified
    title: |
      KKP push guard is flags-agnostic and the menu's Return behavior is asserted from reasoning
    detail: |
      3rd in family. RULE: a claim about what the harness does with bytes we emit
      is either backed by a capture or a live drive, or it is recorded in the
      acknowledged-gap tables — never asserted in a comment as if settled. The
      mechanism already exists and expires; the rule is to route unverified
      harness-reaction claims through it. Two instances: (a)
      harness_tty_fixture_test.go:390 matches "\x1b[>0u" (KKP disabled) and is
      agnostic to bit 1, the flag that makes CSI 13;2u the encoding of
      Shift+Enter, so it establishes "Muse spoke KKP" not "Muse parses this key";
      (b) harness_tty_fixture_test.go:265 states "Enter therefore inserts a
      newline instead of picking the highlighted command" for the slash menu with
      nobody having pressed Return there — contrast the Agy entry three lines
      above, which states the same tradeoff as a checked property. The escape
      hatch (Alt+Return to bare CR, exactly what Muse receives natively) is sound
      by construction, so the consequence is bounded.
```

---

## Re-review — 2026-09-16T12:29:05-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 266 — muse Alt+Return from draft stays in composer; agent pane Return should be Send |
| repo | pair |
| issue file | workshop/issues/000266-muse-alt-return-from-draft-stays-in-composer-agent-pane-return-should-be-send.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6edff3386b9269d3327242898932477a75601af..89a9a243a342ad2ca0c4780bd7971651b2709f7a |
| command | sdlc close --issue 266 |
| reviewer | claude |
| timestamp | 2026-09-16T12:29:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All three open findings are genuinely addressed, and I verified each mechanically rather than from prose: reverting the in-paste interception into a scratch copy of HEAD turns five subtests red (`translate_test.go` ×3 + `TestMuseDraftBodyPasteStaysLiteral` ×2), planting a dead `testdata/tty/muse/9.9.9-nope/composer.raw` comment turns `TestTTYFixtureReferencesResolve` red, and every muse fixture really does carry a `CSI > u` push with the disambiguate bit (`>3u` on 1.3.0-R3233.1, `>1u` on 0.1.0-R708.1) so `assertKittyKeyboardPrecondition` passes on real bytes, not vacuously. `go build ./...` and `go vet ./cmd/internal/wrapcmd` are clean, `lua nvim/draft_send_test.lua` passes, and the focused wrapcmd suite is green (the only red, `TestOrientationChildEnvironmentAndReadinessStatus` and friends, is the known `mkdir /tmp/...: operation not permitted` environment trap, reproduced across unrelated tests in the same run). What keeps this from SHIP is cheap and confined to the new evidence machinery: the reaction-gap ledger's expiry predicate cannot prove the property it names and exempts the one screen this issue turns on, and the commit that deleted the in-paste branch took three test rows with it that pinned a *surviving* behavior — Alt+Enter arriving after `bpEnd` in the same read, which is the production draft path. Neither is a broken build; both are one commit.

## 1. Strengths

- **The non-behavior is pinned in both places it matters, and it is real evidence.** `translate_test.go:52-75` plus `muse_draft_submit_test.go:TestMuseDraftBodyPasteStaysLiteral` go red the moment the branch comes back — I confirmed by reapplying `41812c20^`'s `wrap.go` in a scratch tree. Four flips across this issue produced zero red tests; the fifth cannot.
- **`TestTTYFixtureReferencesResolve` is a real enforcement, not a wrapper around the two known sites** (`harness_tty_fixture_test.go:458`). It walks every `.go` in the package, checks comments as well as code, and `t.Fatal`s if it ever stops finding references — the self-disabling guard is the part most such tests omit.
- **BR-2's precondition is checked against the *capture*, not the profile** (`harness_tty_fixture_test.go:540`), in both the frozen replay and `assertHarnessTTYLiveDecision`. Requiring the disambiguate bit rather than any `CSI > … u` is the correct tightening: `CSI > 0 u` is a push that disables every enhancement.
- **`shortcuts.raw` is the right kind of evidence.** Muse printing "shift + enter for newline / enter to submit message" turns this profile's inverted keymap from a Log entry into a captured, replayed artifact — and it incidentally documents `alt + enter to queue while running`, which is worth knowing given pair rewrites Alt+Enter before Muse sees it.
- **One prompt authority, pinned against drift.** `musePromptGlyphs` (`composer_recognizers.go:100`) with `TestMusePromptAuthorityIsShared` and `TestMuseComposerActive_RejectsSelectionMarkers` closes BR-3 and BR-4 together, and the live prompt on 1.3.0-R3233.1 really is `❯` — so the relaxation was load-bearing, not speculative.

## 2. Critical findings

None.

## 3. Important findings

**I-1 `cmd/internal/wrapcmd/harness_tty_fixture_test.go:200` — the reaction-gap retirement oracle cannot prove what it names, and exempts the screen that matters.**
This is the **4th finding in family `boundary-semantics-unverified`.** Do not fix the instance — state the rule and fix the class.

The rule: **a check that retires an acknowledged gap must test the exact property the gap names; if the data model cannot express that property, add the field rather than approximating it — an approximate expiry retires evidence-free.** Two ways this instance breaks it:

(a) `drivenReturnOnOpenGateScreen` accepts `strings.Contains(scenario.send, "\r")` as proof that Return was pressed *on* the captured screen. It cannot be: `driveHarnessTTYScenario`'s `Input` callback (`harness_tty_live_test.go:809-816`) fires `scenario.send` exactly once, the instant the *composer* is recognized, in order to reach the target screen. A `\r` in `send` is therefore always a Return pressed on the composer. The doc comment at `:196` claims the opposite ("actually presses Return on the screen a fixture file captures"). Nothing misfires today — `claude`'s `permission prompt` (`send: "…ls -la\r"`) targets `overlay.raw` and `codex`'s `working status` targets `working.raw`, neither of which appears in its harness's `ttyFixtureExpectation` map — but the first harness that pins an open-gate screen reached by a Return will silently be told to drop its gap entry. Fix: add an explicit field (`returnOnScreen bool`, or a `sendAfter` dispatched once `settled`) and key the check on that.

(b) `anyOpenGateScreen` (`:212`) skips `composer.raw` by construction, so the ledger demands reaction evidence for menus and exempts the composer — the one screen where this issue's whole change lives. Muse reading `ESC[13;2u` as a newline is currently inferred from `shortcuts.raw` plus the KKP push, never driven and never operator-smoked under the shipped mapping (the `## Plan` manual-smoke checkbox predates `204bbe24`, which introduced it). The inference is strong and I would not block on it, but it should be named in `sdlc close --verified` rather than left implicit.

The class, enumerated: three gap ledgers live in this file, and only one has an exact expiry. `ttyFixtureNegativeGaps` errors on `found && acknowledged` (exact). `ttyFixtureReactionGaps` errors on `driven && acknowledged` where `driven` is the approximation above (new this round). `ttyFixtureDiscriminationGaps` at `:145` has **no expiry branch at all** — `if _, listed := …; !listed && !found` only demands an entry; a harness that later gains a discriminating negative keeps a stale entry forever (pre-existing). Sweeping the enumeration means giving all three the same exact-expiry shape.

**I-2 `cmd/internal/wrapcmd/translate_test.go` — deleting the in-paste branch took three rows that pinned a *surviving* behavior.**
This is the **3rd finding in family `regression-evidence-missing`.** Do not fix the instance — state the rule and fix the class.

The rule: **when a branch is deleted, its test rows are triaged, not deleted with it — a row that pinned behavior the deletion leaves intact has to move, or the deletion silently drops coverage.** `41812c20` removed `"Alt+Enter inside bracketed paste is still a submit"` (`\x1b[200~hello\x1b[201~\x1b\r`), its KKP twin, and `"Alt+Enter with paste end in same chunk before submit"`. All three pinned Alt+Enter arriving *after* `bpEnd` in the same read — which is not the removed branch's behavior at all, it is the ordinary post-paste path, and it is exactly the shape the production draft send produces when `write-chars` and `send-keys Alt Enter` coalesce with the close marker first. I verified the behavior still works (`\x1b[200~ok\x1b[201~\x1b\r` → `\x1b[200~ok\x1b[201~\r`), so this is a lost pin, not a bug — but it is the pin on the positive half of the very contract this issue exists to defend, removed in the same commit that added the pin on the negative half. Restore one row per protocol.

## 4. Minor findings

- **M-1 `harness_tty_fixture_test.go:320` and `harness_tty_live_test.go:736`** — the Agy gap's *status* is now stated in two places that disagree: the expectation comment says the newline-on-LF behavior is "pinned so that stays a checked property" and the driven-scenario comment states it flatly, while the new `ttyFixtureReactionGaps["agy"]` says it "was never sent." **3rd finding in family `duplicated-authority`** — the rule is that the ledger is the single authority for whether a harness-reaction claim is verified; comments cite it, never restate a verdict about it. The BR-14 sweep corrected the Muse comment and left the two Agy ones, which is the instance-not-class shape (`ARCH-PURPOSE`).
- **M-2 `harness_tty_fixture_test.go:196`** — the doc comment opens `// drivenReturnScenario …` for a function named `drivenReturnOnOpenGateScreen`. **4th finding in family `stale-comment`** — the rule generalizes the one this round enforced for paths: *a comment's leading identifier is a claim the tree can check, and a doc comment must name the declaration it precedes*. Measured prevalence in this package: 2 — this one, and `wrap.go:1768`, where `observationProfile` was inserted underneath `checkOverlayOpen`'s doc comment back in `799fb6c3` (#184). A sibling of `TestTTYFixtureReferencesResolve` over `go/ast` doc groups retires the family; fixing only these two does not.
- **M-3 `composer_recognizers.go:196`** — a 118-char comment line in a file that wraps at ~72; the mechanical edit that inserted "the admitted glyphs are musePromptGlyphs" left the rest of the sentence unwrapped.
- **M-4 `translate_test.go:69`** — the row named "Alt+Enter chord split across a paste boundary" is not split across a boundary; it is a complete chord inside a continuing paste (`startPase: true`, single `translateChunk` call). The table cannot express a two-chunk feed, so a genuinely split chord (chunk 1 ending `…\x1b[13;3`, chunk 2 `u\x1b[201~`) is still unpinned — which is the case the deleted `trailingPartial` holdback actually governed.
- **M-5 `nvim/draft_send.lua:44-48`** — two separate `if cmd.kind == 'write'` branches three lines apart; fold the `settle()` into the existing one.
- **M-6** `draft_send_test.lua` pins the settle count on the submit path only; the `no_submit` (Alt+Shift+Return) path now settles too and has no count assertion.

## 5. Test coverage notes

- Regression evidence for all three dispositions is real and I reproduced it: interception restored → 5 red; dead fixture path planted → red; the KKP guard reads the capture, and the captures carry flag words 3 / 1 / 5 so the assert is exercised rather than short-circuited.
- `TestHarnessTTYFixtureConformance` grew 13.8 s → 16.7 s between base and head (measured) for two added Muse fixtures. The stride bound (`harnessTTYExhaustiveSplitBytes` + `harnessTTYSplitStride`) already governs this and reports what it skipped; no action needed, but the per-fixture cost is roughly 1.5 s and worth remembering before the next capture.
- Gap relative to the issue's Done-when: the agent-pane half is asserted entirely wrapper-side. Every green assertion about `\x1b[13;2u` reads the expectation from the profile or is a deliberate restatement of it; the only capture-side evidence is `shortcuts.raw`'s prose and the KKP push. See I-1(b).

## 6. Architectural notes

- **ARCH-DRY — pass.** `musePromptGlyphs` is one authority read by both gates and pinned by `TestMusePromptAuthorityIsShared`; expectations read from the profile via `composerReturnBytes`. M-1 and M-5 are the residual duplications.
- **ARCH-PURE — pass.** Recognizers stay pure over `terminalSnapshot`; `translateChunk` is pure; the 100 ms settle is injected as a callback so `draft_send_test.lua` pins it with no sleep and no IO.
- **ARCH-PURPOSE — flag (M-1, I-1b).** BR-14's fix named the class correctly and built the mechanism, then swept two of the three comment sites and scoped the mechanism to exclude the composer. That is the instance, not the class.
- **ARCH-MOCK — pass, strengthened.** Capture + driven scenario + live conformance is the stateful-fake triad, and this round added the first assertion that reads the capture rather than the profile. Residual limit is I-1: the fake models bytes we emit, never Muse's reaction to them.
- **ARCH-CONSTRAINTS — pass.** Fixture replay cost measured and bounded (above). The blocking `vim.cmd('sleep 100m')` now on every send including short drafts is BR-11, already filed as **#269**; not re-raised.
- **ARCH-SECURE — pass, improved.** Removing the interception removes the path where arbitrary paste payload containing one chord was reinterpreted as a trusted out-of-band submit, and that removal is now pinned. Unknown glyph → `false`; absent profile → fails closed to bare CR.
- **ARCH-ORDER — pass, strongest part of the diff.** The illegal `(inPaste=true, altCR emitted, ObservationUserSubmission published)` state is now unreachable, and `replayHarnessTTYFixture` exercises every chunk boundary rather than the one the author happened to get — a real ordering oracle. M-4 is the one ordering case the table shape cannot reach.
- **ARCH-FUNERAL — pass, with an unwritten rule (round 2 raised this; still unapplied).** Fixture version directories are a growing per-harness-release family, and this window demonstrated the removal path by deleting R3057.1 when superseded — but that rule lives only in the issue Log. `atlas/architecture.md`'s conformance paragraph documents capture and recapture and says nothing about pruning; Muse now carries two versions, Codex two. One sentence ("a version is retained only while a test names it; superseded captures are deleted, not archived") makes the practice checkable before the directory count becomes the question.

## 7. Plan revision recommendations

Round 2's two section-7 recommendations were **not applied** — I re-state them, since the close gate reads Done-when:

- **`## Spec`, bullet 4** still reads "relaxed to allow any prompt glyph in `{⟩,›,❯,>,!,●,▶,▸}`". BR-3 narrowed it to the chevron family `{⟩ › ❯ >}`. Add a `## Revisions` entry naming `musePromptGlyphs` (`composer_recognizers.go:100`) as the authority and `TestMuseComposerActive_RejectsSelectionMarkers` / `TestMusePromptAuthorityIsShared` as the pins.
- **`## Spec` bullet 2 and `## Done when` bullet 2** still say bare Return "must insert a **newline** (`\n`)". The shipped contract is `ESC [13;2u`. Extend the "Muse composer newline mapping clarified" revision to restate the Done-when explicitly — a closer reading it today would find the criterion contradicted by the code that satisfies it.
- New this round: **`## Plan`, the manual-smoke checkbox** ("operator verified live") and the line "Operator re-confirmed the fix at HEAD (`0a05b283`)" both read as covering the agent-pane contract, but the smoke predates `204bbe24` (the Shift+Return mapping) and the re-confirmation is recorded as being about draft submission. Say which half each confirmed, and name the agent-pane Return check in `sdlc close --verified`.
- Process note, not a finding: a 24-file / ~1200-line window landed with no durable design plan under `workshop/plans/` — only the two generated gate ledgers. AGENTS.md §2 asks for one above 3 files / 100 lines. Worth a Log line acknowledging the deviation rather than leaving it implicit.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Verified by reapplying 41812c20^'s wrap.go in a scratch copy of HEAD: 5 subtests go red (translate_test.go x3, TestMuseDraftBodyPasteStaysLiteral x2).
  - id: BR-13
    disposition: addressed
    note: |
      TestTTYFixtureReferencesResolve walks every .go incl. comments and self-disables loudly; verified red on a planted testdata/tty/muse/9.9.9-nope path. Both cited instances corrected; the `?` sheet is now a registered driven scenario with shortcuts.raw captured.
  - id: BR-14
    disposition: addressed
    note: |
      ttyFixtureReactionGaps exists and is enforced, and the KKP guard now requires the disambiguate bit (muse fixtures push >3u/>1u, so the assert runs on real bytes). Residual soundness gaps in the new mechanism raised fresh as I-1, not as a re-raise.
findings:
  - id: new
    severity: Important
    family: boundary-semantics-unverified
    title: |
      Reaction-gap retirement oracle cannot prove what it names, and exempts composer.raw
    detail: |
      4th in family. RULE: a check that retires an acknowledged gap must test the
      exact property the gap names; if the data model cannot express it, add the
      field rather than approximating it. (a) drivenReturnOnOpenGateScreen
      (harness_tty_fixture_test.go:200) accepts a "\r" anywhere in scenario.send as
      proof Return was pressed ON the captured screen, but driveHarnessTTYScenario's
      Input callback (harness_tty_live_test.go:809) dispatches send exactly once, on
      the COMPOSER, to reach the target screen — so the predicate can only ever
      observe a Return pressed elsewhere, while :196 claims the opposite. Nothing
      misfires today (claude's "\r" scenario targets overlay.raw, codex's targets
      working.raw; neither is in its harness's ttyFixtureExpectation map), but the
      first open-gate screen reached via Return silently retires its gap.
      (b) anyOpenGateScreen (:212) skips composer.raw by construction, exempting the
      one screen this issue turns on: Muse reading ESC[13;2u as a newline is inferred
      from shortcuts.raw plus the KKP push, never driven, and the Plan's manual-smoke
      checkbox predates 204bbe24 which introduced that mapping.
      CLASS, enumerated: three gap ledgers in this file, one exact expiry.
      ttyFixtureNegativeGaps errors on found && acknowledged (exact);
      ttyFixtureReactionGaps uses the approximation above (new this round);
      ttyFixtureDiscriminationGaps (:145) has NO expiry branch at all, so an entry
      outlives its gap forever (pre-existing). Sweep = give all three the same shape.
      (ARCH-PURPOSE, ARCH-MOCK)
  - id: new
    severity: Important
    family: regression-evidence-missing
    title: |
      Deleting the in-paste branch took three rows that pinned a surviving behavior
    detail: |
      3rd in family. RULE: when a branch is deleted, its test rows are triaged, not
      deleted with it — a row pinning behavior the deletion leaves intact must move,
      or the deletion silently drops coverage. 41812c20 removed three translate_test
      rows ("Alt+Enter inside bracketed paste is still a submit", its KKP twin, and
      "Alt+Enter with paste end in same chunk before submit"). All three pinned
      Alt+Enter arriving AFTER bpEnd in one read — not the removed branch's behavior,
      but the ordinary post-paste path, and the exact shape the draft send produces
      when write-chars and send-keys coalesce with the close marker first. Verified
      still correct at HEAD ("\x1b[200~ok\x1b[201~\x1b\r" -> "\x1b[200~ok\x1b[201~\r"),
      so this is a lost pin rather than a bug — but it is the pin on the POSITIVE half
      of the contract this issue exists to defend, dropped in the same commit that
      added the pin on the negative half. Restore one row per protocol.
  - id: new
    severity: Minor
    family: duplicated-authority
    title: |
      Agy gap status is stated in two places in one file, contradictorily
    detail: |
      3rd in family. RULE: the gap ledger is the single authority for whether a
      harness-reaction claim is verified; comments may cite it, never restate a
      verdict about it. harness_tty_fixture_test.go:320 says the Agy newline-on-LF
      behavior is "pinned so that stays a checked property" and
      harness_tty_live_test.go:736 states it flatly, while the new
      ttyFixtureReactionGaps["agy"] added in this same window records it as "never
      sent". The BR-14 sweep corrected the Muse comment and left both Agy ones — the
      instance, not the class (ARCH-PURPOSE). The new check cannot detect this: it
      only requires an entry to exist, never that prose agrees with it.
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      Doc comment opens with drivenReturnScenario, a symbol that does not exist
    detail: |
      4th in family. RULE, generalized from the paths this round enforced: a
      comment's leading identifier is a claim the tree can check, and a doc comment
      must name the declaration it precedes. harness_tty_fixture_test.go:196 opens
      "drivenReturnScenario" for func drivenReturnOnOpenGateScreen. Measured
      prevalence in this package: 2 — this one, and wrap.go:1768 where
      observationProfile was inserted underneath checkOverlayOpen's doc comment
      (799fb6c3, #184). A go/ast sibling of TestTTYFixtureReferencesResolve over doc
      groups retires the family; fixing these two sites does not.
```

---

## Re-review — 2026-09-16T12:55:33-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 266 — muse Alt+Return from draft stays in composer; agent pane Return should be Send |
| repo | pair |
| issue file | workshop/issues/000266-muse-alt-return-from-draft-stays-in-composer-agent-pane-return-should-be-send.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6edff3386b9269d3327242898932477a75601af..ea415e980673a0bea767d42d462f7134b7804745 |
| command | sdlc close --issue 266 |
| reviewer | claude |
| timestamp | 2026-09-16T12:55:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shipped behavior is delivered and coherent: Muse's profile now maps plain Return to its native Shift+Return (`ESC[13;2u`) and Alt+Return to bare CR, the draft path settles after *every* body write, the unsafe in-paste submit interception is gone for good, and one `musePromptGlyphs` authority feeds both the Return gate and orientation. I ran the pinned range's tests: `go build ./...` and `go vet ./cmd/internal/wrapcmd` clean, `lua nvim/draft_send_test.lua` passes, and every Muse/translate/fixture/doc-comment test passes (`TestHarnessTTYFixtureConformance` 18.9s, green). The remaining package failures are all environmental in this sandbox (pty child `start … operation not permitted`, `mkdir /tmp/pair255-notify-*`), not from this diff. Three of the four open findings are genuinely addressed — I verified the new oracles by mutation rather than by reading them, and both expiry branches and both doc-comment branches fire. What blocks a clean SHIP is that BR-17's class was not swept (a third site still asserts the Agy reaction as a checked fact while the new ledger records it as undriven), plus one soundness hole the BR-15 fix introduced in its sibling ledger: `discriminating` is a bare declared boolean, so flagging a screen the gate stays *open* on retires a discrimination gap.

## 1. Strengths

- **The oracles are reachable, not decorative.** I reverted each new guard in a scratch copy: dropping `ttyFixtureReactionGaps["codex"]` yields `codex remaps Return with no driven Return and no ttyFixtureReactionGaps entry` (`harness_tty_fixture_test.go:168`); setting `pressesReturn: true` yields the expiry error at `:166`; and planting both a renamed and a stranded doc comment yields exactly the two `TestDocCommentsNameTheirSubject` branches (`:518`, `:521`). That is the class fix BR-18 asked for, and it is measurably narrow — it flags 2 sites, not the 17 a stricter rule would.
- **Dropping the `composer.raw` exemption made the honest state visible.** `harness_tty_fixture_test.go:159` now iterates `required`, so all four harnesses need an entry and all four have one. "No Return has ever been pressed on any captured screen" is an uncomfortable fact that the previous oracle structurally could not surface.
- **BR-16's triage is correct and slightly over-delivered.** `translate_test.go:56-72` restores the post-`bpEnd` chord for both protocols plus a trailing-text variant, with names that say where the chord actually is. Verified against `41812c20`: of the three deleted rows only one was the surviving path, and the replacement covers it for both protocols.
- **`assertKittyKeyboardPrecondition` checks the one thing the profile can't self-confirm** (`harness_tty_fixture_test.go:717`), and it runs in both the frozen replay and the live path (`harness_tty_live_test.go:655`). Both muse captures pass; the disambiguate-bit requirement is the right sharpening.
- **The in-paste removal is the right ARCH-SECURE/ARCH-ORDER call and is now pinned on both halves** — `TestMuseDraftBodyPasteStaysLiteral` on the profile the regression was found on, plus three `translate_test` rows. `ObservationUserSubmission` is no longer published for a non-submit.

## 2. Critical findings

None.

## 3. Important findings

**`discriminating` retires a gap without any conjunct the data model already offers** — `cmd/internal/wrapcmd/harness_tty_fixture_test.go:240` (`harnessHasDiscriminatingNegative`), field at `harness_tty_live_test.go:685`.

This is the **5th finding in family `boundary-semantics-unverified`.** Do not fix the instance — state and apply the rule.

The rule: *when a declared retirement property has components the existing data model can check, the oracle must conjoin them; only the genuinely uncheckable residue may be honor-system, and the field doc must say which part that is.* `ttyFixtureDiscriminationGaps` acknowledges the absence of **a declining screen painted in the composer's own shape**. The oracle reads only `scenario.discriminating`. Two components are already expressible and unused: `!scenario.wantComposer` (it must *decline*) and "`scenario.file` is a captured fixture" (it must exist in the tree). Demonstrated: setting `discriminating: true` on muse's `slash menu` scenario — `wantComposer: true`, a screen the gate deliberately stays **open** on, categorically not a negative — makes the test demand muse's ledger entry be dropped (`harness_tty_fixture_test.go:181`). That is the exact silent-retirement failure BR-15 named, reintroduced in the sibling ledger by BR-15's own fix. Note the asymmetry the sweep left: `negatives[agent]` is anchored to `rawFiles["overlay.raw"]` existing (`:95`), so the negative ledger is exact; the other two now rest on booleans. `pressesReturn` has no checkable component, which is fine — but its doc should say that nothing verifies a scenario setting it actually asserts a reaction.

**BR-17 re-raised as not-addressed** — see the dispositions below; the class was not enumerated.

## 4. Minor findings

- **`atlas/how-to-bring-up-a-new-harness-cli.md:66`** still reads "`composer.raw` is required and must remap to LF" while `:29` was corrected in this same window. **5th in family `stale-comment`** — the rule: the per-harness Return-newline mapping has one authority (`harnessTTYProfiles`) and is restated in prose in four places (`atlas/architecture.md:707`, `how-to:29`, `how-to:66`, `README.md:120`); the tree now checks *code* comments (`TestTTYFixtureReferencesResolve`, `TestDocCommentsNameTheirSubject`) but nothing checks markdown. Fix the class by stopping the per-harness enumeration in the how-to (point at the profile), not by patching line 66. Measured prevalence: 4 prose restatements, 1 currently wrong (and it was already wrong for Claude before this window).
- **The three new Muse captures are the only fixtures in the tree that embed the capture machine's filesystem** — `testdata/tty/muse/1.3.0-R3233.1/{composer,menu,shortcuts}.raw` each contain Muse's `warning: rules file at /Users/xianxu/workspace/pair/CLAUDE.md is ignored`; the other 11 fixtures have 0 occurrences. **2nd in family `fixture-provenance`** — the rule: *a frozen capture must not embed capture-environment-specific content, because a recapture elsewhere then differs for reasons unrelated to harness drift, which is the one thing the fixture exists to detect.* `readHarnessTTYFixture` (`harness_tty_fixture_test.go:259`) is the natural place to assert it (no raw file contains `os.UserHomeDir()` or a `/Users/<name>/` path), with capture driven from a neutral directory. Not a credential leak — `/Users/xianxu` already appears in 98 tracked files.
- **Fixture version dirs have no retirement rule; the reader is capped and the writer isn't** (ARCH-FUNERAL). `harnessTTYExhaustiveSplitBytes`/`harnessTTYSplitStride` bound the replay cost *per file*, but nothing bounds how many version dirs × screens accumulate, and every one replays on every `go test`. Measured: removing this window's three new files takes `TestHarnessTTYFixtureConformance` from 18.9s to 13.0s (+45% for one capture set). Muse now keeps two version dirs and codex two. One line in the atlas conformance section naming the retention rule (newest per harness, plus any older one a test names by path) would close it.
- **`translate_test.go:91` "Alt+Enter chord split across a paste boundary stays literal" does not split the chord** — `startPase: true` with `in: "body\x1b[13;3u\x1b[201~"` feeds a *complete* chord inside a paste that began in an earlier chunk. **4th in family `regression-evidence-missing`** — the rule: *a test-case name that asserts the shape of its input is a claim; the chunk-boundary cases are exactly the ones the removed holdback used to handle, so the split must be fed, not named.* The genuinely uncovered row is a chunk ending mid-chord (`in: "body\x1b[13;3"`, `wantHold` empty, `wantPaste: true`) — with the Alt holdback removed, nothing pins that those bytes now flow through untouched.

## 5. Test coverage notes

Coverage for the shipped behavior is good and mostly anchored to production entities rather than restatements: `TestMuseComposerActive_RelaxedPrompt` iterates `musePromptGlyphs` itself, `TestMusePromptAuthorityIsShared` pins the two gates against each other, `TestComposerReturnExpectationMatchesProfile` reads the profile, and `TestMuseDraftAltEnterSubmission` keys its expectation off a table field (BR-10). The gap that remains after this round is not in the diff's own logic but in the *evidence* layer: `pressesReturn` is set nowhere, so the reaction-retirement branch has no fixture that enters it in the committed tree (I had to synthesize one), and no test pins the ledger oracles themselves — a future edit that loosens `harnessPressesReturn` or `harnessHasDiscriminatingNegative` goes green. A small table-driven test over synthetic scenario/ledger pairs would pin all three ledgers' both-direction behavior without needing a live harness.

One environment note for the close: I could not run the full `make test` here — the sandbox refuses pty-child spawn and `/tmp` mkdir, which accounts for every remaining red test in `./cmd/internal/wrapcmd` (all `operation not permitted`). Per `workshop/lessons.md` this class needs the retention-owner env scrubbed and a non-symlinked `TMPDIR`; that run is the operator's.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass on code (`musePromptGlyphs` is one authority, pinned). Flagged in prose only (Minor above).
- **ARCH-PURE** — pass. Recognizers stay pure functions over `terminalSnapshot`; `orientationPromptOK` is a pure helper; the new oracles are pure over tables and the AST.
- **ARCH-PURPOSE** — pass on the shipped behavior (both entry points, operator-confirmed live). Flagged on BR-17: instance fixed, enumerable sibling left.
- **ARCH-MOCK** — mostly pass; the fixture seam plus three gap ledgers *is* the evidence model, and live conformance is the (documented, manual) drift check. Flagged: the `discriminating` oracle now lets the model claim coverage the artifacts don't back.
- **ARCH-CONSTRAINTS** — flagged: frozen-replay cost +45%, capped per file but not per capture. The 100 ms per-send settle stays bounded by #269, as agreed in round 1.
- **ARCH-SECURE** — the in-paste removal is the right call: a paste payload can no longer be reinterpreted as a trusted submit. Flagged only on capture-environment content.
- **ARCH-ORDER** — pass. The paste state machine reduces to `inPaste → bpEnd`, the non-behavior is pinned in two places across four historical flips, and no submission observation is published for a non-submit.
- **ARCH-FUNERAL** — flagged (fixture family, above). Worth noting for #269: whatever replaces the fixed settle should not add a per-send artifact without naming its removal.

## 7. Plan revision recommendations

Two `## Plan` items are still checked and describe code that HEAD does not contain. The `## Revisions` entries do record the reversal, so the artifact is self-correcting for a careful reader — but the bullets name a grep-able test that no longer exists:

- `- [x] Fix paste-aware Alt handling: wrap.go:translateChunk now scans for Alt+Enter … before pasteEnd when inPaste, emitting an unconditional \r submit even inside the paste window …` — that branch was reverted; the confirmed fix is the 100 ms settle plus the pinned non-behavior.
- `- [x] Add paste tests: … muse_draft_submit_test.go:TestMuseDraftAltEnterSubmission_InsidePaste …` — that symbol does not exist; the surviving test is `TestMuseDraftBodyPasteStaysLiteral`.

Add a `## Revisions` entry (or amend those two bullets in place with a pointer) stating the delta: the in-paste interception is removed, the non-behavior is pinned in `translate_test.go` + `TestMuseDraftBodyPasteStaysLiteral`, and the post-`bpEnd` positive half is pinned by the three restored rows. Also worth one line: the round-3 Log entry says the revert "took three `translate_test` rows with it that pinned the *surviving* path" — only one of the three did (verified against `41812c20`); the commit message states this correctly, the Log does not.

```findings
dispose:
  - id: BR-15
    disposition: addressed
    note: |
      Both parts fixed and verified by mutation, not by reading: the `\r`-in-send inference is gone (pressesReturn is declared at harness_tty_live_test.go:685), the composer.raw exemption is gone (the loop at harness_tty_fixture_test.go:159 iterates `required`, and codex gained the entry that exposes), and all three ledgers now carry both-direction expiry plus the profile-existence check. Setting pressesReturn:true fires :166; the discrimination expiry fires :181. See the new Important finding for the conjunct the discrimination half still lacks.
  - id: BR-16
    disposition: addressed
    note: |
      translate_test.go:56-72 restores the post-bpEnd chord for both protocols plus a trailing-text variant, with names that place the chord correctly. Verified against 41812c20 that only one of the three deleted rows was the surviving path, so the restoration covers it for both protocols.
  - id: BR-17
    disposition: not-addressed
    note: |
      The two named sites now cite the ledger, but the class was not enumerated: composer_recognizers.go:280-281 still states "Tolerable because Agy inserts a newline on LF there rather than selecting" as the checked tolerability fact. `grep -n "inserts a newline" cmd/internal/wrapcmd/*.go` finds it in one line. harness_tty.go:55 is a borderline sibling (Muse's reaction asserted flatly, though shortcuts.raw is documentary basis) — the enumeration should rule on it explicitly.
  - id: BR-18
    disposition: addressed
    note: |
      TestDocCommentsNameTheirSubject retires the family mechanically, and both branches fire under mutation (planted rename -> :518, stranded comment -> :521). Both instances fixed (harnessPressesReturn doc, and checkOverlayOpen's comment moved back off observationProfile in wrap.go).
findings:
  - id: new
    severity: Important
    family: boundary-semantics-unverified
    title: |
      discriminating retires a gap with no conjunct tying it to a declining screen or a captured file
    detail: |
      5th in family. RULE: when a declared retirement property has components the
      data model can already check, the oracle must conjoin them; only the
      genuinely uncheckable residue may be honor-system, and the field doc must
      say which part that is. harnessHasDiscriminatingNegative
      (harness_tty_fixture_test.go:240) reads only scenario.discriminating, while
      the gap it retires names "a DECLINING screen painted in the composer's own
      shape". Two components are expressible and unused: !scenario.wantComposer,
      and scenario.file existing as a capture. Demonstrated: flagging muse's
      `slash menu` scenario (wantComposer:true, a screen the gate stays OPEN on)
      makes the test demand muse's ledger entry be dropped (:181) — the exact
      silent retirement BR-15 named, reintroduced by BR-15's own fix in the
      sibling ledger. Note the sweep's asymmetry: negatives[] is anchored to
      rawFiles["overlay.raw"] existing (:95) and is therefore exact; the other two
      now rest on booleans. pressesReturn has no checkable component, which is
      fine, but its doc should say nothing verifies that a scenario setting it
      actually asserts a reaction. (ARCH-MOCK, ARCH-PURPOSE)
  - id: new
    severity: Minor
    family: stale-comment
    title: |
      atlas how-to still says composer.raw must remap to LF, 37 lines from the line this window fixed
    detail: |
      5th in family. RULE: the per-harness Return-newline mapping has one
      authority (harnessTTYProfiles) and is restated in prose in four places —
      atlas/architecture.md:707, how-to:29, how-to:66, README.md:120. This window
      corrected :29 and left :66 ("composer.raw is required and must remap to
      LF"), which is false for Muse now and was already false for Claude. The
      tree now checks code comments (TestTTYFixtureReferencesResolve,
      TestDocCommentsNameTheirSubject) but nothing checks markdown, so the class
      fix is to stop enumerating per-harness mappings in the how-to and point at
      the profile — not to patch line 66. Measured prevalence: 4 restatements, 1
      wrong. (ARCH-DRY)
  - id: new
    severity: Minor
    family: fixture-provenance
    title: |
      The three new Muse captures are the only fixtures embedding the capture machine's filesystem
    detail: |
      2nd in family. RULE: a frozen capture must not embed
      capture-environment-specific content, because a recapture elsewhere then
      differs for reasons unrelated to harness drift — the one thing the fixture
      exists to detect. All three of testdata/tty/muse/1.3.0-R3233.1/{composer,
      menu,shortcuts}.raw contain Muse's "warning: rules file at
      /Users/xianxu/workspace/pair/CLAUDE.md is ignored"; the other 11 fixtures
      have 0 occurrences, so the driven scenarios' Dir:repoRoot made these the
      first machine-bound captures. Mechanize in readHarnessTTYFixture
      (harness_tty_fixture_test.go:259): no raw file may contain
      os.UserHomeDir() or a /Users/<name>/ path. Not a credential leak —
      /Users/xianxu already appears in 98 tracked files. (ARCH-SECURE)
  - id: new
    severity: Minor
    family: unbounded-artifact-family
    title: |
      Fixture version dirs have no retirement rule; the replay reader is capped and the writer is not
    detail: |
      harnessTTYExhaustiveSplitBytes and harnessTTYSplitStride bound replay cost
      per file, but nothing bounds how many version directories and screens
      accumulate, and every one replays on every `go test`. Measured: removing
      this window's three new files takes TestHarnessTTYFixtureConformance from
      18.9s to 13.0s, so one capture set cost +45%. Muse now keeps two version
      dirs and codex two, and this window's own deletion of R3057.1 was ad hoc.
      One line in the atlas conformance section naming the retention rule
      (newest per harness, plus any older one a test names by path) closes it.
      (ARCH-FUNERAL, ARCH-CONSTRAINTS)
  - id: new
    severity: Minor
    family: regression-evidence-missing
    title: |
      The "chord split across a paste boundary" row does not split the chord
    detail: |
      4th in family. RULE: a test-case name that asserts the shape of its input is
      a claim, and the chunk-boundary cases are exactly the ones the removed Alt
      holdback used to handle — so the split must be fed, not named.
      translate_test.go:91 feeds startPase:true with a COMPLETE chord
      ("body\x1b[13;3u\x1b[201~"); the paste spans the boundary, the chord does
      not. The uncovered row is a chunk ending mid-chord ("body\x1b[13;3",
      wantHold empty, wantPaste true), which is what pins that those bytes now
      flow through untouched after the holdback's removal. Same misnaming class
      BR-16 corrected in the sibling rows.
```
