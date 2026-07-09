# Lessons

## Async buffer requests need live anchors, not saved coordinates

Pair review definitions originally stored the selected line/column range while
the agent produced an answer. If the user inserted text before the selected term
before the result arrived, the response applied to stale coordinates and inserted
the footnote reference into the wrong text.

**Rule.** Any Neovim request that crosses an async boundary and later mutates the
same buffer must anchor the target with an extmark (or re-locate/validate the
target from content) before applying the result. Raw row/column pairs are only a
snapshot. Add an integration regression that mutates text before the target while
the request is pending, then verifies the result follows the target or aborts
cleanly. Caught in #000112 close review.

## Generated review sidecars must stay bounded

`sdlc close` writes a review sidecar, and that sidecar becomes part of later
diffs. If it stores the full raw prompt/transcript, it can bloat the reviewed
diff and carry whitespace-sensitive embedded patches.

**Rule.** Keep committed review sidecars to the durable facts: verdict, window,
findings, verification, and resolution. Avoid committing full prompt/diff
transcripts unless the generator normalizes them and they remain small enough
for future review prompts.

Caught while closing #000108.

## Path precedence contracts need explicit divergent-env tests

#90's embedded runtime implementation documented extraction under
`$PAIR_DATA_DIR/runtime/<digest>/pair-home`, but the first OS-backed
implementation only used the XDG/home resolver. The copied-binary smoke unset
`PAIR_DATA_DIR`, so the bug survived until boundary review tried
`PAIR_DATA_DIR` and `XDG_DATA_HOME` with different roots.

**Rule.** When a feature promises environment-variable precedence, add a test
where the higher-priority and fallback variables are both set to different
directories, then assert the selected path. Also include every Go source file
that can change build output in Make prerequisites; a generated or embedded
artifact path should have a dependency test or an explicit review checklist
entry. Caught in #000090 boundary review.

## Lua patterns: `\0` is empty-position match, not NUL byte

The unescape function in `nvim/scrollback.lua` first attempt used a
placeholder dance: `s:gsub('\\\\', '\0')` to swap `\\` for NUL,
then `gsub('\\(.)', '%1')` to strip remaining `\X`, then
`gsub('\0', '\\')` to restore the NUL → `\`. The result was
absurd: `unescape("plain")` returned `\p\l\a\i\n\` — the NUL pattern
matches between every byte (empty-position match), not the NUL byte
character. Each "match" inserted a `\` between every char.

**Rule.** When you need to match a literal NUL byte in a Lua pattern,
use `%z` or wrap as a character class `[%z]`. But the cleaner answer
is usually to skip patterns entirely for character-by-character
walks: a tiny while-loop with `s:sub(i, i)` is unambiguous and avoids
all the pattern-syntax footguns. Caught in #000018 review.

## Escape on insert, scan-with-parity on extract — for delimited markers

When user-supplied content is embedded in a delimited container
(e.g. `🤖<X>[Y]`), and X or Y can contain the delimiter chars,
the choice is "escape at insert + unescape at extract" vs "find
the closing delimiter cleverly." The first attempt at `🤖<X>[Y]`
parsing tried the latter — find first `>`, peek for `[`, give up
otherwise. Result: any selection with `>` was silently dropped on
extract, since the user couldn't tell the marker had been written
malformed.

**Rule.** Escape the delimiter chars in user-supplied fields at
insert time; have the parser walk byte-by-byte counting backslash
parity to find the *next unescaped* delimiter; unescape the
extracted content. The escape→walk→unescape chain handles every
delimiter-collision case uniformly, including `\\>` (literal `\`
followed by `>`). Don't try to be clever with "find first `>[`
adjacent" patterns — they fail when X contains `>[` literally,
and the failure mode is silent data loss. Caught in #000018 review.

## Shared delimiter codecs beat subsystem-local marker parsing

M4b's review pane added `Alt+q` visual wrapping as `🤖<selection>[]` but initially
embedded the selected text raw, even though annotate already had delimiter escaping for
the same marker family. A selection containing `>` or `]` could truncate the parsed marker
and make accept/reject leave stray syntax in the document.

**Rule.** When a second feature writes the same delimited marker format, reuse or extract
the existing codec before adding parser/writer code. Add tests for delimiter collisions
(`>`, `]`, backslash) at the write path and the consume path. A parser unit test alone is
not enough; the UI wrapper that inserts the marker must also be covered. Caught in #000066
M4b review.

## Shell scripts should use JSON builders, not `printf` JSON

`pair-review-readiness` originally printed JSON with `printf` and unescaped string fields.
A review branch named `review/a"b` produced invalid JSON, even though all the boolean
fields were correct.

**Rule.** In shell seams that emit JSON, use `jq -n --arg/--argjson` (or an equivalent
structured encoder) for every field. Do not hand-build JSON with `printf` unless every
string field is impossible by construction — and then document why. Guard it with a test
using quotes in a branch/path/name. Caught in #000066 M4b review.

## `#table` is 0 on string-keyed tables — never use it for ID generation

Adding nvim/scrollback.lua's hl-group cache: `local name = 'PairScrollback_' .. (#hl_cache + 1)` was meant to give each new (state→hl-group) entry a unique numeric suffix. `hl_cache` is a string-keyed dict (cache key is `(fg|bg|attrs)`), and Lua's `#` on a non-array table returns 0. Result: every group resolved to `PairScrollback_1`, `nvim_set_hl(0, "PairScrollback_1", def)` overwrote on each call, and all extmarks ended up sharing whatever the last-written attrs were. Caught only by an end-to-end test that checked extmark hl_groups against expected fg/bg ints.

**Rule.** When you need monotonic IDs in Lua, use an explicit counter (`local counter = 0; ... counter = counter + 1`). Do not use `#table` unless `table` is provably array-shaped (`{[1]=..., [2]=..., ...}`). The bug is silent — `nvim_set_hl` doesn't error on overwrite, it just wins-last. Filed during #000017 M4.

## Empty fields in delimited parsing — `[^;]+` drops them; semantics may differ

ECMA-48 SGR semantics: an omitted field is `0` (reset). So `\x1b[;1m` = "reset; bold". The first SGR parser pass used `params:gmatch('[^;]+')`, which silently skips empty fields — `[;1m` produced just `1` (bold), and any standing fg/bg/decoration leaked through. Caught in code review of #000017 (no real input from pair-scrollback-render's output would have triggered it, but it's a correctness footgun for any future caller pointing the viewer at non-pair-rendered ANSI).

**Rule.** When the protocol says "empty field has meaning," parse with `([^;]*);` on a `string + ';'` so the trailing-delimiter trick yields every field including empties. Generally true for any delimiter-separated format where omission has semantic value (CSV with empty cells, env-var lists, SGR, etc.).

## Sparse data structures: iterate by index, not by `.keys()`, when count must be exact

pyte's `screen.buffer` is a `StaticDefaultDict` — accessing `buffer[y][x]` lazily creates a default Char, but `buffer.keys()` only contains rows that were *written to*. The renderer originally did `for y in sorted(screen.buffer.keys())`, which silently dropped trailing blank rows when the agent cleared and paused mid-redraw. That shifts every subsequent line number — directly breaking the feature's core promise that `:880` lands where zellij showed line 880. Caught in code review of #000017.

**Rule.** When iterating over a sparse-by-design structure where every slot has a logical existence (even if unwritten), use `range(0, total)` and let the structure's `__getitem__` materialize defaults. `.keys()` is only correct when "absent" really means "doesn't exist." Same shape applies to anything with lazy materialization: defaultdicts, JS Maps with default fallbacks, sparse arrays.

## Atomic write for files a feature can race on its own

`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.

**Rule.** Any output file that a user-triggered keybind (or any concurrently-fireable mechanism) writes to should use the tempfile + atomic rename pattern. The cost is one extra file path; the gain is that readers see only "old complete file" or "new complete file," never "torn file." Apply uniformly even when a race is unlikely — discipline reduces the cognitive load for future readers.

## Verify zellij action and flag names against the installed version

Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:

- Used `TogglePaneFullscreen` for the Alt+u bind. The actual action name in zellij 0.44.1 is `ToggleFocusFullscreen`. Caught by `zellij setup --check --config-dir <pair>/zellij`.
- Used `--layout PATH --session NAME` to "create a new named session with this layout." Zellij's actual semantic: when `--session` is set, `--layout` means "add as tab to that session" and errors if the session doesn't exist. The right flag is `--new-session-with-layout` (`-n`).

**Rule.** Before writing zellij KDL or invoking the zellij CLI:

1. Run `zellij setup --dump-config` to see the canonical action names used in default keybinds.
2. Run `zellij --help`, `zellij attach --help`, `zellij setup --help` against the installed version, and read the flag descriptions in full — they have non-obvious conditional semantics.
3. Always validate config and layout files with `zellij setup --check --config-dir <dir>` and `zellij setup --dump-layout <path>` before committing.

The verification tools are cheap and authoritative. Memory of "I think it's called X" is not.

## Stage content edits before `git mv` when closing an issue

Closing an issue means (a) editing the file (`status: done`, plan checkboxes), then (b) moving it to `workshop/history/`. Done in that order with `Edit` then `git mv`, the rename gets staged but the unstaged content edits do *not* — they stay in the working tree. `make issue-sync` only stages `workshop/issues/`, so the edits silently miss the commit. End state: history file with stale `status: working`.

**Rule.** When closing an issue:
1. Edit the file in place under `workshop/issues/` and `git add` it (or use `git add -u` after editing).
2. Then `git mv` to `workshop/history/` — git carries the staged content into the rename.
3. Or simpler: `git mv` first, edit second, `git add` the new path.

After running `make issue-sync` on a close, verify with `git show HEAD:workshop/history/<file> | grep status:` that the committed file actually has `status: done`. Don't trust the rename alone.

## On cancel, restore the prior visible state

When a confirmation prompt or interactive flow is dismissed, the cancel path must put the UI back exactly how it was — not just "do nothing." Issuing a prompt via `nvim_echo`/`getchar` (or any flow that paints over a region: cmdline, statusline, floating windows, virtual text, highlights) leaves that region in the prompt's state. The proceed branch usually triggers a redraw incidentally (state changes → statusline refresh → cmdline cleared). The cancel branch does not, so the prompt residue lingers until the next user input.

**Rule.** For every interactive surface, the cancel path is responsible for the same restoration the proceed path gets for free:

- Prompts that overdraw the cmdline/statusline → call the same redraw/refresh helper the success path calls (e.g. `refresh_statusline()`), not just `return`.
- Operations that mutated buffer text/cursor/window before asking for confirmation → snapshot first, restore on cancel.
- Highlights, virtual text, floating windows added as part of the flow → tear them down on cancel just like on success.

Treat cancel as an active branch with cleanup duties, not an early return. If you find yourself writing `if ch == 'n' then return end`, ask: what did the proceed branch do that I'm now skipping, and is any of it visual cleanup that cancel also needs?

## Transcript summarization must bias toward USER turns, not a flat tail

`cmd/pair-slug` (#000027) summarized "what is this session about" by feeding
the last N text-bearing transcript turns to a small model. On a tool-heavy
session that window is almost entirely assistant narration: a real Claude
transcript had ~16 genuine user prompts vs ~200 assistant entries (most
`user` entries carry only `tool_result` blocks, correctly dropped as
text-less). Measured: the last 10 text-bearing turns were 10/10 assistant,
0 user. So the slug tracked what the agent was *saying*, not what the user
*asked for* — the orientation signal was pushed out of the window. The unit
tests passed because their fixtures used only text-content messages, never
the dominant `tool_result`-only user shape — green tests masked the bug.
Caught in #000027 M1 review.

**Rule.** When sampling a conversation transcript to infer user intent:
- Don't take a flat tail of turns. Guarantee a minimum number of recent
  *user* turns are in the window (extend backward until satisfied, capped).
- Model test fixtures on the *real* transcript shape, including
  `tool_result`-only user entries and any sidechain/summary types — not the
  clean text-only case. A fixture that can't reproduce the bug can't guard
  against it.

## `gofmt -w <dir>` reformats files you didn't touch

Running `gofmt -w cmd/pair-wrap/` to format M3's edited `main.go` also
rewrote four pre-existing `*_test.go` files (struct-field alignment) that the
milestone never touched, staging unrelated churn into the commit. Caught at
`git status` review before commit; reverted with `git checkout -- <files>`.

**Rule.** Format only the files the change actually touches: `gofmt -w
path/to/file.go` (or `gofmt -w $(git diff --name-only '*.go')`), not the whole
package directory. If a dir-wide gofmt lights up files outside the change,
revert them — don't smuggle repo-wide reformatting into a feature commit.
Caught in #000027 M3.

## Dogfooding a Go-binary change needs `make install`, not just `make build`

M3's pair-wrap trigger "didn't fire" on restart. Trace: pair-slug worked in
isolation, but the running `pair-wrap` (pid via `pair-wrap-pid-<tag>`, binary
via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
earlier — the *installed* copy, with no spawn. I had only `go build -o bin/…`;
the layout (`zellij/layouts/main.kdl`) execs `pair-wrap` by bare name and the
pane's PATH resolved `~/.local/bin` first.

**Rule.** `bin/` is the repo build; `~/.local/bin` (via `make install`) is what
actually runs in a live pair session. To dogfood a change to a Go binary
(pair-wrap/pair-slug/…): `make install`, *then* restart pair. Verifying with
`bin/<binary>` alone proves nothing about the running session. When a "live"
change seems inert, confirm the running binary: `lsof -p $(cat
$PAIR_DATA_DIR/pair-wrap-pid-<tag>) | awk '$4=="txt"{print $NF}'`. Caught in
#000027 M3 dogfood.

## Queue items: resolve by filename key, not display index, across a mutation

Sending from a future-queue slot (`+N`) while the draft `*` was non-empty left
the sent item in BOTH the queue (`+N`) and history (`-1`). Root cause:
`send_and_clear` resolved the item to remove via `queue_key_for_n(nav.pos.n)` —
the *display index* — but the new "park the draft into the queue first"
(`push_front`) step shifts every index by one. Resolving by the stale index
then removed the wrong file (or `nil`), so the actually-sent item was logged to
history but never deleted from the queue → duplication.

**Rule.** A `+N` display index is only valid against the queue snapshot it was
read from. The moment any queue mutation (`push_front`/`push_back`/remove) can
intervene, capture the item's **filename key first** (`queue_key_for_n(n)` →
`NNNNNN`), then mutate, then remove by that stable key. Keys don't move on
insert; indices do. Verified the duplication via a headless driver
(`nvim --headless -u nvim/init.lua` + `maparg().callback`) before fixing, and
guarded it with `tests/queue-send-test.sh` (`make test-queue`).

## strings.ToLower can change byte length — don't cross-index a folded copy

`promptShape` matched against `strings.ToLower(visible)` but then sliced the
**original** `visible` at the match offset. `ToLower` is not length-preserving
(e.g. `Ⱥ` U+023A, 2 bytes → `ⱥ` U+2C65, 3 bytes), so on agent output with such a
rune the offset exceeded `len(visible)` and panicked the slice. The panic was
swallowed by `handleChunk`'s `recover`, but that `recover` wraps the whole
detect block, so OSC-notification + bell handling were silently skipped for that
chunk — a diagnostic-only feature altering proxy behavior. Surfaced in #000045
M1 review (C1).

**Rule.** If you compute a byte offset in one string, slice the *same* string —
never a transformed copy whose length can differ. For case-insensitive matching
where you need offsets back in the original, use a **length-preserving** fold
(ASCII-only `asciiFold`) and clamp slice indices defensively. Add a multibyte
test case (`Ⱥ`/`İ`/`Å`) — ASCII-only tests can't catch this.

## jq slurp (`-s`) over a JSONL file aborts on one bad line

`doctor.sh` read the flight recorder with `jq -rs '…'`, which parses the whole
file as one array — so a single malformed/partial line (a writer crashing
mid-line; O_APPEND only guarantees atomicity below PIPE_BUF) made jq error and,
under `set -euo pipefail`, killed the script. The operator got a jq stack trace
and zero diagnostics exactly when they needed the tool. Surfaced in #000045 M1
review (I1).

**Rule.** Parse append-only JSONL **tolerantly**: pre-filter with
`jq -R 'fromjson? // empty'` to drop bad lines, then slurp; and `|| true` the jq
calls so a parse hiccup can't trip `set -e` in an always-exit-0 diagnostic.
Guard it with a fixture containing a deliberately truncated line
(`doctor/doctor_test.sh`, `make test-doctor`).

## One schema, three languages → pin it with a golden test, not three unit tests

The flight recorder is emitted from Go (`cmd/internal/adapt`), shell
(`bin/lib/adapt-log.sh`), and Lua (`nvim/adapt.lua`); `doctor.sh` only works if
all three produce byte-identical lines. Per-emitter unit tests can't catch the
three drifting apart. Three real divergences surfaced: (1) Go's `encoding/json`
HTML-escapes `<>&` by default — jq and `vim.json` don't; fixed with
`SetEscapeHTML(false)`. (2) field order — Go marshals struct order, jq preserves
object-construction order, Lua needed manual assembly to match. (3) detail
truncation — Go is rune-safe, Lua's `string.sub`/`#` are byte ops and split
multibyte runes (invalid UTF-8). Surfaced in #000045 M2 review.

**Rule.** When N emitters must share a wire format, add ONE golden fixture and
assert every emitter reproduces it byte-for-byte (normalizing only genuinely
variable fields like timestamps). `tests/adapt-schema-test.sh` + the Go
`TestGoldenMatchesFixture` leg do this. Watch the three usual divergence
sources: default escaping, key order, and multibyte/locale-dependent length caps.

## A "momentary mode" flag leaks unless every popup-swap path clears it

#000049 added `spell_popup_active` so bare digits pick a `z=` spell suggestion
instead of inserting. The first cut cleared it only on `CompleteDone` /
`InsertLeave`. But the `TextChangedI/P` autocmd runs `word_complete`, which can
fire a *new* `vim.fn.complete()` (swapping the menu) without any `CompleteDone`
for the in-place replacement — so the flag stayed true under a non-spell popup
and a digit would mis-pick. The naive fix (clear the flag in `run_completers`)
risks killing the feature: if showing a popup re-entered that path it'd clear
the flag before the user could pick.

**Rule.** When a boolean marks "the visible popup is mode X", clear it at the
*exact* sites that replace the popup with a not-X menu (here: right before
`complete()` in `path_complete`/`word_complete`), not in a broad event handler
that might also fire for X itself. Verify the X-shower doesn't trip the clear —
`z=`'s own `complete()` is `noinsert`, so it fires no `TextChanged` and the flag
survives. Prefer clearing at the state-transition source over the event funnel.

## Doc-sync: sweep ALL prose when a design detail moves, and verify claims against code

Shipping #53 hit the merge `atlas/README sync` judge three times because docs
drifted from the code. Two distinct failures:

1. **A relocated UI detail left stale pointers in many files.** Moving the
   changelog spinner from the winbar to a bottom virtual line (one commit) left
   "winbar spinner" in five places — `atlas/architecture.md`, the issue Spec, the
   plan, the `changelog.lua` docstring, and (implicitly) the README. The merge
   judge gates only `atlas/` + `README.md`, so it caught those one at a time
   across re-runs; the Spec/plan/code-comment copies it does *not* gate drifted
   silently. **Rule:** when you rename or relocate a behavior/UI element
   mid-implementation, `grep -rn '<old-term>'` across **atlas + issue spec + plan
   + README + code docstrings** in one pass and fix every hit — don't let the
   merge judge find them serially.

2. **A doc claim overstated the code.** The atlas/spec said the distiller uses a
   "quality/capable-tier model"; the code passes no `--model`, so it falls back to
   `DefaultModel` = the *same small model the slug uses*. **Rule:** doc claims
   about *behavior* ("uses model X", "runs in parallel", "caches Y") must be read
   off the code, not the original intent — aspiration in a spec silently becomes a
   false statement once the implementation takes the simpler path.

Also: `sdlc close --issue N --milestone Mx` is the **no-auto-review** escape; the
reviewed milestone close is `sdlc milestone-close`. Using `close --milestone`
ticks the box without dispatching the boundary review or emitting the
`Review-Verdict:` trailer the issue-close gate then requires — leading to a
restart. Use `milestone-close` for a reviewed boundary.

## #58 — feature removal + anchor semantics (boundary review caught both)

1. **Removing a feature: grep EVERY test layer, not just the unit tests.** Dropping
   the change-log date headers, I updated the Go + lua assertions but missed the
   shell **smoke** test's `grep -q '^## '` ("assert a header exists"), so `make
   test` went red while `go test ./...` was green. **Rule:** when you delete output
   a feature produced, grep the whole `tests/` tree (and `*.sh`) for assertions on
   that output — shell/e2e tests don't show up in `go test`.
2. **Close evidence must name the suite that actually gates.** My `--verified` said
   "go + lua + test-statusline green" — true, but it never ran `test-changelog`
   (the smoke), which was the one that was red. **Rule:** the VERIFIED line must
   cover `make test` (or name each suite incl. the e2e/smoke), not a convenient
   subset; a claim that omits the failing suite is how a red build ships.
3. **An anchor/cursor tracks POSITION, not whether the payload changed.** I gated
   the change-log anchor advance on `newLog \!= priorLog`, so a turn that distilled
   to no textual change left the turn count behind → every later press re-ran the
   model. **Rule:** "processed up to here" and "the output changed" are different
   facts — advance the position marker when you've consumed the input; gate only
   the user-visible side effect (the notification) on an actual change.

## #60 — a stuck headless-nvim boot hangs the whole suite (boundary review: §4/§8)

1. **A buffer-mutating headless driver must `qall!`, not `qall`.** A driver that
   modifies its buffer (`nvim_buf_set_lines`) then ends in bare `vim.cmd('qall')`
   hits `E37: No write since last change`, refuses to quit, and `nvim --headless`
   blocks in its main loop **forever** — even with stdin=`/dev/null`. One such
   driver hung `make test` for 12m54s and leaked week-old nvim corpses. **Rule:**
   any headless driver that mutates a buffer ends in `qall!`; the hazard is latent
   across drivers — audit *every* sibling, don't fix only the one that bit.
2. **Never run a subprocess boot unbounded in a test suite.** Bound it with a
   timeout watchdog that fails loud (kill + exit 124 + diagnostic naming the issue),
   and don't `>/dev/null 2>&1 || true` it — that swallows both the hang and the boot
   error (`tests/lib/run-headless.sh`). Reproduce a suspected hang *streaming*, not
   through `… | tail`, which buffers until EOF and makes a progressing run look
   frozen.
3. **When a fix removes the only trigger of a safety path, pin that path with a
   fixture.** Once `qall!` lands, a green `make test` never exercises the watchdog's
   timeout branch — so the contract is pinned directly with a deliberate-hang
   fixture (`tests/run-headless-test.sh`), else the safety net ships unproven.

## #64 — confirm the file you're fixing is actually tracked (a one-line fix exposed a lost-source regression)

1. **Before editing to fix a bug, confirm the target is git-tracked — `git
   ls-files --error-unmatch <path>`.** The #64 prompt fix was a 3-line edit to
   `bin/pair`, but `bin/pair` turned out to be gitignored AND untracked: a normal
   commit/PR would have committed the atlas + issue edits and **silently dropped
   the actual code change**. **Rule:** when a fix lands in a file under a
   blanket-ignored dir (`bin/`, `dist/`, generated trees), check tracking first; a
   green local edit that isn't in `git status` never ships.
2. **A base-layer `propagate-base`/weave sweep can `git rm` a leaf's OWN source.**
   The cutover (`90c0c6c` "ariadne#107: propagate-base") deleted pair's 15 bin/
   shell scripts (3588 lines) from `main` tracking — they lived under a blanket
   `bin/` ignore (for built Go binaries) so the sweep treated tracked-but-ignored
   source as disposable. No source survived anywhere (not in the substrate, not
   woven by a manifest). **Rule:** after any weave/propagate-base run, verify
   critical dirs still track their source (`git ls-files bin/ | wc -l`); the very
   next ariadne commit (#109, dirty-tree precheck) confirms this sweep is hazardous.
3. **A dir holding BOTH source and build output should use explicit negations, not
   a blanket ignore.** `bin/` had both shell scripts (source) and Go binaries
   (built). The fix: `bin/*` + `!bin/<script>` negations — binaries stay ignored
   (safe default: a new build artifact is never committed by accident) while source
   is provably tracked. A blanket `bin/` relied on "gitignore doesn't untrack
   already-tracked files," which is exactly the invariant a `git rm` sweep breaks.

## #63 — when a spec keys behavior on an identifier, check WHEN that id exists on every path

1. **An identifier's *availability timing* can differ across code paths — confirm
   it before you make it a key.** #63's spec keyed the change-log on `session_id`,
   framing it as "minted on a fresh start." True for **claude** (pre-injected
   `--session-id` at launch) and **any resume** (`--resume <id>` on argv) — but a
   **codex/agy fresh session has no such flag**: the id is discovered *async* by
   `pair-session-watch.sh` and written to the config ~seconds *after* zellij/nvim
   already started. A design that read the id only from a launch-time env var would
   silently mis-key (or skip keying) for those agents. **Rule:** before keying
   anything on an id, trace every code path that produces it and ask "is it known
   *here, now*?" — synchronous for one path ≠ synchronous for all.
2. **Make the canonical store the source of truth; the env var is a fast-path
   cache, not a second fact.** Resolution order in *both* consumers (shell opener +
   Lua watcher) is `PAIR_SESSION_ID → per-tag config → none`. The config (which the
   watcher writes for the async agents) is authoritative; the exported env var is a
   launch-time optimization that just happens to be present for the sync paths.
   This keeps ARCH-DRY (one fact) while still covering the async case — and the
   nvim watcher **re-resolves each tick** so a late-landing id is picked up without
   a restart. **Rule:** when an env var and a file both hold "the same" value,
   pick one as canonical and make the other an explicit cache with a fallback.
3. **Decline a cosmetic transform that introduces a correctness risk that didn't
   exist.** The spec offered "truncate/hash the uuid for the filename (cosmetic)."
   Truncating buys a shorter name but adds a (tiny) collision risk and a transform
   to keep in sync across two languages. Full uuids are path-safe and ~36 chars —
   under any limit. **Rule:** "cosmetic" suggestions that trade away a correctness
   invariant (here: zero-collision keys) for nothing the user sees should be
   declined and the decision logged, not adopted by default.

## A no-`pattern` nvim autocmd on `BufWinEnter` fires for scratch/floating buffers too

`nvim/changelog.lua`'s viewer-setup autocmd was registered on
`{ 'BufReadPost', 'BufWinEnter' }` with **no `pattern`**. That matches every
buffer shown in a window. When #57 added the shared `Alt+q` annotate flow, its
floating prompt — a nameless scratch buffer (`nvim_create_buf(false, true)`) —
triggered `BufWinEnter` on display, so the viewer callback ran `M.setup` on the
*prompt* and locked it `modifiable=false`. The dialog appeared but was
un-typeable. The scrollback viewer dodged the identical bug only by accident:
its autocmd is `BufReadPost`-only, and a scratch buffer (created + `set_lines`,
never read from a file) never fires `BufReadPost`. Found in operator live
dogfooding, not by any headless test.

**Rule.** A read-only viewer's setup autocmd must only act on *its own* buffer,
not every buffer that enters a window. `BufWinEnter` in particular fires for
floating prompts, plugin scratch panes, etc. Guard the callback — discriminate
on a stable property of the real target (here: the change-log buffer is the
named file nvim was launched with, so `nvim_buf_get_name(buf) == ''` → skip the
scratch prompt) and early-return for anything else. Extract the guard into a
testable function (`M.on_buf_enter` returns true/false) so a headless test can
assert the skip path even when the floating UI itself can't be driven. Whenever
you add a floating/scratch UI inside a buffer-scoped viewer, re-check every
`BufWinEnter`/`BufEnter`/`WinEnter` autocmd in that viewer for this collision.

## Changing a shared insert-mode keymap: enumerate ALL its consumers, not just the spec's

#65 fixed the draft `<CR>`: when a completion popup is up and nothing is
Tab-selected, a bare `<CR>` only closes the menu and swallows the newline, so it
now feeds `<C-e><CR>` (cancel completion, then newline). The Spec's three-state
table reasoned about ONE consumer — as-you-type draft completion. But the insert
`<CR>` map is a **shared chokepoint**: it also serves the momentary normal-mode
`z=` spell popup (`spell_suggest_popup`, gated by `spell_popup_active`), whose
contract is "dismiss leaves the text intact — no newline." The first cut would
have injected a spurious newline into the draft on a `z=`-dismiss-via-Return
(the deferred `stopinsert` keeps you in insert mode when the `<CR>` lands). The
fresh-eyes milestone review caught it; the doer's spec never modeled the second
caller.

**Rule.** Before changing a shared keymap / dispatch function, grep for *every*
caller and popup/mode that routes through it (`z=`, as-you-type completers,
future pickers) and write a decision for each — don't let the spec's single
use-case stand in for the contract. Keep the decision **pure and testable**:
thread the distinguishing state in as an argument (`cr_keys(visible,
has_selection, momentary)`, fed `momentary = spell_popup_active` at the map
site) rather than branching on a global inside the handler, so each consumer's
behavior is unit-asserted without a live UI. A chokepoint shared across N
callers needs N tested cases, not one.

## init.lua is at Lua's 200-local-per-chunk ceiling (E5112)

**What happened (#66 M3).** Adding a handful of new file-level `local`s to
`nvim/init.lua` (review toggle + indicator helpers) broke sourcing with
`E5112: main function has more than 200 local variables`. Lua caps locals per
function scope at 200; init.lua's main chunk was already at the edge, so the new
locals silently tipped it over — and a sourcing error there isn't loud in the
headless tests (nvim still runs the `-c` driver, so functions defined *after* the
error line just come back `nil`, looking like "not exposed" rather than "chunk
broke").

**Rule.** New top-level helpers in `nvim/init.lua` go in a `do ... end` block
(their locals are block-scoped, off the main chunk's count); share across blocks
via a `_G.<table>` (e.g. `_G._pair_review = { … }`), not file-level locals. When a
headless probe reports a function as `nil` despite being defined, suspect a
mid-chunk sourcing error first — run `nvim -u nvim/init.lua -c 'lua …' 2>&1` and
grep for `E5112`/`E5108`, don't assume the definition is wrong.

## Test-only debug probes must sit before the guards they are meant to bypass

`tests/pair-continue-test.sh` uses `PAIR_DEBUG_ARGS=1` to ask the real
`bin/pair` parser what it resolved (`AGENT`, `FORCED_TAG`, forwarded args,
continuation doc) without launching zellij. That probe lived below the
in-session ancestry guard. When the test was run from inside a real pair/Codex
pane, `in_zellij_pane` returned true and `bin/pair` exited with "already running
inside a zellij session" before printing any debug fields. The parser was fine;
the seam was below the guard it needed to avoid.

**Rule.** A test-only probe that promises "parse and exit before side effects"
must be placed immediately after the state it reports is resolved, and before
environment/process guards, launch checks, cleanup sweeps, or IO side effects.
For live-session-sensitive tools, verify the seam from inside the real host
environment too — ancestry checks can fail even after env vars are scrubbed.

## Atlas gates apply to invisible workflow semantics too

#70 fixed a race in Codex session-id capture by changing the meaning of
`agent-pid-<tag>` consumption: the watcher no longer accepts any non-empty
pidfile, it waits for one whose mtime is fresh for the current launch. The code
and test were right, but the first close used `--no-atlas` because the change
felt like a narrow bugfix. Boundary review caught that `atlas/architecture.md`
still described the old fallback trigger and omitted the new freshness rule.

**Rule.** When a bugfix changes a persisted file's semantics, a process
boundary, or a recovery/fallback contract, check `atlas/` even if no public UI
changed. A "small" watcher/launcher fix can still alter the architecture map's
truth. If you pass `--no-atlas`, verify the atlas does not already document the
surface you changed; otherwise update the existing entry before close.

## Default command paths need their own assertions

#75's Go launcher prototype parsed an empty launch arg list as the default
agent in `launcher.ParseArgs`, but the dispatcher intercepted `pair-go launch`
with no args and returned help before the parser ran. The narrow parser tests
passed while the command path violated the issue's "default agent" requirement.
The same close review also caught a plan table that claimed `HistorySource`
wrapped `queue-*` even though the implementation only scanned draft/log
sidecars.

**Rule.** For every command parser default, add at least one test at the outer
dispatch/process layer that proves the empty/default invocation reaches the same
decision path as explicit inputs. When revising scope during implementation,
re-read the plan's core-concepts/integration tables and either implement every
listed surface or add a `## Revisions` entry narrowing the table before close.

## `git mv` of source must be swept through the atlas before merge

#92 relocated `slug`/`changelog`/`continuation` logic from `cmd/pair-<name>/`
into shared `cmd/internal/<name>cmd/` runner packages. The milestone/close
reviews all passed, but the `sdlc merge` **atlas/README-sync judge blocked the
merge**: the atlas still had clickable pointers to moved files
(`cmd/pair-slug/slug.go` in `architecture.md` + `how-to-bring-up-a-new-harness-cli.md`),
a Coverage Ledger listing ~10 moved-away paths that no longer exist, and
contract-table rows describing the helpers in their pre-move shape. Updating the
prose (dispatcher section, sequence notes) was not enough — the *structured*
atlas surfaces (file-pointer links, the inventory contract table's Files column
+ disposition, the Coverage Ledger path list) each independently go stale on a
rename.

**Rule.** After any `git mv`/rename of tracked source, before `sdlc merge`, run
`grep -rn '<old/path>' atlas/ README.md` for every moved file and repoint the
hits. Specifically sweep: clickable `file://` / path links, per-file lists like
a Coverage Ledger, and any contract/inventory table row whose Files column names
the moved path (update its disposition too). The boundary-review judges look at
the *diff*; the merge atlas-sync judge looks at *whether the atlas still matches
the tree* — a rename passes the former and fails the latter.

## Atlas prose describing a call graph goes stale when a *caller* changes, not just on renames

The `git mv` lesson above covers renamed **files**. #93 M1 surfaced the sibling
failure: a change that alters **who-calls-what** (not a file location) leaves
distant prose that narrated the old call relationship stale, and the merge
atlas-sync judge blocks on it. M1 folded the title poller's context count
in-process (dropping its `pair context` subprocess) and updated the poller's own
architecture section — but two untouched "#92 M2 repointed call-sites" narrative
blocks (`architecture.md`, `go-migration-inventory.md`) still listed
`bin/pair-title.sh` as a `pair context` caller and called
`bin/pair-session-watch.sh` "the one remaining shim-name caller." One of them
directly **contradicted** the line M1 rewrote (in-process vs. subprocess) — an
internal atlas self-contradiction.

**Rule.** When a change alters a call graph — X stops calling Y, a new shim-name
caller appears, a subprocess becomes in-process — updating the primary section
isn't enough. Before `sdlc merge`, grep the atlas for *other* mentions of the
old relationship: `grep -rn '<caller>' atlas/` and `grep -rn '<callee>\|pair <sub>' atlas/`,
and specifically re-read any "repointed call-sites" / changelog-style narrative
that enumerates callers or counts ("the one remaining …", "N callers still on
…"). Those enumerations and any edited-in-place prose that now disagrees with an
untouched distant line are exactly what the merge atlas-sync judge (matches
atlas *against the tree/behavior*) fails on — the boundary review (diff-only)
won't catch it.

## Porting shell→Go: a side-effect's semantics are a decision, not the Go idiom's default

#93 M4 ported `clipboard-to-pane.sh` et al. The shell wrote its diagnostic with
`> "$LOG"` (truncate each run); the first Go cut used `os.OpenFile(..., O_APPEND)`
— the idiomatic Go default — which quietly changed the behavior: the log now grew
unbounded. The boundary review caught it (Minor), but it's the kind of drift that
ships silently because the *feature* still works and no test observes a
diagnostic. Same class: exit codes (a shell `exit 0` on empty input vs a Go
error return), `set -e` short-circuits, `>>` vs `>`, backgrounded-and-`disown`ed
subshells (→ setsid-detached in Go), and "found-but-failed vs not-found" tool
cascades (`command -v` chains).

**Rule.** When porting a shell script, treat every side effect — not just the
happy-path logic — as a spec line to consciously preserve or *deliberately*
change: log truncate-vs-append, exit codes per branch, file-write atomicity,
process detachment, and the found/failed/absent distinctions of external-tool
cascades. If you improve on the source (M4 moved the truncate to the pipeline
head so it bounds growth *and* keeps the head's lines), say so in the plan
`## Revisions` as a deliberate delta — don't let a Go idiom silently redefine
behavior the source pinned.

## `sdlc milestone-close`'s auto review-window can pick a wrong far-back base on a fresh ticket → `fork/exec claude: argument list too long`

#99 M1 (the first milestone of a brand-new ticket branched off `main`) failed at
`sdlc milestone-close`: the auto-computed boundary-review window was
`<far-back-unrelated-commit>^..HEAD` — a **566-file, ~6.8 MB diff** — and the
review dispatch `fork/exec`s the `claude` CLI with the diff/prompt inline, so the
oversized arg vector tripped **E2BIG (`argument list too long`)**. The close then
aborts with verdict `not-run` and leaves the issue `working`. It is NOT a PATH-size
problem (a minimal PATH still fails) and NOT a code problem — it's the window base.

**Rule / workaround.** When a milestone-close boundary review fails with
`argument list too long`, check the window it printed: `git diff <base>^..HEAD
--stat`. If `<base>` is a wrong far-back commit (huge diff), run the review
yourself against the real branch base and finalize with `--no-judge`:

    sdlc judge milestone-review --base "$(git merge-base main HEAD)" --head HEAD --issue N
    # …address findings, then:
    sdlc milestone-close --issue N --milestone Mx --actual A --verified '…' --no-judge

Put the **real** verdict in the milestone commit's `Review-Verdict:` trailer (the
final `sdlc close` greps commits for it, not sdlc's `not-run` record), and note the
workaround in the issue Log. This is an ariadne/sdlc bug in the first-milestone
window computation — worth filing upstream, not just working around each time.

**Second manifestation (#99 M2): `milestone-close`'s ATLAS-gate window can pick
`base = HEAD` → empty window.** After committing the M2 code (with the atlas
updates in an *earlier* commit of the same milestone) and running
`sdlc milestone-close`, the atlas gate reported "no atlas/ changes in
`<lastCommit>..HEAD`" and aborted — its window base was the just-made HEAD commit,
so the (real, in-milestone) atlas edits a commit or two back were outside it. Same
window-computation bug class as the review-window one above, different gate. **Fix:**
confirm the atlas *was* updated in the true milestone window (`git diff --stat
<prev-boundary>..HEAD -- atlas/`), then pass the precise `--no-atlas` with the
rationale in `--verified` naming the commit that carries the atlas change. Don't
scramble to re-touch the atlas into the narrow window — the requirement is met; the
gate's window is wrong. Both variants point at one upstream fix: milestone-close
should derive its gate/review windows from the milestone's first commit (or the
prior `Mx` boundary), not a far-back base or HEAD itself.

## A milestone that defers scope must narrow its own Plan bullet in the same close

#99 M3's plan/issue bullet listed "in-session compaction" as M3 work, but the
implementation deferred compaction + the continue/rename restart re-entries + the
fzf pick to M5 (all → `ErrFallbackToShell`). The code was right and the deferral
was architecturally sound, but the tracker still claimed undelivered scope — the
M3 milestone-review flagged it Important (ARCH-PURPOSE / traceability). This is a
**recurring** shape: M1 also front-loaded/deferred pieces from its bullet. A
milestone-close that ticks `- [x] Mx` against a bullet the code doesn't deliver
silently over-reports progress.

**Rule.** When a milestone ships less (or different) than its Plan bullet
literally says, narrow it *in the same close*: add a plan `## Revisions` entry
naming what moved and to which milestone, edit the `- [ ] Mx` bullet's wording to
the shipped surface, and only then tick it. The tracker must never assert scope
the diff doesn't contain. Corollary (from the same review, I-2): don't cite an
ephemeral/uncommitted artifact (a scratchpad smoke, a `/tmp` script) as "coverage"
in committed code or comments — either commit the artifact or describe it honestly
as a one-time boundary verification recorded in the issue Log. Caught in #99 M3
milestone-review.

## Making a launcher flow native can silently break shell-seam tests

#99 M5a made the fzf session **pick** native (was `ErrFallbackToShell`). That
removed the *only* path by which `PAIR_TEST_CALL=... bin/pair` (a bare `pair`,
no verb) reached `bin/pair-shell`: under M4 a bare pair with sessions decided
`ActionPick → ErrFallbackToShell → shell`, which then ran the shell helper the
seam names. Native pick calls real `fzf`, which opened the agent's `/dev/tty`
and **blocked forever** — `make test` looked hung for 28 min.

**Rule.** `PAIR_TEST_CALL` (and `PAIR_DEBUG_*`, `PAIR_FORCE_IN_SESSION`,
`PAIR_FAKE_IN_ZELLIJ`, `PAIR_REEXEC_CAPTURE`) are **shell-only** dispatch/debug
seams with no native equivalent — `bin/pair-shell` short-circuits them early
(shell 930). When you port the *next* flow native (M5b compaction / continue /
rename), first ask *which shell-test seam reached the shell only via that flow's
fallback*, and route those seams to the shell explicitly (M5a did this in
`LaunchNative`: `PAIR_TEST_CALL != "" → ErrFallbackToShell`, before any
zellij/fzf). Corollary: a native `fzf`/`vared` pick with a live controlling tty
but no interactive user **hangs**, it doesn't error — never let a headless/test
invocation reach it. Caught in #99 M5a (the pair-continue / cmux-ownership
contract tests). Route removed at M5c when the shell + fallback arm retire.

## `| tail` hides a running suite; `sdlc milestone-close --dry-run` mutates

Two process gotchas from the #99 M5a close:

1. **`make test 2>&1 | tail -N` shows NOTHING until the pipe closes** (the whole
   suite finishes). A legitimately-running multi-minute suite then looks stalled
   at an empty/stale log — and killing it "because it hung" throws away real work.
   **Rule.** Redirect the suite straight to a file (`> log 2>&1`) and watch the
   file (line count + mtime) to see progress and detect a *real* hang (mtime
   idle > ~150s), instead of piping through `tail`.
2. **`sdlc milestone-close --dry-run` actually ticks the milestone + appends the
   `## Log` line** despite the flag (help says "skip close mutation"). **Rule.**
   Don't trust `--dry-run` to be side-effect-free here — `git checkout` the issue
   file and run for real, or fold the mutation. (Genuine `sdlc` gap; fix the
   `--dry-run` guard in `milestoneclose.go` when convenient.)

## The command sandbox blocks `ps` — breaking `InZellijPane()` ancestry detection

Diagnosing a #99 M5b "hang": `tests/pair-continue-test.sh` stalled at the
tag-mismatch compaction case (157). Root cause was NOT the native code — the
sandbox denies `ps` (`operation not permitted`, rc=127), and both the shell's
`in_zellij_pane` and the Go `InZellijPane()` walk the PPID ancestry via
`ps -o comm=/-o ppid=`. With `ps` blocked they return **false**, so the "already
inside a pane" guard never fires and the launch falls through to the create
name-prompt (fzf/vared on `/dev/tty`) → hangs. Run with the sandbox off (`ps`
available) and the same test PASSES (the guard fires → exit 1).

**Rule.** Any launcher test that depends on `InZellijPane()` / process-ancestry
detection (the compaction tag-guard, the in-pane reject) must run with the
**sandbox disabled** — `ps` is blocked in-sandbox and silently flips the
detection to false. When a launcher contract test "hangs" at an in-pane case,
check `ps -o comm= -p $$` first; if it's denied, re-run sandbox-off before
suspecting the code. (Sibling of the "tail hides a running suite" gotcha above,
and the cmux-broken-pipe-from-agent-shell memory.) Caught in #99 M5b.

## A validity/existence marker must exist in EVERY deployment layout

Retiring `bin/pair-shell` (#99 M5c) meant the entrypoint could no longer key
"is this a valid Pair asset root?" on `bin/pair-shell` existing. The tempting
replacement was `bin/pair-wrap` (a sibling the launch already needs) — but
`bin/*` is gitignored (built binaries), so `bin/pair-wrap` is **absent in a fresh
checkout before `make build`**. Keying the marker on it would make
`ResolveAssetRoot` reject an un-built source tree — a launch that works after
`make build` but not on a clean clone. The right marker is
`zellij/layouts/main.kdl`: a **tracked source file** AND **bundled into the
embedded runtime** AND the exact file the launch reads — so it exists in all three
layouts (source checkout, Homebrew/adjacent install, extracted embedded pair-home)
and can't drift from what the launch needs.

**Rule.** When choosing a file whose presence marks a directory as "a valid
install/asset root," verify it exists in **every** layout that root can take —
source checkout, packaged install, and any embedded/extracted copy. Prefer a
tracked, bundled asset the code actually consumes over a built artifact (gitignored
binaries fail the clean-checkout case). Caught in #99 M5c.

## A straggler grep that filters out comments hides stale doc as findings

When a change renames or deletes a referenced symbol (e.g. #94 deleting the
`bin/*.sh` shims), the sweep for lingering references must include comments —
`grep ... | grep -v '// '` to drop lineage noise ALSO drops stale present-tense
comments that describe the now-gone mechanism as current. Two such comments
(`cmd/pair-session-watch/main.go`, `atlas/how-to-bring-up-a-new-harness-cli.md`)
survived the M2 sweep precisely because the comment-filter hid them, and the
end-of-issue integration review turned FIX-THEN-SHIP over exactly those two lines.

**Rule.** In a rename/delete straggler sweep, grep WITHOUT filtering comments;
then hand-classify each hit as (a) legitimate provenance ("ported from X", "mirrors
X") — keep, or (b) a present-tense claim that X is the current mechanism — fix.
Don't let a `grep -v` that suppresses lineage also suppress stale docs. Search
`Makefile`, `atlas/*`, and every `cmd/*/main.go` package-doc, not just the files
you edited. Caught in #94 M2 / close.

## Commit milestone-close's OWN edits WITH the printed Review-Verdict trailer

`sdlc milestone-close` (like `sdlc close`) edits the issue file (ticks the box,
appends the Log line) and PRINTS a `Review-Verdict:`/`Review-Window:` trailer to
paste — it does NOT commit. If you commit the milestone's CODE first and then run
milestone-close, its file edits land in some later unrelated commit whose message
lacks the trailer — and the eventual `sdlc close --issue N` verdict gate refuses
("milestones M1, M2 lack Review-Verdict trailer in close commits"), forcing a
`--no-verdict` bypass (the reviews really ran; only the bookkeeping is missing).

**Rule.** Per milestone: finish the code, run `sdlc milestone-close`, then make the
NEXT commit carry both the milestone-close's issue-file edits AND the printed
`Review-Verdict:`/`Review-Window:` trailer lines in its message. One commit =
{the milestone's tick+Log edit} + {the trailer}. Same for the final `sdlc close`.
That keeps the trailer anchor on the close commit and avoids the `--no-verdict`
detour. (Corollary: FIX-THEN-SHIP → fix → the fixes move HEAD past the reviewed
anchor, so `sdlc merge`'s publish gate refuses → re-run `sdlc close` to re-review
the delta + re-anchor, then merge.) Caught in #94.

## `sdlc actual` can collide on a same-numbered issue from another context

Closing pair #95, `sdlc actual` suggested **8.46h** (est 2.4h) with attribution
sprayed across ~80 issues (#1–#151). The window start `1a372eb` turned out to be a
**2026-06-15** commit — "#95 M5: pair cutover prep — untrack AGENTS.md symlink" — an
*unrelated* "#95" from a different numbering context that the mention-fallback
window detector (`gitx.CommitWindow` greps commit messages for the issue number)
latched onto, scoping the window from mid-June to now instead of the 5-commit #95
branch (~40 min actual). `sdlc actual` has no `--base`/`--since` flag to correct the
window.

**Rule.** Treat an `sdlc actual` figure that's wildly over estimate AND attributed
across many unrelated issues / a long time span as suspect. Verify the window start:
`git log -1 --format='%h %ci %s' <window-start-sha>` (the sha printed as `window
<sha> → HEAD`). If it's an unrelated same-number mention (a cross-context / historical
collision), the measurement is polluted — close with `--no-actual` and record the
collision + the real rough figure in `--verified`, rather than committing the
inflated number to the velocity ledger. Do NOT hand-type a "corrected" value either
(that's the guessing the gate forbids); N/A-with-reason is the honest handling.
Caught in #95.

## Don't run slow, multi-round-trip orchestration inside a hook the invoker reaps

#100: the whole copy-on-select paste chain (mirror → in_nvim probe → flash →
focus → write, ~5 sequential `zellij action` client spawns at ~400ms each cold,
~1.5–2s total) ran *inside* zellij's `copy_command` child. zellij SIGKILL-reaps
that child after ~1s, so when the binaries were cold (dev-mode fleet rebuild on
every restart → macOS first-run scan + cold page-in on first exec) the first copy
after a restart was killed mid-chain and the paste silently dropped. Warm copies
finished under the deadline, so it looked intermittent ("first copy fails, rest
work"). The Go migration surfaced it: shell helpers needed no rebuild and had no
fresh-binary first-exec cost.

**Rule.** A hook invoked by an external supervisor (zellij `copy_command`, a git
hook, an editor `formatexpr`, a shell `PROMPT_COMMAND`) runs on *that
supervisor's* deadline, not yours — assume it can be reaped. Keep the hook to the
one fast thing it owes the supervisor (here: mirror the selection to the
clipboard) and `setsid`-detach anything slow so it outlives the reap. Diagnostic
signature of an external reap vs. a code bug: the process dies at a **variable**
point across runs with **no catchable signal** logged (SIGKILL) and nothing hung
in `ps` — a code bug dies at the *same* spot every time. Prewarming only narrows
the window and stays machine-speed dependent; detaching removes the deadline
dependency entirely (the root-cause fix). Also: once a side effect is detached its
stderr is `/dev/null`, so a debug-log line becomes the *only* channel a failure
can surface on — log failures explicitly. Caught in #100 (diagnosis + close review).

## Removing a transitional alias: sweep every caller, and never let a test pin the doomed token

#104 M3 removed the transitional flat dispatch aliases `scrollback-render` /
`changelog` (kept in M2 so callers could migrate incrementally). The M2 caller
sweep repointed the obvious call sites but missed `nvim/scrollback.lua`'s
`renderer_command` (the Alt+/ viewer's in-buffer refresh), which kept emitting
`pair scrollback-render`. Once M3 dropped the alias, that argv classified as the
public launcher (`ModePublicPair`) and fell through to *launch a session* → the
refresh silently failed in every session. **The unit test made it worse:** the
one test exercising `renderer_command` asserted `rc[2] == 'scrollback-render'` —
it *pinned the value the removal was about to invalidate*, so `make test` stayed
green while the runtime path was dead. Only a fresh-eyes boundary review that
actually ran `pair scrollback-render` against the built binary caught it.

**Rule.** Removing a token/alias/flag that other code passes as a *string
argument* (not a symbol the compiler checks) is a repo-wide grep obligation, not
a "rewrite the callers I know about" task — sweep `*.lua`, `*.kdl`, `*.sh`, shell
heredocs, and any arg-table/command-string builder for the literal before
deleting it, because the type system won't. And when a test *pins* a value you
plan to remove, that test is load-bearing for the migration: update it **in the
same change** as the removal (and make it assert the *new* form), or it will
enshrine the broken value and green-light the regression. Prefer, where possible,
a runtime assertion that the built binary actually routes the string (an e2e that
execs `pair <sub>`), since a pure-unit test that only inspects the arg table
proves the table's shape, not that the dispatcher accepts it. Caught in #104 M3
boundary review.

## Ownership files must store the canonical resource id, not only a display key

#107's repo-scoped session model moved zellij ownership from `pair-<tag>` to a
public session name assigned by `session-names.jsonl`, but the cmux owner file
still stored only the repo-local tag. The title poller then tested a foreign
owner's liveness by reconstructing `pair-<tag>`, so a live scoped owner such as
`pair-pair-work` looked stale and another session could reclaim the workspace
title.

**Rule.** When a lock/owner/lease file guards a resource whose runtime identity
can differ from its display key, store the canonical runtime id alongside the
display key. Readers may support old one-field files as legacy, but new writes
must include the canonical id and liveness probes must use it. Add a regression
where the display key and runtime id deliberately differ. Caught in #107 close
review.
