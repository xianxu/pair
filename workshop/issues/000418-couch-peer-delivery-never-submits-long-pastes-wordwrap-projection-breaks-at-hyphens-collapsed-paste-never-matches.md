---
id: 000418
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'b9b6a78ec09a3d49882d353dc077de0520cce794' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T20:15:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a2c55c9d", done: "c19cec0c"}
---

# Couch peer delivery never submits long pastes: wordwrap projection breaks at hyphens; collapsed paste never matches

## Problem

Reported by ariadne:1 (TL) on 2026-10-09 around 20:00 PT. Three peer messages
from ariadne:1 to ariadne:2, :3 and :4 all expired with "delivery deadline
elapsed: waiting for pasted envelope to render". Each envelope was pasted into
an idle Claude Code composer and never submitted; the operator pressed
alt+return by hand. The report itself, delivered to pair:1, failed the same way
(receipt 02f9e80d…, `cancelled` by operator input).

After the paste, `dispatchPeer` submits only when `peerComposerMatches`
confirms that the rendered composer equals a width projection of the envelope
(`cmd/internal/wrapcmd/peer_composer.go`). Replaying each recipient's raw
scrollback through wrap's terminal model, at 94×39, and running the real
matcher on the composer showed two independent causes.

1. **The word-wrap projection breaks at hyphens; Claude Code does not.**
   On ariadne:2 (Claude Code 2.1.295, 689-char envelope) and ariadne:3
   (2.1.296, 602 chars), the full text rendered, but `matches=false`. Claude
   kept `(workshop/projects/ariadne-robustness-1.md):` whole and wrapped it to
   the next line. `ansi.Wordwrap(expected, width-4, "")` broke it after
   `ariadne-`, and every later line shifted. Claude wraps at spaces only. Any
   hyphenated token that straddles the wrap column fails the match, and long
   bodies of paths, UUIDs and dates hit it almost every time.
2. **A collapsed paste never matches.** On ariadne:4 (2.1.296), and on
   pair:1 (2.1.295, about 1,100-char envelope), Claude rendered only
   `[Pasted text #N +1 lines] paste again to expand`; the envelope text was
   never drawn before submission. `peerComposerMatches` refuses a summarized
   paste marker by design. Envelopes of 602 and 689 chars rendered in full,
   which fits a collapse threshold somewhere between 689 and about 1,100
   chars; the exact threshold is unmeasured.

Evidence: recipient raw logs under `~/.local/share/pair/repos/`
(`7ec48ff0…/scrollback-1-ariadne-9`, `9a404a6b…/scrollback-1-ariadne-32`,
`5771eb3e…/scrollback-1-ariadne-31`, and pair:1's
`68f32487…/scrollback-couch-9d6fdf5eecace248-claude.raw`). They are replayable
with a throwaway test that feeds the bytes into `newTerminalModel(94, 39)` and
calls `peerComposerText`/`peerComposerMatches`.

## Spec

- Project Claude's composer wrapping as Claude does it: break at spaces only,
  and hard-break a token longer than the width. Keep the char-wrap fallback.
  Turn the captured ariadne:2 and :3 composers into fixtures.
- Collapsed paste: choose between (a) accepting the placeholder as render
  evidence when the composer was verified empty before our paste, the
  placeholder is the composer's only content, and its `+M lines` equals the
  envelope's newline count; and (b) refusing at admission, or splitting,
  envelopes above Claude's collapse threshold, which must be measured first.
  (a) keeps long messages working. Its evidence is weaker (no content check),
  but the empty-before-paste precondition already rules out a human draft.

## Done when

- A long single-line envelope with hyphenated tokens at wrap boundaries
  submits on Claude Code (fixture from the captured ariadne:2/:3 composers).
- A collapsed-paste envelope either submits or is refused at admission with an
  explicit reason, never left in the composer until the deadline.
- A live check with a ≥1,000-char body containing paths and UUIDs submits
  without operator input.

## Plan

- [x] Claude projection wraps at spaces only (`peerSpaceWordwrap`); Codex keeps
      `ansi.Wordwrap`. Fixture: the captured ariadne:2 composer.
- [x] Strict collapsed-marker acceptance (`peerClaudeCollapsedPaste`), chosen by
      the operator (option a).
- [x] Live `TestPeerLiveConformance -peer-live-body=hyphen-wrap|collapsed` submit.
- [x] Docs: atlas/couch.md, README, fixture README.
- [x] Over-width words hard-break like wrap-ansi (close review BR-1), captured live.

## Log

### 2026-10-09 (implementation)
- 2026-10-09: closed — Live Claude Code 2.1.296 (isolated PTY, safe mode, no tools, production dispatcher), TestPeerLiveConformance -peer-live-submit: -peer-live-body=overwidth (160-col token) render=wrapped submitted; hyphen-wrap (~625 chars) render=wrapped submitted; collapsed (~1300 chars) render=collapsed submitted. The hyphen-wrap run with the Claude branch reverted to ansi.Wordwrap never submits (harness timeout). Over-width rule captured live (wrap-ansi hard: start on current line unless next line needs fewer breaks) and pinned by TestPeerComposerClaudeCapturedOverwidthWord; hyphen fixture from the captured ariadne:2 composer; strict collapsed-marker cases; peerSpaceWordwrap table. wrapcmd and couchmessage green under clean env (unsandboxed); couchcmd cold-resume flake passed on rerun and does not import wrapcmd. Root cause evidence: recipient scrollback replays in the issue Problem.; review verdict: SHIP
- Operator chose option (a): accept the collapsed marker strictly. It must be the
  composer's whole content, with the cursor right after it, `+M` equal to the
  envelope's newline count, and the composer verified empty before the paste.
  This reverses #353's deliberate "collapsed stays unsupported".
- Live, Claude Code 2.1.296, isolated PTY, safe mode, no tools:
  - `-peer-live-submit -peer-live-body=hyphen-wrap` (about 630 chars):
    render=wrapped, submitted.
  - `-peer-live-body=collapsed` (about 1,300 chars): render=collapsed,
    submitted.
  - The same hyphen-wrap run with the Claude branch switched back to
    `ansi.Wordwrap` never submitted (harness timeout), so the live check
    detects the bug.
- Unit: the captured hyphen-wrap fixture asserts that it distinguishes the two
  projections. The collapsed tests cover a wrong line count, extra text, a
  misplaced cursor, the marker on a second line, and an image marker.

### 2026-10-09
- Close review round 1 (BR-1): the Spec promised hard-breaking over-width
  tokens, but the first cut declined them. Captured live on 2.1.296
  (`-peer-live-body=overwidth`): Claude starts a 160-column token on the
  current line and breaks it at the width, which is wrap-ansi's `hard` rule
  (start on the next line only if that needs fewer breaks). Implemented exactly
  and pinned by `TestPeerComposerClaudeCapturedOverwidthWord`. Live submit
  re-run: overwidth, hyphen-wrap and collapsed all submit. README no longer
  overstates: unverifiable whitespace still waits for a human.
