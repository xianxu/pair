# Pair-owned VT fork

Base: `github.com/charmbracelet/x/vt@v0.0.0-20260510215043-e3181689be6b`
(commit `e3181689be6b`). The original MIT license and tests are retained. The
root module replaces only x/vt with this directory; other Charm modules remain
normal pinned dependencies. Pair owns these patches and must rerun both the
module suite and the root terminal qualifier before updating the base.

Upstream comparison on 2026-09-15: latest VT change
`a5dee49b28632257cd9a475e8ca36e98a62ff155` fixes whole-write combining and OSC8
field ordering, but still finalizes graphemes at transport write boundaries.
The field-order correction here agrees with upstream; incremental handling is
owned here instead of importing the incomplete upstream algorithm.

## Patch families

- `utf8.go`: render one provisional grapheme and extend it across arbitrary
  writes; seal at terminal control boundaries. Recompute bounded cluster width,
  repair occupied cells, and wrap wide clusters before the right edge. Default
  cluster limit is 256 UTF-8 bytes: additional extensions are discarded until a
  grapheme boundary/control, without growing retained state.
- `handlers.go`, `screen.go`, `csi_mode.go`, `osc.go`: CSI cursor restoration,
  blink callback polarity, standard operating-status reply, retained mode47,
  mode1047 exit clearing and mode1049 save/clear/restore, correct OSC8 field
  order and URI semicolons.
- `pair_keyboard.go`, `key.go`: separate main/alternate Kitty flag stacks,
  push/pop/set/query/reset, bounded stack eviction, modifiers and
  press/repeat/release, alternate key identities, associated text, functional
  key wire numbers and modified legacy navigation keys. Legacy text and keypad
  handling stays available when negotiation is off. Unmapped internal key IDs
  are not emitted as Unicode.
- `csi_cursor.go`: saturating tab motion terminates immediately; REP counts
  above the cell limit are rejected so one short command cannot monopolize
  a serialized endpoint transaction. Margin requests outside screen geometry
  and unknown cursor shapes are rejected before they can reach UV buffer loops
  or immutable frame metadata.
- `mouse.go`, `mode.go`: tracking modes replace each other; motion/release
  filtering depends on the active mode. Store only recognized mode IDs.
- `pair_limits.go`, `scrollback.go`, `screen.go`: validated geometry,
  checked resize, exact-size screen replacement on resize (UV otherwise keeps
  removed cell payloads), bounded history by lines/cells/bytes, and release of
  cleared/evicted references. `Usage` reports conservative payload/capacity
  accounting with a 16 KiB fixed-overhead allowance, 25% plus 16 bytes per
  nonempty string and 16 bytes per boxed color reference. This is a conservative
  engineering estimate, not a Go heap census or mathematical upper bound across
  runtime versions. History byte limits still use logical payload sizes.
- `emulator.go`, `pair_limits.go`: synchronous replaceable reply destination,
  first-error retention via `TakeReplyError`, and short-write detection. Legacy
  `Read`/pipe behavior remains the default. Endpoint code must provide a bounded
  non-reentrant collector and check the error after every input transaction.
- `osc.go`, `dcs.go`, `callbacks.go`: synchronous typed clipboard write/query
  and notification callbacks. A 64 KiB parser buffer plus one sentinel byte
  rejects oversized string commands before dispatch instead of executing a
  truncated prefix. Titles/cwd and notification title/body are each limited
  to 4 KiB; hyperlink parameters plus URL to 2 KiB. Clipboard data is decoded
  only after the encoded string bound passes. Effect policy and origin routing
  remain outside this package.

Default geometry cap is 262144 cells per screen; history is at most 1000 lines,
65536 cells and 4 MiB per buffer; keyboard stack depth is 16. Limits do not cap
caller-owned callback data, reply collectors, immutable frames or endpoint
queues. Callbacks and emulator methods require external serialized ownership.

### Erase follow-up

ED0/ED1 erase through the cursor boundary, ED2 preserves the current erase
background, and ED3 erases saved lines without changing visible cells (xterm
CSI J semantics). EL/ECH use the same background-only blank cell. Fill areas
are clipped before UV iteration to bound oversized ECH counts. Literal tests
cover all display erase modes, first/last boundaries, scrolling margins, cursor
position, attributes/links and foreground/background behavior. The inherited
`ED Simple Erase Above` oracle formerly encoded the bug by erasing F to the
right of CUP2;2; its expected second row is corrected to `  F     `. The
original 68 root qualification predicates remain unchanged. Four inherited
origin/outside-scroll-region fixtures requested a bottom margin one row beyond
their declared height; their geometry now includes that row, with a matching
blank expected row, so they exercise valid margins. Oversized margins are
independently tested as ignored requests.

## Verification

```sh
(cd third_party/vt && go test ./... && go test -race ./...)
go run ./cmd/probes/terminalqualify
```

The qualifier retains its original predicates: 68 executable cases must pass;
14 production integration obligations remain separately tracked. Its exit1
while those obligations are not covered is intentional. Additional `pair_*test`
cases cover every cluster byte split, bytewise input, right/bottom wrap,
oversized cluster recovery, exact string limits and recovery, history limits,
resize retention, negotiation state, effect limits and related regressions.

- Parameter commands retain overflow evidence across writes and reject CSI/DCS
  atomically, including subparameter counts and numeric values that would collide
  with parser flags/sentinels. A fixed extra parser slot preserves exactly 32
  parameters despite upstream final-slot accounting; overflow parameter bytes
  are not accumulated. Cancellation and subsequent commands recover normally.
- `MouseEpoch` records real tracking/SGR mode transitions, including reset and
  transitions that return to the same state within one input chunk.

M2 BR8 adds `Emulator.Cursor()` as a copied value from the active screen. Endpoint
and qualification snapshots use this state directly instead of reconstructing
cursor shape/visibility through incomplete callbacks. Literal tests cover reset,
saved-cursor restoration and alternate-buffer transitions at every byte split.

- Zero-width input extends a still-open printable grapheme when segmentation
  permits; controls seal that cluster. Orphan/non-attaching zero-width graphemes
  (including combining marks, ZWJ and variation selectors after controls) are
  consumed without cells, cursor movement or pending-wrap changes. This follows
  native Zellij's orphan behavior and preserves nonempty cells' positive-width
  invariant. Pinned xterm/headless 5.5's default representation instead stores
  nonempty width-zero cells for these orphan cases; that representation is not
  adopted. Contiguous supported combining/ZWJ/VS clusters remain intact.

M3 typed history publication:
- Primary scrollback has monotonic append-attempt IDs and a clear epoch. Loss
  gaps remain observable; snapshots own cells/colors and never include alternate
  history. Full-width top scrolling admits rows; interior regions do not.
- Row metadata preserves incoming soft wrap and used text extent. Unprinted
  blank cells have empty content with Width1; printed spaces remain text. Legacy
  String/Render and upstream visual test adapters normalize only that display
  distinction. Erase, partial-wide overwrite, IL/DL and resize retain coherent
  metadata and cells.
- Primary width changes reflow visible logical lines independently of retained
  history, preserving early-wide gaps, cursor position and pending wrap. Narrowing
  admits displaced rows through existing limits. Alternate resize still clips.
- At width1, an indivisible wide glyph is retained privately and painted blank;
  widening restores it, overwrite discards it, and scroll admits its full source
  to typed history. This is a deliberate coherent clipped-view policy, not native
  equivalence for one-column copy. Retained memory estimates include this source.
- ED2 preserves the visible prefix through the last printed row, including blank
  hard separators and explicit-space soft rows, matching measured native Zellij.
  Pinned xterm's direct ED2 discards viewport rows rather than admitting them;
  production oracle tests explicitly distinguish these two baseline behaviors.
