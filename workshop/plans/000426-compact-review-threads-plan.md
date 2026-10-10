# Compact review threads implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the integrated pane work; delegate bounded codec work if useful. Track the steps below.

**Goal:** Port parley.nvim#312's compact comment chains and editable thread float to Pair's review pane.

**Architecture:** Adapt the landed Parley implementation at `420b2b3109fb`, retaining Pair's parser and edit semantics. One buffer-aware projection of parsed markers supplies conceal geometry, cursor protection, and float targeting. A pure thread model owns serialization and save/close decisions; Neovim glue performs edits and owns temporary resources.

**Tech Stack:** Lua, Neovim extmarks/acwrite buffers, existing shell/headless tests, Go runtime bundle tooling.

Status: approved by the operator on 2026-10-10 ("continue"). This exceeds 100 added production lines and requires the full flow. One atomic implementation and one close review; no artificial milestone boundary.

## Scope and choices

The issue's existing spec defines the UI: anchors remain visible, earlier turns
collapse to colored brackets around an ellipsis, and a last human turn remains
editable. Enter opens a chat-like thread, with `:w` saving, `q`/`:x` saving and
closing, and `:q!` discarding. Enter elsewhere retains native counted behavior.

Port standalone modules rather than adding a runtime dependency on Parley.
Using raw text plus a read-only popup would be smaller but would omit the
requested editing behavior. Loading Parley's plugin would couple standalone
Pair sessions to the user's editor installation. The local port fits existing
`markers.lua` and `projection.lua` practice (ARCH-DRY).

Only the review pane is included. Scrollback/changelog annotation behavior is
unchanged. Legacy multiline markers, including Pair's reconciliation hunks,
remain raw and retain current parsing/highlighting/resolution; they are not
partially concealed or opened in this line-local editor. Migrating these
writers requires a separate representation decision because anchors must stay
verbatim. New float saves emit single-line encoded turns.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Turn newline codec | `nvim/review/comment_codec.lua` | new |
| Parsed marker turn representation | `nvim/review/markers.lua` | modified |
| Compact layout and cursor policy | `nvim/review/comment_view.lua` | new |
| Thread lines and save/close transition model | `nvim/review/comment_thread.lua` | new |
| Turn resolution | `nvim/review/resolve.lua` | modified |

The turn codec preserves the canonical odd/even backslash rule around `<br>`
while supporting Pair's delimiter escapes. Add `raw_text` to parsed sections,
retaining `text` with its existing delimiter-decoded contract. `decode_turn`
reads raw_text directly: recognize slash runs plus `<br>` BEFORE generic
backslash unescaping; 2n slashes mean n literal slashes followed by newline,
2n+1 mean n slashes followed by literal `<br>`. Elsewhere use existing delimiter
escape semantics. `encode_turn` is the inverse, escaping bracket delimiters
without re-escaping the already encoded `<br>` runs. Factor any shared generic
escape primitive through `nvim/marker_codec.lua`; do not concatenate two whole
string encoders/decoders whose escape alphabets overlap.

Single-line markers use this canonical interpretation regardless of producer;
there is no version bit to distinguish a formerly literal unescaped `<br>`.
This deliberate compatibility change applies only to turn-derived text. Legacy
multiline markers retain the existing section-text/resolve path. Anchors always
retain current delimiter decoding and literal `<br>` behavior. Thread conversion
and single-line turn resolution use raw_text through the same turn codec.
Enumerate existing section consumers and pin their contracts with regression
tests. Independently authored canonical raw fixtures with 0–4 slashes before
`<br>` must test float decoding and resolution; paired round trips alone cannot
prove compatibility with agents/Parley.

Add `markers.scan(lines)` returning `{markers, diagnostics}`. `markers` carries
the same successful records as `parse_markers`; add `complete=false` when the
next byte is an unmatched section opener, plus an exclusive end position.
Diagnostics are `{row,col,end_row,end_col,kind="malformed"}` from the first
marker byte to that line's end for an unclosed opening/continuation; code
exclusions are applied by the same scanner. Wholly malformed chains produce a
diagnostic even without a successful record. Multiline successful records are
legacy, not malformed. `parse_markers(lines)` remains a wrapper returning only
successful records, preserving legacy callers. All compact consumers accept
only complete single-line records. The view consumes this scan, not an independent line scan:
inline/fenced code exclusions and multiline eligibility apply equally to all
three consumers. It yields byte ranges for hidden text, visible anchor, colored
brackets, editable final human turn, and malformed-marker warnings. Warnings
must not conceal malformed input. UTF-8 cursor landing respects character starts.

Thread conversion uses `💬:`/`🤖:` prefixes and escaped prefix-like continuation
lines. Preserve all intentional content, including trailing newlines, rather
than inheriting upstream's lossy trailing-blank normalization. An appended empty
reply slot is omitted on save; a pre-existing empty human turn remains. Save
re-parses the whole serialization and checks turn count/types/content, prefix,
and exact consumption; malformed thread structure is refused visibly.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Thread float controller | `nvim/review/comment_float.lua` | new | Neovim buffer/window/extmark/write events |
| Compact view attachment | `nvim/review/comment.lua` | new | window options and buffer autocmds |
| Review pane wiring | `nvim/review.lua` | modified | existing activation/render/keymap lifecycle |
| Runtime packaging/inventory | `Makefile.local`, `cmd/internal/artifactpath/manifest.go`, generated runtime bundle | modified | test discovery and embedded assets |

Use real isolated Neovim for UI integration and reuse the stateful pane host in
`tests/review-controls-test.sh` for review handoff. No new external service or
binary dependency (ARCH-MOCK). Runtime resource classification must include
source files and generated mirrors.

## Architecture and operating envelope

- **ARCH-PURE:** codec/layout/thread conversion and lifecycle decisions are
  deterministic; API calls remain in the controller and attachment.
- **ARCH-PURPOSE:** prove actual painted cells and actual pane keymaps, then
  save the human round through the existing handoff. Extmark assertions alone
  do not prove the requested rendering.
- **ARCH-CONSTRAINTS:** keystroke path operates on the active line's cached
  layout; reparse on content changes, not every cursor move. No IO/process
  launch on cursor movement. One live float per review activation. Clamp float
  dimensions to usable editor space, including small terminals; wrapping and
  scrolling handle long threads. Test a 1,000-line document with 100 markers
  and a 100-turn thread. Initial supported compact envelope: <=1,000 lines and
  <=128 KiB (conservative UI assumption); larger buffers retain raw highlighting,
  with one notice per activation, no compact cache or float. Within the envelope,
  target <=50ms per refresh; measure the representative workload and investigate
  any breach. Cursor moves use cached layout only. These limits bound the new
  projection; they do not promise to improve the legacy parser's own costs.
- **ARCH-SECURE:** document and float text are untrusted data, never commands.
  Validate the serialized marker and source range immediately before replacing
  bytes. Anchor the opened instance with a range extmark, not a text search
  that could select another identical marker. Source disappearance/refusal
  cannot write to another buffer or silently discard user text. Tests isolate
  PAIR/COUCH/ZELLIJ variables and use an owned short temporary directory.
- **ARCH-ORDER:** pure lifecycle states are closed, editing, and conflicted;
  source identity and last-saved raw bytes belong to the active state. Open
  creates one resource set. Save with a matching source emits replace; success
  updates the expected raw bytes and clears dirty state. Changed/missing source
  emits refusal and preserves float edits; retry requires revalidation, never
  overwrites unknown text. Close-after-save happens only on confirmed success.
  Discard closes explicitly. Forced source/activation/window teardown rescues
  unsaved text to the unnamed register and notifies, then closes. Duplicate
  cleanup is harmless; a second open focuses the existing float. Pure transition
  tests and real event tests exercise changed source, repeated save, undo,
  source wipe, retarget, and late/duplicate close.
- **ARCH-FUNERAL:** activation owns the float, scratch buffer, tracking mark,
  maps/autocmds, caches, and saved window options. Stop/retarget, source wipe and
  window close release them; restore window options on leave. No durable new
  file or background process is created. Existing recovery owns document edits;
  thread text pending save lives only in the float or rescue register.

## Chunk 1: Implement and verify the port

### Task 1: Codec and thread representation

Files: new `nvim/review/comment_codec.lua`, `comment_thread.lua` and colocated
`*_test.lua`; modify `nvim/review/markers.lua`, `resolve.lua`, shared
`nvim/marker_codec.lua` only for reusable primitives, and their tests.

- [x] Enumerate section-text consumers with `rg 'sections|last.text' nvim/review nvim/review.lua`; pin existing anchor/delimiter/resolve behavior.
- [x] Test `encode_turn`/`decode_turn` with seeded delimiter-alphabet properties plus independent canonical wire fixtures; test `to_lines`/`from_lines` by role/content preservation and full parser consumption; test `resolve` by literal anchor preservation versus canonical turn decoding.
- [x] Run `nvim -l nvim/review/comment_codec_test.lua` and `nvim -l nvim/review/comment_thread_test.lua`; verify behavioral failures before implementation.
- [x] Implement the canonical raw-turn codec above, adapting upstream codec/thread code through Pair's parser. Add newline decoding only for turn-derived resolution, preserving anchors literally.
- [x] Run the new tests plus `nvim -l nvim/review/markers_test.lua` and `nvim -l nvim/review/resolve_test.lua`; commit this coherent component with an issue reference.

### Task 2: Shared compact projection and attachment

Files: new `nvim/review/comment_view.lua`, `comment.lua`, colocated view tests;
modify `nvim/review.lua`; new `tests/review-comments-test.sh`.

- [x] Test `markers.scan` with malformed/code-context generated documents, asserting diagnostics and completeness never conceal a broken prefix. Test `comment_view.layout`/`marker_at` with eligibility invariants; test `snap` over every byte/insertion point, asserting legal UTF-8 destinations and unchanged visible input.
- [x] Build the real-render test around existing `tests/lib/run-headless.sh`/isolated test environment, attaching a real UI/pty before `screenstring()` assertions. Assert actual colored bracket/ellipsis cells, visible anchor and final reply, not only extmark metadata.
- [x] Run the failing view/render tests, then implement the shared projection, conceal and directional cursor policy. Reuse Pair parser exclusions, original multiline highlight fallback, and existing highlight setup; update on TextChangedI as well as ordinary review events.
- [x] Attach/detach through start_review/stop_review, including restore rollback. Rendering must not enter the undo history or mutate source bytes. Verify undo/redo, buffer switch, theme change and small-window rendering; commit.

### Task 3: Float lifecycle and review handoff

Files: new `nvim/review/comment_float.lua` and colocated lifecycle tests;
modify `nvim/review.lua`, `tests/review-comments-test.sh`,
`tests/review-controls-test.sh` or a dedicated thread handoff test.

- [x] Test `comment_thread.new_session().transition` using generated event sequences and independent no-overwrite/no-loss/resource-count invariants. Test `comment_float.open_thread`/save/close effects with real Neovim source edits and teardown; drive Enter/save/discard through real keymaps and commands.
- [x] Implement acwrite float with role highlighting and editable final reply. API glue executes pure model effects, preserving text on conflict/forced close and refusing close-after-failed-save. Escape exits insert mode normally; review pane float dismissal must use the same cleanup path.
- [x] Save a multiline reply, assert exactly one encoded marker line in the source, submit via the actual Alt+Return mapping and inspect the saved document read by the stateful agent host. Assert no automatic agent submission from :w alone.
- [x] Rerun existing accept/reject, diagnostic floats, navigation, controls and restore tests. Commit.

### Task 4: Packaging, documentation and final verification

Files: `Makefile.local`, `cmd/internal/artifactpath/manifest.go`, generated assets,
`README.md`, `atlas/review-workbench.md`, `atlas/index.md` if a new map is added,
and the issue/plan Log and checkboxes.

- [x] Register every new test in test-lua/test-review; classify every production source/mirror and regenerate via `make runtimebundle-generate`.
- [x] Document compact display, Enter/thread save/discard controls, literal/multiline fallback and rescue behavior in README and the existing atlas page. Preserve the existing index link if no new atlas file is needed.
- [x] In an environment cleared of PAIR_*, COUCH_* and ZELLIJ* with a short dedicated TMPDIR, run `make test-lua test-review`, `make test-runtimebundle`, `go test ./cmd/internal/artifactpath/...`, `make build` and `git diff --check`. Inspect every failure, including pre-existing ones that name new files.
- [x] Mutation-check core properties (break newline parity, remove source compare guard, bypass fence eligibility); tests must fail for their own asserted outcome. Record representative render/performance evidence and restore mutants via overlays/temporary copies, not tracked-file churn.
- [ ] Tick completed steps and record test evidence. Run `sdlc close --issue 426 --verified '<concrete evidence>'` for the mandatory fresh-context review; fix findings and record prevention rules in lessons. Publish through sdlc pr/merge when authorized.

## Approval and implementation entry

After approval, run `sdlc change-code --issue 426 --flow full --worktree=no`
before any test or production edit. Address the plan gate's findings, derive the
estimate only after plan-quality passes, and set it through the issue setter.
The boundary judge owns the code review; do not dispatch a redundant one.

## Revisions

### 2026-10-10 — fresh-eyes review: canonical wire compatibility

Supersedes the initial whole-string newline/delimiter codec composition, which
lost canonical slash parity. Preserve raw turn payloads and decode the `<br>`
rule before generic escapes; encode with one coordinated scanner. Specify
single-line canonical precedence and retain legacy multiline resolution. Add
independent 0–4 slash fixtures alongside round-trip properties. The reviewer
found no other Critical/Important issues in the draft.

### 2026-10-10 — plan gate PQ-1/PQ-2 and envelope refinement

PQ-1: introduce markers.scan completeness and diagnostic records with shared
code exclusions; parse_markers retains its successful-prefix legacy API. PQ-2:
replace test inventories with named risky functions and adversarial strategies.
PQ-3: bound compact projection to 1,000 lines/128 KiB with visible raw fallback
and a measured 50ms refresh target. These refine implementation/testing without
changing the approved user interaction. Operator approved the plan before the
gate. No implementation edits occurred before acceptance.

### 2026-10-10 — integration refinements

Keyboard help reads only review.lua; extend its source discovery to the new
comment modules so Enter and thread q are discoverable. The parser's fenced-code
exclusion is extended to ordinary tilde/indented fences so all compact consumers
honor the stated literal-code contract. Canonical writers escape backticks to
keep arbitrary editable turn content from manufacturing cross-turn code spans.
The fresh close review remains the single implementation review boundary.
