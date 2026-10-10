# Boundary Review — pair#418 (whole-issue close)

| field | value |
|-------|-------|
| issue | 418 — Couch peer delivery never submits long pastes: wordwrap projection breaks at hyphens; collapsed paste never matches |
| repo | pair |
| issue file | workshop/issues/000418-couch-peer-delivery-never-submits-long-pastes-wordwrap-projection-breaks-at-hyphens-collapsed-paste-never-matches.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5b19184c667b41c8a4ae38fa7a7517202b48137..2dd7a654c9500f2fcaec96811ca7f24b403d283d |
| command | sdlc close --issue 418 |
| reviewer | claude |
| timestamp | 2026-10-09T20:26:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm finishing a read-only review of #418's commits; nothing is changed in the repo.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change fixes both causes in the Spec, and I traced each one in the code.

- **Hyphen wrapping:** Claude's composer is now projected by `peerSpaceWordwrap`, which breaks only at spaces. Codex still uses `ansi.Wordwrap`.
- **Collapsed paste:** the `[Pasted text #N +M lines]` marker is accepted only in a strict form (`peerClaudeCollapsedPaste`), which is the operator's option (a).

The Log claims an empty composer is verified before the paste. I checked the dispatcher and it is enforced: `peer_delivery.go:312` pastes only when `composer == PeerComposerEmpty`. Rendering is checked separately, after the paste sequence. Every Done-when clause has evidence: a fixture taken from the captured ariadne:2 composer, strict-marker tests, and two live Claude 2.1.296 runs recorded in the Log.

What stands between this and SHIP is one Spec deviation that nothing records. The Spec says "hard-break a token longer than the width", but the code gives up on such a word. The issue file has no `## Revisions` entry for that.

**Strengths**
- `peer_composer_test.go` `TestPeerComposerClaudeCapturedHyphenWrap` asserts that `ansi.Wordwrap` gives a *different* layout from the captured one. If the fixture ever stops telling the two apart, the test fails instead of passing for nothing.
- `peerClaudeCollapsedPaste` checks four things:
  - the regex is anchored, and `+M` must be a positive integer;
  - `M` must equal the envelope's newline count;
  - the cursor column must be exactly `2+width`;
  - the cursor row must hold `❯`.

  The tests reject a wrong count in both directions, typed text after the marker, a misplaced cursor, the marker on a second line, an image marker, and the singular "line" form.
- The Codex and Claude projections are separate cases in a `switch`, so the Codex behaviour cannot change by accident.
- The new `-peer-live-body` flag is backed by a recorded negative check: with the old projection, the live run never submitted.

**Critical findings:** none.

**Important findings**
- **Spec deviation not recorded** (`peer_composer.go` `peerSpaceWordwrap`): the Spec says to hard-break a token wider than the line. The code returns `ok=false` instead and relies on the char-wrap fallback.
  - That fallback matches only when the *whole* message char-wraps. A normal sentence holding one URL or path longer than about 90 columns matches neither projection, so it still expires unsubmitted.
  - The code comment explains why, but the issue's Spec still promises the hard break.
  - Fix: add a `## Revisions` entry that defers or drops the hard-break clause. Also file a follow-up to capture Claude's real hard-break behaviour.

**Minor findings**
- `README.md`: "Messages submit automatically once the composer shows them" overstates it. Whitespace the projection can't map and words wider than the line still expire.
- `peer_live_test.go`: the flag help says "about 600" and "about 1200" chars. The bodies are actually about 625 and 1,304, and the Log says about 630 and 1,300.
- `peerClaudeCollapsedPaste` calls `s.CellAt(0, s.Cursor.Y)` twice; bind it to a variable once.

**Test coverage**
- The pure functions are tested directly without IO: `peerSpaceWordwrap`, the marker regex, and the matcher on terminal snapshots.
- The dispatcher-level rule (empty composer before paste, then render) predates this change and is not re-tested here. The live collapsed run covers it end to end.
- No unit fixture covers a word wider than the line mixed with normal words. That would pin down the deviation above.

**Architecture**
- **ARCH-DRY: pass.** `peerSpaceWordwrap` differs from both the char-wrap builder and `ansi.Wordwrap`, so it duplicates neither.
- **ARCH-PURE: pass.** All the new logic is pure functions of a snapshot and strings. The live-test flag is test-only.
- **ARCH-PURPOSE: pass.** Both root causes are fixed, and the docs are swept consistently: the atlas, README and fixture README. The hard-break gap is a Spec clause left undone, not the issue's main purpose.

**Plan revision recommendation**
- Add to `## Revisions`: "2026-10-09 — Spec's 'hard-break a token longer than the width' not implemented: Claude's hard-break layout is uncaptured, so `peerSpaceWordwrap` declines such text and the char-wrap fallback applies. Mixed text with an over-width token still expires; tracked as follow-up."

```findings
findings:
  - id: new
    severity: Important
    family: spec-deviation-unrecorded
    title: |
      Spec promises hard-breaking over-width tokens; peerSpaceWordwrap declines them with no Revisions entry
    detail: |
      peer_composer.go peerSpaceWordwrap returns ok=false for a word wider than the line, and the char-wrap fallback only matches all-char-wrapped text. A normal sentence with one path or URL longer than about 90 columns still expires. Add a ## Revisions entry deferring the clause (and a follow-up to capture Claude's hard-break), or implement and fixture it.
  - id: new
    severity: Minor
    family: doc-claim-overstates-behavior
    title: |
      README says messages submit "once the composer shows them"; unsafe whitespace and over-width words still expire
    detail: |
      README.md line ~530. Also peer_live_test.go flag help says about 600/1200 chars; the bodies are about 625/1304 (Log says about 630/1300).
  - id: new
    severity: Minor
    family: style-nit
    title: |
      peerClaudeCollapsedPaste calls s.CellAt(0, s.Cursor.Y) twice
```

---

## Re-review — 2026-10-09T20:31:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 418 — Couch peer delivery never submits long pastes: wordwrap projection breaks at hyphens; collapsed paste never matches |
| repo | pair |
| issue file | workshop/issues/000418-couch-peer-delivery-never-submits-long-pastes-wordwrap-projection-breaks-at-hyphens-collapsed-paste-never-matches.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5b19184c667b41c8a4ae38fa7a7517202b48137..03beb29511cd4ca4dcc4501260597ad0c992a409 |
| command | sdlc close --issue 418 |
| reviewer | claude |
| timestamp | 2026-10-09T20:31:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are fixed, and I found no new Critical or Important issues. This round's main fix was BR-1: `peerSpaceWordwrap` now hard-breaks words wider than the line. I checked its tie-break rule against wrap-ansi's hard mode, including the case where the added space fills the row exactly. I also confirmed the regression evidence. Making over-width words return `"", false` through a `go test -overlay` copy turns `TestPeerComposerClaudeCapturedOverwidthWord` and four `TestPeerSpaceWordwrap` cases red. The tree was left untouched. At HEAD, the peer composer tests pass. The one new finding is Minor: three prose sites still say Claude wraps "at spaces only", which leaves out the hard-break rule this round added.

1. **Strengths**
   - `peer_composer.go` `peerSpaceWordwrap` follows wrap-ansi's hard rule closely. The `thisLine`/`nextLine` break counts include the added space, just as wrap-ansi's `rowLength` does. When the space fills the row (`column == width`), the word correctly starts on the next line.
   - `TestPeerComposerClaudeCapturedHyphenWrap` checks that its own fixture tells the two projections apart (`ansi.Wordwrap(...) == lines` → Fatal). Without that check the fixture could drift and prove nothing.
   - `peerClaudeCollapsedPaste` is strict in the right ways: the regex is anchored, the line count must equal the newline count, the cursor must sit right after the marker, and the `❯` prompt must be on the cursor row. Delivery also requires `composer == PeerComposerEmpty` before it pastes (`peer_delivery.go:311-312`), so the claim "empty before paste" holds.
   - I checked the "paste again to expand" hint against pair:1's raw scrollback. It is drawn two rows below the marker (`\e[2B`), outside the composer box, so the strict regex does not see it.
   - The Codex path is unchanged: it still uses `ansi.Wordwrap`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Prose describes Claude's wrapping as "spaces only" and leaves out the over-width hard-break.
     - **This is the 2nd finding in family `doc-claim-overstates-behavior`.**
     - **Rule:** describe Claude's wrapping in one canonical sentence — "breaks at single spaces; a word wider than the line hard-breaks at the width" — or point to `peerSpaceWordwrap`. Don't paraphrase a partial rule.
     - **Instances in this window:**
       - `atlas/couch.md` (~L267): "Claude breaks lines at spaces only"
       - `testdata/peer/claude/2.1.286/README.md` last paragraph: "Claude wraps at spaces only, never after a hyphen"
       - `peer_composer.go` comment in the `case "claude"` branch: "Claude Code breaks only at spaces"
     - The repo README already says it correctly ("so do over-long words").

5. **Test coverage notes**
   - Every Done-when clause has evidence:
     - Hyphen wrap: captured fixture plus a live run.
     - Collapsed paste: strict acceptance with five negative cases plus a live submit.
     - The ≥1,000-char live body: the collapsed variant is 1,291 chars by my measurement; the flag help says "about 1300".
   - The test case named "pair:1 capture shape" is synthesized and is not a replay of the pair:1 bytes. That's acceptable, because the real hint sits outside the composer box.

6. **Architectural notes**
   - **ARCH-DRY:** pass. Each harness gets its own projection, and nothing is duplicated.
   - **ARCH-PURE:** pass. `peerSpaceWordwrap` is a pure string function tested directly, and the marker check reads only the snapshot.
   - **ARCH-PURPOSE:** pass. Both causes in the issue (hyphen wrap and collapsed marker) are fixed, and BR-1's deferred clause is now delivered.

7. **Plan revisions:** none needed. The Plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      peerSpaceWordwrap implements wrap-ansi hard mode; overlay revert turns TestPeerComposerClaudeCapturedOverwidthWord and 4 TestPeerSpaceWordwrap cases red.
  - id: BR-2
    disposition: addressed
    note: |
      README now scopes auto-submit to verifiable layouts and says tabs/leading/trailing/repeated spaces wait for a human; flag help says about 625/1300 (measured 619/1291).
  - id: BR-3
    disposition: addressed
    note: |
      CellAt(0, s.Cursor.Y) is now called once, into prompt.
findings:
  - id: new
    severity: Minor
    family: doc-claim-overstates-behavior
    title: |
      Three prose sites say Claude wraps "at spaces only", omitting the over-width hard-break
    detail: |
      2nd in family. Rule: state Claude's wrap rule once in full (single spaces; over-width words hard-break at the width) or point to peerSpaceWordwrap. Instances: atlas/couch.md (~L267), testdata/peer/claude/2.1.286/README.md last paragraph, peer_composer.go case "claude" comment. README.md is already correct.
```

---

## Re-review — 2026-10-09T20:34:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 418 — Couch peer delivery never submits long pastes: wordwrap projection breaks at hyphens; collapsed paste never matches |
| repo | pair |
| issue file | workshop/issues/000418-couch-peer-delivery-never-submits-long-pastes-wordwrap-projection-breaks-at-hyphens-collapsed-paste-never-matches.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5b19184c667b41c8a4ae38fa7a7517202b48137..717273c735811b7160f6b569024c8ad37ce22412 |
| command | sdlc close --issue 418 |
| reviewer | claude |
| timestamp | 2026-10-09T20:34:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change is ready to ship. The diff does what the Spec and Done-when ask for:
- **Wrapping:** Claude's composer now wraps on Claude's own rule (`peerSpaceWordwrap`: break only at single spaces, and hard-break over-wide words the way wrap-ansi does). Codex keeps `ansi.Wordwrap`.
- **Collapsed pastes:** Claude's `[Pasted text #N +M lines]` marker is accepted only in a strict form, which the operator chose.

The prior round's only open finding, BR-4, is fixed at all three sites it named: the atlas, the fixture README and the `case "claude"` comment now point to the doc comment on `peerSpaceWordwrap` instead of repeating the rule. One more copy turned up in a test comment (`peer_composer_test.go:283`). It is minor and doesn't block. The targeted tests pass (`go test -run 'PeerComposer|PeerSpaceWordwrap'` → ok).

1. **Strengths**
   - `TestPeerComposerClaudeCapturedHyphenWrap` checks that its fixture actually tells the two wrap rules apart (`ansi.Wordwrap(...) == lines` makes it fail). Without that check, the test could pass while proving nothing.
   - The collapsed-marker check is tight. It requires an exact regex match on the whole composer content, the cursor directly after the marker, `+M` equal to the message's newline count, and the `❯` prompt. Negative cases cover a wrong line count, extra text after the marker, a misplaced cursor, the marker on a second line, and an image marker.
   - The over-wide-word rule was captured from a live run (`-peer-live-body=overwidth`) and pinned in a fixture; it wasn't guessed. The table tests cover the "start on the next line if that needs fewer breaks" boundary in both directions.
   - The live harness is extended (`-peer-live-body`) so the regression shows up end to end. The log records that the reverted code never submits.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `peer_composer_test.go:283`: a test comment still says "Claude wraps at spaces only". It's correct about hyphens but leaves out the hard-break rule. It belongs to BR-4's family (rule: state Claude's wrap rule only on `peerSpaceWordwrap`, everything else points there). Fix: reword it to "Claude never breaks after hyphens (peerSpaceWordwrap)". I searched README, atlas and wrapcmd for "spaces only" / "at spaces"; this is the only remaining copy.

5. **Test coverage notes**
   - Each Done-when clause has a test or live run: hyphen wrap (captured fixture plus live), collapsed paste (strict-form unit tests plus live), and a body over 1,000 characters with paths and UUIDs (live `collapsed`).
   - `peerSpaceWordwrap`'s inputs with repeated, leading or trailing spaces are only reachable after `peerWordwrapSafe` filters them out, so they're correctly not tested through the matcher.

6. **Architecture**
   - **ARCH-DRY: pass.** The wrap rule is stated once and the docs point to it. Codex and Claude projections are separate functions on purpose, because their wrap rules really differ.
   - **ARCH-PURE: pass.** `peerSpaceWordwrap` and the marker check are pure functions of the snapshot and text. The tests use terminal snapshots, not mocks.
   - **ARCH-PURPOSE: pass.** Both root causes are fixed, with live evidence, and nothing that matters was deferred.

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      All three named sites now point to peerSpaceWordwrap; one more copy remains in a test comment (peer_composer_test.go:283), minor and not blocking.
```
