# Review workbench

An embedded nvim **document workbench** for agentic document review (issue #66):
a persistent agent (pair's session) applies edit records through
`pair review apply`, and an nvim review pane reloads the file and decorates the
round; the pane saves only for the operator's own actions and pokes the agent. The
agent is the producer **and the only git writer**: it creates/resumes the
`review/<slug>` branch and commits `docflow` rounds with the body `pair review apply`
returns. The contract between them is a small set of seam files + git commits.

## Modules (`nvim/review/`)

Rounds carry optional `Span.presentation` (`direct`, `highlight`, `propose`).
An absent span value inherits `Round.presentation`, then highlight. Direct spans
have no change decorations; highlight spans carry change ink, and proposal markers
retain their marker rendering. `round_watch.lua` and `reconstruct.lua` share the
fallback. `RoundBody` materializes each record's effective presentation so resume
keeps a legacy round's inherited setting. Go codecs reject recursive duplicate
keys and invalid owned fields while retaining opaque extension values through
round/artifact/record rewrites. Pair's Lua record codec retains additive fields.
The producer/Parley consumer fixture is
`cmd/internal/sessionsync/testdata/editorial-rounds/presentation.json`: source and
result hashes, valid byte positions, mixed explicit/absent modes, both legacy
fallbacks and a mixed commit/resume round-trip. Parley #419 pins this fixture
after landing; its consumer acceptance is independent of Pair's headless tests.

Pure core (run under `nvim -l`, colocated `*_test.lua`, `make test-lua`):

- `marker_codec.lua` — shared 🤖 delimiter escaping/unescaping used by both
  annotate and review markers. Keeps `🤖<selection>[]` safe when selected text
  contains marker delimiters such as `>`, `]`, or backslashes.
- `record.lua` — the `Record` `{old, occurrence, new, explain}` and its **one**
  JSON serialization in the agent commit body. Records carry `new_occurrence` (Nth
  match of `new` post-apply) so resume can re-anchor; `occurrence` (Nth `old` in
  base) and `new_occurrence` are never crossed.
- `reconstruct.lua` — pure `records, content, which → {highlights, diagnostics}`
  (0-based line ranges + explains). `which='old'` anchors by `occurrence`,
  `which='new'` by `new_occurrence`. The resume / from-commit render path; exports
  `nth_offset`/`line_of`.
- `markers.lua` (M2) — pure 🤖 review-request parser (ported from parley):
  `🤖<quoted>?(~strike~)?([user]|{agent})*` → marker records with `ready`/`pending`
  (last-section rule), excluding markers in fenced/inline code. The human's
  in-doc review requests; M3 highlights from it, M4's agent reads it. `spans_multiline`
  (#89 M1) derives multi-line-aware highlight spans (`{row,col,end_row,end_col}`)
  from the parser — a `🤖<…>` may cross rows (retiring the per-line `highlight_spans`);
  `MULTILINE_LINE_BUDGET` = 200 so a large multi-line quote still parses.
- `mode.lua` (M2/M4d) — pure pair-side UI metadata for the 3 human assistance
  levels: Generate, Edit, Proofread. Pair does not carry prompt prose for these;
  their meanings live in ariadne's `xx-fix` skill. `menu.lua` (M4c/M4d) presents
  those modes in the review pane with a one-round optional instruction buffer.

Integration seams (headless shell tests, `make test-review`):

- `apply.lua` — the decoration layer: `render` (records + content → highlights and
  diagnostics), `snapshot`/`apply_snapshot` (read/restore ranged extmarks
  ({line,end_line}) + diagnostics as two independent layers), `clear_at_line`,
  `clear_all` and definition placement. It edits no buffer text; its buffer apply
  was retired by pair#664 with the legacy handoff.
- `docflow.lua` — thin wrapper shelling `$DOCFLOW_BIN` (ariadne's `docflow`):
  `start`/`round --side`/`status`/`ship`. The review nvim no longer calls it;
  it remains as a contract test surface for the commit shape the **agent**
  produces (and as a future candidate for removal if no read use earns it).
- `round_watch.lua` / `oplock.lua` — the agent-round consumer and the operator
  lock (see *The applied-round path* below).
- `artifact.lua` — shared file-generation receipts for definition responses;
  consuming an observed response preserves a replacement published during
  processing.
- `projection.lua` (M2) — decoration coherence across undo/redo (ported from
  parley): per-buffer snapshots keyed by content hash; on undo/redo restore the
  matching snapshot, on a novel state capture the riding decorations. The
  `record_empty_for` guard keeps a prior round's styling when round-2's base is
  round-1's output. No more clear-on-each-apply.
- `restore.lua` / `restore_controller.lua` — pure activation/admission policy and
  pane-owned RPC transaction. The controller revalidates branch/document before
  effects, blocks pending work, retains buffers/undo and reconstructs the selected
  document from a resolver-captured snapshot on both initial opening and later
  activation. Neovim's normal buffer load supplies editor setup, not byte authority.
  A host that refuses the restore socket (an agent sandbox) still gets the review:
  the pane warns and publishes its open state without an endpoint, which
  `restore_client` reads as a legacy pane that cannot restore branches and says
  the host refused its socket (pair#450).
- `identity.lua` / `restore_client.lua` — resolver adapter and asynchronous draft
  activation client; verify acknowledgments before target publication or visibility.
- `recovery.lua` — private bounded unsaved-text snapshots for checkout mismatch;
  explicit recovery/discard, matching successful-save cleanup. Snapshots retain
  line-ending, BOM and encoding options as well as text.
- `document_bytes.lua` — shared byte/line conversion for activation, asynchronous
  refresh and exact content checks; preserves LF/CRLF, BOM and final-newline
  state, refusing unsupported text instead of silently converting it.
- `recovery_observer.lua` — coalesces edit events into asynchronous identity
  observations for proactive snapshots, and focus events for clean-buffer disk
  refresh on the matching branch. Refresh uses bounded file bytes captured by the
  resolver inside its branch/HEAD validation window, never a later checkout read.
  Admission rejects a checkout-owned index lock, changing index generation, or
  an index tree different from pinned HEAD. This deliberately also refuses staged
  changes until commit/unstage, because checkout can publish its index before HEAD.
  Late results must still belong to the captured activation and unchanged buffer;
  activation stop and exit cancel outstanding work. Write/apply/quit boundaries
  retain fresh authority checks.
- `poke_bodies.lua` — pure builders for the prose signals sent to the agent:
  review target prep, human turn finished, ship requested and definition requested.
- `readiness.lua` + `cmd/pair-review-readiness` (`cmd/internal/reviewcmd`, Go since
  #93 M3) — pure/classified git readiness for review-start: stop / track / resume /
  new / interact. The nvim proposes; the agent acts. The 4-case decision stays
  single-source in `readiness.lua`, invoked via `nvim --headless`; the Go helper
  gathers the git facts and emits JSON via `encoding/json`. The shared Go
  `identity.go` resolver adds read-only `pair review readiness --resolve <directory>`:
  exact current-slug round subjects and changed paths must identify one tracked,
  existing in-repository regular file; no slug-to-filename guess or first-hit choice.
  It bounds history to 10,000 matching commits / 8 MiB and a 2-second total deadline;
  incomplete reads refuse instead of claiming unique identity.
- `resolve.lua` — pure parley §5 accept/reject resolution for `🤖` marker chains;
  `nvim/review.lua` binds it to the review pane (`\a`, `\r`, `]m`, `[m`).
- `spinner.lua` — pure compact spinner/elapsed helper wired into the review pane
  statusline while the pane is awaiting the agent.
- `init.lua` — the orchestrator: `start` a review (`undofile` + round watch +
  reconstruct-on-open); on each applied round `on_applied_round` reloads the file
  (`:edit!`; an empty round only ends the turn), decorates via `apply.apply_snapshot`, records the projection and calls
  `after_applied_round` (the pane clears the awaiting spinner and redraws the active
  diagnostic); `human_round` saves only. It calls no `docflow` writer and never
  writes the file for an agent round.

## The loop (round = two docflow commits)

`:PairReview <file>` proposes a review target → the agent runs readiness prep
(track/new/resume/interact) and marks the target ready → Alt+c opens the pane.
The agent runs `pair review apply` → pair writes the file and the round artifact →
the pane reloads and decorates → the agent commits the agent round with the `Body`
from the apply reply. Human edits → Alt+Return saves and pokes
`human_finished` with the selected posture plus any one-round instruction → the
agent commits the human round and re-reviews using the standing `xx-fix` review
rules. Finishing a human turn clears stale agent-applied highlights and diagnostics;
the next applied round repaints current styling.
Diagnostics always render because they carry the agent's rationale. Change
highlights are only a direct-edit affordance: direct replacements highlight the
exact inserted `new` span, marker-rendered proposals (`🤖<old>{new}`, `🤖{new}`,
`🤖~old~`) do not get a redundant highlight, and empty direct deletions get no
fake highlight. Generative modes should use direct replacements when marker noise
would be too high, but deletion-only changes should remain visible as `🤖~old~`.
`:PairReviewShip` pokes the agent to run `docflow ship`; the pane does not shell
docflow. History lives in git (round commits + per-hunk explains in the agent commit
body); fine-grained undo lives in nvim's `undofile`. Unsaved checkout-mismatch
recovery uses the bounded snapshot lifecycle below. The doc must
be in a git repo.

### The applied-round path (pair#661 M2)

The agent applies every round through pair:
`pair review apply EDITS.json FILE --if-hash H --json` (`cmd/internal/reviewapply`) runs the shared engine (`sessionsync.BuildEdit`, the one `couch --session-sync apply` uses) with no Couch owner, ledger or registry.

- **Order:** a data-dir flock (`review-apply-<tag>.flock`), then the S1 operator lock (`FILE.lock`, exit 3), then the whole-file hash (exit 6), then the edit, then the round artifact (`review-round-<tag>.json`: `{file, round, body}`). The artifact is removed before the document moves, so a failed apply never leaves a stale one.
- **Commit body:** `body` is the agent's commit body, the round's spans as review-records with `new_occurrence`. The agent commits it verbatim with docflow, so resume reconstructs unchanged.
- **`Warning` in the reply:** the file was written but its round artifact was not. Commit `Body` now; the pane shows no highlight for that round.
- **The pane** (`round_watch.lua`) polls the artifact. It acts on an unseen round (`result_hash` plus `at`) of its own file only while the file on disk still hashes to it: it reloads (`edit!`) and decorates the spans in `review_diag`, and writes nothing for an agent round.
- **The empty round** (pair#690; producer side ariadne#453): a reply with no edits is still an apply, with records `[]`. The engine writes no file, only an artifact whose round keeps the hash (`base_hash` = `result_hash`, no spans, empty `body`). The pane reloads nothing and ends the agent's turn (spinner, restore pending block); the agent commits nothing.
  - A round it declines (unsaved operator edits) is warned about once and redelivered later.
- **The operator lock:** the pane takes `FILE.lock` (`oplock.lua`) on the first edit of the operator's turn and drops it on send and at exit, so an apply waits for the send.
  - The lock records `pid:`. A lock whose owner is gone (`ESRCH`) is reclaimed at review start and on the next take, so a crashed pane can't block the agent.
  - Readiness ignores the review's own untracked lock (`cleanBesidesLock`, porcelain `-z`).
- **Recovery:** `pair review round FILE --json` (`{Round, Body, Current}`) recovers a lost apply reply.
- **Under `presentation: propose`:** the same rules as couch apply (atlas/couch.md).
- **History:** the legacy handoff file, the landed artifact, nvim's buffer apply, the concurrent-edit reconcile and the apply gate were retired by pair#664 once ariadne#437 moved `xx-fix` to this path. `agent_round_rule_test.lua` defends that nvim never writes the reviewed buffer or file for an agent round.

## The review window (M3)

The document workbench in a live pair session — a **floating** nvim pane (the
proven scrollback/changelog pattern), opened on a file, alongside pair's agent+draft.

- `nvim/review.lua` — the pane init (`nvim -u nvim/review.lua <file>`): dofiles the
  review core + poke + markers, `review.start{}`, wires **Alt+Return = finish human
  turn** (`human_round` save + `human_finished` poke), renders 🤖 markers
  (`markers.spans_multiline` → multi-line `ParleyReview*` extmarks, re-rendered on
  TextChanged), supports accept/reject on any line a marker spans (`Alt+a`/`Alt+r`,
  with `\a`/`\r` fallbacks; multi-line-aware since #89 M1); when `Alt+a` is pressed outside a marker but inside an
  agent-applied highlight, it clears that highlight + matching diagnosis as an
  acceptance gesture. It inserts human comment markers (`Alt+q` bare marker or visual
  quote), exposes `:PairReviewShip` as an agent-owned ship request, plus marker
  navigation (`Alt+n`/`Alt+Shift+N`, with `]m`/`[m` fallbacks), sets review-local clipboard/cursor/search defaults
  (`unnamedplus`, blinking cursor, `ignorecase`+`smartcase` so `/foo` matches
  case-insensitively but `/Foo` stays case-sensitive — #101), writes the
  open-state file (line 1 = pane nvim `pid` for
  liveness, line 2 = the absolute doc path for the indicator, line 3 = JSON
  `{version:1,endpoint,token,session,context}` for the private same-pane RPC endpoint).
  The owning incarnation tears it down on `VimLeave`. Also defines `PairReviewToggle()` = hide-self (the case where Alt+c
  fires from inside the focused floating review pane). Pane-open no longer sends a
  separate "review workbench open" poke; the prep and human-finished pokes carry the
  workbench protocol context. The command line is hidden until `:` commands, and the
  pane statusline shows mode, `Alt+Return review · Alt+c draft · Alt+a/r accept/reject`,
  filename and line position in both idle and awaiting states. Normal-mode Esc
  dismisses internal floats without hiding the review; insert/visual Esc keeps
  its Vim behavior. Alt+c hides review and focuses the draft by pane ID from
  normal/insert mode. Alt+h derives review help from the
  mapping descriptions, with a drift test covering every local mapping. After a send
  it stays focused in the review pane and shows a 100ms braille spinner plus elapsed
  time until the agent's applied round lands.
- `bin/pair-review-open <file>` — validates + spawns the **full-screen** floating pane
  (`zellij run --floating --close-on-exit --name review --width 100% --height 100%`;
  percentage dims, not `tput`, which measured the wrong pane). A live singleton
  refuses replacement; document changes use the existing pane's activation RPC.
- `:PairReview <file>` (in draft `nvim/init.lua`, `complete=file`) — proposes the
  review target. It writes exact `$PAIR_REVIEW_TARGET_PATH` with `status=proposed`,
  runs `pair-review-readiness --prepare <file>` locally for deterministic
  start-up work (track file / create or resume `review/<slug>` / mark target
  `ready`), then sends the agent only a concise "review prepared; ack" message.
  It does **not** open the pane; Alt+c opens it once ready.
- **Alt+c** (`zellij/config.kdl`) — routed through the draft nvim like Alt+d
  (`MoveFocus Down` → `<C-\><C-n>` → `:lua PairReviewToggle()`), **not** a spawned
  shell pane. The draft's `PairReviewToggle()` (`nvim/init.lua`) branches on the
  current checkout's branch identity before target-cache or visibility decisions.
  A matching pane toggles; a different clean idle pane activates in the same Neovim
  process. Modified buffers and awaiting-agent or pending-definition work refuse
  switching. An old pane lacking RPC refuses
  safely; no live pane is killed. RPC has a five-second operation deadline and
  uncertain outcomes probe the same pane/incarnation rather than spawning a fallback.
  The target updates only after acknowledgment and checkout revalidation.
  Off a review branch, Alt+c prompts unless this conversation explicitly selected a
  document in a different repository: that verified peer selection remains usable,
  while a current review branch always wins. No-history first opening
  requires the current conversation's explicit preparation receipt matching canonical
  repo, branch, relative file and prepared HEAD; committed round history wins once
  present. A matching authenticated live pane can maintain that selection across an
  empty human round that advances HEAD without recording a path. Fresh sessions can
  restore committed branch context but never another
  conversation's transient requests or selection receipt. Session identity resolves
  inherited `PAIR_SESSION_ID`, then the shared inventory's established owner projection;
  fresh asynchronous IDs remain unscoped until the watcher publishes a durable binding.
  Restoration sends no automatic review request and makes no Git writes.
  `Alt+r` remains reject inside the review pane.
- `nvim/pair_poke.lua` — id-based agent poke: relative `move-focus` does NOT escape a
  floating pane, so it resolves the agent pane from `list-panes --json` and writes
  directly with `write-chars --pane-id <agent>` (the body framed as one bracketed paste by the draft send's own `frame`, #211) + `send-keys --pane-id <agent> "Alt Enter"`.
  The review pane keeps focus while the agent receives the poke.
- **review-mode bar** (`nvim/init.lua`, `do`-block; `_pair_review_bar` count source +
  `_pair_review_segment` cached segment) — while a review is open, the draft's
  **statusline** carries `-H < pos > +Q • 🪄 <Mode> • <file> •     🤖 A/H`: `H` is
  prompt history count, `Q` is future queue count, `A` is agent/robot review rounds,
  and `H` after the slash is human review rounds. `pair_compose_statusline` swaps the
  cached segment in for the rightmost cheatsheet and right-aligns the review-round count,
  so review mode is visible even when the pane is hidden. A 1.5s timer recomputes the segment (counts parsed from `git log` round
  subjects, **branch-scoped** to the active `review/<slug>` so other docs' shipped reviews
  don't leak in — `🤖 0/0` off a review branch / in M3 render-only; mode from
  exact `$PAIR_REVIEW_MODE_PATH`, defaulting to Edit) and triggers a redraw
  only on change; the hot render path never shells git. (This **supersedes** an earlier
  line-1 `=== review … ===` indicator — line 1 is the user's to edit. New draft-side
  review helpers live in `do`-blocks sharing `_G._pair_review` — init.lua is at Lua's
  200-local chunk ceiling.) The cross-process `review-<tag>.open` path is centralized in
  `nvim/review/seam.lua` (one fallback rule for writer + reader).
- **send menu + waiting cue** (`nvim/review.lua`, `nvim/review/menu.lua`,
  `nvim/review/spinner.lua`) — the exported `PairReviewPane.open_mode_menu(file)`
  API presents a mode selector plus a one-round optional instruction editor,
  then finishes the human turn with the selected mode/instruction. Its former
  `Alt+Shift+Return` binding is retired: that chord now invokes global
  right-terminal fullscreen directly from this editor, preserving its pane ID
  for focus restoration. There is no buffer-local map overriding the global.
  `Alt+Return` keeps the current mode and sends directly. Send and ship pokes mark the
  pane as awaiting the agent, displayed by the statusline spinner until the next
  applied round clears it.
- **inline definitions** (`nvim/review/define.lua`,
  `nvim/review/definition_seam.lua`, `pair review definition`; #112) —
  visual-select a term in the review pane and press `Shift+Alt+d` to ask the existing
  pair agent for a concise definition. The pane writes
  `review-definition-request-<tag>.json` with the selected term, byte range, file,
  request id, stripped document text in `.context`, and activation identity in
  `.review_context` (the text strips only the managed definition footer); then it pokes the agent to answer by running
  `pair review definition --term <term> <request-id> <definition>`, which writes
  `review-definition-result-<tag>.json`, echoing the activation in `.context`.
  The pane checks both request ID and activation before consumption. On result, the pane rewrites the selected
  text to `term[^id]`, appends or updates a managed final `---` footnote block,
  and rehydrates diagnostics/highlights from the durable footnotes. Definition
  highlights live in a dedicated `review_define` extmark namespace but diagnostics
  share the review diagnostic namespace, so the existing cursor-scoped diagnostic
  display works; projection snapshots include the definition extmarks so undo/redo
  preserves exact column spans. Re-defining the same term updates the footer
  without duplicating the inline reference.
- **durability** (`nvim/review.lua`) — save-on-`VimLeave` runs only on the
  matching checkout. A branch mismatch suspends save/send/ship. Unsaved text
  on exit goes to private `review-recovery/` beside the open-state file, keyed by
  repository/branch/document. Return to the original branch, then use
  `:PairReviewRecover` for an undoable restore or `:PairReviewDiscardRecovery` to
  discard explicitly. A matching successful save removes only the snapshot this
  buffer/incarnation wrote or restored. Prior-process snapshots cannot be silently
  overwritten. Storage is bounded to 32 entries and 8 MiB per snapshot; capacity or
  oversized content refuses preservation with a recovery instruction, never eviction.
  The apply gate that deferred a round while the operator was mid-edit was retired
  by pair#664; the operator lock now makes the agent's apply wait instead.
- **docflow degradation** (`nvim/review/docflow.lua`) — missing `docflow` still has
  a calm contract-test path, but the review pane no longer shells docflow at runtime.
  Round commits are agent-side. See `workshop/targets/review-protocol.md` for the
  full agent↔nvim state machine.

The agent pane is pair's **existing** agent — ordinary chat still works; the shared `xx-fix` skill
owns its producer behavior (ariadne#121; activation-scoped effects ariadne#268).
It echoes the exact request context (canonical absolute repo, exact branch,
repo-relative file, opaque activation) to the pair commands that take `--context`,
and checks live pane metadata plus the checkout immediately before human/agent round
or ship Git effects. Record-body encoding is unchanged.

## State

M1 (contract + history spine), M2 (consumer-half port), M3 (review window + live
smoke), M4a (nvim writes no git), and
M4a' pair-side review-start/resume are implemented and headless-tested. M4b adds
pair-side accept/reject + marker navigation, fulfill-or-punt default Edit posture,
and ship request. M4c adds pair-side mode display/menu and the awaiting-agent
spinner. M4d starts workflow-detail tuning with one-round instruction menu polish,
three-mode assistance semantics, minimal-marker Edit behavior, exact-span
direct-change highlighting, and diagnostic-only marker proposals. Fact-check is
not a mode; it is requested through the one-round instruction field or agent
prompt and handled by the review agent's skill workflow.

**Concurrent edits (#89, pair#661, pair#664)** — #89 kept multi-line `🤖<…>`
markers and matching-context `VimLeave` durability with mismatch recovery. Its
reconcile engine and apply gate were retired by pair#664: the operator lock
(`FILE.lock`) now serializes the agent's apply against the operator's turn.

The authoritative real-agent instructions live in ariadne's shared `xx-fix` skill,
including ariadne#268's activation checks. `review-applied-round-test` drives real
`pair review apply` rounds in integration tests; live smoke additionally checks the
persistent agent's recognition and round-trip behavior.

## Tests

- `make test-lua` — `record`, `reconstruct`, `markers`, `seam`, `mode`, `poke_bodies`,
  `readiness`, `resolve`, `restore`, `restore_controller`, `restore_client`,
  `recovery`, `spinner`, `menu`, `round_watch`, `round_authority`, and
  `agent_round_rule` (nvim never writes the reviewed buffer or file for an agent
  round: `on_lines`, `BufWritePre` and Lua write spies, with a positive control on
  the operator send) (pure/headless).
- `make test-review` — `docflow` (+ hermetic `tests/lib/fake-docflow.sh` and a
  gated smoke against the real ariadne `docflow.sh`), `apply` (render, snapshot,
  `apply_snapshot`, `clear_at_line`), `applied-round` (real `pair review apply`
  rounds end to end), and `projection` (undo/redo coherence + riding + round-2 idempotence); M3 adds
  `poke` (id-based agent poke, no relative move-focus), `window` (:PairReview +
  pair-review-open + review.lua: keymap/state/markers + Alt+Return round-trip; an
  applied round clears the spinner),
  `toggle` (mode-aware branch, explicit show/hide, no toggle-floating-panes),
  `review-readiness-cli` (quoted git facts stay valid JSON), `resume`,
  and `review-branch-restore` (real branches and same-process activation, with a
  real `pair review apply` round preserving CRLF/BOM).
- `review-resume-isolation-test.py` (#591) runs the complete resume fixture with
  foreign inherited `PAIR_HOME` and `PAIR_BIN`, requiring the checkout executable
  and unchanged caller artifact sentinels. The resume fixture reuses
  `ReviewTestEnvironment` before its Git/editor subprocesses; its original
  legacy, scoped/ambiguous and read-only assertions and deadlines are preserved.
- `review-observation-diagnostics-test.py` (#586) validates exit/status and
  complete wanted identities before forwarding, and tests incomplete fresh
  observations, real checkout refusal, and a successful restoration. The
  fixture-only `tests/lib/review_observation_trace.lua` observes existing wait
  and callback results in memory, retaining bounded identity metadata and
  request/call ordering without copying document snapshots. Failure output
  includes both observations before editor teardown; setting the fixture's
  `PAIR_TEST_OBSERVATION_REPORT` retains a successful probe report too. This
  instrumentation changes no production resolver deadline or admission rule.
  `tests/lib/review_observation_gate.lua` (#623) controls only delivery of an
  already-completed asynchronous resolver result. Fixture phases capture
  their ID boundary atomically with arming, then retain start, result, release,
  settled and byte-application evidence. No artificial child sleep spends the
  native 2500ms resolver budget. The gate retains at most one held result,
  bounds metadata to32 entries and discards callbacks on editor teardown;
  timeout results pass through unchanged. Its lifecycle and forced arm-order
  tests run through the diagnostics command in the normal suite. There is no
  production hook.
- The observation ordering fixture (#662) uses the test-only stateful resolver
  in `tests/lib/review_resolver_fake.lua` and its Python wanted-observation
  transport. Both read a shared catalog of the fixture's two known commits,
  actual loose HEAD/ref files and current document bytes. Async results capture
  identity and bytes before callback release; unknown refs or heads refuse.
  UI assertions count synchronous waits rather than requiring a host to finish
  within 250 ms. Its fake unit checks 8/16 query costs and bounds retained
  metadata. Real Git identity collection remains covered by `reviewcmd` tests;
  `review-resolver-native-deadline-test.lua` separately verifies native Neovim
  sync/async timeout outcomes with the unchanged 2500 ms deadline. Native empty
  stderr now retains an explicit timeout/exit refusal reason; authorization
  still rejects unknown observations without changing edits or activation.
- `review-branch-restore-diagnostics-test.py` (#620) and its landed-artifact
  timing diagnostics were deleted by pair#664 with the landed artifact.

Review marker navigation is buffer-local and normal-mode only. Couch's existing
client-focus probe recognizes the review role and forwards Alt+n unchanged;
Ctrl+Alt+n still relaunches from review, and draft/switcher relaunch is unchanged.
Alt+Shift+N overrides agent restart only in the review buffer's normal mode.

The draft statusline includes `Alt+c review` in its normal hints and while a
review is open. Both review return keys hide the overlay and focus the draft.

Scrollback and changelog exits use that same return-to-draft operation after
annotation emission, hiding the floating layer instead of revealing a review
underneath. The review stays alive and can be shown again with Alt+c.

Recovery write failures leave buffers modified and block ordinary non-bang quit.
Explicit `:qa!` retains Neovim's discard semantics if storage fails; callback
errors cannot prevent that forced exit.


## Compact comment threads (#426)

The pane uses `review/comment.lua` for activation-owned rendering and window
options. `markers.scan` supplies successful markers, completeness and malformed
ranges to `comment_view.layout`; the same cached geometry drives conceal,
cursor snapping and Enter targeting. Earlier turns collapse to role-colored
brackets around an ellipsis; anchors and the final human turn remain visible.
Malformed ranges, code examples and legacy multiline reconciliation markers
stay raw. Above 1,000 lines/128 KiB, the activation reports a raw-view fallback.
Cursor movement reads the cache; text edits refresh it, including insert mode.

`comment_codec` interprets canonical `<br>` slash parity directly from each
section's raw payload. Generic delimiter unescape cannot run first: it erases
that parity. Only single-line turn-derived text gets newline decoding; anchors
and legacy multiline resolution retain their previous behavior. `apply.lua`
uses that same codec when shortening displayed replacements.

`comment_thread` converts raw markers to editable `💬:`/`🤖:` lines and owns an
encapsulated save/close state. `comment_float` runs its effects in an acwrite
scratch window. Enter opens it; `:w` saves to the source buffer, `q`/`:x` saves
and closes, `:q!` discards. The entire thread is editable. A range extmark plus
exact byte comparison protects the opened instance from concurrent edits;
refusals keep the float dirty. Save is an ordinary undoable human edit and sends
no agent message; Alt+Return in the pane saves the document and submits it.
Forced teardown rescues unsaved text to the unnamed register, then removes the
float, scratch buffer, tracking mark and autocmds. Deactivation restores window
options and Enter's previous mapping.

Tests: colocated comment codec/thread/view/float/attachment tests,
`tests/review-comments-test.sh` (real PTY painted cells and thread controls), and
`tests/review-controls-test.sh` (saved thread through the human round).
