# Boundary Review — pair#413 (whole-issue close)

| field | value |
|-------|-------|
| issue | 413 — couch --broadcast-list: print the current broadcast session and its viewer count |
| repo | pair |
| issue file | workshop/issues/000413-couch-broadcast-list.md |
| boundary | whole-issue close |
| milestone | — |
| window | f904c1172c6bb4e2093bfef4e494e0dc0c6c26bb..c8bb70e7f0caa773d0b7b0225e6230c9bbbd248f |
| command | sdlc close --issue 413 |
| reviewer | claude |
| timestamp | 2026-10-08T10:15:00-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature works as the Spec describes. `couch --broadcast-list [--json]` routes a new identity-free `broadcast-status` broker op to `Console.BroadcastStatus`. The viewer count comes from `Hub.Viewers`, which reads on the hub's loop, so the answer comes from memory. The no-link rule holds by construction: neither `broadcast.Status` nor `couchmessage.BroadcastStatus` has a field that could carry the link or token. I checked the main risk, a socket-path mismatch for an "any shell" CLI. The message CLI goes through `prepareRuntime`, which sets `selection`, so `rt.StoreDir()` is the same canonical `Roots.StoreDir` the server hashes into its socket path. That makes the path correct. The new tests pass (`TestFormatBroadcastStatus`, `TestParseBroadcastListForms`, `TestBroadcastListNeverPrintsTheLink`, `TestBroadcastListWithNoCouchRunning`, `TestBroadcastStatusNeedsNoSlotIdentity`, `TestHubViewersCountsLiveSubscribers` ×5, `TestSessionStatusReportsModeStartAndViewers` ×5, `TestConsoleBroadcastStatusFollowsThePhase`). Other packages failed in my run: the `couchmessage` and `couchtty` transport and PTY tests hit sandbox limits on `/tmp` and pty-child processes. `artifactpath` reported 33 unrelated sources (`nvim/review/*`, `wrapcmd/peer_*`) missing from its inventory, but did not flag `broadcast_list.go`. One gap holds up SHIP: the README's couch command list doesn't include the new command.

## 1. Strengths
- **The no-link rule holds by construction.** Neither status type has a field for the link (`cmd/internal/couchmessage/protocol.go:38`, `cmd/internal/broadcast/session.go:121`). `TestBroadcastListNeverPrintsTheLink` checks both output formats against a real session's token.
- **The console lock never waits on the hub.** `Console.BroadcastStatus` (`console_broadcast.go:81`) copies the phase and session under `c.mu` and reads the viewer count after releasing it.
- **`Hub.Viewers` is safe after the hub ends.** It reuses `h.do`, which returns without running `f` once the hub is done, so an ended hub reads 0 and cannot hang.
- **Validation is strict.** `ValidateRequest` refuses any populated field on `broadcast-status`, and the parser refuses every form except the two documented ones.
- **The provider is installed safely.** It goes in through an atomic pointer after the service starts serving, and a nil provider answers `unsupported`.

## 2. Critical
None.

## 3. Important
- **README update is missing for `couch --broadcast-list [--json]`.** README.md:378-395 lists every couch CLI command, including `--actors` and `--peek`, and README.md:808 is the broadcast section. Neither was updated. The fix is to add a line to the command list and one sentence in the broadcast section saying the command lists the broadcast and never the link.

## 4. Minor
- **Spec wording disagrees with the code.** The Spec lists states as "starting, live, ending"; the code and atlas use `stopping`. Fix the Spec.
- **The "predates" branch never fires for a real older couch.** In `broadcast_list.go:31`, a pre-#413 couch rejects the request with `invalid-request: operation requires the calling conversation identity`, not "unknown message operation". The user sees the raw error instead of the restart hint, and no test feeds that response.
- **`Hub.Viewers` ignores the handler's context.** The transport requires handlers to honour their deadline. The hub loop never blocks, so the wait is bounded in practice.
- **Mode detection is fragile and untested.** The mode comes from a type assertion `cfg.Tunnel.(LocalOnly)`. A `*LocalOnly` would report `tunnel`, and no test covers the local-only mode.
- **Import placement.** The `couchmessage` import in `console_broadcast.go:5` sits in the standard-library group.
- **The resync case is assumed, not asserted.** `TestHubViewersCountsLiveSubscribers` never checks that `slow` actually entered resync (`_ = slow`); it only checks the count.
- **Plan detail is stale.** Plan Task 3 says the provider is wired in `run.go` beside `SetForget`; it is actually wired in `startMessageService`.

## 5. Test coverage notes
- Each layer has tests: hub, session, console, service, CLI parse and format, and the end-to-end CLI against a fake call.
- The leak test feeds the CLI a snapshot built from `session.Status()`, not from the real `messageService`/`Console` path. That is acceptable, because the type has no field that could carry the link.
- Nothing checks `local-only` mode or an older couch's response.

## 6. Architecture
- **ARCH-DRY: pass.** The parse and route follow the existing message-CLI pattern. The error mapping in `broadcast_list.go:30-37` copies the `messages.go:186` switch; it is small, but a shared `classifyOpResponse` helper would make sense if a third caller appears.
- **ARCH-PURE: pass.** `formatBroadcastStatus` is pure and table-tested, and the IO stays in the thin CLI and service.
- **ARCH-PURPOSE: pass.** Every Done-when item is delivered. The README gap is a docs-gate issue, not under-delivery.
- **ARCH-MOCK: pass.** The tests use the existing `FakeTunnel` and the injected `messageCall` seam.
- **ARCH-CONSTRAINTS: pass.** The answer comes from memory with an O(1) read on the hub loop, inside the client's `AdmissionTimeout`.
- **ARCH-SECURE: pass.** The socket is `0600` in a `0700` uid-owned directory (`transport.go:103,112-120`). The op is read-only and no type can carry the credential.
- **ARCH-ORDER: pass.** It holds no state between events; it takes one snapshot under `c.mu` and then makes one hub query. A phase change between the two only makes the reply momentarily stale, which is harmless.
- **ARCH-FUNERAL: pass.** It creates nothing durable; `startedAt` and `mode` die with the session.

## 7. Plan revision recommendations
- Add a `## Revisions` entry: the provider is wired in `startMessageService` (`message_service.go:311`), not in `run.go` beside `SetForget`.
- Add a `## Revisions` entry or Spec edit: the states are `starting`, `live`, `stopping`, not `ending`.

```findings
findings:
  - id: new
    severity: Important
    family: readme-tracks-cli-surface
    title: |
      README couch command list and broadcast section omit couch --broadcast-list [--json]
    detail: |
      README.md:378-395 enumerates every couch CLI flag (incl. --actors, --peek) and README.md:808 documents broadcasting; neither mentions the new command or that it never prints the link.
  - id: new
    severity: Minor
    family: spec-matches-implementation
    title: |
      Spec lists broadcast states as starting/live/ending; code and atlas use stopping
  - id: new
    severity: Minor
    family: version-skew-detection
    title: |
      The predates-couch hint in broadcast_list.go never fires for a real pre-#413 couch
    detail: |
      An older couch answers invalid-request "operation requires the calling conversation identity", not "unknown message operation", so the user sees the raw error instead of the restart hint; untested.
  - id: new
    severity: Minor
    family: handler-honors-context
    title: |
      handleBroadcastStatus calls Hub.Viewers without the request context
    detail: |
      The transport contract says handlers honor their deadline; h.do has no ctx case. Bounded in practice because the hub loop never blocks.
  - id: new
    severity: Minor
    family: test-asserts-precondition
    title: |
      Mode derives from a LocalOnly type assertion with no local-only test; hub test never asserts the slow viewer resynced
```
