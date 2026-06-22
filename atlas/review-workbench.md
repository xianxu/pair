# Review workbench

An embedded nvim **document workbench** for agentic document review (issue #66):
a persistent agent (pair's session) *proposes* edit records, and an nvim review
pane *applies* them undo-ably, decorates them, saves, and pokes the agent. The
agent is the producer **and the only git writer**: it creates/resumes the
`review/<slug>` branch and commits `docflow` rounds from the nvim's landed
artifact. The contract between them is a small set of seam files + git commits.

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
  `start`/`round --side`/`status`/`ship`. The review nvim no longer calls it;
  it remains as a contract test surface for the commit shape the **agent**
  produces (and as a future candidate for removal if no read use earns it).
- `handoff.lua` — the ephemeral `review-handoff-<tag>.json` (in XDG data dir):
  the agent writes it atomically; nvim **timer-polls** (not fs_event — macOS
  FSEvents precedent in `init.lua`), decodes, unlinks, fires a callback. Data and
  signal in one file. Also owns the reverse channel,
  `review-landed-<tag>.json`: `{summary, body, applied, dropped}` for the agent
  to commit verbatim after nvim applies a handoff.
- `apply.snapshot`/`apply.apply_snapshot` (M2) — read/restore the decoration
  state: ranged extmarks ({line,end_line}) + diagnostics, as two independent
  layers (they decouple after riding) sharing a `clear()` helper with `place`.
- `projection.lua` (M2) — decoration coherence across undo/redo (ported from
  parley): per-buffer snapshots keyed by content hash; on undo/redo restore the
  matching snapshot, on a novel state capture the riding decorations. The
  `record_empty_for` guard keeps a prior round's styling when round-2's base is
  round-1's output. No more clear-on-each-apply.
- `poke_bodies.lua` — pure builders for the prose signals sent to the agent:
  review opened, review target prep, handoff applied, and human turn finished.
- `readiness.lua` + `bin/pair-review-readiness` — pure/classified git readiness
  for review-start: stop / track / resume / new / interact. The nvim proposes;
  the agent acts.
- `resolve.lua` — pure parley §5 accept/reject resolution for `🤖` marker chains;
  `nvim/review.lua` binds it to the review pane (`\a`, `\r`, `]m`, `[m`).
- `spinner.lua` — pure compact spinner/elapsed helper for M4c; currently unwired
  pre-work.
- `init.lua` — the orchestrator: `start` a review (`undofile` + handoff watch +
  reconstruct-on-open); on each handoff `on_agent_round` = undo-able apply →
  snapshot (projection record/watch) → save → write landed-artifact → poke the
  agent to commit; `human_round` saves only. It calls no `docflow` writer.

## The loop (round = two docflow commits)

`:PairReview <file>` proposes a review target → the agent runs readiness prep
(track/new/resume/interact) and marks the target ready → Alt+c opens the pane.
Agent writes a records handoff → nvim watcher applies undo-ably, decorates, saves,
writes the landed-artifact, and pokes `agent_applied` → the agent commits the
agent round from that artifact. Human edits → Alt+Return saves and pokes
`human_finished` in Copy Edit posture, with `🤖[]` comments treated as
fulfill-or-punt instructions → the agent commits the human round and re-reviews.
`:PairReviewShip` pokes the agent to run `docflow ship`; the pane does not shell
docflow. History lives in git (round commits + per-hunk explains in the agent commit
body); fine-grained undo lives in nvim's `undofile`; no bespoke sidecar. The doc must
be in a git repo.

## The review window (M3)

The document workbench in a live pair session — a **floating** nvim pane (the
proven scrollback/changelog pattern), opened on a file, alongside pair's agent+draft.

- `nvim/review.lua` — the pane init (`nvim -u nvim/review.lua <file>`): dofiles the
  review core + poke + markers, `review.start{}`, wires **Alt+Return = finish human
  turn** (`human_round` save + `human_finished` poke), renders 🤖 markers
  (`markers.highlight_spans` → `ParleyReview*` extmarks, re-rendered on
  TextChanged), supports accept/reject on the cursor line (`Alt+a`/`Alt+r`, with
  `\a`/`\r` fallbacks), inserts human comment markers (`Alt+q` bare marker or visual
  quote), exposes `:PairReviewShip` as an agent-owned ship request, plus marker
  navigation (`]m`/`[m`), writes the open-state file (line 1 = pane nvim `pid` for
  liveness, line 2 = the absolute doc path for the indicator), and tears down on
  `VimLeave`. Also defines `PairReviewToggle()` = hide-self (the case where Alt+c
  fires from inside the focused floating review pane).
- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
  (`zellij run --floating --close-on-exit --name review --width 100% --height 100%`;
  percentage dims, not `tput`, which measured the wrong pane), replacing any live
  review (single pane).
- `:PairReview <file>` (in draft `nvim/init.lua`, `complete=file`) — proposes the
  review target. It writes `review-target-<tag>.json` with `status=proposed` and
  pokes the agent to run `pair-review-readiness`; it does **not** open the pane.
  The agent marks the target `ready`; Alt+c opens it.
- **Alt+c** (`zellij/config.kdl`) — routed through the draft nvim like Alt+d
  (`MoveFocus Down` → `<C-\><C-n>` → `:lua PairReviewToggle()`), **not** a spawned
  shell pane. The draft's `PairReviewToggle()` (`nvim/init.lua`) branches on the
  state-file liveness and review-target status: live review → flip visibility from
  this *tiled* draft (`are-floating-panes-visible` → `show`/`hide-floating-panes`;
  **never** `toggle-floating-panes`); no live review + ready target → open;
  proposed target → "prep in progress"; no target → drop into `:PairReview `
  (file-select). Pure decision `_pair_review_toggle_action(alive, visible, status)`.
  Review-targets are scoped to the current conversation id so fresh sessions ignore
  stale targets while resumed sessions keep their in-progress target. Resolution is
  `PAIR_SESSION_ID` → `config-<tag>-<agent>.json` → live Codex rollout via
  `agent-pid-<tag>`; Codex/agy learn ids asynchronously, so review target handling must
  not rely on the launch-time env alone. `Alt+r` is deliberately free inside the review
  pane for reject.
- `nvim/pair_poke.lua` — id-based agent poke: relative `move-focus` does NOT escape a
  floating pane, so it resolves the agent + caller panes from `list-panes --json` and
  `focus-pane-id`s them (focus agent → write-chars + `write 27 13` submit → restore).
- **review-mode bar** (`nvim/init.lua`, `do`-block; `_pair_review_bar` count source +
  `_pair_review_segment` cached segment) — while a review is open, the draft's
  **statusline** carries `Review • <file> • 🤖N/M`: `pair_compose_statusline` swaps the
  cached segment in for the rightmost cheatsheet, so review mode is visible even when the
  pane is hidden. A 1.5s timer recomputes the segment (counts parsed from `git log` round
  subjects, **branch-scoped** to the active `review/<slug>` so other docs' shipped reviews
  don't leak in — `🤖0/0` off a review branch / in M3 render-only) and triggers a redraw
  only on change; the hot render path never shells git. (This **supersedes** an earlier
  line-1 `=== review … ===` indicator — line 1 is the user's to edit. New draft-side
  review helpers live in `do`-blocks sharing `_G._pair_review` — init.lua is at Lua's
  200-local chunk ceiling.) The cross-process `review-<tag>.open` path is centralized in
  `nvim/review/seam.lua` (one fallback rule for writer + reader).
- **docflow degradation** (`nvim/review/docflow.lua`) — missing `docflow` still has
  a calm contract-test path, but the review pane no longer shells docflow at runtime.
  Round commits are agent-side. See `workshop/targets/review-protocol.md` for the
  full agent↔nvim state machine.

The agent pane is pair's **existing** agent — free-form chat works for free; the SKILL
that makes "please review" / "ship it" review-aware is the ariadne #000121 half of M4.

## State

M1 (contract + history spine), M2 (consumer-half port), M3 (review window + live
smoke), M4a (nvim writes no git; fake-agent commits from landed artifacts), and
M4a' pair-side review-start/resume are implemented and headless-tested. The current
open boundary is M4b skeleton: pair-side accept/reject + marker navigation,
fulfill-or-punt default Copy Edit posture, and ship request are implemented; full
suite verification/milestone close is pending. M4c is the thickening pass; one spinner
helper exists as unwired pre-work.

The real-agent half lives in ariadne #000121. Until that lands, pair proves the
protocol with `tests/lib/fake-review-agent.sh`; the real live smoke remains the
cross-repo proof that the persistent agent recognizes review mode and owns the
docflow rounds.

## Tests

- `make test-lua` — `record`, `reconstruct`, `markers`, `mode`, `poke_bodies`,
  `readiness`, `resolve`, `spinner` (pure).
- `make test-review` — `docflow` (+ hermetic `tests/lib/fake-docflow.sh` and a
  gated smoke against the real ariadne `docflow.sh`), `apply` (incl. snapshot
  round-trip), `handoff`, the `loop` e2e (with `tests/lib/fake-review-agent.sh`),
  and `projection` (undo/redo coherence + riding + round-2 idempotence); M3 adds
  `poke` (id-based agent poke, no relative move-focus), `window` (:PairReview +
  pair-review-open + review.lua: keymap/state/markers + Alt+Return round-trip),
  `toggle` (mode-aware branch, explicit show/hide, no toggle-floating-panes),
  `resume`, and the agent-owns-git loop.
