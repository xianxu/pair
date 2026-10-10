---
id: 000426
status: working
deps: [parley.nvim#312, ariadne#316]
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours: 2.806
card_mirror: 'fcc8217fb03747a2d0d725af557cf531e10510c8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T11:43:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:0
    worktree: /Users/xianxu/workspace/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Review pane: render 🤖 comment chains compactly with a thread float (port parley#312)

## Problem

The review workbench pane (`nvim/review/`, atlas `review-workbench.md`) shows
🤖 review markers raw. A long comment chain (`🤖<X>[q]{a}[q2]{a2}…`) floods
the line and is hard to read. parley#312 made these chains first class in
parley's markdown buffers. Pair's review pane should render them the same way,
so a review reads the same in both tools.

## Spec

Port parley#312's behavior (parley `lua/parley/comment/`; atlas
`parley.nvim/atlas/modes/review.md`):

- **Compact display** (extmark conceal, `conceallevel=2`):
  - every turn collapses to `[…]` / `{…}`, except a last human turn, which
    stays visible and editable inline (`🤖[…]{…}[this is ok]`);
  - `🤖<X>…` highlights X with the 🤖 and `<>` hidden;
  - `🤖~D~{N}` shows D struck through plus `{…}`;
  - turn brackets are colored by speaker.
- **Edit protection:** the cursor snaps off hidden bytes. Insert mode only
  allows these points: before the 🤖, inside the anchor, inside the last human
  turn, after the marker. A marker broken by a stray edit paints visibly
  (`ParleyReviewBroken`-style).
- **Thread float:** `<CR>` on a marker opens the chain like a chat: `💬: `
  human turns, `🤖: ` robot turns, unprefixed lines continue a turn. `:w`
  writes it back as one line, `q` saves and closes, `:q!` discards. Save
  re-parses the joined marker and refuses unbalanced brackets.
- **Single-line grammar** (ariadne#316, review-convention §3): a newline inside
  a turn is `<br>`, a literal `<br>` is `\<br>`, and a backslash run before
  `<br>` doubles (odd = literal, even = newline). Only turn text is decoded;
  anchors stay verbatim.

Open questions, to settle at start-plan:

- **Multi-line markers.** Pair #89 made `markers.lua` (`spans_multiline`) and
  `reconcile.conflict_marker` multi-line aware. ariadne#316 now says markers
  are single-line. Should the agent's conflict markers move to `<br>`, with
  legacy multi-line markers painting as broken?
- **Escaping.** Pair's `marker_codec.lua` already escapes `>`, `]` and `\`
  inside quotes. Reconcile it with the `<br>` codec, keeping one codec per
  delimiter set.
- **Reuse vs port.** Pair ports parley's parser (`markers.lua`) rather than
  depending on parley. Decide whether to port `comment/view.lua` (pure layout
  and snap) and `comment/thread.lua` (pure float layout) the same way.
- **Other viewers.** Should the scrollback and change-log viewers
  (`nvim/annotate.lua`) get the compact display too, or only the review pane?

## Done when

- In the review pane, a chain of 3 or more turns displays as
  `🤖[…]{…}[last human turn]`, checked on a real render (a pty
  `screenstring()` probe, not just extmark assertions; see parley's lesson on
  treesitter link conceal hiding `[text]` brackets).
- `<CR>` on a marker opens the thread float. A reply saved with `:w` lands as
  one `<br>`-encoded line, and the agent's next round reads it.
- The codec and float layout round-trip under property tests that draw input
  from their own delimiters.

## Plan

- [ ] Implement the reviewed plan in `workshop/plans/000426-compact-review-threads-plan.md`, including property, real-render, and review-round integration tests.

## Log

### 2026-10-10

- Claimed in pair:0 and ran start-plan. Both dependencies have landed:
  parley.nvim#312 at `420b2b3109fb`, ariadne#316 at `4164e66ece11`.
- Read the actual issue file after discovering `issue show` prints section
  headings, not bodies. No implementation changes made.
- Drafted the durable plan; implementation awaits operator approval.
- Fresh-eyes spec/plan review approved after correcting canonical escape parity;
  no remaining Critical/Important findings. Issue schema validation and committed
  diff whitespace checks pass. Plan checkpoint: `8885b6b2`.

### 2026-10-10 — implementation and verification

- Full change-code gate accepted; plan validity/testing findings addressed.
  Estimate-quality was advisory: runtime packaging and final suites are absorbed
  by the projection/float implementation rows, mutation checks by their component
  rows. The midpoint estimate remains provisional; actual work is measured.
- Implemented canonical raw-turn codec, shared completeness/diagnostics scanner,
  compact projection, cursor protection, full editable thread and encapsulated
  save/close model. Initial failing tests covered missing behavior before code.
- Corrected painted bracket colors after the real PTY assertion failed; assigning
  role highlights to conceal replacement marks fixes the actual screen cells.
- `make test-lua test-review` passed with isolated session variables and TMPDIR.
  `make test-runtimebundle` passed. Inventory suite failure is being compared to
  the base; none of the new comment modules appears in its current failure list.
- Mutation checks caught canonical parity, missing source comparison, and missing
  fence exclusion. Pure layout: 1,000 lines/100 markers p95 0.836ms (max 0.972ms),
  100-turn marker p95 0.222ms (max 0.287ms), 100 samples each.

- Baseline comparison proved the artifact classifier's 53 findings are identical
  at base `78d860907984` and current HEAD; no new modules are unclassified.
  Both isolated checkouts generated runtime assets before the comparison.
- PTY tests pass including actual undo/redo repaint after yielding to Neovim's
  input loop; no synthetic render/autocmd is used as an oracle repair.
- Complete rendering (API decorations included), 1,000 lines/100 markers:
  initial 4.10ms; 50 redraws median 2.33ms, p95 2.85ms, max 2.87ms.
- Keyboard help derives Enter/thread-q descriptions from the new Lua modules;
  its package tests pass, including embedded-source drift.

## Revisions

### 2026-10-10 — proposed port boundaries for approval

Resolve the Spec's open questions as follows (pending operator approval):
review pane only; port standalone Lua modules from Parley's landed commit;
coordinate newline and delimiter escaping through a raw-turn codec rather
than composing whole-string encoders (see the correction below). Compact only complete single-line markers. Preserve
legacy multiline parsing, highlighting, resolution, and reconciliation output
in this issue: a multiline quoted hunk cannot be converted to an encoded anchor
without changing its meaning. Such markers stay fully visible, with no compact
cursor snapping or thread float. Newly saved thread turns use the canonical
single-line encoding. No broad migration of existing writers or other viewers.

Additional acceptance criteria: code-fenced/inline examples stay literal;
rendering, cursor protection and float targeting agree on marker eligibility;
float save refuses a changed source marker without losing the edited thread;
repeated saves, undo/redo and teardown are covered. The original three
Done-when requirements remain in force.

### 2026-10-10 — correct raw-turn encoding after plan review

The fresh-eyes reviewer found that delimiter unescape before newline decode
loses the canonical odd/even slash rule. The plan now retains raw turn text,
uses one coordinated turn encoder/decoder, and pins canonical wire fixtures
independently of paired round trips. Single-line turns follow canonical `<br>`
semantics; anchors and legacy multiline resolution retain existing behavior.

### 2026-10-10 — implementation approval and gate refinement

Operator approved implementation ("continue"). Plan gate requested a concrete
malformed-marker scanner contract and function-level test strategies; both are
now specified, together with a bounded compact-render envelope. Full flow,
one atomic close boundary.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration source is marked stale, so these
are provisional. Three focused Lua/Neovim units: codec/thread representation,
compact projection/attachment, and float lifecycle/handoff. Each uses table
midpoint design 2h × 0.2 for the approved detailed plan, and implementation
1h × 0.4 v3.1 scaling. Familiarity 1.0: existing Lua/Neovim patterns and landed
Parley implementation; no novel library needed. Docs use 0.1h × 0.2 design and
0.1h × 0.4 implementation. One close review uses 0.1h × 0.2 design and
0.35h × 0.4 implementation. Design buffer 15% for the thorough plan.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.4 impl=0.4
item: lua-neovim design=0.4 impl=0.4
item: lua-neovim design=0.4 impl=0.4
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.14
design-buffer: 0.15
total: 2.806
```
