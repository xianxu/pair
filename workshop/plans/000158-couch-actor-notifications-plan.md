# Couch Actor Notifications Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route Pair-normalized terminal notifications through Couch without duplication, retain a bounded ephemeral inbox for inactive actors, and make the newest unread actor one Enter away from the switcher.

**Architecture:** A shared pure Go package owns the canonical OSC 777 envelope and message sanitization. Pair normalizes native and hook-driven signals at the outer-TTY boundary; Couch's existing per-child terminal parser observes the same bytes while preserving raw forwarding, then one Console-owned pure attention ledger projects state into the status row and switcher. A successful switch acknowledges only the exact retained identities captured when it was dispatched.

**Tech Stack:** Go, POSIX PTYs, ANSI/OSC framing, the existing Couch reducer/rendering architecture, Bash compatibility shim, Go unit/integration/race/benchmark tests.

---

## Chunk 1: Canonical transport, attention state, and Couch presentation

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `notifyosc.Notification` and codec | `cmd/internal/notifyosc/notification.go` | new |
| `wrapcmd.NotificationRewriter` | `cmd/internal/wrapcmd/notification_rewriter.go` | new |
| `ptychild.Screen` notification observation | `cmd/internal/ptychild/screen.go` | modified |
| `ptychild.ReplayWindow` | `cmd/internal/ptychild/replay.go` | modified |
| `couchtty.AttentionLedger` | `cmd/internal/couchtty/attention.go` | new |
| `couchtty.MenuState` attention projection | `cmd/internal/couchtty/menu.go` | modified |
| `couchtty.StatusActor` attention presentation | `cmd/internal/couchtty/reserve.go` | modified |

- **`notifyosc.Notification` and codec** — owns the exact `ESC ] 777 ; notify ; pair ; <message> BEL` representation, UTF-8 repair, C0/DEL/C1 removal, and the post-sanitization 4 KiB limit.
  - **Relationships:** One notification maps to one encoded envelope; every Pair emitter and Couch decoder depends N:1 on this codec.
  - **DRY rationale:** Pair and Couch must not carry separate opinions about sanitization, bounds, or canonical framing (`ARCH-DRY`, `ARCH-PURE`).
  - **Future extensions:** A versioned envelope or structured fields widen this package while leaving agent interpreters and Couch presentation unchanged.

- **`wrapcmd.NotificationRewriter`** — incrementally separates recognized native OSC from ordinary agent output, emits one normalized message event, and returns all unknown bytes unchanged.
  - **Relationships:** One rewriter belongs 1:1 to a proxy run and emits 0:N normalized notifications while producing one ordered ordinary-output stream.
  - **DRY rationale:** The current rolling-regex observer cannot suppress a recognized OSC or prove chunk-boundary equivalence; one stateful framing owner replaces detection plus forwarding decisions.
  - **Future extensions:** Additional native OSC families extend the recognition predicate, not the stream state machine.

- **`ptychild.Screen` notification observation** — recognizes canonical Pair envelopes inside the existing bounded terminal framer, withholds only an exact in-progress canonical candidate, and emits ordered forward parts (ordinary bytes or one complete notification) alongside the legacy bare-BEL latch.
  - **Relationships:** One Screen belongs 1:1 to one child and produces 0:N ordered parts for each atomically delivered child batch.
  - **DRY rationale:** Reusing the screen parser's pending/skip state prevents a second terminal parser with divergent chunk semantics.
  - **Future extensions:** Other Pair-owned terminal metadata can become typed observations at the same framing boundary.

- **`ptychild.ReplayWindow`** — identifies the absolute ring offset whose bytes have been processed and are safe to replay, plus absolute completed-notification spans to exclude even when ring retention bisects one.
  - **Relationships:** Each child owns one monotonically advancing ring position and bounded span index; each Console pane records at most one processed replay-safe cutoff.
  - **DRY rationale:** Replay and live delivery use the same observer decision instead of independently guessing whether a partial OSC is safe.
  - **Future extensions:** Snapshot consumers can request other explicit historical cutoffs without changing the ring's retention policy.

- **`couchtty.AttentionLedger`** — the sole pure authority for per-actor retained messages, monotonic unread order, deduplication, eviction, newest-source selection, overflow rebasing, and sequence-qualified clearing.
  - **Relationships:** One Console owns exactly one ledger; it contains 0:3 messages per actor and projects N:1 into status and switcher models.
  - **DRY rationale:** It replaces `pane.bell` plus `MenuState.Bells`, preventing compound attention state from diverging across UI consumers.
  - **Future extensions:** A later attach/detach protocol could add attention classes without introducing persistence.

- **`couchtty.MenuState` attention projection** — receives immutable attention rows when opened/rendered; its operation attempt identifies a ledger-owned dispatch-time acknowledgement capture but does not own unread state.
  - **Relationships:** Each visible actor has 0:3 display-only notification children; a switch attempt identifies at most one ledger-owned capture.
  - **DRY rationale:** The menu remains a pure interaction reducer while the Console ledger remains the single state authority.
  - **Future extensions:** Message actions can be added later only by deliberately promoting children into selectable identities.

- **`couchtty.StatusActor` attention presentation** — colors an unread actor's existing label without adding width-bearing symbols or tokens.
  - **Relationships:** One status actor is a read-only projection of one attached pane plus its ledger state.
  - **DRY rationale:** Existing roster spacing and clipping stay centralized in `RenderStatusRow`.
  - **Future extensions:** Theme-selected attention colors can replace the initial ANSI style without changing ledger semantics.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `notifycmd.Run` | `cmd/internal/notifycmd/run.go` | new | environment, outer-TTY sidecar, nonblocking terminal write |
| `pair-notify` compatibility shim | `bin/pair-notify` | modified | hook-facing shell command |
| `wrapcmd.proxy` output pump | `cmd/internal/wrapcmd/wrap.go` | modified | agent PTY stdout and Pair outer TTY |
| `ptychild.Child` observation handoff | `cmd/internal/ptychild/child.go` | modified | actor PTY read pump |
| `couchtty.Console` attention routing | `cmd/internal/couchtty/console.go` | modified | actor chunks, host terminal, switch completion |
| Runtime bundle and architecture map | `cmd/internal/runtimebundle/assets/runtime/files/bin/pair-notify`, `cmd/internal/runtimebundle/assets/runtime/manifest.json`, `atlas/architecture.md`, `atlas/index.md` | modified | packaged shim and operator-facing system map |

- **`notifycmd.Run`** — validates hook arguments and Pair environment, reads the exact recorded outer-TTY path, and writes `notifyosc.Encode(message)` through an injected runtime using nonblocking semantics.
  - **Injected into:** Tests use a stateful runtime that records files, writability, and bytes written; production uses filesystem/open syscalls.
  - **Future extensions:** Other hook sources can invoke the same command without learning OSC.

- **`pair-notify` compatibility shim** — preserves the installed hook command name and tolerant failure behavior, then delegates to `pair notify`; deprecated `--osc 9|777` is accepted for compatibility but both values produce the canonical 777 envelope.
  - **Injected into:** Existing Claude hooks and manual invocations.
  - **Future extensions:** Remove the option after documented compatibility telemetry; the shim itself can eventually become a busybox alias.

- **`wrapcmd.proxy` output pump** — feeds raw bytes to existing overlay/terminal observers, sends rewriter passthrough bytes to inner Zellij, and writes exactly one canonical envelope per accepted native/marker/idle event to the recorded outer TTY.
  - **Injected into:** `NotificationRewriter`; tests use byte buffers and a fake outer-TTY sink.
  - **Future extensions:** Agent modes remain configuration on the interpreter, not Couch behavior.

- **`ptychild.Child` observation handoff** — keeps Screen mutation and ordered-part draining under the existing child mutex, then offers one owned `OutputBatch` for that exact PTY read and waits for Console acknowledgement before that actor reads again.
  - **Injected into:** Console consumes the typed batch; fake and real child share the same Screen state machine.
  - **Future extensions:** Additional terminal observations become new typed part kinds without adding pull-based latches.

- **`couchtty.Console` attention routing** — consumes ordered output parts stamped with the focus epoch at child delivery, forwards focused output once, forwards only canonical notification envelopes from inactive output, updates the ledger under `Console.mu`, projects it to both UIs, and acknowledges on successful `forceSwitch` completion.
  - **Injected into:** Existing stateful fake child/host and real PTY conformance harness.
  - **Future extensions:** An outer workspace can observe the same forwarded standard OSC without Couch-specific IPC.

- **Runtime bundle and architecture map** — regenerates the tracked runtime manifest/mirror and documents the singular notification byte path and ephemeral Couch ownership.
  - **Injected into:** Installed Pair distributions and future maintainers.
  - **Future extensions:** None; these mirror the implemented surface.

### Task 1: Build the canonical notification codec and hook command

**Files:**
- Create: `cmd/internal/notifyosc/notification.go`
- Test: `cmd/internal/notifyosc/notification_test.go`
- Create: `cmd/internal/notifycmd/run.go`
- Test: `cmd/internal/notifycmd/run_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `bin/pair-notify`
- Modify: `cmd/internal/entrypoint/alias.go`
- Modify: `cmd/internal/entrypoint/alias_test.go`

- [ ] **Step 1: Write codec tests that pin exact bytes and all sanitization boundaries**

  Table-test BEL and ST input recognition, invalid UTF-8 replacement, every C0/DEL/C1 code point, embedded `ESC ]` text after control stripping, an exact 4096-byte result, a multibyte rune crossing the limit, wrong OSC family/title, malformed termination, and encode/decode round trips. Assert exact canonical bytes, not semantic equivalence.

- [ ] **Step 2: Run the codec test and verify the package is absent**

  Run: `go test -p 20 ./cmd/internal/notifyosc -count=1`

  Expected: FAIL because `Notification`, `Sanitize`, `Encode`, and `DecodeOSC` do not exist.

- [ ] **Step 3: Implement the minimal pure codec**

  Use these contracts:

  ```go
  const MaxMessageBytes = 4 << 10

  type Notification struct { Message string }

  func Sanitize(raw []byte) string
  func Encode(message string) []byte
  func DecodeOSC(sequence []byte) (Notification, bool)
  ```

  `Sanitize` must decode invalid input with replacement, remove runes in `U+0000..001F`, `U+007F..009F`, then append whole UTF-8 runes until the 4096-byte bound. `DecodeOSC` accepts only complete canonical Pair envelopes terminated by BEL or ST, defensively sanitizes the body, and rejects all other OSC.

- [ ] **Step 4: Run codec tests and verify they pass**

  Run: `go test -p 20 ./cmd/internal/notifyosc -count=1`

  Expected: PASS.

- [ ] **Step 5: Write failing stateful-runtime tests for `notifycmd.Run` and dispatcher routing**

  Cover canonical bytes written once, missing message exit 2, both legacy `--osc` spellings accepted but normalized to 777, invalid `--osc` exit 2, and tolerant exit 0 with a warning for missing tag/path sidecar/stale TTY/write failure. The fake must model sidecar contents and writes, not assert function-call choreography.

- [ ] **Step 6: Run the command tests and verify the route is missing**

  Run: `go test -p 20 ./cmd/internal/notifycmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`

  Expected: FAIL because `notifycmd.Run`, the `notify` family, and shim delegation are not implemented.

- [ ] **Step 7: Implement `pair notify` and reduce the shell helper to delegation**

  Add `notify` as a buffered dispatcher family. Keep filesystem/open behavior behind a `notifycmd.Runtime` with a production implementation; use `unix.Open(..., O_WRONLY|O_NONBLOCK, 0)` and the shared encoder. Keep `pair-notify` in `entrypoint.nonBusybox`: it remains a hook-facing shell shim. In that shim, resolve `BASH_SOURCE[0]` through symlinks exactly as `bin/pair-dev` does, derive its real sibling directory, and `exec "$here/pair" notify "$@"`. This works in source, installed, and extracted-runtime layouts without depending on the caller's `PATH` or requiring `PAIR_HOME`.

- [ ] **Step 8: Run focused command tests and shell syntax checks**

  Run: `go test -p 20 ./cmd/internal/notifyosc ./cmd/internal/notifycmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1 && bash -n bin/pair-notify`

  Expected: PASS and no shell syntax output.

- [ ] **Step 9: Commit the canonical emission boundary**

  Run:

  ```bash
  git add cmd/internal/notifyosc cmd/internal/notifycmd cmd/internal/dispatcher bin/pair-notify cmd/internal/entrypoint
  git commit -m "#158: canonicalize Pair notification emission"
  ```

### Task 2: Normalize native agent notifications without corrupting output

**Files:**
- Create: `cmd/internal/wrapcmd/notification_rewriter.go`
- Test: `cmd/internal/wrapcmd/notification_rewriter_test.go`
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/osc_test.go`
- Modify: `cmd/internal/wrapcmd/stdout_batch_test.go`
- Modify: `cmd/internal/wrapcmd/picker_overlay_test.go`

- [ ] **Step 1: Write failing arbitrary-split tests for the native notification rewriter**

  For every byte split of representative OSC 9 and OSC 777 sequences, assert: recognized actionable native OSC is absent from passthrough and yields one sanitized notification; unknown/non-actionable/malformed OSC is byte-identical passthrough; adjacent text preserves order; BEL and ST terminators agree; incomplete and overlong frames remain bounded and recover only at their real terminator. Include a non-actionable OSC followed by an actionable one and vice versa. Pin extraction: OSC 9 uses its entire body after `9;`; OSC 777 applies `strings.SplitN(body, ";", 3)`, requires fields `[0] == "notify"`, treats `[1]` as title and the entire `[2]` remainder (including embedded semicolons) as body, prefers non-empty body, falls back to non-empty title, then to `agent attention` when both sanitize empty. Fewer than three fields and OSC 9 `4;...` are non-actionable passthrough.

- [ ] **Step 2: Run the rewriter tests and verify they fail**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestNotificationRewriter' -count=1`

  Expected: FAIL because `NotificationRewriter` does not exist.

- [ ] **Step 3: Implement the minimal incremental rewriter**

  Give it one bounded pending buffer and explicit OSC skip state. Replace the boolean-only classifier with one `nativeNotification(ps, body) (message string, actionable bool)` decision implementing the extraction table above, so classification and extraction cannot disagree, and return ordered results:

  ```go
  type RewriteResult struct {
      Passthrough []byte
      Notifications []notifyosc.Notification
  }
  func (r *NotificationRewriter) Feed(chunk []byte, normalize bool) RewriteResult
  ```

  Do not make the rewriter an overlay detector or terminal emulator. It owns only notification replacement; callers continue feeding raw bytes to those existing observers.

- [ ] **Step 4: Run the rewriter tests and verify they pass**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestNotificationRewriter' -count=1`

  Expected: PASS.

- [ ] **Step 5: Write failing proxy regressions for singular emission**

  Assert native actionable OSC reaches the inner stdout zero times and the outer sink once as canonical 777; unknown OSC reaches inner stdout unchanged and outer sink zero times; marker/idle/BEL fallback emissions use the same encoder; overlay recognition still sees raw native bytes; rate limiting remains one decision at `emitOuter`.

- [ ] **Step 6: Run proxy singular-emission regressions red**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestProxy.*(NativeNotification|CanonicalEmission|UnknownOSC|Overlay)' -count=1`

  Expected: FAIL because the proxy still forwards native actionable OSC to inner stdout and emits the legacy OSC 9 envelope.

- [ ] **Step 7: Wire the rewriter into the stdout pump and shared encoder into `emitOuter`**

  Feed raw chunks to screen/overlay detection first, then write only rewriter passthrough to ordinary stdout. Replace the rolling regex notification emission with rewriter events and change `emitOuter` to accept semantic text and write `notifyosc.Encode`. Preserve panic/error isolation and nonblocking outer-TTY writes.

- [ ] **Step 8: Run all wrapcmd tests including race-sensitive batch cases**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -count=1`

  Expected: PASS.

- [ ] **Step 9: Commit native normalization**

  Run:

  ```bash
  git add cmd/internal/wrapcmd
  git commit -m "#158: normalize native agent notifications"
  ```

### Task 3: Observe canonical OSC at the existing child parser boundary

**Files:**
- Modify: `cmd/internal/ptychild/screen.go`
- Modify: `cmd/internal/ptychild/screen_test.go`
- Modify: `cmd/internal/ptychild/child.go`
- Modify: `cmd/internal/ptychild/child_test.go`
- Modify: `cmd/internal/ptychild/fake.go`
- Modify: `cmd/internal/ptychild/replay.go`
- Modify: `cmd/internal/ptychild/replay_test.go`
- Modify: `cmd/internal/couchcore/ptyrunner.go`
- Modify: `cmd/internal/couchcore/ptyrunner_test.go`
- Modify: `cmd/internal/termcmd/run.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_menu_test.go`
- Modify: `cmd/internal/couchtty/console_live_test.go`
- Modify: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchtty/vtscreen_test.go`
- Test: `cmd/internal/ptychild/notification_benchmark_test.go`

- [ ] **Step 1: Add failing Screen tests for notification observations**

  Test whole/every-split canonical envelopes, BEL/ST, exact raw-envelope retention, defensive sanitized message, unknown/malformed byte-for-byte passthrough, and ordered text/notification/text events. Pin the exception precisely: bytes are withheld only while they remain an exact prefix of `ESC ] 777 ; notify ; pair ;`; a mismatch flushes the whole prefix immediately; once the header matches, storage remains bounded by header + 4096 message bytes + terminator. Overrun flushes buffered bytes and passes subsequent bytes transparently through the real terminator with no notification. An in-bound unterminated candidate stays withheld while later unrelated input processing and independent Screens remain live. Keep legacy bare BEL as a separate message-less attention event.

- [ ] **Step 2: Run Screen tests and verify notification access is missing**

  Run: `go test -p 20 ./cmd/internal/ptychild -run 'TestScreen.*(Notification|OSC|Bell)' -count=1`

  Expected: FAIL because `TakeOutputParts` and typed observations do not exist.

- [ ] **Step 3: Extend Screen classification and child access atomically**

  Add:

  ```go
  type NotificationObservation struct {
      Message string
      Raw []byte
  }
  type OutputPart struct {
      Bytes []byte
      Notification *NotificationObservation
  }
  type OutputBatch struct {
      Raw []byte
      Parts []OutputPart
      Bell bool
      RowDirty bool
      RingEnd uint64
      ReplaySafeEnd uint64
  }
  func (s *Screen) TakeOutputParts() []OutputPart
  ```

  Decode only a complete bounded candidate through `notifyosc.DecodeOSC`. Ordinary bytes and mismatched/overlong candidates become byte-faithful `OutputPart.Bytes`; a canonical envelope becomes one notification part carrying its exact raw bytes. Track absolute ring positions and set `ReplaySafeEnd` only through bytes for which the observer has emitted a decision; an in-progress canonical candidate therefore holds the cutoff before its prefix. Record each completed canonical envelope's absolute `[start,end)` span in a ring-bounded index and prune spans only after their end precedes the retained oldest offset. Change `ptychild.Options.Sink` and fake `SetSink` from `func([]byte)` to `func(OutputBatch)` and, inside the existing pump critical section, feed Screen and atomically drain parts/latches produced by that exact raw chunk into one owned batch before invoking Sink. Update every finite consumer listed above: non-Couch consumers use `batch.Raw`; Couch's `Deliver` blocks until the Console acknowledges that actor's batch, guaranteeing at most one unacknowledged batch per child pump. Copy retained bytes. Do not start a goroutine or perform IO in Screen.

- [ ] **Step 4: Write and run failing focus-order, takeover, and replay tests**

  Add deterministic Console tests for delivery-before-switch (consumed, no inbox), switch-before-delivery (retained), and a canonical envelope split across takeover in both focus directions. Cover every append/deliver/replay ordering: queued ordinary batches are excluded by the last Console-processed cutoff; every partial canonical split is excluded by `ReplaySafeEnd`; mismatch, overrun, termination, and teardown advance or discard the cutoff correctly. Raw parts use focus at processing time: queued bytes from the actor switched into paint once, while queued bytes from the actor switched away stay hidden and appear once on its later replay. Notification consumption alone uses the delivery stamp. Assert no canonical prefix reaches the host before completion, takeover/status bytes remain outside the OSC, and the complete ordered host stream contains exactly one full envelope.

  In `replay_test.go`, add failing cases proving completed Pair envelopes are removed while ordinary text, unknown complete OSC, and non-notification controls remain byte-faithful. Set the ring's retained-oldest boundary at every byte offset through a completed Pair envelope and prove neither its prefix, body tail, nor terminator survives replay.

  Also create a focused actor partial CSI and partial OSC, then deliver an inactive actor notification. Assert that actor's `Deliver` remains blocked with exactly one unacknowledged batch while the focused actor continues to deliver its terminator; the notification then forwards exactly once and releases only its source actor. Takeover must reset the boundary and release it as well. Verify 100 blocked inactive actors consume at most one batch each and do not prevent focused-actor progress.

  Run: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchtty -run 'Test(OutputBatchFocusOrder|SplitNotificationAcrossTakeover|StripReplayNotifications|CrossActorNotificationDeferral)' -count=1`

  Expected: FAIL because Console has no focus-stamped notification batches and replay still includes Pair envelopes.

- [ ] **Step 5: Implement Console batch routing and replay filtering, then pass focused tests**

  Implement Console behavior only after the red run. `Deliver` stamps notification visibility under `Console.mu`, sends a chunk carrying a private acknowledgement channel, and waits for acknowledgement or Console stop. `onChunk` decides raw visibility from current focus, advances the pane cutoff only after processing, and acknowledges ordinary/safe batches immediately. Takeover calls `ReplayThrough(processedSafeEnd)`; Replay uses absolute notification spans to remove every intersection, including a retention-bisected envelope. If `hostScan` is unsafe, retain that one chunk without acknowledging it, continue processing other actors, then forward and acknowledge it at the next real boundary or immediately after takeover resets the scanner, before later Console paint bytes. Then rerun: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchtty -run 'Test(OutputBatchFocusOrder|SplitNotificationAcrossTakeover|StripReplayNotifications|CrossActorNotificationDeferral)' -count=1`.

  Expected: PASS.

- [ ] **Step 6: Add allocation and throughput benchmark coverage**

  Benchmark 10 MiB of 4 KiB chunks through an oversized unterminated OSC. Add a portable `AllocsPerRun` assertion that steady skip-mode chunks allocate zero and a benchmark reporting bytes/sec; reserve the target `>=10 MiB/s` assertion for `PAIR_MENU_PERF_TARGET=m2-max`.

- [ ] **Step 7: Run all sink consumers and benchmark smoke**

  Run: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/termcmd ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1 && go test -p 20 ./cmd/internal/ptychild -run '^$' -bench 'NotificationSkip' -benchtime=100x`

  Expected: PASS; benchmark reports throughput and allocations without a CI wall-clock gate.

- [ ] **Step 8: Commit terminal observation**

  Run:

  ```bash
  git add cmd/internal/ptychild cmd/internal/couchcore/ptyrunner.go cmd/internal/couchcore/ptyrunner_test.go cmd/internal/termcmd/run.go cmd/internal/termcmd/run_test.go cmd/internal/couchcmd/run.go cmd/internal/couchtty
  git commit -m "#158: observe Pair notifications on actor PTYs"
  ```

### Task 4: Add the sole attention ledger and conditional acknowledgement

**Files:**
- Create: `cmd/internal/couchtty/attention.go`
- Test: `cmd/internal/couchtty/attention_test.go`
- Modify: `cmd/internal/couchtty/menu.go`
- Modify: `cmd/internal/couchtty/menu_test.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchtty/console_menu_operation_test.go`
- Modify: `cmd/internal/couchtty/notice.go`
- Modify: `cmd/internal/couchtty/notice_test.go`

- [ ] **Step 1: Write failing pure ledger transition tests**

  Cover focused immediate consumption, inactive insertion, exact-string dedup moving newest, three-message eviction, stable actor order, newest unread source, actor removal, attempt capture and exact-identity acknowledgement, arrival/deduplication after dispatch survival, failed-switch capture cancellation, zero/out-of-range sequence safety, `uint64` overflow rebasing that atomically remaps retained messages plus pending captures, the combined dispatch → overflow rebase → new arrival → successful completion race, and a newly constructed ledger/Console containing no state from a prior instance.

- [ ] **Step 2: Run ledger tests and verify they fail**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestAttentionLedger' -count=1`

  Expected: FAIL because the ledger does not exist.

- [ ] **Step 3: Implement the pure ledger with immutable projections**

  Use actor identity as `couchcore.ThreadAddress`; retain at most three `AttentionMessage{Text, Sequence}` values per address. Mutators execute under `Console.mu`, while `Projection()` returns owned copies for renderers. Legacy bare BEL calls `Mark(address, "")`, highlights the actor, and creates no child message row.

- [ ] **Step 4: Write failing deterministic Console integration and race tests**

  Use the existing stateful console fixture to prove: inactive canonical OSC is forwarded outward exactly once and retained; focused OSC is forwarded exactly once but not retained; unknown OSC follows ordinary focused/hidden rules; switch failure retains; a notification injected after dispatch but before success survives; success acknowledges only the captured identities; actor exit/park drops state; a fresh Console begins empty. Preserve all ordinary operation and actor-exit notices—the change removes only bell/notification authority. Add `TestAttentionHandlingStartsNoAuxiliaryWork`: inject a notification synchronously and assert inventory-provider calls, refresh generation, operation queues, and goroutine-launch seams remain unchanged; use fakes that fail on any persistence/filesystem access. Pure ledger tests import no IO runtime and use no mocks.

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(Console.*Attention|Switch.*Attention|Expected.*Exit|AttentionHandlingStartsNoAuxiliaryWork)' -count=1`

  Expected: FAIL because Console still owns parallel bell state and does not route typed notification parts.

- [ ] **Step 5: Replace parallel bell authority, capture the switch watermark, and pass integration tests**

  Delete `pane.bell`, `MenuState.Bells`, `MenuEventBell`, and their reducers. Add the ledger to Console. At switch dispatch, call `ledger.Capture(attempt, address)` so the ledger owns the exact retained identities under the same lock as sequences. On successful `forceSwitch`, call `ledger.Acknowledge(attempt)`; on failure call `ledger.Cancel(attempt)`. Rebase remaps retained identities and every pending capture in one pure transition, so later arrivals never match an older attempt accidentally. No other action clears attention. Actor exit and successful park call `DropActor` and remove captures for that actor.

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(Console.*Attention|Switch.*Attention|Expected.*Exit|AttentionHandlingStartsNoAuxiliaryWork)' -count=1`

  Expected: PASS.

- [ ] **Step 6: Run ledger and menu reducer suites together**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(AttentionLedger|Menu|ConsoleMenuOperation)' -count=1`

  Expected: PASS with no `Bells` or `pane.bell` authority remaining (`rg -n 'Bells|\.bell' cmd/internal/couchtty` finds only intentional terminal-bell compatibility test prose, if any).

- [ ] **Step 7: Commit attention ownership**

  Run:

  ```bash
  git add cmd/internal/couchtty
  git commit -m "#158: centralize Couch actor attention state"
  ```

### Task 5: Render attention in the status row and hierarchical switcher

**Files:**
- Modify: `cmd/internal/couchtty/reserve.go`
- Modify: `cmd/internal/couchtty/reserve_test.go`
- Modify: `cmd/internal/couchtty/menu_render.go`
- Modify: `cmd/internal/couchtty/menu_render_test.go`
- Modify: `cmd/internal/couchtty/menu_perf_test.go`
- Modify: `cmd/internal/couchtty/console.go`

- [ ] **Step 1: Write failing status-row presentation tests**

  Pin that attention changes only ANSI color around the existing inactive label, adds no star/dot/count, preserves actor spacing and visible width, clips narrow rows correctly, and never marks the focused actor. Include three active actors with attention on one inactive actor.

- [ ] **Step 2: Write failing switcher hierarchy and selection tests**

  With 100 actors and three messages each, assert each message is an indented line immediately below its actor, children are display-only, Up/Down visit actor rows only, filtering considers actor fields but not message text, actor order is unchanged, and opening selects the actor with greatest unread sequence. Pin viewport behavior when children expand beyond terminal height and width-aware clipping of hostile/wide text.

- [ ] **Step 3: Run render tests and verify the old star/bell UI fails**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderStatus|RenderMenu|Menu.*Attention)' -count=1`

  Expected: FAIL because current renderers use bell stars and have no notification children.

- [ ] **Step 4: Implement read-only attention projections in both render paths**

  Change `StatusActor.Bell` to `Attention bool` and apply one existing palette-compatible attention style without changing cell width. Add `Attention map[ThreadAddress][]AttentionMessage` (or an equivalent immutable projection) to the menu render model, not its reducer authority. Calculate viewport height from visual lines while selection indexes actor rows. On `Ctrl-Space`, initialize the root frame selection from `ledger.NewestActor()` without sorting inventory.

- [ ] **Step 5: Extend the 100-row performance fixture and target protocol**

  Give every actor three retained messages. Keep 20 warmups and 200 samples at 120x40, four load workers, `<50 ms p95` first frame, and `<16 ms p95` selection/navigation/render computation. Update portable allocation bounds only from measured stable results; do not loosen them speculatively.

- [ ] **Step 6: Run render, Console, and portable performance tests**

  Run: `go test -p 20 ./cmd/internal/couchtty -count=1`

  Expected: PASS, including `TestMenu100Bounds`; target-only latency test skips unless explicitly enabled.

- [ ] **Step 7: Commit the user-facing attention UI**

  Run:

  ```bash
  git add cmd/internal/couchtty
  git commit -m "#158: surface actor attention in Couch UI"
  ```

### Task 6: Prove the complete PTY path, package it, and update the map

**Files:**
- Test: `cmd/internal/couchtty/notification_pty_test.go`
- Modify: `cmd/internal/runtimebundle/assets/runtime/files/bin/pair-notify`
- Modify: `cmd/internal/runtimebundle/assets/runtime/manifest.json`
- Modify: `cmd/internal/runtimebundle/embed_test.go`
- Modify: `cmd/internal/artifactpath/coverage_test.go`
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md`
- Modify: `workshop/issues/000158-couch-actor-notifications.md`

- [ ] **Step 1: Write a real-PTY conformance test over the completed production path**

  Start a controlled child on a real PTY that invokes the production canonical notification command against its recorded outer TTY. Prove the host receives the exact envelope once and Console attributes the decoded message to the inactive actor. Repeat focused to prove exact once-forwarding with no retained attention. Use bounded deadlines only as safety guards and always join/reap the child.

- [ ] **Step 2: Run the conformance test as independent verification**

  Run: `go test -p 20 ./cmd/internal/couchtty -run '^TestNotificationPTYConformance$' -count=1 -v`

  Expected: PASS. If it fails, return to the owning earlier task and add a focused failing regression before changing production code; do not add test-only injection or an unspecified wiring patch here.

- [ ] **Step 3: Regenerate and verify the runtime bundle from tracked inputs**

  Run: `make runtimebundle-generate && go test -p 20 ./cmd/internal/runtimebundle ./cmd/internal/artifactpath -count=1`

  Expected: PASS; generated manifest and `pair-notify` mirror match tracked sources in both directions.

- [ ] **Step 4: Update architecture and issue log**

  In `atlas/architecture.md`, replace the old independent OSC 9/777 helper description with the singular canonical Pair envelope, native replacement path, Couch tee behavior, bounded ephemeral ledger, and successful-switch acknowledgement. Ensure `atlas/index.md` still links the map. Append implementation discoveries and verification commands to the issue `## Log`; do not rewrite prior revisions.

- [ ] **Step 5: Run the complete bounded verification suite**

  Run:

  ```bash
  go test -p 20 ./cmd/internal/notifyosc ./cmd/internal/notifycmd ./cmd/internal/wrapcmd ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/termcmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint ./cmd/internal/runtimebundle ./cmd/internal/artifactpath -count=1
  go test -p 20 -race ./cmd/internal/notifyosc ./cmd/internal/wrapcmd ./cmd/internal/ptychild ./cmd/internal/couchtty -count=1
  go test -p 20 ./cmd/internal/ptychild -run '^$' -bench 'NotificationSkip' -benchtime=100x
  bash -n bin/pair-notify
  git diff --check
  ```

  Expected: all tests PASS, benchmark reports bounded allocations/throughput, shell syntax is clean, and `git diff --check` prints nothing. Never exceed 20 Go test workers.

- [ ] **Step 6: Run the target M2 Max performance protocol**

  Run on the operator's otherwise normal development machine:

  ```bash
  PAIR_MENU_PERF_TARGET=m2-max go test -p 20 ./cmd/internal/couchtty ./cmd/internal/ptychild -run 'Test(MenuTargetPerformance|NotificationSkipTarget)$' -count=1 -v
  ```

  Expected: PASS with the 100-actor/300-message UI p95 limits and malformed-stream throughput `>=10 MiB/s` under the documented four-worker co-tenancy load.

- [ ] **Step 7: Commit the conformance evidence and documentation**

  Run:

  ```bash
  git add cmd/internal/couchtty/notification_pty_test.go cmd/internal/runtimebundle cmd/internal/artifactpath atlas workshop/issues/000158-couch-actor-notifications.md
  git commit -m "#158: verify actor notification routing"
  ```

- [ ] **Step 8: Stop at the SDLC close boundary**

  Review `git status --short`, `git diff main...HEAD --stat`, and the issue checklist. Then run `sdlc close --issue 158 --verified '<exact commands and observed results>'` only after every task above is checked and the implementation is smoke-testable. The close command owns the mandatory fresh-context boundary review; do not dispatch a redundant review manually.

## Revisions

### 2026-08-31T22:11:42-07:00 — preserve canonical framing across asynchronous takeover

**Reason:** plan review proved that immediate canonical-prefix forwarding cannot
remain byte-faithful when Couch takes over the one outer terminal stream between
PTY reads. Subsequent review also found replay/delivery races, cross-actor
sequence splicing, and acknowledgement rebase invalidation.

**Delta:** only an exact bounded Pair-envelope candidate is withheld; every
other byte flushes immediately. Output batches now carry processed replay-safe
cutoffs plus absolute notification spans, replay removes completed Pair
envelopes even across ring-retention cuts, unsafe cross-actor forwarding
backpressures one batch per source actor, and acknowledgement captures live
inside the ledger so overflow rebase updates them atomically.
Tests cover every split and append/deliver/replay ordering plus the complete
outer host byte stream (`ARCH-PURPOSE`, `ARCH-PURE`, `ARCH-CONSTRAINTS`).
