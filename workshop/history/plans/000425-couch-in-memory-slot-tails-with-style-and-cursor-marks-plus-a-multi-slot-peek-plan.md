# #425 — couch in-memory slot tails with style and cursor marks, plus a multi-slot peek

**Goal:** `couch --peek` reads a slot's tail from its live wrapper's memory, keeps
the faint/reverse cues and the cursor as light markup, and takes several slots in
one call. Pair renders; it never classifies.

## Architecture

The wrapper already holds the agent pane in memory: `terminalModel` wraps a vt
emulator with the screen plus a 10k-line scrollback (claude and codex, the agents
with a terminal model — the same set that runs a peer endpoint). So the "tail
store" is not a new buffer; it is a render of that emulator on request.

```
couch --peek pair:1:2,ariadne:0           (CLI process, DirectStoreExecutor)
  └─ per slot: resolve thread (existing) ─→ SlotTail(address, n)
        └─ broker socket, op "tail" {TailScope, TailTag, Lines}   (identity-free, read-only)
             └─ messageService: connected binding with that scope+tag
                  └─ RemoteEndpoint.Tail ─→ wrapper endpoint op "tail"
                       └─ peerDelivery.tailProbe → terminalModel.Tail(n)  (render under m.mu)
  fallback (no couch running, no endpoint, agent without a model, error):
        existing recording render (plain text) + the live reason in `unavailable`
```

Why through the broker: the wrapper's endpoint socket name hashes its full
binding (PID, nonce, version), which only the broker knows. The broker socket is
in the operator-only store dir; `broadcast-status` (#413) already sets the
precedent for an identity-free, read-only broker op. Peek must work from any
operator shell, so it cannot require a caller identity.

## Markup (generic terminal bookkeeping, no agent knowledge)

- `‹dim›…‹/dim›` around faint (SGR 2) runs; `‹rev›…‹/rev›` around reverse video.
  Spans close at line end and reopen on the next line.
- `‹cursor›` inserted before the cursor cell when the cursor is visible.
- One summary: `cursor: ROW,COL SHAPE` (1-based, ROW counts the returned lines;
  SHAPE ∈ default/block/underline/bar, `steady` when non-blinking), or
  `cursor: hidden at ROW,COL`, or `cursor: outside the tail`.
- Trailing blank cells trimmed (padded out to the cursor column on its row);
  trailing all-blank rows below the cursor dropped. Wide-char continuation cells
  skipped. Alt screen: screen rows only (its scrollback belongs to the main screen).

## Bounds (ARCH lifecycle)

Creates nothing durable: the tail is rendered per request from the emulator the
wrapper already keeps and dies with the response. Request bound: `Lines` ≤ 200;
response bound: 128 KiB of rendered text — oldest lines dropped first, counted in
`Truncated`. Both under the 590 KB transport frame.

## Multi-slot syntax

`repo:N[:M…]` expands to `repo:N, repo:M…`; `,` separates groups. One ref keeps
today's single `PeekResult` (JSON compatibility); two or more return
`PeekSnapshot{Slots []PeekResult}`, rendered as one section per slot. A segment
that does not resolve becomes a section whose `unavailable` names why — the
snapshot never fails as a whole. Slots are read concurrently.

## Files

- `cmd/internal/wrapcmd/terminal_tail.go` (new): `terminalModel.Tail(n) couchmessage.Tail` + markup renderer.
- `cmd/internal/wrapcmd/peer_runtime.go`: endpoint op `tail` → `d.tailProbe`; wired to `p.terminal`.
- `cmd/internal/couchmessage/endpoint.go`: `tail` op validation, `Lines`; `Tail` type; `TailReader`; `RemoteEndpoint.Tail`.
- `cmd/internal/couchmessage/protocol.go`: `TailScope/TailTag/Lines`, `tail` validation (identity-free), `Response.Tail`.
- `cmd/internal/couchcmd/message_service.go`: `handleTail` — connected binding by scope+tag → endpoint `TailReader`.
- `cmd/internal/couchcore/peek.go`: `SlotTail` seam (live first, recording fallback), `Source`, `Cursor`, `Truncated`; `ExpandPeekReferences`; `PeekSnapshot`.
- `cmd/internal/couchcore/operationdispatch.go`: peek dispatches single vs multi.
- `cmd/internal/couchcmd/run.go`: `SlotTail` wiring (broker call), `renderPeek` cursor/source lines, `renderPeekSnapshot`.
- `cmd/internal/couchcmd/cli.go`: help text unchanged shape (`--peek ref[,ref…]`).
- Docs: `atlas/couch.md`, `cmd/internal/couchcmd/skills/couch/SKILL.md` (judging state is the caller's).

## Tests

- `terminal_tail_test.go`: captured Claude ghost suggestion (`testdata/tty/claude/…/composer.raw`
  or `peer/claude/2.1.286/startup-fix-lint.raw`) renders `‹dim›`; the typed draft
  (`paste-short.raw`) renders the text without `‹dim›`; cursor marker/summary; reverse;
  scrollback + screen tail ordering; byte budget truncation; alt screen.
- endpoint: `tail` request validation table (lines bounds, extra fields refused).
- protocol: `tail` identity-free validation; other ops refuse Tail fields.
- message_service: tail of a connected binding; not-connected → `unavailable`; endpoint without TailReader → `unsupported`.
- couchcore: `ExpandPeekReferences` table; PeekThread uses live tail and skips the recording; falls back with reason; multi snapshot with one unresolvable segment.
- CLI router test: `couch --peek pair:1:2,ariadne:0 --lines 10` parses through the top-level dispatcher (lesson #424).

## Steps

- [ ] terminal tail renderer + tests (TDD)
- [ ] endpoint `tail` op + wrapper wiring + tests
- [ ] broker `tail` op + message_service handler + tests
- [ ] couchcore peek live-first, multi-slot expansion/snapshot + tests
- [ ] CLI wiring, rendering, router test
- [ ] atlas + skill docs
- [ ] live check: build, `couch --peek` on 3+ live slots, timing
