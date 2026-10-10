---
id: 000418
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '679cdbe15a2e7c5b758135a84b43dcc2ad79195b' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-09
