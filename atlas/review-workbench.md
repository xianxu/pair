# Review workbench

An embedded nvim **document workbench** for agentic document review (issue #66):
a persistent agent (pair's session) *proposes* edit records, and an nvim review
pane *applies* them undo-ably, decorates them, and journals each round via
ariadne's `docflow`. The agent is the producer; nvim is the applier/renderer;
the contract between them is a records file + git commits.

## Modules (`nvim/review/`)

Pure core (run under `nvim -l`, colocated `*_test.lua`, `make test-lua`):

- `record.lua` — the `Record` `{old, occurrence, new, explain}` and its **one**
  JSON serialization, written verbatim to both the handoff file and the agent
  commit body. `apply` enriches each record with `new_occurrence` (Nth match of
  `new` post-apply) so resume can re-anchor; `occurrence` (Nth `old` in base) and
  `new_occurrence` are never crossed.
- `reconstruct.lua` — pure `records, content, which → {highlights, diagnostics}`
  (0-based line ranges + explains). `which='old'` anchors by `occurrence`,
  `which='new'` by `new_occurrence`. The resume / from-commit render path; exports
  `nth_offset`/`line_of`.
- `markers.lua` (M2) — pure 🤖 review-request parser (ported from parley):
  `🤖<quoted>?(~strike~)?([user]|{agent})*` → marker records with `ready`/`pending`
  (last-section rule), excluding markers in fenced/inline code. The human's
  in-doc review requests; M3 highlights from it, M4's agent reads it.
- `mode.lua` + `modes/*.md` (M2) — pure mode-brief parser (`parse`/`directives` +
  IO `load`/`list`) and the 6 stock modes (developmental→free-form). `directives()`
  renders the scope/frontier/deletions block M4's agent SKILL.md composes in.

Integration seams (headless shell tests, `make test-review`):

- `apply.lua` — applies records as ONE undo block (first edit breaks the undo
  sequence so the round is separate from prior history; edits 2..N `undojoin`,
  E790-safe), **bottom-to-top** to avoid offset drift, decorates from the actual
  edited ranges, returns records enriched with `new_occurrence`. No file reload
  (a reload would reset undo).
- `docflow.lua` — thin wrapper shelling `$DOCFLOW_BIN` (ariadne's `docflow`):
  `start`/`round --side`/`status`/`ship`. No commit/branch logic reimplemented.
- `handoff.lua` — the ephemeral `review-handoff-<tag>.json` (in XDG data dir):
  the agent writes it atomically; nvim **timer-polls** (not fs_event — macOS
  FSEvents precedent in `init.lua`), decodes, unlinks, fires a callback. Data and
  signal in one file.
- `apply.snapshot`/`apply.apply_snapshot` (M2) — read/restore the decoration
  state: ranged extmarks ({line,end_line}) + diagnostics, as two independent
  layers (they decouple after riding) sharing a `clear()` helper with `place`.
- `projection.lua` (M2) — decoration coherence across undo/redo (ported from
  parley): per-buffer snapshots keyed by content hash; on undo/redo restore the
  matching snapshot, on a novel state capture the riding decorations. The
  `record_empty_for` guard keeps a prior round's styling when round-2's base is
  round-1's output. No more clear-on-each-apply.
- `init.lua` — the orchestrator: `start` a review (docflow start + `undofile` +
  handoff watch); on each handoff `on_agent_round` = undo-able apply → snapshot
  (projection record/watch) → save → `docflow round --side agent` with the records
  embedded in the commit body; `human_round` commits incoming human edits.

## The loop (round = two docflow commits)

human edits → `human_round` (docflow human commit) → poke agent → agent writes a
records handoff → nvim watcher → `apply` (undo-able) + decorate → `docflow round
--side agent` (records in the commit body). History lives in git (round commits +
per-hunk explains in the agent commit body); fine-grained undo lives in nvim's
`undofile`; no bespoke sidecar. The doc must be in a git repo.

## The review window (M3)

The document workbench in a live pair session — a **floating** nvim pane (the
proven scrollback/changelog pattern), opened on a file, alongside pair's agent+draft.

- `nvim/review.lua` — the pane init (`nvim -u nvim/review.lua <file>`): dofiles the
  review core + poke + markers, `review.start{}`, wires **Alt+Return = finish human
  turn** (`human_round` + poke "updated, please review"), renders 🤖 markers
  (`markers.highlight_spans` → `ParleyReview*` extmarks, re-rendered on TextChanged),
  writes the open-state file (`pid\nvisible`), and tears down on `VimLeave`.
- `bin/pair-review-open <file>` — validates + spawns the floating pane (`zellij run
  --floating --close-on-exit --name review`), replacing any live review (single pane).
- `:PairReview <file>` (in draft `nvim/init.lua`, `complete=file`) — the `:e`-style
  opener; shells to `pair-review-open`.
- `bin/pair-review-toggle` (Alt+r, `zellij/config.kdl`) — mode-aware: no review →
  focus the draft + open `:PairReview ` (file-select); review open → flip visibility
  (`show`/`hide-floating-panes`, visibility tracked on state-file line 2 to dodge the
  toggle's own floating pane confounding `are-floating-panes-visible`; **never**
  `toggle-floating-panes`).
- `nvim/pair_poke.lua` — id-based agent poke: relative `move-focus` does NOT escape a
  floating pane, so it resolves the agent + caller panes from `list-panes --json` and
  `focus-pane-id`s them (focus agent → write-chars + `write 27 13` submit → restore).

The agent pane is pair's **existing** agent — free-form chat works for free; the SKILL
that makes "please review" / "ship it" review-aware is M4.

## State

M1 (contract + history spine, fake-agent-driven e2e), M2 (consumer-half port:
projection / markers / modes), and M3's **headless parts** (window/poke/toggle/marker
rendering wiring) are complete + tested. **M3 manual real-session smoke is pending**
(live zellij pane open/toggle, the Alt+Return round-trip, and the projection
`TextChanged` autocmd firing live — which headless can't reach). Still to come (see
`workshop/issues/000066`, `workshop/plans/000066-*`): M3 the interactive
styling-accumulation semantics (human round adds without clearing agent's; clear on
next conversation turn), M4 the real agent SKILL.md (composing `mode.directives()`) +
memory discovery + cross-session resume.

## Tests

- `make test-lua` — `record`, `reconstruct`, `markers`, `mode` (pure).
- `make test-review` — `docflow` (+ hermetic `tests/lib/fake-docflow.sh` and a
  gated smoke against the real ariadne `docflow.sh`), `apply` (incl. snapshot
  round-trip), `handoff`, the `loop` e2e (with `tests/lib/fake-review-agent.sh`),
  and `projection` (undo/redo coherence + riding + round-2 idempotence); M3 adds
  `poke` (id-based agent poke, no relative move-focus), `window` (:PairReview +
  pair-review-open + review.lua: keymap/state/markers + Alt+Return round-trip), and
  `toggle` (mode-aware branch, explicit show/hide, no toggle-floating-panes).
