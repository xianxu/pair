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

---

## Re-review — 2026-10-08T10:16:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 413 — couch --broadcast-list: print the current broadcast session and its viewer count |
| repo | pair |
| issue file | workshop/issues/000413-couch-broadcast-list.md |
| boundary | whole-issue close |
| milestone | — |
| window | f904c1172c6bb4e2093bfef4e494e0dc0c6c26bb..38253a18c998681462aefebafe7e486bded61006 |
| command | sdlc close --issue 413 |
| reviewer | claude |
| timestamp | 2026-10-08T10:16:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All five prior findings are dealt with, and this round adds nothing new. The README now covers `couch --broadcast-list [--json]` in both places it was missing: the command list (README.md:388) and the broadcasting section (README.md:822). The Spec's state names now match the code (`starting`, `live`, `stopping`). The CLI now gives the restart hint when it reaches an older couch. I checked that the string it matches is the real refusal at base: `ValidateRequest` returns "operation requires the calling conversation identity" (protocol.go:51 at f904c117) for any op that has no identity fields, and an op added in #413 has none. A table test covers both kinds of refusal. The hub test now asserts that its slow viewer really is resyncing, and there is a new test for local-only mode. The targeted tests pass in `couchcmd` and `broadcast`.

1. **Strengths**
   - `broadcast_list.go:31-37`: the restart hint covers both older shapes, the identity refusal and the unknown-op error. `TestBroadcastListAgainstAnOlderCouchSaysRestart` feeds it the exact text from base.
   - `hub_test.go:508-512`: the test checks its own setup first, so it can't pass without actually covering a resyncing viewer.
   - No status type has a field that could carry the link (`protocol.go:38-44`), so the rule that the link is never printed holds by construction.
   - `ValidateRequest` refuses any populated field on `broadcast-status`, so the request has exactly one valid shape.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** two leftovers from last round's prose, neither tracked:
   - The plan still says the provider is wired in `run.go`; it is actually wired in `startMessageService`.
   - The `couchmessage` import in `console_broadcast.go` is in the standard-library group.

5. **Test coverage:** every layer has a test: hub, session (tunnel and local-only), console, service, and the CLI's parsing, formatting, no-link rule, no running couch, and older couch.

6. **Architecture:**

   | Principle | Result | Why |
   |---|---|---|
   | ARCH-DRY | pass | The error mapping still mirrors `messages.go:183`. Pull out a shared helper if a third caller appears. |
   | ARCH-PURE | pass | `formatBroadcastStatus` is pure. |
   | ARCH-PURPOSE | pass | Every Done-when item is delivered, including the README. |
   | ARCH-MOCK | pass | Tests use the `messageCall` seam and `FakeTunnel`. |
   | ARCH-CONSTRAINTS | pass | The count is an O(1) read from memory on the hub loop. |
   | ARCH-SECURE | pass | The socket is owner-only and no type can carry the token. |
   | ARCH-ORDER | pass | It holds no state between events; it takes one snapshot and then makes one hub read. |
   | ARCH-FUNERAL | pass | It creates nothing durable. |

7. **Plan revisions:** optionally add a `## Revisions` line saying the provider is wired in `startMessageService`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:388 command-list line and README.md:822 broadcast-section paragraph, including that the link is never printed; verified in the 38253a18 diff.
  - id: BR-2
    disposition: addressed
    note: |
      Issue Spec line 42 now reads starting, live, stopping, matching the plan and code.
  - id: BR-3
    disposition: addressed
    note: |
      broadcast_list.go:31-34 matches the base refusal text (protocol.go:51 at f904c117); TestBroadcastListAgainstAnOlderCouchSaysRestart covers both refusal shapes, and the identity case fails without the fix.
  - id: BR-4
    disposition: withdrawn
    note: |
      h.do returns once the hub is done, and the loop only runs non-blocking closures, so the wait is bounded; the rationale in the Log is accepted.
  - id: BR-5
    disposition: addressed
    note: |
      TestSessionStatusReportsLocalOnly pins the local-only mode; hub_test.go:508-512 asserts resyncing >= 1 before checking the count.
```
