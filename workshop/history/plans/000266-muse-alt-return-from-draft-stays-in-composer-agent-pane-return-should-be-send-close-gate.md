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
    - "n": 2
      timestamp: "2026-09-16T12:05:07-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: In-paste branch removed; wrap.go:1942-1956 scans only bpEnd, no altCR or ObservationUserSubmission is reachable inside a paste.
          round: 2
        - id: BR-2
          disposition: addressed
          note: assertKittyKeyboardPrecondition checks the capture (not the profile) in both the frozen replay and the live path; both muse fixtures carry a real CSI > u push.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Selection markers dropped and pinned by TestMuseComposerActive_RejectsSelectionMarkers; slash menu driven live and captured as muse/1.3.0-R3233.1/menu.raw.
          round: 2
        - id: BR-4
          disposition: addressed
          note: One musePromptGlyphs authority read by both gates, pinned by TestMusePromptAuthorityIsShared.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The untested holdback is gone with the branch it served.
          round: 2
        - id: BR-6
          disposition: addressed
          note: harness_tty_fixture_test.go:418 and nvim/init.lua:733 both corrected; a new instance appeared elsewhere, raised separately under the family rule.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Test removed with the branch; the surviving cases carry rationale comments.
          round: 2
        - id: BR-8
          disposition: addressed
          note: Single indexOfSubseq call at wrap.go:1945.
          round: 2
        - id: BR-9
          disposition: addressed
          note: R3057.1 deleted; muse/1.3.0-R3233.1/metadata.json records captured_at 2026-09-16T18:44:17Z, a real capture second.
          round: 2
        - id: BR-10
          disposition: addressed
          note: muse_draft_submit_test.go:19 keys composer state and expected bytes off table fields.
          round: 2
        - id: BR-11
          disposition: addressed
          note: Filed as pair#269 with the defect stated correctly (budget from one measurement, no detector) and the phase machine named as the seam; settle retained as the confirmed fix.
          round: 2
      findings:
        - id: BR-12
          severity: Important
          title: No test pins that an Alt+Enter chord inside a bracketed paste stays literal
          detail: |-
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
          family: regression-evidence-missing
          round: 2
        - id: BR-13
          severity: Minor
          title: Comment cites a fixture directory this same window deleted
          detail: |-
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
          family: stale-comment
          round: 2
        - id: BR-14
          severity: Minor
          title: KKP push guard is flags-agnostic and the menu's Return behavior is asserted from reasoning
          detail: |-
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
          family: boundary-semantics-unverified
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-16T12:29:05-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: addressed
          note: 'Verified by reapplying 41812c20^''s wrap.go in a scratch copy of HEAD: 5 subtests go red (translate_test.go x3, TestMuseDraftBodyPasteStaysLiteral x2).'
          round: 3
        - id: BR-13
          disposition: addressed
          note: TestTTYFixtureReferencesResolve walks every .go incl. comments and self-disables loudly; verified red on a planted testdata/tty/muse/9.9.9-nope path. Both cited instances corrected; the `?` sheet is now a registered driven scenario with shortcuts.raw captured.
          round: 3
        - id: BR-14
          disposition: addressed
          note: ttyFixtureReactionGaps exists and is enforced, and the KKP guard now requires the disambiguate bit (muse fixtures push >3u/>1u, so the assert runs on real bytes). Residual soundness gaps in the new mechanism raised fresh as I-1, not as a re-raise.
          round: 3
      findings:
        - id: BR-15
          severity: Important
          title: Reaction-gap retirement oracle cannot prove what it names, and exempts composer.raw
          detail: |-
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
          family: boundary-semantics-unverified
          round: 3
        - id: BR-16
          severity: Important
          title: Deleting the in-paste branch took three rows that pinned a surviving behavior
          detail: |-
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
          family: regression-evidence-missing
          round: 3
        - id: BR-17
          severity: Minor
          title: Agy gap status is stated in two places in one file, contradictorily
          detail: |-
            3rd in family. RULE: the gap ledger is the single authority for whether a
            harness-reaction claim is verified; comments may cite it, never restate a
            verdict about it. harness_tty_fixture_test.go:320 says the Agy newline-on-LF
            behavior is "pinned so that stays a checked property" and
            harness_tty_live_test.go:736 states it flatly, while the new
            ttyFixtureReactionGaps["agy"] added in this same window records it as "never
            sent". The BR-14 sweep corrected the Muse comment and left both Agy ones — the
            instance, not the class (ARCH-PURPOSE). The new check cannot detect this: it
            only requires an entry to exist, never that prose agrees with it.
          family: duplicated-authority
          round: 3
        - id: BR-18
          severity: Minor
          title: Doc comment opens with drivenReturnScenario, a symbol that does not exist
          detail: |-
            4th in family. RULE, generalized from the paths this round enforced: a
            comment's leading identifier is a claim the tree can check, and a doc comment
            must name the declaration it precedes. harness_tty_fixture_test.go:196 opens
            "drivenReturnScenario" for func drivenReturnOnOpenGateScreen. Measured
            prevalence in this package: 2 — this one, and wrap.go:1768 where
            observationProfile was inserted underneath checkOverlayOpen's doc comment
            (799fb6c3, #184). A go/ast sibling of TestTTYFixtureReferencesResolve over doc
            groups retires the family; fixing these two sites does not.
          family: stale-comment
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-16T12:55:34-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: addressed
          note: 'Both parts fixed and verified by mutation, not by reading: the `\r`-in-send inference is gone (pressesReturn is declared at harness_tty_live_test.go:685), the composer.raw exemption is gone (the loop at harness_tty_fixture_test.go:159 iterates `required`, and codex gained the entry that exposes), and all three ledgers now carry both-direction expiry plus the profile-existence check. Setting pressesReturn:true fires :166; the discrimination expiry fires :181. See the new Important finding for the conjunct the discrimination half still lacks.'
          round: 4
        - id: BR-16
          disposition: addressed
          note: translate_test.go:56-72 restores the post-bpEnd chord for both protocols plus a trailing-text variant, with names that place the chord correctly. Verified against 41812c20 that only one of the three deleted rows was the surviving path, so the restoration covers it for both protocols.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: 'The two named sites now cite the ledger, but the class was not enumerated: composer_recognizers.go:280-281 still states "Tolerable because Agy inserts a newline on LF there rather than selecting" as the checked tolerability fact. `grep -n "inserts a newline" cmd/internal/wrapcmd/*.go` finds it in one line. harness_tty.go:55 is a borderline sibling (Muse''s reaction asserted flatly, though shortcuts.raw is documentary basis) — the enumeration should rule on it explicitly.'
          round: 4
        - id: BR-18
          disposition: addressed
          note: TestDocCommentsNameTheirSubject retires the family mechanically, and both branches fire under mutation (planted rename -> :518, stranded comment -> :521). Both instances fixed (harnessPressesReturn doc, and checkOverlayOpen's comment moved back off observationProfile in wrap.go).
          round: 4
      findings:
        - id: BR-19
          severity: Important
          title: discriminating retires a gap with no conjunct tying it to a declining screen or a captured file
          detail: |-
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
          family: boundary-semantics-unverified
          round: 4
        - id: BR-20
          severity: Minor
          title: atlas how-to still says composer.raw must remap to LF, 37 lines from the line this window fixed
          detail: |-
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
          family: stale-comment
          round: 4
        - id: BR-21
          severity: Minor
          title: The three new Muse captures are the only fixtures embedding the capture machine's filesystem
          detail: |-
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
          family: fixture-provenance
          round: 4
        - id: BR-22
          severity: Minor
          title: Fixture version dirs have no retirement rule; the replay reader is capped and the writer is not
          detail: |-
            harnessTTYExhaustiveSplitBytes and harnessTTYSplitStride bound replay cost
            per file, but nothing bounds how many version directories and screens
            accumulate, and every one replays on every `go test`. Measured: removing
            this window's three new files takes TestHarnessTTYFixtureConformance from
            18.9s to 13.0s, so one capture set cost +45%. Muse now keeps two version
            dirs and codex two, and this window's own deletion of R3057.1 was ad hoc.
            One line in the atlas conformance section naming the retention rule
            (newest per harness, plus any older one a test names by path) closes it.
            (ARCH-FUNERAL, ARCH-CONSTRAINTS)
          family: unbounded-artifact-family
          round: 4
        - id: BR-23
          severity: Minor
          title: The "chord split across a paste boundary" row does not split the chord
          detail: |-
            4th in family. RULE: a test-case name that asserts the shape of its input is
            a claim, and the chunk-boundary cases are exactly the ones the removed Alt
            holdback used to handle — so the split must be fed, not named.
            translate_test.go:91 feeds startPase:true with a COMPLETE chord
            ("body\x1b[13;3u\x1b[201~"); the paste spans the boundary, the chord does
            not. The uncovered row is a chunk ending mid-chord ("body\x1b[13;3",
            wantHold empty, wantPaste true), which is what pins that those bytes now
            flow through untouched after the holdback's removal. Same misnaming class
            BR-16 corrected in the sibling rows.
          family: regression-evidence-missing
          round: 4
      blocked: false
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

## Round 2 — 2026-09-16T12:05:07-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — In-paste branch removed; wrap.go:1942-1956 scans only bpEnd, no altCR or ObservationUserSubmission is reachable inside a paste.
- BR-2 — addressed — assertKittyKeyboardPrecondition checks the capture (not the profile) in both the frozen replay and the live path; both muse fixtures carry a real CSI > u push.
- BR-3 — addressed — Selection markers dropped and pinned by TestMuseComposerActive_RejectsSelectionMarkers; slash menu driven live and captured as muse/1.3.0-R3233.1/menu.raw.
- BR-4 — addressed — One musePromptGlyphs authority read by both gates, pinned by TestMusePromptAuthorityIsShared.
- BR-5 — addressed — The untested holdback is gone with the branch it served.
- BR-6 — addressed — harness_tty_fixture_test.go:418 and nvim/init.lua:733 both corrected; a new instance appeared elsewhere, raised separately under the family rule.
- BR-7 — addressed — Test removed with the branch; the surviving cases carry rationale comments.
- BR-8 — addressed — Single indexOfSubseq call at wrap.go:1945.
- BR-9 — addressed — R3057.1 deleted; muse/1.3.0-R3233.1/metadata.json records captured_at 2026-09-16T18:44:17Z, a real capture second.
- BR-10 — addressed — muse_draft_submit_test.go:19 keys composer state and expected bytes off table fields.
- BR-11 — addressed — Filed as pair#269 with the defect stated correctly (budget from one measurement, no detector) and the phase machine named as the seam; settle retained as the confirmed fix.

### Raised

- **BR-12** [Important] `regression-evidence-missing` No test pins that an Alt+Enter chord inside a bracketed paste stays literal
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
- **BR-13** [Minor] `stale-comment` Comment cites a fixture directory this same window deleted
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
- **BR-14** [Minor] `boundary-semantics-unverified` KKP push guard is flags-agnostic and the menu's Return behavior is asserted from reasoning
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

## Round 3 — 2026-09-16T12:29:05-07:00 (claude) — BLOCKED

### Disposed

- BR-12 — addressed — Verified by reapplying 41812c20^'s wrap.go in a scratch copy of HEAD: 5 subtests go red (translate_test.go x3, TestMuseDraftBodyPasteStaysLiteral x2).
- BR-13 — addressed — TestTTYFixtureReferencesResolve walks every .go incl. comments and self-disables loudly; verified red on a planted testdata/tty/muse/9.9.9-nope path. Both cited instances corrected; the `?` sheet is now a registered driven scenario with shortcuts.raw captured.
- BR-14 — addressed — ttyFixtureReactionGaps exists and is enforced, and the KKP guard now requires the disambiguate bit (muse fixtures push >3u/>1u, so the assert runs on real bytes). Residual soundness gaps in the new mechanism raised fresh as I-1, not as a re-raise.

### Raised

- **BR-15** [Important] `boundary-semantics-unverified` Reaction-gap retirement oracle cannot prove what it names, and exempts composer.raw
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
- **BR-16** [Important] `regression-evidence-missing` Deleting the in-paste branch took three rows that pinned a surviving behavior
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
- **BR-17** [Minor] `duplicated-authority` Agy gap status is stated in two places in one file, contradictorily
  3rd in family. RULE: the gap ledger is the single authority for whether a
  harness-reaction claim is verified; comments may cite it, never restate a
  verdict about it. harness_tty_fixture_test.go:320 says the Agy newline-on-LF
  behavior is "pinned so that stays a checked property" and
  harness_tty_live_test.go:736 states it flatly, while the new
  ttyFixtureReactionGaps["agy"] added in this same window records it as "never
  sent". The BR-14 sweep corrected the Muse comment and left both Agy ones — the
  instance, not the class (ARCH-PURPOSE). The new check cannot detect this: it
  only requires an entry to exist, never that prose agrees with it.
- **BR-18** [Minor] `stale-comment` Doc comment opens with drivenReturnScenario, a symbol that does not exist
  4th in family. RULE, generalized from the paths this round enforced: a
  comment's leading identifier is a claim the tree can check, and a doc comment
  must name the declaration it precedes. harness_tty_fixture_test.go:196 opens
  "drivenReturnScenario" for func drivenReturnOnOpenGateScreen. Measured
  prevalence in this package: 2 — this one, and wrap.go:1768 where
  observationProfile was inserted underneath checkOverlayOpen's doc comment
  (799fb6c3, #184). A go/ast sibling of TestTTYFixtureReferencesResolve over doc
  groups retires the family; fixing these two sites does not.

## Round 4 — 2026-09-16T12:55:34-07:00 (claude) — passed

### Disposed

- BR-15 — addressed — Both parts fixed and verified by mutation, not by reading: the `\r`-in-send inference is gone (pressesReturn is declared at harness_tty_live_test.go:685), the composer.raw exemption is gone (the loop at harness_tty_fixture_test.go:159 iterates `required`, and codex gained the entry that exposes), and all three ledgers now carry both-direction expiry plus the profile-existence check. Setting pressesReturn:true fires :166; the discrimination expiry fires :181. See the new Important finding for the conjunct the discrimination half still lacks.
- BR-16 — addressed — translate_test.go:56-72 restores the post-bpEnd chord for both protocols plus a trailing-text variant, with names that place the chord correctly. Verified against 41812c20 that only one of the three deleted rows was the surviving path, so the restoration covers it for both protocols.
- BR-17 — not-addressed — The two named sites now cite the ledger, but the class was not enumerated: composer_recognizers.go:280-281 still states "Tolerable because Agy inserts a newline on LF there rather than selecting" as the checked tolerability fact. `grep -n "inserts a newline" cmd/internal/wrapcmd/*.go` finds it in one line. harness_tty.go:55 is a borderline sibling (Muse's reaction asserted flatly, though shortcuts.raw is documentary basis) — the enumeration should rule on it explicitly.
- BR-18 — addressed — TestDocCommentsNameTheirSubject retires the family mechanically, and both branches fire under mutation (planted rename -> :518, stranded comment -> :521). Both instances fixed (harnessPressesReturn doc, and checkOverlayOpen's comment moved back off observationProfile in wrap.go).

### Raised

- **BR-19** [Important] `boundary-semantics-unverified` discriminating retires a gap with no conjunct tying it to a declining screen or a captured file
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
- **BR-20** [Minor] `stale-comment` atlas how-to still says composer.raw must remap to LF, 37 lines from the line this window fixed
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
- **BR-21** [Minor] `fixture-provenance` The three new Muse captures are the only fixtures embedding the capture machine's filesystem
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
- **BR-22** [Minor] `unbounded-artifact-family` Fixture version dirs have no retirement rule; the replay reader is capped and the writer is not
  harnessTTYExhaustiveSplitBytes and harnessTTYSplitStride bound replay cost
  per file, but nothing bounds how many version directories and screens
  accumulate, and every one replays on every `go test`. Measured: removing
  this window's three new files takes TestHarnessTTYFixtureConformance from
  18.9s to 13.0s, so one capture set cost +45%. Muse now keeps two version
  dirs and codex two, and this window's own deletion of R3057.1 was ad hoc.
  One line in the atlas conformance section naming the retention rule
  (newest per harness, plus any older one a test names by path) closes it.
  (ARCH-FUNERAL, ARCH-CONSTRAINTS)
- **BR-23** [Minor] `regression-evidence-missing` The "chord split across a paste boundary" row does not split the chord
  4th in family. RULE: a test-case name that asserts the shape of its input is
  a claim, and the chunk-boundary cases are exactly the ones the removed Alt
  holdback used to handle — so the split must be fed, not named.
  translate_test.go:91 feeds startPase:true with a COMPLETE chord
  ("body\x1b[13;3u\x1b[201~"); the paste spans the boundary, the chord does
  not. The uncovered row is a chunk ending mid-chord ("body\x1b[13;3",
  wantHold empty, wantPaste true), which is what pins that those bytes now
  flow through untouched after the holdback's removal. Same misnaming class
  BR-16 corrected in the sibling rows.

## Open findings

- **BR-17** [Minor] `duplicated-authority` Agy gap status is stated in two places in one file, contradictorily
- **BR-19** [Important] `boundary-semantics-unverified` discriminating retires a gap with no conjunct tying it to a declining screen or a captured file
- **BR-20** [Minor] `stale-comment` atlas how-to still says composer.raw must remap to LF, 37 lines from the line this window fixed
- **BR-21** [Minor] `fixture-provenance` The three new Muse captures are the only fixtures embedding the capture machine's filesystem
- **BR-22** [Minor] `unbounded-artifact-family` Fixture version dirs have no retirement rule; the replay reader is capped and the writer is not
- **BR-23** [Minor] `regression-evidence-missing` The "chord split across a paste boundary" row does not split the chord
