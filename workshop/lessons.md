# Lessons

## Historical source contracts must pin bytes as well as paths

A historical declaration guard derived its filenames from an immutable commit
range but parsed each file from the current worktree. Later declarations in an
old filename therefore changed the supposedly frozen digest and made every
unmarked current export look like a missing historical concept.

**Rule.** A historical source oracle must read both its path set and file bytes
from the same pinned Git objects. Never combine pinned names with mutable
worktree contents. Caught during #000155 close verification.

## Append-only authority needs an explicit commit outcome

A ledger writer returned ordinary errors from sync, close, directory sync, and
unlock after a complete JSON row was already readable. Recovery then accepted
the row as authority even though the caller inferred that the append failed;
an unterminated but otherwise complete JSON object widened the same mismatch.

**Rule.** For append-only authority, make the record terminator part of the
commit framing and model non-authoritative, indeterminate, and committed results
explicitly. Fault-test every byte boundary plus write, file sync, close,
directory sync, and unlock, then recover through the production parser and
assert that its authority agrees with the reported outcome. A store-level label
is not enough: enumerate every production consumer and every other writer that
publishes equivalent authority. Post-publication retry needs a stable operation
identity so it can finish durability without duplicating evidence. Caught in
#000155 close review.

## Verification must not inherit ignored generated state

A boundary suite passed in the developer checkout because an ignored generated
runtime mirror was present, while the same focused test failed from a clean
archive after those files were correctly removed from Git.

**Rule.** Any test that validates generated output must create that output in a
temporary directory from tracked inputs, then compare it with the manifest in
both directions. Run the focused gate once from a clean archive or checkout;
the working tree's ignored build residue is never admissible evidence. Caught
in #000149 M5 review.

## Compound event state needs one synchronization owner

An overlay used an atomic boolean plus a separately locked text tail. Enter
loaded the boolean, a new overlay re-armed it, and Enter then stored false and
cleared carryover—losing the newer event without any data race.

**Rule.** When one logical event spans a flag, carryover, generation, or other
fields, mutate and consume the whole state under one owner. Atomic primitives
do not make a multi-step protocol atomic. Add a deterministic re-arm-during-
consume interleaving that proves both the new flag and its associated data
survive. Caught in #000139 Task 5 review.

## Cross-system resize needs an exclusive transaction token

The terminal model and child PTY could temporarily or permanently disagree on
geometry. A simple validity boolean fixed one resize but failed when two
prepare/commit sequences overlapped; an earlier commit could reopen
authorization while the later resize remained incomplete.

**Rule.** For a state transition spanning two systems, validate first and hold
exclusive transaction ownership across prepare, external mutation, and exactly
one commit or abort. Prepared and aborted states must stay fail-closed; commit
must discard pre-transaction authorization and require fresh evidence. Test
overlapping transactions through both commit and abort, external failure, and
recovery. Caught in #000139 Task 5 review.

## Panic recovery must not strand a critical section

`handleChunk` intentionally recovered detector panics, but the detector wrapper
manually unlocked its mutex after the call. A panic skipped the unlock, so the
process survived while the next Return deadlocked.

**Rule.** Any callback invoked inside a critical section must be wrapped by a
helper that defers unlock before calling it. If an outer boundary recovers
panics, add a regression that injects a panic and then proves the next operation
using the same lock completes. Caught in #000139 Task 5 review.

## Differential migrations must transform every state axis

The first Muse snapshot oracle covered an empty composer at the captured cursor
column but omitted typed text and the legacy tracker's cursor-row ±1 behavior.
Both omissions produced unallowlisted old-true/new-false transitions even
though the literal startup fixture stayed positive.

**Rule.** A differential migration must enumerate transformations of every
state axis the old predicate consumes: content, style, locality, cursor row,
cursor column, visibility, and lifecycle mutation. Include representative
positive transforms—not only the captured empty state—and reject any behavior
change not named by the contract. Caught in #000139 Task 3 review.

## Process cleanup is one observable transaction

The first live-harness capture helper hid cleanup errors behind a primary
timeout, could skip its final reap after a kill error, and requested reader
cancellation without joining the goroutine. Happy-path child tests still
passed, but callers could not know whether capture had actually finished.

**Rule.** A subprocess/PTY helper must have one teardown owner: cancel and
close IO, signal, reuse one wait-result channel, continue through bounded
kill/reap even after operation failures, and boundedly join every reader.
Return `errors.Join(primary, cleanup)` so the original failure and cleanup
failure are both observable. Pair injected operation-failure tests with a real
controlled child on the same seam. Caught in #000139 Task 2A review.

## Capacity tests must finish on capacity, not elapsed throughput

A 1 MiB retention test waited 100 ms and then required all 1 MiB to have
arrived. Under concurrent load it retained only 377,856 bytes, even though the
cap implementation was correct.

**Rule.** Test a byte/item cap by completing when the observed retained count
reaches the cap, with time only as a generous safety bound. Keep timeout
behavior in a separate test. Never make scheduler throughput the oracle for a
capacity invariant. Caught in #000139 Task 2A review.

## Authorization enums need a fail-safe zero value

The first Return gate enum assigned its legacy remap policy to zero. An absent
or corrupt profile therefore fell through as authorized; an all-zero keymap
could report `Fired` while emitting no bytes and swallow Enter.

**Rule.** For any enum controlling a rewrite, permission, route, or destructive
action, reserve zero for unknown/disabled and switch exhaustively. Only named
authorizing values may reach configured behavior; zero and invalid values must
take the safe observable fallback. Test both an all-zero owner struct and an
out-of-range enum. Caught in #000139 Task 2 review.

## Terminal observers must share the parser's state model

A raw C1 CSI byte can be a control in terminal ground state and ordinary data
inside UTF-8, OSC, or DCS. A side observer that scans framed escapes without
the terminal parser's state therefore authorizes controls the screen owner did
not parse, especially across caller chunk boundaries.

**Rule.** When security- or routing-relevant evidence shadows a terminal
parser, use the same bounded parser state semantics as the screen owner. Test
the same control byte in ground, UTF-8, OSC, and DCS contexts at every split;
do not infer controls from raw byte values alone. Caught in #000139 Task 1
review.

## Dependency boundaries define the property-test oracle

x/vt flushes extended graphemes at each `Write`, so one-shot and chunked writes
of the same valid ZWJ stream can produce different cell grids. Requiring grid
equality would force Pair to become a second grapheme renderer without proving
the Return-routing behavior the issue exists to protect.

**Rule.** Before asserting chunk-equivalent representations, prove the owning
dependency promises that representation invariant. If it does not, keep
boundary tests to safety, bounds, and coherent state, then assert equivalence at
the product decision seam using literal production streams. Seed fuzzers with a
deterministic multi-codepoint grapheme such as `👩‍💻`. Caught in #000139 Task 1
review.

## Validate dimensions before allocation boundaries

Rejecting only zero and negative dimensions still allowed huge positive PTY
sizes to panic inside x/vt allocation and made snapshot multiplication unsafe.

**Rule.** Any externally influenced width/height pair must pass one shared,
overflow-safe per-axis and total-area validator before construction, resize, or
`width*height` allocation. Rejected mutations must preserve the prior complete
state. Include max-int-shaped rejection tests. Caught in #000139 Task 1 review.

## Local predicates must count local evidence

The #142 close review caught a composer detector that required one painted row
near the cursor but counted the second required row anywhere on screen. That
kept the reported Codex composer bug fixed, but it weakened the positive
detection contract with a sparse-row false positive.

**Rule.** When a predicate is anchored to proximity, selection, cursor position,
or any other local evidence, count only evidence inside that same local window.
Add a negative regression with one local match plus one far-away match so global
aggregation cannot accidentally satisfy a local threshold. Caught in #000142
close review.

## OS command helpers need one reusable seam

The #141 close review caught duplicated `ps` process-tree and `lsof` parsing in
two command paths (`pair slug` and launcher restart recovery). Both paths were
correct locally, but parallel shell-output parsers drift easily and tests tend to
cover only one consumer.

**Rule.** When two features consume the same external command shape (`ps`,
`lsof`, `git`, `zellij`, etc.), extract the command parser/traversal into a
shared internal package before adding the second consumer. Keep one fake-command
test at the real OS seam for each production consumer that depends on environment
or filesystem inputs.

## Release smokes must use clean archive inputs

The Homebrew v1.24/v1.25 publish path first looked fine from the working tree:
ignored generated runtime-bundle assets were present locally, and formula syntax
and style passed. The real Homebrew source build failed only when it built from
GitHub's clean tarball, where ignored assets were absent and the generator's
import cycle/order assumptions became visible.

**Rule.** For release/package work, run the same clean source path the package
manager uses before treating the release as published: generated ignored assets
must be regenerated from tracked inputs, and install recipes must run generators
before moving source trees into their install location. Add a clean-source
regression for any generator that package builds depend on. Caught in #000131
Homebrew publish.

## Sidecar filenames do not validate sidecar identity

`config-<tag>-<agent>.json` names the intended lookup axis, but the JSON still
has its own `agent` field. Treating the filename as sufficient let a mismatched
config reach the tag-restart picker, and stale saved session IDs were silently
downgraded to fresh sessions despite the spec requiring a warning.

**Rule.** When consuming persisted sidecars that duplicate identity in their
filename and body, validate the body identity before offering UI/actions. On
malformed or mismatched persisted state, warn and fall through to the next
source of truth; on stale resumable IDs, warn before using saved args for a fresh
launch. Add integration-level regressions at the consuming flow, not only pure
policy tests. Caught in #000115 close review.

## Zellij's pane report cannot identify action-created panes

The tiled split (`action new-pane --direction down`) creates panes for which
zellij 0.44.3 reports `terminal_command: null`, and pane titles are pair-owned
mutable UI (#118 tab strips, user-renamable). Classifying workbench panes from
the zellij report alone therefore silently fails for exactly the panes pair
creates at runtime — live smoke showed split halves invisible to chord routing
and to the focus picker.

**Rule.** Pair-owned pane identity comes from self-registration (the process
writes its own `$ZELLIJ_PANE_ID` + pid to a sidecar; readers filter by pid
liveness), never from report heuristics. When adding a new pair-owned pane
kind, register it and overlay the registry onto `RoleForPane`
(`RoleForPaneWith`). Zellij `is_focused` is per-client and stale for
unfocused-side panes — a pair-authored record outranks it. Caught in #123
tiled-pivot smoke.

## Drive zellij live smokes through a real attached client

CLI actions (`zellij --session X action write|focus-pane-id|new-pane`) run as
ephemeral clients: their focus state diverges from the attached client, writes
land on stale focus, splits target the wrong pane, and `--near-current-pane`
creates invisible orphan panes. Results look like product bugs but are harness
artifacts.

**Rule.** Smoke zellij interactively via a PTY-attached client (expect spawn +
fifo-fed keystrokes) sending the real byte encodings (`\x1bk`, `\x1bD`, SGR
mouse). Use CLI `list-panes` only for observation. Restart the session after
every rebuild — resident pair processes do not pick up new binaries. Caught in
#123 tiled-pivot smoke.

## Zellij forwarded bytes must preserve every focused surface using the chord

`Alt+Shift+d` was added as a right-terminal split shortcut by rebinding Zellij
to forward the KKP sequence `ESC[68;4u`. The terminal wrapper understood that
sequence, but the review pane already used the same physical chord as `<M-D>` for
visual definitions, and Neovim did not treat the forwarded KKP bytes as `<M-D>`.

**Rule.** When changing a Zellij binding for a physical chord, inventory every
focused surface that already uses that chord and test the exact forwarded bytes
against each consumer. For Neovim surfaces, add a map for the raw forwarded byte
sequence when KKP does not resolve to the existing `<M-...>` mapping. Caught in
#000123 close review.

## Activating an empty terminal tab must still redraw

`Alt+t` created a new terminal tab and made it active, but `newTab` only updated
the pane title and waited for async child PTY output. The old tab's viewport
stayed visible until the new shell wrote over part of it, leaving confusing
residue in the newly selected tab.

**Rule.** Any terminal-tab activation path must redraw the selected tab
immediately, even when its buffer is empty. The clear-screen prefix is the
observable behavior; child output arriving later is not a substitute for the
activation redraw. Add a regression that creates a fresh tab and asserts stdout
starts with the redraw clear sequence. Caught after #000118 close.

## Async terminal modes must keep target identity

Terminal tab rename originally looked up `activeTabLocked()` again at commit
time. If the tab being renamed exited while rename mode was open, `removeTab`
could promote another tab to active and Enter would rename that replacement tab.

**Rule.** When an async mode starts against a terminal tab, capture the tab's
stable ID at mode entry and pass that ID through every refresh/finish path.
Never re-resolve by "current active" after an async boundary. Add a regression
where the target tab exits mid-mode and the replacement active tab keeps its
original name. Caught in #000118 re-close review.

## Zellij pane self-mutations must pass `--pane-id`

Terminal tab rename originally called `zellij action rename-pane <title>` from
inside `pair term`, relying on Zellij's focused pane. Live layout-3 smoke showed
the floating terminal and draft pane can both appear focused in `list-panes`, and
the implicit rename targeted the draft pane instead of the terminal pane.

**Rule.** Any process running inside a Zellij pane that mutates its own pane
state must pass `--pane-id "$ZELLIJ_PANE_ID"` when the action supports it
(`rename-pane`, geometry, close/focus variants, etc.). Add a fake-runtime test
asserting the exact `--pane-id` action shape, then run a live smoke for focus
ambiguity when floating panes are involved. Caught in #000118 close review.

## Unknown escape terminators are part of the escape sequence

Rename-mode input first treated some unknown CSI sequences as malformed prefixes
and preserved their final byte for reprocessing. `ESC[1;5D` then consumed the
escape prefix but inserted `D` into the tab name, violating the "unknown
controls are consumed" contract.

**Rule.** When consuming an unknown terminal control sequence, consume through
the protocol terminator (`A`-`Z`, `a`-`z`, `~`, etc.) and reprocess only bytes
after that terminator. Add regression cases with known-looking but unsupported
controls such as `ESC[1;5D`; recognized-control tests alone do not prove the
malformed/unknown path. Caught in #000118 close review.

## Global keymaps need post-setup buffer-local shadow tests

Pair installed shared workbench-global mappings before scrollback buffer setup,
but older buffer-local safety maps later replaced Alt+x and Alt+Up/Down. Pure
router tests and static “module loaded” checks stayed green while the live
buffer used the wrong callbacks.

**Rule.** For a global Neovim mapping consumed by specialized buffers, open a
real representative buffer after every setup autocmd and inspect `maparg(...,
false, true)`. Assert the resolved description/callback and that no unintended
buffer-local mapping shadows it. Static source greps do not prove effective
mapping precedence. Caught in #000117 close review.

## Plan entity tables must name implemented symbols

The #117 plan described conceptual entities (`DraftLuaTarget`,
`OverlayRoutePlan`, then `draftroute.Router`) that never existed as named code
symbols. The implementation was sound, but the boundary review repeatedly had
to reconcile the durable design record with the actual API.

**Rule.** Before a boundary review, mechanically walk every Core concepts table
row: `rg` the exact entity name at the declared path, and either point to the
real symbol or revise the row to the implemented function/type. Conceptual
groupings must be explicitly labeled as such, not formatted like nonexistent
APIs. Also search completed task prose and unchecked rows—the revisions section
does not cancel stale contradictory instructions elsewhere in the same plan.
Categorization is part of the audit: a row under Pure must be free of IO and
have a direct unit test; a future-milestone entity must say “planned,” not “new.”
Caught in #000117 close review; the classification extension was caught again
in #000146 M3 review.
After repeated drift, the audit must be executable: parse the table and make a
wrong symbol, path, deletion status, or PURE classification fail. A manual grep
record can correct today’s rows but cannot prevent the next edit from restoring
the same family. Caught in #000146 M3 review round 3.
An executable prose-table audit must also pin the expected row set. Validating
only rows that remain is one-way: deleting a whole row deletes the test input
and passes. Reject missing, extra, and duplicate rows before checking their
contents. Caught in #000146 M3 review round 4.

## Cross-language cache tests must use the producer's exact JSON types

Draft Neovim wrote its PID with `vim.fn.getpid()`, producing a JSON number.
The Go cache reader modeled PID as a string, so decoding failed and quietly
re-enabled the slow fallback. Tests passed because they marshaled the Go
consumer struct—thereby generating the consumer’s preferred string shape,
not the producer’s real numeric shape.

**Rule.** For a cache or sidecar crossing language boundaries, keep at least
one consumer fixture as literal output in the producer’s exact schema,
including number-vs-string types. Producer-derived fixtures catch wire-format
drift; consumer-self-marshaled fixtures do not. Caught in #000117 close review.

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

## Async lifecycle paths must respect active modal ownership

#118's terminal rename mode correctly consumed stdin bytes in the frame-title
editor, but PTY-exit cleanup still called the ordinary title redraw and active
viewport redraw path. A background tab exit could erase the rename title while
stdin stayed in rename mode.

**Rule.** When adding a modal interaction, audit async lifecycle callbacks
separately from the direct input path. The mode owner needs enough shared state
for cleanup/repaint code to preserve the mode's visible surface, and tests
should trigger the lifecycle event while the mode is active.

## Escape decoders must distinguish prefixes from unknown complete controls

#118's rename decoder treated every `ESC[<...` byte string without `M`/`m` as an
incomplete SGR mouse report. A malformed complete sequence such as
`ESC[<0;12;4X` could then stay pending and swallow later input.

**Rule.** For terminal escape parsing, buffer only when the byte string is still
a real prefix with no final byte. Once a CSI/SS3 final byte arrives, unsupported
or malformed controls must be consumed as complete input. Add split-boundary
tests where the final byte arrives in a later read. Use the same final-byte
predicate for both "is this sequence complete?" and "how much malformed input
should be consumed?" so a control like `ESC[@z` cannot swallow the following
printable `z`.

## Never disable an input layer without auditing the escape hatches it provides

#123 set `mouse_mode false` to stop mouse drags from moving workbench panes.
That single global switch also removed click-to-focus — which turned a latent
keyboard trap (left→right `move-focus right` landing on the invisible
terminal-filler pane, which swallows all keys) into a total focus lockout,
and silently killed copy-on-select and scroll.

**Rule.** Before disabling a whole input modality (mouse, a keyboard protocol,
a bind table), enumerate what recovery paths and features ride on it; prefer
the narrowest mechanism that fixes the reported problem. And test focus
navigation as full round trips from *every* pane, driven through the real
input path (`zellij action write` into the pane), not only via `--test-shortcut`
harness calls — the trap here lived in the pane the chord *lands on*, not the
pane that handles it.

## Pane navigation must be id-based, not relative

The draft/agent → terminal jump used `zellij action move-focus right`, which
addresses the tiled layer only and cannot reach a floating pane; it focused the
filler behind the terminal. tests/review-poke-test.sh already encoded this rule
for the review pane ("no relative move-focus (must be id-based)").

**Rule.** Any cross-pane focus change in the workbench goes through pane-id
addressing (`focus-pane-id`, now via `pair layout focus-terminal` for the right
terminal). Relative `move-focus` is acceptable only within the tiled left stack
(agent ↕ draft) where no floating layer is involved.

## Paired protocol terminators need one constant, not one per site

#127: an SGR mouse event ends with `M` (press) **or** `m` (release). Two sites
framed it — `parseSGRMousePressPrefix` and `isSGRMousePrefix` — and both looked
only for `M`, so a release read as "sequence not finished". It went into the
`held` buffer along with every keystroke behind it: a dead keyboard, plus a
child stuck in an unmatched mouse drag (nvim: stuck in visual mode). A third
site, `sgrMouseSize`, already knew both forms — the protocol wasn't unknown, the
sites just disagreed.

**Rule.** When a protocol has paired/alternate terminators, define them once and
derive every framing site from that constant. If you find two sites making the
same framing decision independently, that IS the bug, not a style issue —
consolidate before adding a third. Same for "where does this CSI end": one
scanner per package.

## Any pure byte-scanner gets a fuzz test, not just valid-input cases

#127's close review found a panic in brand-new code: `isParameterizedOSCQuery`
checked a 4-byte prefix and a 2-byte suffix that **overlap** on a short body, so
`\x1b]4;?` satisfied both and inverted a slice bound. Reachable from ordinary
child output, on the tab-switch path the same issue had just added — it would
have killed `pair term` and every shell in the pane. Thirty hand-written test
cases missed it because every one fed a *syntactically valid* sequence.

**Rule.** A function that scans attacker-shaped or device-shaped bytes (terminal
output, transcripts, clipboard) gets a `Fuzz*` asserting "never panics" plus a
cheap invariant (output ≤ input) — seeded with the malformed forms. Cost is ~10
lines. Separately: when a prefix check and a suffix check can overlap, guard the
minimum length explicitly; `len(x) >= prefix+suffix` is load-bearing, not
defensive.

## Assert behavior, not the implementation's current location

`tests/scrollback-open-test.sh` grepped for a literal `'<M-x>'` inside
`nvim/scrollback.lua`. #117 moved global chord handling into the shared
`workbench_route` / `workbench_actions` pair — behavior unchanged, assertion
dead. `make test` was red on main and stayed that way.

**Rule.** Pin the wiring plus the generated table (does the viewer install the
global maps, does the action table still carry the chord), not a string in
whichever file happens to hold it today. Corollary: a test that would stay green
if the behavior were *fixed* isn't a pin — #127 shipped a "pins the accepted
residual" test that never built a second tab, making it a duplicate of the test
above it.

## A stateless plan judge converges by consequence — read each round, don't just count them

`sdlc change-code`'s plan-quality judge starts fresh every round, so the standing
worry is that plans converge by exhaustion (ariadne#187). #130 ran six rounds,
and the useful distinction turned out not to be round *count* but round
*provenance*: rounds 4 and 5 blocked on defects that were **consequences of
rounds 3 and 4's own fixes** — the ledger short-circuit only became reachable
once the sweep was correct, and the duplicate-row bug only existed because
round 4 introduced the liveness gate. That is the fix surface moving, not the
gate re-litigating settled ground.

**Rule.** Classify each round's findings before deciding whether to loop. If they
are consequences of your last round's change, the gate is still earning its cost
— keep going. If they re-derive ground a previous round settled, stop and close
with `--no-judge`, recording why in the issue. Round 6 of #130 dropped to one
Important + two Minor with no new blocking design defect; that is the signal, not
the round number. Never loop silently — the operator is cost-sensitive about
gate ceremony and wants the trade-off surfaced.

## Land the behavior-flip hunk first and watch which test goes red

#130's plan was reviewed six times and named the unbounded-ledger-append defect
precisely — in its *designed* form. It did not name the *intermediate-state*
form: applying the short-circuit flip while the name generator still emitted the
old scheme makes a legacy row fall through and re-mint **another legacy name**,
appending a duplicate row per create. Landing the one-line hunk and running the
package tests surfaced it in seconds, via
`TestAssignSessionNameReusesSameScopeBinding`.

**Rule.** In a multi-hunk change where one hunk flips a decision and another
changes what that decision *produces*, land the flip on its own, deliberately, to
see what breaks. The failing test names the ordering constraint better than
review does — review reasons about the end state, tests reason about the state
you are actually in. Then revert it, record the constraint at the call site, and
ship the hunks together.

## User-visible identity changes need README and pasted-name tests before close

#130 changed the public zellij/cmux/list session name from `pair-...` to
`📁repo[-tag]`. The atlas and issue were updated, but close review still caught
two boundary gaps: README did not explain the new public name vs repo-local tag,
and pasted `📁...` resume/rename paths were implemented without direct tests.

**Rule.** When a change alters text a user sees or may paste back into a command,
update README in the same window and add tests for the paste-back entry points.
Atlas explains architecture; README explains what the operator types and sees.

## Shell `printf` recycles its format — a dropped `%s` corrupts silently

#133 removed a field from the `printf` in `zellij/layouts/main-{2,3}.kdl` that
writes `pane-<tag>-<agent>.json`. Drop the `%s` but leave its argument and POSIX
`printf` **reapplies the whole format** to the surplus argument: the file gets two
concatenated JSON objects, `json.Unmarshal` fails, and every consumer degrades to
its zero value — `paneCwd` returns `""`, the title poller skips the pane. Nothing
goes red, because the producer was a shell line no test executed; all four
fixtures in the tree hand-wrote that JSON.

**Rule.** A shell line that emits structured data is a producer, and it needs a
test that *executes* it: extract the command from its layout/script, run it under
`sh` with the external binaries stubbed on `PATH`, and assert the real consumer
decodes the result. Assert the payload is exactly **one** value — that single
assertion is what distinguishes format-recycling from a merely-wrong field. Write
it BEFORE the edit so it passes first, then mutation-test it: make the wrong edit
on purpose and watch it fail. Verified both directions in #133.

Corollary: a stub that discards argv only proves a command *ran*. Where the
arguments are the thing you depend on (`rename-pane --pane-id N <title>` carries
the startup pane title), make the stub record argv and assert it — otherwise
mangling the expansion keeps the whole tree green. Close review caught exactly
this gap in #133's first cut.

## Grep the rendered SHAPE, not the symbol, when deleting a display format

#130's lesson already said "update README in the same window". #133 missed README
anyway — while correctly updating `atlas/architecture.md`, `Makefile.local`, both
KDLs, the generated bundle, and the code. The reason the existing rule didn't
fire: the sweep grepped for **symbols** being deleted (`TildeAbbrev`, `abbrevCwd`,
`cwd_display`) and every one of those was clean. README never mentioned a symbol —
it described the *output*, `<agent> (<count>) [<cwd>]`. Same class of miss hit the
`titlepoller` package doc, which contradicted the function 58 lines below it.

**Rule.** When changing what a format *renders*, grep for a distinctive fragment
of the rendered shape (here `[<cwd>]`, or `) [`) across `README.md`, `atlas/`,
`Makefile*`, and package doc comments — in addition to grepping the symbols. Prose
restates output, not identifiers, so a symbol-only sweep reports all-clear on the
surfaces humans actually read. Do this sweep before the close review, not in
response to it.

## A verification command must itself be verified — an invalid flag reads as "clean"

#133's Done-when told the reader to confirm the gitignored runtime bundle with
`grep -rn --no-ignore …`. While implementing #132 that command turned out to be
**invalid**: this environment's `grep` is ugrep 7.5.0, where `--no-ignore` does not
exist (the flag is `--no-ignore-files`). ugrep exits 2 and prints an error — but
the error had been swallowed by `2>/dev/null`, so the output was zero lines, which
is indistinguishable from "no matches found". I reported a clean bundle on the
strength of a command that never ran a search. (Re-checked with
`--no-ignore-files` and with `/usr/bin/grep`: the bundle really was clean, so the
conclusion survived — but only by luck.)

The trap generalizes past flags: `grep pattern file | head` reports `head`'s exit
status, not grep's, so `echo "exit=$?"` after a pipe tells you nothing about
whether the search matched or even ran.

**Rule.** When a check's *passing* output is empty, prove the check can fail
before believing it passed. Cheapest proofs, in order: (1) run it against a string
you know is present and watch it print; (2) drop `2>/dev/null` so an invalid flag
surfaces; (3) confirm with a second, independent tool (`/usr/bin/grep` alongside
ugrep). Prefer `--no-ignore-files` in this repo, and never put the assertion
behind a pipe whose exit code you then read. An "empty = clean" idiom is a
false-pass generator unless you have seen it produce a non-empty result at least
once in the same session.

## A differential oracle proves output equivalence, not storage — check aliasing separately

#128 replaced a regex with a byte scanner and fuzzed the new `Strip` against the
retired `otherEscRe.ReplaceAll` as an oracle: **20M executions, zero
disagreements**. It still shipped a Critical. `Strip` had a "fast path" returning
the *input slice* when it held no ESC, and both callers pipe the result into an
in-place compactor — so on ordinary ESC-free output it rewrote the caller's own
buffer, corrupting the image-capture file and racing the mutex that guarded it.

The oracle compared `Strip(buf)` to `ReplaceAll(buf, nil)` **by value**. Two
functions can agree on every byte of output forever and differ completely in whether
the output shares storage with the input. The regex allocated unconditionally; the
replacement did not, and nothing in the comparison could see that.

Worse, the unit test *asserted the defect*: `&got[0] == &in[0]` with the comment
"should not copy when there is nothing to strip". A plausible-sounding optimisation
got locked in as intent.

**Rule.** When replacing something that returns a slice, the contract includes
**storage**, not just contents. Write it down in the API doc, and add the two-line
property to the fuzzer alongside the value comparison:

```go
before := append([]byte(nil), buf...)
got := f(buf)
if !bytes.Equal(buf, before) { t.Fatal("mutated its input") }
for i := range got { got[i] = 0 }
if !bytes.Equal(buf, before) { t.Fatal("returned storage aliasing its input") }
```

Corollary for reviews: "the fuzzer is green" answers *which bytes*, never *whose
memory*. Ask what the oracle is blind to before treating a large exec count as
coverage.

## A timeout can describe a phase without describing a process lifetime

#143 changed session discovery from a terminal 60-second deadline into two
phases: fast startup discovery, then low-frequency polling while the bound agent
process lives. The implementation and tests captured the distinction, but two
atlas entries still called the component a "60s watcher" or promised a failure
when that window elapsed.

**Rule.** When a timeout changes from terminal to transitional, grep prose for
the duration, "timeout", "deadline", "window", and the component name. Update
each description to name both the bounded phase and the lifecycle exit
condition; otherwise operational docs will mistake a scheduling phase for the
component's lifetime.

## Reconcile the plan's entity table and promised cases before boundary review

#144's implementation centralized Codex root identity correctly, but its first
close review still returned REWORK. The durable plan named an exported function
in the wrong file, called an existing reused process seam "modified", and listed
integration cases that the implementation's broader tests did not assert
literally. The functional suite was green; the review contract was not.

**Rule.** Before a boundary close, mechanically reconcile every Core concepts
row against `rg` and `git diff <base> -- <path>`: exact symbol spelling,
visibility, location, and new/modified/reused status. Then turn each promised
test bullet into a named-test checklist and verify it directly; adjacent coverage
is not fulfillment. Finally, when centralizing a behavior, grep both code symbols
and the old prose description across every atlas file so older maps do not keep
teaching the retired rule.

## Raw review transcripts are not safe source artifacts

#144's first close-review transcript was committed as workflow evidence. It
embedded thousands of lines of raw prompts and diffs, including upstream
trailing whitespace and space-before-tab sequences. The implementation files
were clean, but branch-wide `git diff --check` then failed on 897 lines inside
the generated transcript.

**Rule.** Do not commit raw boundary-review transcripts unless their generator
normalizes embedded patches and `git diff --check <base>..HEAD` passes with the
artifact included. Prefer the gate ledger, verdict trailers, and concise issue
log as durable evidence; generated diagnostic capture is disposable.

## Negative environment tests must clear every required variable

#143's focused wrapper command failed only inside a live Pair session because a
test for an incomplete readiness environment set two variables but silently
inherited the other two required variables from the harness.

**Rule.** A test asserting behavior when environment input is absent must call
`t.Setenv(key, "")` for every absent key in that contract. Unsetting variables
only in the outer test command hides the isolation defect and makes the checked
command non-reproducible for the exact environment where Pair is developed.

## Long-lived process ownership needs an incarnation, not a PID

#143 extended a watcher from a bounded startup window to the lifetime of an
agent, but initially polled only `kill -0 <pid>`. A numeric PID can be recycled
between slow polls, letting an unrelated process inherit the old watcher's
authority.

**Rule.** Any sidecar that owns a process across polling intervals must capture
an OS process-start token and compare `(pid, start-token)` on every poll. Its
stateful fake must support “old process dies; same PID, new token” as a distinct
transition; a `map[pid]bool` liveness fake cannot test ownership.

## Validate a PID before using OS pseudo-filesystem paths

#143's Linux process-identity boundary accepted the `/proc/self` alias from a
malformed pidfile, which could bind a long-lived watcher to its own process.

**Rule.** Before interpolating external PID text into `/proc`, `ps`, `kill`, or
similar process APIs, parse it once as a positive decimal integer and pass only
the normalized integer onward. Tests must include OS aliases, zero, negatives,
and nonnumeric input—not only empty and nonexistent numeric PIDs.

## Revalidate authority after slow IO, not only before it

#143 captured a stable process incarnation before each discovery pass, but the
pass itself crossed `ps`, `lsof`, and filesystem IO before writing the session
binding. PID ownership could change inside that interval.

**Rule.** When external identity authorizes a persistent write, validate it at
both ends of any IO-heavy discovery: before scanning, and again after selecting
the candidate immediately before persistence. Give the fake an in-call hook so
tests can change identity during the external operation; between-poll state
changes do not cover TOCTOU races.

## A screen-scraping recognizer must be validated against derived states, not just the captured screen (#139)

Three composer recognizers were each derived from a *startup* capture and each
rejected ordinary composing states — a blank line inside the message, a composer
grown past one line. The startup screen is the one state a capture makes easy
and the one state users spend the least time in. When a predicate keys on screen
structure, enumerate the states the feature is meant to *produce* (after one
newline, after several, with an empty line, grown) and pin each one; a fixture
of the initial paint proves almost nothing about them.

## Do not infer a discriminator's power without capturing the state it must reject (#139)

Agy's recognizer was tightened to require the composer prompt's bright blue,
on the assumption that its pickers paint markers unstyled. Driving the real CLI
showed Agy paints slash-menu selection markers in exactly the same bright blue.
The rule was a necessary condition sold as a sufficient one. If a rule exists to
reject state X, capture X and assert the rejection; otherwise record explicitly
that the rule is unproven rather than describing it as a defense.

## `go test` hides passing-package output, so a "loud" warning must be a failure (#139)

An evidence gap was reported with `t.Logf`, then `fmt.Fprintf(os.Stderr, ...)`.
Neither appears in `make test` — non-verbose `go test` suppresses output for
packages that pass. A warning that only shows under `-v` is a silent pass. Make
the condition fail, with an in-code acknowledgment list that must name each known
gap and that itself fails when an entry outlives the gap.

## When two states are provably indistinguishable, pin the resolution as policy (#139)

Codex's composer blank line and the gap above its status line are cell-identical;
so are a mid-frame composer row and the settled status row. No predicate can
separate them. The fix is not a cleverer heuristic but an explicit decision,
recorded as a test row with a comment saying which way it resolves and why, so a
later edit flips a policy visibly instead of silently changing behavior.

## Tag implementation commits with the issue number or the close loses its measurement window (#134)

pair#134's implementation landed in `e4d1557` with a descriptive subject and no
`#134` reference. Four days later the close could not measure actual hours at
all — `sdlc actual` derives active time from the commit range that references
the issue, and there was none — so it closed `actual_hours: N/A` and skipped
calibration permanently. The estimate stands unreconciled and that data point is
gone. The commit convention (`<area>: #N: <subject>`) is not cosmetic; it is the
input to velocity measurement. Guessing a number afterwards is worse than N/A,
because a fabricated actual pollutes the calibration the gate exists to protect.

## An issue left at `working` with every box ticked is not done (#134)

pair#134 had 7/7 plan rows ticked, all code merged to main, and passing tests —
and still sat at `status: working` for five days with no boundary review, no
measured hours, and no archive. Ticked boxes track implementation; the status
field tracks the gate. Worse, the durable plan showed 0/30 rows ticked while the
work it described had shipped, so the record read as unstarted. When work lands,
close it or say why it is blocked; a stale `working` hides both the completed
work and the open verification it still carries.

## A height bound on a composer recognizer is a lost-draft bug, not a safety measure (#138)

Claude's recognizer inherited a 20-row ceiling from a sibling harness. Past it
the gate declined and plain Return submitted the draft — on the default agent,
where the same keystroke had inserted a newline the day before. The bound bought
nothing: the box was already pinned by an immediate top rule and by taking the
first painted column-0 row below as the closing rule, so nothing distant could
pair into it at any height. Before copying a bound, ask what it excludes that the
structure does not already exclude; if the answer is "nothing", it is pure
false-negative surface.

## Capture stop-conditions must require a settled screen, not the first matching text (#138)

Two harnesses in a row produced fixtures caught mid-repaint — Codex with the
cursor parked on the status line, Claude with the cursor hidden — because the
capture stopped the instant the marker text appeared. A TUI paints in several
writes and a PTY read can land between them. Stop on the condition under test
actually holding (recognizer fires *and* marker present), not on the first byte
that mentions it.

## Mark byte-exact fixtures binary in .gitattributes (#138)

`git diff --check` flagged trailing whitespace inside a literal PTY capture. Any
tooling that acted on that — an editor, a pre-commit hook, EOL normalisation —
would corrupt a fixture whose SHA-256 is pinned in metadata. Captured evidence is
not text and should be declared as such the moment the first one lands.

## A test that survives deleting the seam it names tests nothing

A plan's test asserted that path resolution went through the `PathOps` seam.
Removing both `Physical` calls from the function left the test green, because
the fake's canned reply already equalled the expected answer. Separately, the
first function written in the same milestone kept an explicit `filepath.Clean`
that no test could miss: `filepath.Abs` already cleans its result, so the line
was dead on the success path.

**Rule.** For every test that asserts a *seam is used* or a *step happens*,
delete that call and confirm the test goes red before trusting it. Do the same
for any line you believe is load-bearing. Verify the deletion actually applied —
a `cp && python` chain that short-circuits on a blocked write leaves the file
untouched and reports a meaningless pass. Caught throughout #000145.

## `kill -0` succeeds for a zombie, so it is not a liveness check

`ExecRunner.Alive()` used `procutil.Alive`, which is `kill -0 <pid>`. That
succeeds for a child that has exited but not been reaped, so a finished process
reported as running until something called `Wait`. The stateful fake reported it
dead — correctly — and only a live real-vs-fake comparison exposed the
divergence. The first hypothesis was that the test scenario was wrong (that
signalling `sh` was not reaching `sleep`); adding `exec` changed nothing, which
is what redirected attention to `Alive`.

**Rule.** For your own children, reap in a background goroutine and make
liveness a closed channel, not a syscall. Reserve `kill -0` plus a kernel start
token for processes you did not spawn, and write down the window where a zombie
could still read as live. Caught in #000145 Task 16.

## Conformance means comparing two implementations, not running each

A live check that starts a real process, then separately drives the fake by hand
to the value it is about to assert, tests only the harness. Both halves pass and
no drift is detectable. The version that found a real bug encoded one scenario —
exit code, signal-ignored, signal-fatal — and asserted the two implementations
*agreed*.

**Rule.** Write the scenario once and assert real and fake produce the same
observation. Where the fake cannot know something a real process decides (a
child's signal disposition), make it scriptable and check both settings against
reality rather than assuming one. Gate with an env var and `t.Skip`, never a
build tag — a tagged file stops compiling under `go test ./...` and rots
invisibly. Caught in #000145 Task 16.

## Run the whole tree under -race, not just the package you touched

`go test ./cmd/... -race` had not been part of the loop. One run surfaced three
races: two in test doubles (a buffer polled across goroutines; a fake's map
written from two goroutines) and one **production** bug — `scribecmd` deferred
`ptmx.Close()` without stopping its SIGWINCH goroutine, which could `ioctl` a
recycled descriptor and resize an unrelated file.

**Rule.** Run the full tree under `-race` before claiming a suite is green, and
read the whole output rather than a `tail` — a truncated pipe reported one
failing package when there were four. When a signal-handling or drain goroutine
shares a resource with a deferred `Close`, stop and drain it first; register
that cleanup *after* the close defer so it runs *before* it, since defers are
LIFO. Caught while building #000145.

## A fixture that fights the policy it sits on deadlocks rather than fails

An actor test sent three identical messages and waited for four callbacks. The
mailbox collapses by kind, so three became one, the count never arrived, and the
test hung until the timeout instead of failing with a diff.

**Rule.** When a test drives a component through a policy layer, check the
fixture against that policy first — collapse, dedup, rate limits and batching
all silently change how many events arrive. Prefer distinct inputs over repeated
ones, and give any test that can block an explicit timeout so a fixture bug
reports as a failure rather than a hang. Caught in #000145 Task 9.

## A guard bypass must never bind positionally

`couch start <path> [same-tree]` bound argv positionally against the declared
argument list, so `couch start /repo true` set the escape-hatch flag and
silently disabled the one-agent-per-tree refusal. The first fix — "optional
arguments never bind positionally" — was a broader rule than the problem, and
it broke `couch describe <ref> <text>`, a command smoke-tested earlier in the
same session.

**Rule.** Mark arguments that bypass a guard as flag-only and bind them by name
alone; leave ordinary optional arguments positional. When a fix generalises past
the finding, exercise the neighbours it now governs before believing it. Caught
in #000145 close review round 3.

## A gated-only pin is not a pin

A regression test for a real bug was written into an opt-in suite behind
`PAIR_LIVE_COUCH`, which no target set. Restoring the bug left the default suite
green in 0.35s; only the gate caught it, and nothing ran the gate. The same
issue had already been raised one round earlier for a different fix, and the
second instance was introduced *while addressing the first*.

**Rule.** A fix is pinned by a test the default suite runs. If a check genuinely
needs an env gate, make the gate runnable from a target and add a hermetic
default-suite test for the same property — building a temp fixture rather than
resolving against the ambient checkout, so it can run anywhere. Caught in
#000145 close review rounds 2 and 3.

## A test that passes with the fix reverted is measuring something adjacent

Three times in one issue, a deletion check came back green: a symlink-seam
assertion whose fake ignored the argument that made it load-bearing; an
operation audit comparing two views of one source; a canonicalisation test fed
an already-canonical path. Each read as confirmation and confirmed nothing.

**Rule.** Revert the fix and require red before believing a regression test, and
check that the revert actually applied — a shell chain that short-circuits on a
blocked write leaves the file untouched and reports a meaningless pass. When the
check is green, the test is wrong, not the fix; fix the fixture so it can fail.
Caught throughout #000145.

## A repeating finding family means the enumeration was never written

The close gate reported "not converging: fix rules, not instances" for two
rounds while repeat families grew from four to six. Each finding carried a
`family:` slug naming the underlying rule and a measured prevalence — *2 of 9
accessors locked*, *7 of 9 tests on the non-fake path*. Fixing the flagged
instance each time left the class intact, and two of the round-3 findings were
families introduced *by* the round-2 fixes.

**Rule.** When a review names a family, the deliverable is the class: enumerate
every instance and state the rule, then grep for the shape rather than trusting
the fix. Prevalence in a finding is a worklist, not a label. Caught in #000145
close review rounds 2 and 3.

## A partly-run checklist ticked as done hides what its unrun steps would find

An operator smoke had five steps; step 1 ran and the box was ticked, with the
issue Log simultaneously recording that four steps had not run. Two of those
four were exactly where the two Critical findings lived — a second-shell read
and a repeat start would have surfaced both in under a minute. Later, the repair
for one of those Criticals shipped a fail-open path that only the same unrun
step caught.

**Rule.** Tick a checklist item only for steps actually performed; record the
rest as unrun in the same breath. When a Log says "steps 2-5 not exercised" and
a checkbox says done, believe the Log. Caught in #000145 close review round 1.

## An aliasing test must force an in-place OVERWRITE, not just a later write

#146 Task 1.1's first `Snapshot` aliasing test appended to a ring with spare
capacity, then asserted the snapshot was unchanged. It passed against
`return r.data` — the aliasing bug it existed to catch — because `append` wrote
*past* the snapshot's bytes rather than over them. The deletion check caught it;
inspection would not have.

**Rule.** To pin "this returns a copy", construct the case where the next
mutation writes **into the same indices** the returned value occupies — for a
bounded buffer that means filling to capacity so the trim's `copy()` shifts, not
leaving headroom so `append` extends. Same shape for any snapshot/defensive-copy
assertion: if the mutation you perform cannot reach the bytes you assert on, the
test passes with the copy removed.

Corollary for bounded buffers: `Snapshot` reports the *window*, so a buffer that
grows its backing array without bound looks identical from outside. Pin the
allocation separately (`cap()`), or "bounded" is untested.

## A smoke instruction must put the affordance in the operator's view

#146 M1 migrated `pair term`'s multiplexer onto extracted packages, and the
smoke instruction sent the operator to standalone `pair term` — the one mode
where `pair term` has **no tab indicator at all**. Its tab strip is rendered as
the *zellij pane title* (`renamePane` → `zellij action rename-pane "[terminal 1]
work"`), so outside zellij the call fails and is swallowed. The operator
reasonably read "I press Alt+t and nothing visibly happens" as a crash, and only
found the tabs by noticing that changing one tab's contents changed what came
back on switch.

**Rule.** Before writing smoke steps, ask *where does the thing I want observed
actually render?* If the affordance lives in a host the reduced-scope harness
does not have (a zellij pane title, a status bar, a notification centre), the
reduced harness cannot smoke it — either give steps in the real environment, or
say up front which signal is missing and what to substitute. A "simpler"
repro that removes the observable is not simpler, it is unfalsifiable.

Corollary: when the operator reports "X seems broken", check whether X is
*observable* in the setup you sent them to before investigating X.

## A deletion check proves nothing until you confirm the mutation APPLIED and traversed

#146 M1's boundary review found a "verified" claim that was false (BR-4: the
`Ring` copy-vs-re-slice change was logged as a bug fix pinned by a deletion
check; reverting it left the named test green). Fixing that round produced the
same failure twice more:

- A mutation written as a Python string containing `\x1b` silently became a real
  ESC byte, so `str.replace` matched nothing. The file was unchanged, the suite
  stayed green, and "the check passed" would have meant *the check never ran*.
- A mutation that did apply removed a `return` the test's input never reached —
  `\x1b[?1049r` exits at an earlier `final != 'h' && final != 'l'` guard, so
  deleting the later one changed nothing for that case.

**Rule.** A deletion check has three obligations, and only the first is usually
performed:
1. **Mutate.** Confirm the file actually changed (`git diff --stat`, or assert
   the replacement matched) — a no-op edit is indistinguishable from a passing
   check.
2. **Compile.** A build failure is not a red test; it proves nothing about the
   assertion.
3. **Traverse.** Confirm the mutated line is on the path the test's input takes.
   Removing a guard the input never reaches is a green check with no meaning.

**The same three obligations apply to ORDINARY edits, not just mutations.** Three
times in #146 M1 a scripted edit silently failed — a `str.replace` whose pattern
did not match, a script that raised before its `write()`, so *nothing* in it
landed — and each time the suite stayed green and the edit was reported as done.
An edit is not applied because you wrote it; it is applied because you checked.
Assert the match inside the script, or grep the result afterwards.

And name the mutation precisely in any log entry. "Ring trim" covered *removing
the trim entirely* and was true; it was written up as covering *copy vs
re-slice*, which it never touched. The gap between the mutation you ran and the
claim you make from it is where this class of lie lives.

## An async assertion must prove the change LANDED, not just poll for a state

#146 M2's operator smoke reported the reserved row appearing and then vanishing.
The emulator tests written to reproduce it **passed** — in 0.01s, on all five
cases. They fed the child a clear and immediately polled "is the row there?",
which was true from *before* the clear: the chunk had not reached the screen
yet. A green suite reported the bug as fixed while it was still live.

The second shape was wrong the other way. Waiting to OBSERVE the damage ("poll
until the row is gone, then poll until it returns") is flaky by construction:
when the repair is fast the damaged state may never be visible at all, and the
case that was already handled (RIS) started failing for the wrong reason.

**The marker must be set by the CONSUMER, not the producer.** This recurred five
times across #146 M2/M3 despite the rule below, and every recurrence had the
same shape: the wait condition polled something the PRODUCER sets synchronously
(`child.Feed` updates the ring immediately), so it was already true before the
consumer had looked at anything. Twice that produced a false PASS on a live bug;
twice a deletion check failed to fire and the test was proving nothing; once a
false FAIL.

Ask of every wait condition: **could this be true before the code under test
ran?** If yes, it is not a marker. Reach for something only the consumer can
set — output it emits, or state it records — and remember the queue is FIFO, so
a later marker proves the earlier item was drained.

**Rule.** For a poll-based assertion over an async pipeline, establish ordering
with a MARKER rather than with timing. Send the stimulus, then send something
whose arrival is observable and ordered behind it; wait for the marker, then
assert. Ordering through a channel or a stream is a guarantee — "it has probably
happened by now" is not, in either direction.

Corollary, and the reason this class keeps recurring: ask what the assertion
reports **before** the action runs. If it is already the value you want, the
test cannot fail for the reason you think it can. Same defect as an aliasing
test whose mutation cannot reach the bytes it asserts on.

## Injecting into a stream you don't own needs a single writer AND a scanner fed only by that stream

#146 M2 shipped a status row painted into the same terminal a child is writing
to. Getting that right took three attempts, and each wrong one looked correct:

1. **Ask the child whether it is mid-sequence.** Wrong stream: the child's
   scanner had already consumed chunks the console had not yet written, so the
   answer described a different point in time.
2. **Track the stream the console writes — but only guard one writer.**
   `applyLayout` (on SIGWINCH) and the hotkey path still wrote from their own
   goroutines, so two of three writers bypassed the check.
3. **Feed the console's own escapes into that scanner.** Appending `\x1b[1;23r`
   to a pending `\x1b[38;2;76` let the scanner frame them together as one
   complete sequence, so it reported "safe" exactly when it was not.

**Rule.** To interleave your own output into a stream produced by something
else: (a) make ONE goroutine the only writer, so everything else sends events
rather than bytes; (b) frame the stream at the point of writing, not at the
point of reading; (c) feed the framing scanner ONLY the other party's bytes —
your own are known-complete and including them corrupts the very state you are
consulting.

Corollary for tests: the bug lives in the SKEW between producer and consumer, so
a test that synchronises them cannot see it. A reviewer's phrase for the version
that waited for the console to catch up before continuing: "avoids the window
rather than covering it."

## A capability audit that checks DECLARATION passes on a list that does nothing

`#146` M3 shipped a panel whose `PanelActions()` returned `start, stop, name,
describe`, with an audit asserting every name is a declared `couchcore`
operation. It passed. Nothing was wired: no keystroke reached any of them, so
the operator opened the panel and had no way to start a second child. The audit
was satisfied by a string slice.

Same shape as a gated-only pin, one level up: the check tested that the CLAIM was
well-formed, never that the claim was true.

**Rule.** When a component declares what it can do, the audit must check the
declaration is REACHABLE, not merely consistent. For a keyboard surface that
means every declared action maps to a key and no two share one; for an API it
means every declared operation has a call path a test exercises. Pair the
subset check ("nothing undeclared") with a coverage check ("nothing declared
that cannot be invoked") — the first alone is passed by an empty implementation.

Corollary, and it is the cheaper detector: if a feature is declared and the
operator asks *"how do I actually do this?"*, the audit that should have caught
it was checking the wrong direction.

## Framing input is not optional once you accept keystrokes

The same `#146` panel took any printable byte as typeahead. An SGR mouse report
is `\x1b[<0;12;4M` — every byte after the ESC is printable — so moving the
mouse over the panel typed `[<;0;M[<;;M…` into the filter, which then matched
nothing, rendered "(nothing running)", and left no way back because Escape was
not handled either.

**Rule.** Any surface that consumes terminal input must FRAME escape sequences
before interpreting bytes, and drop the ones it does not use rather than letting
them decay into text. Route it through the repo's existing scanner
(`cmd/internal/ansi`) — a second framing decision is the bug this repo has paid
for repeatedly. And decide explicitly what the ESCAPE key does: a picker with no
way out is a trap, and "nothing happens" is what the operator sees.

## A key encoding fix must cover EVERY key, not the one that was reported

`#146` M2: ctrl-space never reached couch, because zellij enables the Kitty
keyboard protocol and the terminal sends `\x1b[32;5u` rather than NUL. Fixed —
for ctrl-space. M3 then shipped a panel whose Escape, Enter and arrows were all
dead for the identical reason, and the operator reported the same class of bug a
second time.

The evidence was in the tree both times: pair's own chord table carries BOTH
encodings for every chord (`workbenchshortcut/shortcut.go`), which is what a
keyboard surface in this repo is supposed to look like.

**Rule.** Terminal key encoding is a property of the MODE the terminal is in,
not of a particular key. When one key turns out to arrive in an unexpected
encoding, enumerate every key the surface consumes and handle both forms for all
of them in the same change — a per-key fix guarantees the next key reports the
same bug. Decode the codepoint (`CSI <n> ; <mods> u`) rather than listing byte
strings, so a key nobody thought about still decodes.

Corollary: a surface that takes over the screen inherits whatever keyboard mode
the previous occupant set. It does not get to assume the default.

## A refusal that names an action you cannot perform pushes the operator to the bypass

`#146`'s one-agent-per-tree guard refused correctly and then advised "switch to
it, or --same-tree". couch has no switch verb -- attaching to a session another
process hosts is a different issue's work. So the only followable half of the
advice was the flag that turns the guard OFF.

**Rule.** Every remedy a refusal offers must be a command that exists today. If
the natural remedy is not built yet, say so explicitly ("attaching needs X")
rather than naming it as an option -- an operator who cannot follow the safe
advice will follow the unsafe one. Where the surface has a declared verb set,
assert in a test that each suggested command is in it, so the advice cannot
drift from the implementation.

## A stream split is not an event boundary

`#146` M3's interceptor correctly returned `before / hotkey / rest`, but the
stdin goroutine queued the hotkey and immediately routed `rest`. The Run
goroutine had not necessarily changed focus yet, so bytes logically after the
hotkey could still reach the actor being left. The unit test proved the parser's
split and missed the consumer's scheduling race.

**Rule.** If bytes after an input control depend on that control taking effect,
the control needs an acknowledgment (or all routing belongs on one event loop)
before the suffix is consumed. Enumerate every legal read split in a composed
test; parser tests alone cannot prove routing order. The same rule applies to a
bare ESC that might be the prefix of a following CSI: read boundaries carry no
semantic meaning, so resolve the ambiguity explicitly.
Generate the split cases from the production recognition table; a handwritten
representative split is how the first-byte boundary escaped #000146 M3 twice.

## A displayed model must have one production constructor

`#146` M3 built and thoroughly tested `NewPanelModel(TreeSummary)`, including
parked trees, then production constructed a second `PanelModel` directly from
hosted panes. Both versions rendered, so ordinary live-actor smoke passed while
parked rows and refreshed metadata disappeared.

**Rule.** When a pure model constructor is the declared source of UI state,
production must call it. Runtime-only data may be joined afterward through a
named pure transform; it must not become a parallel constructor. Pin the real
wiring with one fixture containing state that exists only in the domain source
(here: a parked tree), because the intersection of two sources cannot reveal
which one production consumed.

## Printable command keys and direct typeahead cannot share a mode

`#146` M3 treated `s`, `x`, `n`, `d`, and digits as commands only while the
query was empty. That made a query's interpretation depend on its first byte:
some visible names could never be typed even though typeahead appeared to accept
ordinary text.

**Rule.** A typeahead surface that promises direct printable input reserves no
printable prefix for commands in the same mode. Put commands behind an explicit
namespace or modifier and test every command rune as the first query byte. A
help line is part of this contract and must be updated from the same inventory.

## Liveness is not local routability

`#146`'s panel joined a global live-actor summary with console-local child
targets, then treated a missing target as proof that the worktree was parked.
A live actor hosted by another couch process has exactly that shape, so Enter
attempted to start a duplicate instead of explaining that attachment transport
was unavailable.

**Rule.** When global state is joined with process-local capabilities, model
the facts independently. Test all resulting states—in this case local-live,
remote-live, and parked—and authorize an action from the capability it needs,
not from the absence of a different capability (ARCH-PURPOSE).

## A client PID is not the lifetime of the durable session it views

`#149` M1 initially released bounded-path capacity when the recorded Pair
client PID died. Pair is only a zellij client, however; the zellij session and
its workspace-writing panes can survive it, so this admitted a second writer
while the first durable incarnation was still active.

**Rule.** Release concurrency capacity only from evidence about the complete
incarnation named by the policy, not a convenient child or client process.
When a harness has a server/client split, test client death with the server
still live and retain occupancy until a whole-incarnation quiescence seam says
otherwise (ARCH-PURPOSE, ARCH-MOCK).

## Uniqueness must cover every durable representation of an identity

`#149` M1 first checked opaque tags only against ThreadStore. Existing Pair
drafts, configs, logs, and detached-session bindings could therefore already
own the same composite address even when no ThreadStore record existed.

**Rule.** Before claiming a generated identity, enumerate and check every
durable representation that can resolve to it. Prefer a structural ownership
rule that covers the whole namespace (here: an exact tag token with filename
boundaries inside Pair's owned scope) over a prefix enum: constructors in Go,
Lua, layouts, and future consumers otherwise drift independently. Keep the
collision seam explicit until all producers share one transaction (ARCH-DRY,
ARCH-PURPOSE).

A scan followed by a separate claim is still a collision bug: another producer
can create state in the interval. The shared authority must be acquired first
(for files, an O_EXCL marker is sufficient), and every current producer must
participate before writing its first durable representation. Test simultaneous
claimers and the reserved-to-established handoff, not only preexisting files.

## A pidfile is a readiness promise, not merely process metadata

The wrapper wrote `pair-wrap-pid-<tag>` before registering its SIGUSR2 handler.
Tests usually slept long enough to hide the interval, but a loaded full suite
could observe the pidfile, send restart, and lose the signal before the handler
owned it.

**Rule.** Install every handler and initialize every state transition that a
pidfile-triggered caller depends on before publishing the pidfile. Tests should
act immediately when the file appears; adding sleep only widens the disguise.

## Acknowledgement transfers execution permission, not ownership

`#149` M2 initially acknowledged the pre-exec helper and then returned errors
from registration, promotion, or cache persistence while the target kept
running. The caller discarded the handle on error, leaving a workspace writer
with no supervisor responsible for it.

**Rule.** Treat acknowledgement errors as possibly delivered: a successful
write followed by a close error cannot be revoked by `Cancel`. Enumerate every
exit after an acknowledgement attempt and before ownership handoff. Each must
either transfer ownership or quiesce the whole incarnation—not merely the held
client when it can leave a server/session and workspace-writing descendants—
then preserve occupied durable state whenever reconciliation is uncertain.
Test the complete failure-site table with a real orphanable descendant, not one
representative branch or a single-process fake. The regression must perform
cleanup through the production ownership boundary; a fake hook that kills the
descendant itself proves a capability production may not have. Enumerate every
pre-handoff process class: keep ordinary descendants and Couch-launched
sidecars in an actor-owned process group, and clean separately detached servers
through their exact durable binding. A destructive command returning does not
prove absence: observe the exact external state afterward, model re-registration
in the stateful fake, and fail closed on query, deletion, or escalation errors
without returning ownership. A process-table match is not destructive authority:
carry PID plus a kernel start token and reauthorize both identity and exact argv
immediately before signalling. Every stateful external fake also needs a
committed live target and cadence. Retrying while ownership is retained must
reuse one wait-result channel for the exact process; a fresh `Wait` goroutine on
each attempt leaks blocked waiters. A live conformance test must make every
relied-on external operation load-bearing—for zellij teardown, explicitly
observe server enumeration, delete dispatch, and exact-server escalation—not
only assert a final absence that a weaker path can also produce. Instrument the
lowest injected effect seam, and construct a live target that the preceding
operation cannot remove, so a higher-level method-entry flag cannot stand in
for the external effect (ARCH-PURPOSE, ARCH-MOCK).

## Crash-recovery evidence must be atomically published

`#149` M2 changed the durable registration marker from reserved to established
with `O_TRUNC` followed by write. Concurrent readers could observe empty or
partial JSON, and a crash could permanently strand malformed evidence.

**Rule.** Publish a state transition used as concurrent or crash-recovery
evidence by writing and syncing a same-directory temporary file, atomically
renaming it, then syncing the directory. Synchronize a reader before rename and
prove it sees the complete old value, then the complete new value—never a
transient parse error (ARCH-PURPOSE).

## PURE fixtures must be literal at their direct boundary

`#149` called `ThreadRecord` and `StartTransaction` pure, but their shared
fixture created and resolved temporary directories, and a runner lifecycle test
sat beside the direct transition tests. The production functions were pure;
their claimed direct tests were not.

**Rule.** Direct tests for a PURE entity use literal values and no filesystem,
process, network, or integration fake. Put store/runner conformance in an
explicitly integration-named file, and mechanically reject integration seams
from the direct-test files (ARCH-PURE).

## Optional durable indexes need a typed absence state

An optional index reader cannot collapse missing, corrupt, and incomplete into
one error branch. Once durable authority exists, falling back to legacy lookup
can silently create a second identity.

**Rule.** Tolerate only a typed absent-store result. Propagate every malformed
or incomplete authoritative read before any launch or mutation, and test both
directions through the production decision flow (ARCH-PURPOSE).

## Required string arguments validate presence, not truthiness

Empty strings can be meaningful commands, especially clearing metadata. A
schema check using `value == ""` erases the distinction a patch type preserved.

**Rule.** Validate required map keys by membership. Let the operation-specific
executor decide whether an explicitly empty value is valid, and pin every
declared clearing path.

## Composite references carry their collision domain

A tag, path, or human name is not an address when the same value can occur in
several repository scopes. Correct storage does not help if one CLI consumer
drops scope before resolution.

**Rule.** Every composite-address consumer either carries the exact address or
derives scope at its boundary. Test a repeated tag across scopes at the public
entry point, including reads as well as writes (ARCH-PURPOSE).

## A declared effect needs one mechanically enforced authority

Routing most calls through a dispatcher still leaves two semantics if a
startup path can invoke the primitive directly.

**Rule.** Keep the lowest effect primitive private to its executor's package,
then test the typed declaration emitted by each external wiring path. This
turns dispatcher bypass into a compile error rather than a review convention
(ARCH-DRY, ARCH-PURPOSE).

## Durable-read callback types must carry failure

An error-aware store is still fail-open when a UI callback returns only a
slice: the adapter has no representation except “empty,” so corruption becomes
indistinguishable from valid absence.

**Rule.** Every callback crossing a durable-record read returns `(value,
error)`. The consumer preserves the last valid state where possible and shows
the failure in its owned surface. Test the production adapter with a failing
real store read as well as the consumer's visible error behavior
(ARCH-PURPOSE, ARCH-MOCK).

## Compact and diagnostic renderers have different identity duties

Human-first lists reduce noise, but a detail view must still expose immutable
identity for exact commands and support.

**Rule.** Share row rendering mechanics while making identity visibility an
explicit mode: compact lists may hide a named system id; diagnostic views must
always print the full durable address. Pin both halves so one shared renderer
cannot flatten the distinction (ARCH-DRY, ARCH-PURPOSE).

## A portable projection must share the owner's acceptance contract

A read-only consumer can project fewer fields after validation, but a partial
shadow decode schema silently defines a second set of valid durable states.
Missing fields and malformed nested data can then influence decisions even
while the owning store refuses them.

**Rule.** Put the complete persisted wire shape and structural validator below
all readers and writers. Project only after that shared decode succeeds. Keep a
cross-reader mutation table enumerating every required top-level, address,
nested-record, generation, and path-binding invariant; each reader must reject
every mutation (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

## Durable index relocation needs an overlap-read epoch

Moving the writer to a scoped location does not move already-live state. A
reader that switches locations atomically can strand detached sessions even
though both old and new files are individually valid.

**Rule.** During a durable-index relocation, read and strictly merge every
prior authoritative location before the new one, while writing only the new
location. Tolerate missing files only; malformed or unreadable present state
must stop every identity-dependent or destructive caller. Pin a legacy-only
live record, a mixed old/new record, and each fail-closed effect path.

## Constructor closure includes source shape and derived companions

An extension-only source scan misses extensionless scripts, and a token scan
misses `exact_path .. suffix` derivations. Both allow a claimed path authority
to coexist with real constructors outside it.

**Rule.** Enumerate production source by language extension or shebang, route
intentional legacy reads through an explicit compatibility API, and test exact
non-Go bindings against forbidden sibling derivations. When one artifact has
companions, their derivation belongs in the path authority too
(ARCH-DRY, ARCH-PURPOSE).

## Negative scans do not prove a single authority

A blacklist can reject known constructor shapes while a split literal, helper,
or new expression form still derives the same artifact elsewhere. A source's
self-declared classification is not evidence that it consumed the authority.

**Rule.** For every claimed artifact family, require a positive, mechanically
checked derivation witness from the owning resolver through the family-specific
member, and make every production source explicitly participate or explicitly
declare non-participation independent of token discovery. Track lexical value
identity and reject discarded witnesses; then scan each construction
independently, because one valid witness proves participation but cannot bless
a second path in the same file. Include runtime fragment assembly, not only
compile-time constants. Keep exact protocol/CLI vocabulary in a closed, counted
allowance that cannot witness path authority. Derive architecture inventories
from the owning declarations when the source boundary is mechanically
enumerable; do not create another expected-row list. In plans, state
verification recipes from their initial filesystem and environment state—such
as no `.git` and no generated assets—and make required architecture rows an
executable inventory so review can reject proof that depends on developer
residue or corrected-but-unpinned prose (ARCH-DRY, ARCH-PURPOSE).

## Plan review must challenge the proof shape

A plan can name the right authority and still propose evidence that recognizes
only the author's preferred syntax or selected source files. That defect is
cheapest to find before implementation.

**Rule.** For every exclusivity or whole-diff claim, plan review asks what
positive witness proves derivation, what exhaustive source enumerates the
population, and which adversarial mutations vary lexical scope (cross-file
package and local), order, aliasing, helper indirection, runtime composition, and initial
filesystem state. Participation must fail closed: a newly exported authority or
new source/declaration cannot silently receive the non-participating/default
disposition; close the audited declaration population mechanically when
individual detail markers would be noise.
If those witnesses are absent, the plan is not ready for `change-code`, even
when its happy-path tests are precise (ARCH-PURPOSE).

## Static enforcement must name its bounded language

`#149` M5 turned a useful repository inventory into a home-grown Go provenance
analyzer. Each review found another legal construction—ordering, helpers,
package globals, cross-file flow, builders, then compound assignment—and the
response kept extending the evaluator while still calling it exhaustive.

**Rule.** A source-analysis test must state exactly which syntax and boundary it
recognizes. Treat literal and AST scans as defense in depth unless they are
backed by a real typed semantic model; do not call them proof of arbitrary
program behavior. Prove the shipped repository with exhaustive current-source
inventory, positive dependencies, and integration behavior. If a boundary
review repeats the same family after five rounds, stop and bring the proof
shape to the operator instead of adding another syntax case (Simplicity First,
ARCH-DRY, ARCH-PURPOSE).

## Process-boundary tests must own shutdown and join

Writing a release file in `t.Cleanup` is not sufficient when a fake child polls
that file: Go may remove its `t.TempDir` before the child observes the release,
and an unjoined goroutine may restore process-wide environment after the next
test begins. Repeated verification of #154 leaked dozens of orphan fake-Zellij
shells before this failure path was inspected.

**Rule.** A test that starts a goroutine or external process must register one
cleanup owner that releases it and boundedly joins its completion before temp
directories or global environment are restored. Give fake processes their own
bounded timeout, bound every synchronization receive, and verify repeated/race
runs leave no matching processes behind (ARCH-MOCK, ARCH-PURPOSE).

## Milestones inherit repository-wide contracts immediately

A new package can pass every focused test while leaving source inventories and
historical declaration contracts red. Deferring those catalogs to a later
migration task means the branch has no green boundary between milestones.

**Rule.** Before the first commit that introduces production sources, run and
update every exhaustive repository source/declaration contract those files
enter. Include those focused contract commands in the milestone verification,
not only in final migration (ARCH-PURPOSE).

## Core-concept tables need declaration witnesses

A greppable architecture table still drifts when names and paths are maintained
only in prose; adding aliases can make the names look correct while their
declared locations stay false.

**Rule.** For a plan with a Core Concepts table, mark the owning declarations
and mechanically compare both directions across concept name, pure/integration
kind, status, milestone, and source path. The contract must reject an unmatched
row and an unmatched marked declaration (ARCH-DRY, ARCH-PURPOSE).

## Exact schemas need presence-aware types and executable registries

Value-typed nested JSON structs collapse absent objects into valid-looking zero
values. Hand-written diagnostic severities and comparator shortcuts likewise
drift from a precise schema while ordinary happy-path tests stay green.

**Rule.** Represent optional structured input with pointers or explicit
presence bits. Turn exhaustive code/severity sets, stable-ID tuples, and total
ordering tuples into table-driven tests—including equal primary keys—and have
production derive from the same registry (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

## Recursive storage scans must classify before opening

Treating every non-directory entry as a file admits FIFOs, sockets, devices,
and symlinks; opening one can block or escape. Returning a partial walk result
and then discarding it at the caller also defeats partial-inventory semantics.

**Rule.** At the runtime boundary, admit only regular non-symlink files and
return rejected entries structurally. Every caller preserves valid siblings
alongside diagnostics for rejected entries or later traversal errors; tests
include a real special file and an injected partial failure (ARCH-PURE,
ARCH-MOCK, ARCH-PURPOSE).

## Compatibility caches must not survive an authority downgrade

A restart path can correctly read a current typed ledger and still reintroduce
stale identity one function later by falling back to a compatibility config.

**Rule.** Once a stronger authority reports “present but provisional,” carry
that absence explicitly through every restart/composition layer. Fallbacks are
allowed only when stronger authority is absent, never when it deliberately
withholds a value. Test the mixed state: current provisional authority plus a
stale populated cache (`ARCH-PURPOSE`).

## Durable text framing must represent the writer's entire input domain

A visual Markdown delimiter is not a record delimiter when operator-authored
Markdown may contain the same bytes.

**Rule.** For durable arbitrary text, use versioned length framing (or escaping
with a proven inverse), and retain an explicit legacy decoder. Add a
writer-to-parser round-trip containing the old delimiter and a header-shaped
body, not only parser unit cases (`ARCH-PURE`, `ARCH-PURPOSE`).

## Project milestone closure fields move as one state

Project actual/closed fields paired with an unchecked milestone are internally
contradictory. The close command also requires the load-bearing project detail
block before it dispatches review, so omitting all pre-gate metadata is not a
valid workaround.

**Rule.** When preparing a project milestone boundary, update its checkbox,
actual, closed date, and detail block together, exactly as `sdlc` preflight
requires. Never stage only the metadata or only the checkbox. Issue/plan gate
state remains owned by the successful close transaction (`ARCH-PURPOSE`).

## Optional corroboration is available only when it can discriminate

The mere presence of process open files does not mean process evidence can
corroborate a native root. Treating unrelated files as available evidence can
filter every otherwise valid causal match.

**Rule.** Optional corroboration reports “available” only when at least one
observed fact maps through the authoritative model to a candidate it can
distinguish. An irrelevant non-empty observation set is absence, not negative
evidence; test both states through the production fake (`ARCH-PURPOSE`,
`ARCH-MOCK`).

## Public schemas project from internal models explicitly

Serializing an internal evidence struct directly leaked an explanatory root ID
and encoded required empty arrays as null, despite stable schema-v1 prose.

**Rule.** A versioned public schema owns a dedicated DTO containing exactly its
documented fields and presence semantics. Pin a non-empty independent golden;
empty-output goldens cannot detect nested field leakage or null/empty drift
(`ARCH-PURE`, `ARCH-PURPOSE`).

## Evidence filters must distinguish unrequested from unsupported

Silently skipping a recognized evidence record is safe only when the record is
valid, its versioned values are supported, and its supported owner was simply
not requested by the caller.

**Rule.** Exhaustively classify every recognized versioned evidence path. Only
valid supported-but-unrequested evidence may be silent; unsupported enum
values, malformed owners or paths, rejected ownership, unknown native IDs, and
read failures must produce registry-backed diagnostics. Test the filter and
every rejection class together so a new adapter cannot copy an incomplete
default (`ARCH-DRY`, `ARCH-PURPOSE`).

## Optional evidence must stay optional when acquisition is unavailable

Finding a process identifier does not guarantee that the platform can produce
a stable process-incarnation token. Treating that missing token as negative
evidence can suppress a unique portable match before the stronger evidence is
even evaluated.

**Rule.** Optional evidence constrains a decision only after every field needed
to interpret it is usable. If acquisition is unavailable, continue through the
portable authority path; if a usable token later changes, fail closed. Test
both unavailable and changed-token states (`ARCH-PURE`, `ARCH-PURPOSE`).

## JSONL bounds apply to records, not histories

A whole-file cap silently turns healthy long-running sessions into unreadable
ones even when every individual record is small and valid.

**Rule.** Stream append-only record histories through the injected range seam,
bound each record, and test accepted evidence beyond the former whole-file
threshold. Consumers may accumulate content only when their public contract
requires it; causal and metric projections stay streaming (`ARCH-PURE`,
`ARCH-MOCK`, `ARCH-PURPOSE`).

## Mixed-format stores need one row classifier

When a compatibility writer and a typed writer intentionally share one file,
independent readers can label a supported row as corruption while another
reader accepts it.

**Rule.** Put typed, compatibility, and malformed row classification in the
store-owning package. Every reader derives from that classifier, and a mixed
typed/legacy/corrupt fixture pins all three dispositions. Compatibility means
the complete strict historical shape with every required field, allowed
optional fields only, valid field types, and a supported owner—not merely a
recognizable discriminator. Every object at every nesting depth must also have
unique keys; derive from the shared strict JSON decoder instead of assuming
Go's standard decoder rejects duplicates. Required numeric/boolean fields need
explicit presence tracking because zero values cannot distinguish omission or
`null` from a valid zero; optional fields may be absent, but explicit `null`
must not masquerade as absence (`ARCH-DRY`, `ARCH-PURPOSE`).

## Shared identity does not imply shared native parsing

A consumer can correctly query the authoritative root and still recreate an
agent-specific transcript parser after it receives that artifact. That moves
selection authority but leaves schema authority duplicated.

**Rule.** Native paths, record schemas, and record-to-event normalization all
belong to the inventory package. Consumers request bounded typed projections,
never raw transcript bytes. The source shadow sweep names former adapter
symbols as well as native path patterns, and a long-history consumer regression
proves the projection is streaming (`ARCH-DRY`, `ARCH-PURPOSE`).

## Historical contracts need an immutable source boundary

A contract derived from `base..HEAD` can pass against uncommitted edits and fail
immediately after those same edits are committed, because the source universe
changes underneath the assertion.

**Rule.** Historical milestone contracts derive their source universe from the
milestone's immutable base and head commits (or an equally immutable marker
inventory), never the current `HEAD`. Verify once after commit when the contract
itself depends on Git objects (`ARCH-DRY`, `ARCH-PURPOSE`).

## Format classification includes enum ownership

Strict JSON structure is insufficient when a structurally valid record can name
an owner the runtime does not support.

**Rule.** The store-owning classifier validates both shape and supported enum
values for every current and compatibility format. Exercise each row kind with
an unsupported owner through every downstream projection (`ARCH-DRY`,
`ARCH-PURPOSE`).
## 2026-08-28 — Durable intent is not completed action

- When a durable record is written before an external action, encode prepared
  and completed states separately; recovery must never infer that the external
  action happened merely because its intent record is readable.
- Concept-table contracts must enumerate every allowed introduction stage and
  fail on every new exported type in owned packages until it is classified as
  a core concept or an explicit implementation detail.

- A post-action evidence marker must consume the external action's actual
  result. Preserve a commit-only retry identity after confirmed delivery;
  retrying the external action is duplication, while forgetting the identity
  strands true evidence.
- Derive architectural ownership inventories from durable change/plan
  authority. Hand-listed directories and exported-only scans are escape
  hatches, not exhaustive concept contracts.
- External action tests must model state across failure→retry, not merely inject
  isolated return codes. Preserve the last confirmed phase so retries resume
  instead of replaying already-applied effects.
- A transaction regression is not end-to-end when adjacent production seams
  are replaced by independent always-successful stubs. Boot the real wiring,
  inject state only at the external boundary, and assert the durable parser's
  output alongside external state.
- Place test injection beneath the behavior under test. A parallel test-only
  return branch can model the right outcome while mutation proves production
  propagation is completely unguarded.

## 2026-08-29 — A primitive is not an authority until consumers derive from it

- A planned façade, migrator, or reconciliation function needs a production
  caller and an integration test that observes its durable effect; test-only
  reachability does not deliver the architecture (`ARCH-PURPOSE`).
- Serialized publication prevents byte corruption, not semantic regression.
  Same-key writers must merge monotonic cursors and fail-closed disputes against
  the state reread under the lock (`ARCH-PURE`).
- When upgrading an append-only record, projection must select the newest
  compatible record for the same identity; otherwise durable migration can
  succeed while every consumer remains pinned to the legacy row (`ARCH-DRY`).
- A background worker is not production-reachable merely because a production
  binary contains it. Pin the lifecycle event that starts it and the durable
  effect through the real OS seam.
- Monotonic state with multiple cursors needs a partial order over every axis;
  a larger raw EOF cannot justify a smaller parser-complete cursor.
- Authority-required publication must never fall back to a weaker legacy
  format after a prerequisite store fails. Encode “no proof, no binding” at the
  shared persistence boundary, not only in its caller.
- Global cache classification and per-launch causality answer different
  questions. Catalog timing may govern reuse after authorization, but cannot
  erase that an artifact was absent from a particular launch baseline.
- An incremental validator is not incremental across processes unless accepted
  advancement is durably published and the next query starts from that state.
  Pin append-once/query-twice behavior with zero reads on the second query, and
  make the stateful fake consume the same pure publication rule as production
  (`ARCH-DRY`, `ARCH-PURPOSE`, `ARCH-MOCK`).
- An ownership package cannot be exempted wholesale from its own shadow sweep:
  it contains both legitimate definitions and potential consumer shortcuts.
  Parse call sites and use a closed, stale-checked allowlist for the few named
  diagnostic/compatibility callers; mutation-test an offender inside the owner
  package (`ARCH-DRY`, `ARCH-PURPOSE`).
- A call-expression name sweep is not a closed authority boundary because a
  function value can be aliased before invocation. When the forbidden surface
  is a small named API, classify every AST reference (including selector
  expressions), then allow only exact function sites and mutation-test direct,
  local-alias, and selector-alias forms (`ARCH-DRY`, `ARCH-PURPOSE`).
- `testing.T.Cleanup` runs LIFO. Register process/sidecar shutdown after
  `t.TempDir` creation so writers are stopped before the temporary directory's
  automatically registered removal; the reverse order creates intermittent
  “directory not empty” cleanup failures.

## 2026-08-30 — Recovery and conformance must cross the production boundary

- Construction may compose local durable stores, but it must not enter active-
  park reconciliation or external observation. Pin startup with a barrier at
  the first production query, then run recovery in one context-owned serial
  worker (`ARCH-PURE`, `ARCH-CONSTRAINTS`).
- A fake/real conformance claim must drive the complete semantic scenario
  through the production entrypoints and let the shared runner assign trace
  labels. Comparing a hand-selected suffix such as cleanup stages allows the
  transaction boundary that motivated the fake to drift untested (`ARCH-MOCK`,
  `ARCH-PURPOSE`).
- A plan's concept inventory must use delivered symbol names and paths. When the
  table is a review input, parse it and resolve declarations mechanically so a
  refactor cannot leave design-time aliases masquerading as current code
  (`ARCH-PURPOSE`).
- A startup barrier belongs at the first external observation, not merely the
  first destructive call. A read-only daemon query can wedge just as long as
  teardown; constructors should stop at local durable composition and let the
  context-owned worker perform all external recovery (`ARCH-PURE`,
  `ARCH-CONSTRAINTS`).
- A live driver must observe the effect it claims production caused. If the
  driver kills the child, invokes cleanup, or publishes evidence itself, the
  probe can pass after the production trigger is bypassed. Add an intent-only
  mutation that leaves the real handoff blocked and therefore fails
  (`ARCH-MOCK`, `ARCH-PURPOSE`).
- Changing a shared operation's required arguments is a consumer migration,
  not a declaration-only edit. Enumerate every production dispatch site and
  run each consumer package's integration regressions in the same boundary;
  focused executor tests cannot expose a Console rejected before its fake runs
  (`ARCH-DRY`, `ARCH-PURPOSE`).
- A fail-closed projection over durable records must invoke the record's shared
  structural validator before interpreting selected fields. Positive fixtures
  must themselves be valid persisted shapes; otherwise tests normalize corrupt
  evidence as the happy path (`ARCH-PURPOSE`).
- When production-source participation is exhaustive, every new source file
  must be classified in the same commit and the exhaustive inventory test must
  be part of changed-package verification (`ARCH-PURPOSE`).

## Terminal release must reset emulator modes as well as termios

Restoring cooked termios does not revoke DEC private modes previously replayed
from a child. Couch returned to the shell with any-event/SGR mouse reporting
still enabled, so ordinary pointer movement became printable escape input.

**Rule.** A terminal multiplexer/proxy's single teardown owner must restore both
kernel line discipline and a shell-safe terminal-emulator mode baseline. Reset
mouse tracking/encodings, focus events, bracketed paste, synchronized output,
and extended keyboard reporting on every exit path. Test by enabling a mode in
the child stream and asserting the reset follows it before return
(`ARCH-DRY`, `ARCH-CONSTRAINTS`).

## 2026-08-30 — Concurrency controls and negative probes must be consumed

- A concurrency primitive is not an invariant until every production
  entrypoint consumes it. When introducing a single-flight worker, add an
  overlap test through two real callers (for example startup recovery and UI
  dispatch), and require one external effect plus one shared committed result
  (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).
- A negative integration test must prove it crossed the precondition it means
  to mutate and failed at the expected stage; accepting any error lets setup,
  permissions, or an unrelated dependency failure produce a false green.
  Stateful host-daemon fixtures that are not isolated must run sequentially
  even when each package retains its normal internal test parallelism
  (`ARCH-MOCK`, `ARCH-CONSTRAINTS`).

## Staged consumer migrations need a cross-document current-state contract

Correcting one atlas sentence left a project milestone claiming that an
authority introduced in M1 was already consumed by a UI deliberately deferred
to M3.

**Rule.** For a staged consumer migration, distinguish “authority exists” from
“consumer is wired” in every current-state surface: atlas, project milestone,
issue log, plan revision, and operator README. Pin the actual production
provider and those declarations in one regression so the consumer migration
must update the entire class together (`ARCH-PURPOSE`).

## Hierarchical reducers must preserve identity across every projection

- Capture a unique attempt identity plus the operation's target and originating
  frame instance when dispatch occurs, then reject mismatched completions before applying
  their returned inventory. Operation/address identifies a target, not an
  attempt, and frame kind/depth identifies a structural position, not the frame
  that occupied it; otherwise an old completion can mutate newer work.
  Looking only at the currently visible frame loses root-level operations and
  lets stale results rewrite state (`ARCH-PURE`, `ARCH-PURPOSE`).
- Exact origin identity bounds what an asynchronous completion owns; it does
  not grant ownership of UI created afterward. When completion restores or
  collapses the captured stack, transform only its originating prefix and
  preserve unrelated later overlays by instance. Sweep every operation and
  outcome after legal post-dispatch navigation, not only replacement at the
  origin slot (`ARCH-PURE`, `ARCH-PURPOSE`).
- Correlation cannot require identity produced only by success: enumerate every
  operation across success/failure and present/missing result fields. Commit
  optimistic UI changes only after correlated success, so a failed external
  operation preserves the state it did not actually change.
- A list frame's applicability comes from its captured parent/action identity,
  not its filtered selection. Zero matches legitimately means no selection;
  refresh must retain the frame and reconcile selection afterward.
- Keep operation identifiers separate from presentation labels and use one
  descriptor mapping for filtering, rendering, and dispatch. An internal
  operation such as `name` may be presented as `rename` without changing the
  shared operation contract (`ARCH-DRY`).
- Hierarchical layout requires semantic geometry: selected parent-row offsets
  for wide children and measured parent-list height for narrow children.
  Equal partitions are bounded rectangles but do not express the hierarchy.
- Reducer support is not user reachability. Every semantic key must be driven
  through the production decoder in every accepted terminal mode and across
  split reads. Likewise, bounded rendering must reserve mandatory semantic
  cues before clipping variable text; a row that fits but hides state is not
  operationally bounded (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).
