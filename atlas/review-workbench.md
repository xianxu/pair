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
- `init.lua` — the orchestrator: `start` a review (docflow start + `undofile` +
  handoff watch); on each handoff `on_agent_round` = undo-able apply → save →
  `docflow round --side agent` with the records embedded in the commit body;
  `human_round` commits incoming human edits.

## The loop (round = two docflow commits)

human edits → `human_round` (docflow human commit) → poke agent → agent writes a
records handoff → nvim watcher → `apply` (undo-able) + decorate → `docflow round
--side agent` (records in the commit body). History lives in git (round commits +
per-hunk explains in the agent commit body); fine-grained undo lives in nvim's
`undofile`; no bespoke sidecar. The doc must be in a git repo.

## State

M1 (the contract + history spine) is complete and fake-agent-driven end-to-end
(`tests/review-loop-test.sh`). Still to come (see `workshop/issues/000066`,
`workshop/plans/000066-*`): M2 port parley's render/projection/modes, M3 the
`:PairReview` window + zellij agent poke, M4 the real agent SKILL.md + memory
discovery + cross-session resume.

## Tests

- `make test-lua` — `record`, `reconstruct` (pure).
- `make test-review` — `docflow` (+ hermetic `tests/lib/fake-docflow.sh` and a
  gated smoke against the real ariadne `docflow.sh`), `apply`, `handoff`, and the
  `loop` e2e (with `tests/lib/fake-review-agent.sh`).
