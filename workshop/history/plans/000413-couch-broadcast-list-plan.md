# couch --broadcast-list (pair#413) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --broadcast-list [--json]` prints the running couch's broadcast
state, start time, mode and live viewer count. It never prints the link or
token.

**Architecture:** A read-only broker operation, `broadcast-status`, answered by
the running console. The data path is:
1. The CLI sends the request over the broker socket.
2. `messageService` passes it to a console-supplied snapshot function, before
   any slot-identity check.
3. The console reads its own `broadcastState` and asks the session for its
   viewer count.
4. The session asks the hub, through the hub's own loop.

**Tech Stack:** Go; `cmd/internal/broadcast`, `couchtty`, `couchmessage`,
`couchcmd`.

## Design decisions

- **D1: no link, ever** (operator, 2026-10-08). The snapshot type has no field
  that could hold the link or token, so neither format can leak it by
  construction (ARCH-SECURE). A test checks both output formats against a live
  session's token.
- **D2: an operator query, not a slot query.** The CLI finds the broker socket
  through `rt.StoreDir()` (`COUCH_STORE_DIR`, else the default store), so it
  works from any shell. The handler needs no caller identity. The socket lives
  in the operator-only store directory, so it adds no trust boundary.
- **D3: answered from memory.** The viewer count is `len(h.subs)`, read on the
  hub loop through `h.do`. The phase and start time are console memory. There
  is no per-viewer probe on the request path. After the hub ends, `do`
  doesn't run `f` and the count reads 0.
- **D4: mode comes from the tunnel type.** It is captured at `Start`:
  `LocalOnly` gives `local-only`, anything else gives `tunnel`.

**ARCH-ORDER:** the query holds no state between events. It reads phase and
session under `c.mu` and returns. **ARCH-FUNERAL:** creates nothing durable.
**ARCH-PURE:** `BroadcastStatus` and its text formatting are pure;
`formatBroadcastStatus` is table-tested.

## Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `broadcast.Status` | `cmd/internal/broadcast/session.go` | new |
| `Hub.Viewers` | `cmd/internal/broadcast/hub.go` | new |
| `couchmessage.BroadcastStatus` (wire) | `cmd/internal/couchmessage/protocol.go` | new |
| `formatBroadcastStatus` | `cmd/internal/couchcmd/broadcast_list.go` | new |

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.BroadcastStatus` | `cmd/internal/couchtty/console_broadcast.go` | new | console memory |
| `messageService` `broadcast-status` op | `cmd/internal/couchcmd/message_service.go` | modified | broker socket |

### Task 1: hub viewer count and session status (broadcast)

- [ ] Test (`hub_test.go`): subscribe 3, cancel 1, mark 1 resyncing →
      `Viewers() == 2`; after `Close`, `Viewers() == 0`.
- [ ] `func (h *Hub) Viewers() (n int) { h.do(func() { n = len(h.subs) }); return }`.
- [ ] `Session` gains `startedAt time.Time` and `mode string`, set in `Start`.
      `func (s *Session) Status() Status` returns
      `Status{StartedAt, Mode, Viewers: s.hub.Viewers()}`.
      Test via `FakeTunnel`; the Status struct has no link field.

### Task 2: console snapshot (couchtty)

- [ ] `func (c *Console) BroadcastStatus() (couchmessage.BroadcastStatus, bool)`:
      under `c.mu`, map the phase to `starting`, `live` or `stopping`, plus the
      session's Status. It returns false when the phase is off. The viewer
      count is read after releasing `c.mu`, so the console lock never waits on
      the hub loop.
- [ ] Test: off is not ok; starting has state "starting" with no viewers; live
      has its state, mode and viewer count.

### Task 3: wire op and service (couchmessage, couchcmd)

- [ ] `Request.Op == "broadcast-status"`: validation takes no fields.
      `Response.Broadcast *BroadcastStatus`
      (`{State, StartedAt, Mode, Viewers}`, JSON-tagged, no link).
- [ ] `messageService.handle`: route `broadcast-status` before the caller
      check, to `s.broadcastStatus` (a func set at wiring, `run.go` beside
      `SetForget`). With no provider wired it answers `unsupported`.
- [ ] Test: the handler answers without slot identity, and the response holds
      no token from a live fake session.

### Task 4: CLI (couchcmd)

- [ ] Parse `--broadcast-list` and `--broadcast-list --json` in
      `parseMessageCLI`. Any other form is refused.
- [ ] `runBroadcastListCLI`: socket from `rt.StoreDir()`. A connect failure
      prints "couch: no running couch (…)" and exits 1. `unsupported`, or
      unknown-op from an older couch, says to restart couch.
- [ ] Output:
  - `formatBroadcastStatus` gives "no broadcast", or
    `live since 14:02:11 (tunnel) — 2 viewers` (singular "1 viewer").
  - `--json` encodes `{"broadcast": null | {...}}`.
- [ ] Tests:
  - the format table;
  - parse forms;
  - the end-to-end CLI against a fake broker: text and JSON, neither
    containing the token;
  - no couch running: exit 1.
- [ ] `couch --help` lists the command.

### Task 5: close

- [ ] `atlas/broadcast.md` gets a "Listing" section.
- [ ] Full verification (unsandboxed; known failures checked against main).
- [ ] `sdlc close`.
