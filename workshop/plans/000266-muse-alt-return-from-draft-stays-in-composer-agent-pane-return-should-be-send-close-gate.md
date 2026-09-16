---
gate: boundary-review
issue: 266
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-16T11:34:37-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: In-paste submit emits CR inside the still-open bracketed-paste window
          detail: |-
            wrap.go:1960 emits keymap.altCR before bpEnd and leaves inPaste true, so the
            harness receives \x1b[200~body\r\x1b[201~. Muse enables ?2004h (verified in the
            1.3.0 fixture), so a CR inside the window is pasted text, not an Enter key — the
            original "draft sits unsent" symptom, now with a stray newline. Both tests assert
            only the wrapper's output bytes, and the operator's live confirmation covers the
            100ms settle, which prevents this path from firing. ObservationUserSubmission
            fires regardless, so a non-submit opens a turn and later raises a spurious
            "no agent output" alert. Emit bpEnd before altCR and consume the trailing real
            close marker, or drop the branch and keep the settle alone (ARCH-ORDER, ARCH-MOCK).
          family: boundary-semantics-unverified
          round: 1
        - id: BR-2
          severity: Important
          title: Muse plainCR uses a KKP sequence whose precondition is unrecorded and unguarded
          detail: |-
            harness_tty.go:57 sets plainCR to \x1b[13;2u, parseable only while Muse keeps
            Kitty progressive enhancement pushed. Muse 1.3.0 does push it (\x1b[>3u in the new
            fixture), so the mapping is correct today — but no comment records the dependency
            and assertHarnessTTYLiveDecision reads its expectation from the same profile, so
            the live check cannot detect the day Muse stops. Failure mode is silent: Enter does
            nothing and telemetry still reports fired. Assert a KKP push in the captured prefix
            before trusting a KKP-encoded plainCR (ARCH-MOCK, ARCH-SECURE).
          family: boundary-semantics-unverified
          round: 1
        - id: BR-3
          severity: Important
          title: Orientation now accepts menu-bullet glyphs and bang as Muse composer prompts
          detail: |-
            orientation.go:205 extends the speculative Muse glyph set to the orientation
            auto-submit gate, accepting ●, ▶, ▸, > — conventional selection markers — for the
            one harness listed in both ttyFixtureNegativeGaps and ttyFixtureDiscriminationGaps,
            while this same window removes a Muse picker text marker. Accepting "!" also
            defeats the non-coding-mode guard the commit says it preserved: the same glyph is
            rejected as content, and rejected outright for Claude. Drop "!", justify or drop
            ●/▶/▸, and capture one Muse declining state (a slash menu needs no tool call)
            (ARCH-PURPOSE).
          family: positive-gate-needs-declining-evidence
          round: 1
        - id: BR-4
          severity: Important
          title: Muse prompt-glyph set is hand-maintained in two places with nothing pinning them
          detail: |-
            composer_recognizers.go:187 and orientation.go:205 each carry the same literal
            glyph list. a239c856 removed orientation's duplicate authority citing ARCH-DRY;
            cc253e42 reintroduced it. A glyph added to one produces a state where the Return
            remap accepts a composer orientation rejects. Extract one musePromptGlyphs source
            consumed by both, keeping orientation's menu guard layered on top (ARCH-DRY).
          family: duplicated-authority
          round: 1
        - id: BR-5
          severity: Important
          title: Restored split-Alt-partial holdback inside a paste has no test
          detail: |-
            wrap.go:1978 extends the held-back tail to cover a partial Alt chord. It is
            load-bearing — for a chunk ending \x1b[1 inside a paste, bpEnd's partial is 2 bytes
            and the KKP partial is 3 — and no existing case reaches it; the in-paste partial
            test uses \x1b[20, where bpEnd already wins. Add a startPase case ending \x1b[13;3
            expecting that exact holdback.
          family: regression-evidence-missing
          round: 1
        - id: BR-6
          severity: Minor
          title: Fixture expectation comment is stale and ungrammatical after the Muse change
          detail: |-
            harness_tty_fixture_test.go:383-386 still says "Codex, Muse and Agy all remap to
            LF" (false for Muse) and the edited sentence lost its predicate. nvim/init.lua:734
            likewise enumerates claude/codex/agy newline sequences without Muse's new one, in
            the function this diff edited.
          family: stale-comment
          round: 1
        - id: BR-7
          severity: Minor
          title: Coalesced-paste test lost the doc comment explaining why the case exists
          detail: |-
            muse_draft_submit_test.go:50 — the rewrite dropped the rationale from the file's
            least self-evident test.
          family: stale-comment
          round: 1
        - id: BR-8
          severity: Minor
          title: bpEnd index recomputed one line after endIdx already holds it
          detail: |-
            wrap.go:1969 recomputes indexOfSubseq(data[i:], bpEnd) into idx; endIdx at 1947 is
            the same value.
          family: duplicated-authority
          round: 1
        - id: BR-9
          severity: Minor
          title: New Muse fixture records a hand-rounded captured_at
          detail: |-
            testdata/tty/muse/1.3.0-R3057.1/metadata.json uses 06:00:00Z; every other fixture
            records a real capture second, which is what makes the field usable as provenance.
          family: fixture-provenance
          round: 1
        - id: BR-10
          severity: Minor
          title: Muse draft submit test branches on the subtest name string
          detail: |-
            muse_draft_submit_test.go:17 and :39 key both the fixture paint and the expected
            plain-Return bytes off name == "with composer"; a rename silently flips the
            assertion to the wrong expectation. Use a table field.
          family: test-asserts-on-subtest-name
          round: 1
        - id: BR-11
          severity: Minor
          title: 100ms settle is a fixed sleep against an ordering problem, with no detector
          detail: |-
            nvim/draft_send.lua:48 now blocks every draft send for 100ms, budgeted from a
            single 18ms observation, with nothing reporting when delivery exceeds it.
            draft_send.lua already carries a phase state machine that could confirm delivery
            instead of sleeping (ARCH-CONSTRAINTS, ARCH-ORDER).
          family: timing-heuristic-without-bound
          round: 1
      blocked: true
---

# Gate ledger — pair#266 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T11:34:37-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `boundary-semantics-unverified` In-paste submit emits CR inside the still-open bracketed-paste window
  wrap.go:1960 emits keymap.altCR before bpEnd and leaves inPaste true, so the
  harness receives \x1b[200~body\r\x1b[201~. Muse enables ?2004h (verified in the
  1.3.0 fixture), so a CR inside the window is pasted text, not an Enter key — the
  original "draft sits unsent" symptom, now with a stray newline. Both tests assert
  only the wrapper's output bytes, and the operator's live confirmation covers the
  100ms settle, which prevents this path from firing. ObservationUserSubmission
  fires regardless, so a non-submit opens a turn and later raises a spurious
  "no agent output" alert. Emit bpEnd before altCR and consume the trailing real
  close marker, or drop the branch and keep the settle alone (ARCH-ORDER, ARCH-MOCK).
- **BR-2** [Important] `boundary-semantics-unverified` Muse plainCR uses a KKP sequence whose precondition is unrecorded and unguarded
  harness_tty.go:57 sets plainCR to \x1b[13;2u, parseable only while Muse keeps
  Kitty progressive enhancement pushed. Muse 1.3.0 does push it (\x1b[>3u in the new
  fixture), so the mapping is correct today — but no comment records the dependency
  and assertHarnessTTYLiveDecision reads its expectation from the same profile, so
  the live check cannot detect the day Muse stops. Failure mode is silent: Enter does
  nothing and telemetry still reports fired. Assert a KKP push in the captured prefix
  before trusting a KKP-encoded plainCR (ARCH-MOCK, ARCH-SECURE).
- **BR-3** [Important] `positive-gate-needs-declining-evidence` Orientation now accepts menu-bullet glyphs and bang as Muse composer prompts
  orientation.go:205 extends the speculative Muse glyph set to the orientation
  auto-submit gate, accepting ●, ▶, ▸, > — conventional selection markers — for the
  one harness listed in both ttyFixtureNegativeGaps and ttyFixtureDiscriminationGaps,
  while this same window removes a Muse picker text marker. Accepting "!" also
  defeats the non-coding-mode guard the commit says it preserved: the same glyph is
  rejected as content, and rejected outright for Claude. Drop "!", justify or drop
  ●/▶/▸, and capture one Muse declining state (a slash menu needs no tool call)
  (ARCH-PURPOSE).
- **BR-4** [Important] `duplicated-authority` Muse prompt-glyph set is hand-maintained in two places with nothing pinning them
  composer_recognizers.go:187 and orientation.go:205 each carry the same literal
  glyph list. a239c856 removed orientation's duplicate authority citing ARCH-DRY;
  cc253e42 reintroduced it. A glyph added to one produces a state where the Return
  remap accepts a composer orientation rejects. Extract one musePromptGlyphs source
  consumed by both, keeping orientation's menu guard layered on top (ARCH-DRY).
- **BR-5** [Important] `regression-evidence-missing` Restored split-Alt-partial holdback inside a paste has no test
  wrap.go:1978 extends the held-back tail to cover a partial Alt chord. It is
  load-bearing — for a chunk ending \x1b[1 inside a paste, bpEnd's partial is 2 bytes
  and the KKP partial is 3 — and no existing case reaches it; the in-paste partial
  test uses \x1b[20, where bpEnd already wins. Add a startPase case ending \x1b[13;3
  expecting that exact holdback.
- **BR-6** [Minor] `stale-comment` Fixture expectation comment is stale and ungrammatical after the Muse change
  harness_tty_fixture_test.go:383-386 still says "Codex, Muse and Agy all remap to
  LF" (false for Muse) and the edited sentence lost its predicate. nvim/init.lua:734
  likewise enumerates claude/codex/agy newline sequences without Muse's new one, in
  the function this diff edited.
- **BR-7** [Minor] `stale-comment` Coalesced-paste test lost the doc comment explaining why the case exists
  muse_draft_submit_test.go:50 — the rewrite dropped the rationale from the file's
  least self-evident test.
- **BR-8** [Minor] `duplicated-authority` bpEnd index recomputed one line after endIdx already holds it
  wrap.go:1969 recomputes indexOfSubseq(data[i:], bpEnd) into idx; endIdx at 1947 is
  the same value.
- **BR-9** [Minor] `fixture-provenance` New Muse fixture records a hand-rounded captured_at
  testdata/tty/muse/1.3.0-R3057.1/metadata.json uses 06:00:00Z; every other fixture
  records a real capture second, which is what makes the field usable as provenance.
- **BR-10** [Minor] `test-asserts-on-subtest-name` Muse draft submit test branches on the subtest name string
  muse_draft_submit_test.go:17 and :39 key both the fixture paint and the expected
  plain-Return bytes off name == "with composer"; a rename silently flips the
  assertion to the wrong expectation. Use a table field.
- **BR-11** [Minor] `timing-heuristic-without-bound` 100ms settle is a fixed sleep against an ordering problem, with no detector
  nvim/draft_send.lua:48 now blocks every draft send for 100ms, budgeted from a
  single 18ms observation, with nothing reporting when delivery exceeds it.
  draft_send.lua already carries a phase state machine that could confirm delivery
  instead of sleeping (ARCH-CONSTRAINTS, ARCH-ORDER).

## Open findings

- **BR-1** [Important] `boundary-semantics-unverified` In-paste submit emits CR inside the still-open bracketed-paste window
- **BR-2** [Important] `boundary-semantics-unverified` Muse plainCR uses a KKP sequence whose precondition is unrecorded and unguarded
- **BR-3** [Important] `positive-gate-needs-declining-evidence` Orientation now accepts menu-bullet glyphs and bang as Muse composer prompts
- **BR-4** [Important] `duplicated-authority` Muse prompt-glyph set is hand-maintained in two places with nothing pinning them
- **BR-5** [Important] `regression-evidence-missing` Restored split-Alt-partial holdback inside a paste has no test
- **BR-6** [Minor] `stale-comment` Fixture expectation comment is stale and ungrammatical after the Muse change
- **BR-7** [Minor] `stale-comment` Coalesced-paste test lost the doc comment explaining why the case exists
- **BR-8** [Minor] `duplicated-authority` bpEnd index recomputed one line after endIdx already holds it
- **BR-9** [Minor] `fixture-provenance` New Muse fixture records a hand-rounded captured_at
- **BR-10** [Minor] `test-asserts-on-subtest-name` Muse draft submit test branches on the subtest name string
- **BR-11** [Minor] `timing-heuristic-without-bound` 100ms settle is a fixed sleep against an ordering problem, with no detector
