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
| `ptychild.Screen` replay-safe offset | `cmd/internal/ptychild/screen.go` | modified |
| `ptychild.Child.ReplayThrough` and notification spans | `cmd/internal/ptychild/child.go` | modified |
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

- **`ptychild.Screen` replay-safe offset and `ptychild.Child.ReplayThrough`** — Screen identifies the absolute raw-stream offset that does not bisect a withheld canonical candidate; Child retains completed notification spans beside its bounded ring and removes every intersection through the Console-processed cutoff.
  - **Relationships:** Each Screen owns one monotonically advancing stream position; its Child owns the bounded span index; each Console pane records at most one processed replay-safe cutoff.
  - **DRY rationale:** Replay and live delivery derive from the same Screen observations instead of independently guessing whether a partial OSC is safe.
  - **Future extensions:** Other snapshot consumers can request explicit historical cutoffs through Child without changing ring retention policy.

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
| Architecture map | `atlas/architecture.md` | modified | operator-facing notification system map |

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

- **Architecture map** — documents the singular notification byte path and ephemeral Couch ownership. Runtime bundle generation verifies that ignored derived assets already match tracked inputs; `atlas/index.md` already links this map and remains unchanged.
  - **Injected into:** Future maintainers; installed Pair distributions derive their runtime mirror from the tracked shim.
  - **Future extensions:** None; the map describes the implemented surface while generated output stays derived.

### Task 1: Build the canonical notification codec and hook command

**Files:**
- Create: `cmd/internal/notifyosc/notification.go`
- Test: `cmd/internal/notifyosc/notification_test.go`
- Create: `cmd/internal/notifycmd/run.go`
- Test: `cmd/internal/notifycmd/run_test.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/internal/dispatcher/dispatcher_test.go`
- Modify: `bin/pair-notify`
- Verify unchanged: `cmd/internal/entrypoint/alias.go`, `cmd/internal/entrypoint/alias_test.go` (dispatcher discovery already owns the command)

- [x] **Step 1: Write codec tests that pin exact bytes and all sanitization boundaries**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `notifyosc.Sanitize` | Fuzz invalid UTF-8, control runes, and boundary-sized multibyte text; assert valid UTF-8, no C0/DEL/C1, whole-rune output, and `len <= MaxMessageBytes`. |
  | `notifyosc.Encode` | Fuzz arbitrary messages; assert one exact canonical envelope whose decoded message equals `Sanitize(input)`. |
  | `notifyosc.DecodeOSC` | Fuzz arbitrary terminal bytes seeded with BEL/ST canonical and near-canonical forms; accept only one complete exact Pair envelope and never panic. |

- [x] **Step 2: Run the codec test and verify the package is absent**

  Run: `go test -p 20 ./cmd/internal/notifyosc -count=1`

  Expected: FAIL because `Notification`, `Sanitize`, `Encode`, and `DecodeOSC` do not exist.

- [x] **Step 3: Implement the minimal pure codec**

  Use these contracts:

  ```go
  const MaxMessageBytes = 4 << 10

  type Notification struct { Message string }

  func Sanitize(raw []byte) string
  func Encode(message string) []byte
  func DecodeOSC(sequence []byte) (Notification, bool)
  ```

  `Sanitize` must decode invalid input with replacement, remove runes in `U+0000..001F`, `U+007F..009F`, then append whole UTF-8 runes until the 4096-byte bound. `DecodeOSC` accepts only complete canonical Pair envelopes terminated by BEL or ST, defensively sanitizes the body, and rejects all other OSC.

- [x] **Step 4: Run codec tests and verify they pass**

  Run: `go test -p 20 ./cmd/internal/notifyosc -count=1`

  Expected: PASS.

- [x] **Step 5: Write failing stateful-runtime tests for `notifycmd.Run` and dispatcher routing**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `notifycmd.Run` | Drive malformed CLI/environment and every stateful sidecar/TTY state through one fake runtime; assert tolerant hook exits and exactly one canonical write only from a valid state. |
  | `dispatcher.Dispatch` | Resolve legacy option forms and invalid commands through the real dispatcher; assert both supported OSC options converge on `notifycmd.Run`. |

- [x] **Step 6: Run the command tests and verify the route is missing**

  Run: `go test -p 20 ./cmd/internal/notifycmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1`

  Expected: FAIL because `notifycmd.Run`, the `notify` family, and shim delegation are not implemented.

- [x] **Step 7: Implement `pair notify` and reduce the shell helper to delegation**

  Implement the named codec and `notifycmd.Runtime` contracts above. Route `pair notify` through the buffered dispatcher and keep `pair-notify` as a symlink-safe hook shim delegating to its sibling `pair`; all encoding remains in Go.

- [x] **Step 8: Run focused command tests and shell syntax checks**

  Run: `go test -p 20 ./cmd/internal/notifyosc ./cmd/internal/notifycmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint -count=1 && bash -n bin/pair-notify`

  Expected: PASS and no shell syntax output.

- [x] **Step 9: Commit the canonical emission boundary**

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
- Verify unchanged: `cmd/internal/wrapcmd/osc_test.go`, `cmd/internal/wrapcmd/stdout_batch_test.go`, `cmd/internal/wrapcmd/picker_overlay_test.go` (new rewriter tests cover the replacement seam; existing suites remain consumers)

- [x] **Step 1: Write failing arbitrary-split tests for the native notification rewriter**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `nativeNotification` | Fuzz OSC family/body fields seeded with OSC 9/777 and embedded delimiters; apply the documented `SplitN(..., 3)` extraction and reject non-actionable forms. |
  | `NotificationRewriter.Feed` | Fuzz arbitrary chunk partitions and malformed/overlong streams; recognized events normalize once, all other bytes preserve order, and pending memory stays bounded through the real terminator. |

- [x] **Step 2: Run the rewriter tests and verify they fail**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestNotificationRewriter' -count=1`

  Expected: FAIL because `NotificationRewriter` does not exist.

- [x] **Step 3: Implement the minimal incremental rewriter**

  Implement one bounded incremental rewriter using the following contract; `nativeNotification` is the sole classification/extraction decision:

  ```go
  type RewriteResult struct {
      Passthrough []byte
      Notifications []notifyosc.Notification
  }
  func (r *NotificationRewriter) Feed(chunk []byte, normalize bool) RewriteResult
  ```

  Do not make the rewriter an overlay detector or terminal emulator. It owns only notification replacement; callers continue feeding raw bytes to those existing observers.

- [x] **Step 4: Run the rewriter tests and verify they pass**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestNotificationRewriter' -count=1`

  Expected: PASS.

- [x] **Step 5: Write failing proxy regressions for singular emission**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `proxy.handleAgentOutput` | Feed mixed actionable/unknown OSC while observing inner and outer sinks; assert one output owner per byte and raw overlay evidence remains available. |
  | `proxy.emitOuter` | Drive native/marker/idle/fallback sources through the shared sink and rate limiter; assert every accepted source writes `notifyosc.Encode` exactly once. |

- [x] **Step 6: Run proxy singular-emission regressions red**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -run 'TestProxy.*(NativeNotification|CanonicalEmission|UnknownOSC|Overlay)' -count=1`

  Expected: FAIL because the proxy still forwards native actionable OSC to inner stdout and emits the legacy OSC 9 envelope.

- [x] **Step 7: Wire the rewriter into the stdout pump and shared encoder into `emitOuter`**

  Integrate `NotificationRewriter.Feed` as the sole native replacement decision while preserving the existing raw overlay observer and failure-isolated, nonblocking outer sink.

- [x] **Step 8: Run all wrapcmd tests including race-sensitive batch cases**

  Run: `go test -p 20 ./cmd/internal/wrapcmd -count=1`

  Expected: PASS.

- [x] **Step 9: Commit native normalization**

  Run:

  ```bash
  git add cmd/internal/wrapcmd
  git commit -m "#158: normalize native agent notifications"
  ```

### Task 3: Observe canonical OSC at the existing child parser boundary

**Files:**
- Modify: `cmd/internal/ptychild/screen.go`
- Verify unchanged: `cmd/internal/ptychild/screen_test.go` (notification-specific Screen coverage lives in `notification_test.go`)
- Modify: `cmd/internal/ptychild/child.go`
- Modify: `cmd/internal/ptychild/child_test.go`
- Modify: `cmd/internal/ptychild/fake.go`
- Verify unchanged: `cmd/internal/ptychild/replay.go` (query stripping is unchanged; notification spans belong to Child)
- Modify: `cmd/internal/ptychild/replay_test.go`
- Modify: `cmd/internal/couchcore/ptyrunner.go`
- Modify: `cmd/internal/couchcore/ptyrunner_test.go`
- Modify: `cmd/internal/termcmd/run.go`
- Verify unchanged: `cmd/internal/termcmd/run_test.go`, `cmd/internal/couchcmd/run.go` (compile/full-suite coverage verifies their existing typed wiring)
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_menu_test.go`
- Modify: `cmd/internal/couchtty/console_live_test.go`
- Modify: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchtty/vtscreen_test.go`
- Test: `cmd/internal/ptychild/notification_benchmark_test.go`

- [x] **Step 1: Add failing Screen tests for notification observations**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `Screen.Feed` / `TakeOutputParts` | Fuzz arbitrary PTY partitions seeded with canonical-prefix mismatch, termination, overrun, and incompletion; only an exact bounded candidate is withheld, all other bytes preserve order, and independent Screen state remains isolated. |
  | `Screen.TakeBell` | Mix bare BEL with framed terminators; only a ground-state BEL produces the compatibility event. |

- [x] **Step 2: Run Screen tests and verify notification access is missing**

  Run: `go test -p 20 ./cmd/internal/ptychild -run 'TestScreen.*(Notification|OSC|Bell)' -count=1`

  Expected: FAIL because `TakeOutputParts` and typed observations do not exist.

- [x] **Step 3: Extend Screen classification and child access atomically**

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

  Implement the typed contracts above. `Screen.Feed` owns canonical-candidate decisions and replay-safe offsets; `Child.pump` atomically packages those decisions with the raw ring append. Non-Couch sinks consume `Raw`; Couch uses ordered parts and an acknowledged one-batch-per-actor handoff.

- [x] **Step 4: Write and run failing focus-order, takeover, and replay tests**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `Child.ReplayThrough` | Fuzz ring-retention cuts and processed cutoffs seeded at every canonical-envelope offset; absolute spans remove every intersecting notification while preserving unrelated retained bytes. |
  | `Console.Deliver` / `onChunk` | Deterministically permute append, delivery, focus switch, processing, and teardown; processing-time focus owns raw bytes, delivery-time focus owns notification consumption, and each byte appears once on the correct screen. |
  | `Console.flushDeferredNotifications` | Hold the host scanner in partial CSI/OSC while many actors notify; one unacknowledged batch backpressures each source, focused progress reaches a safe boundary, and arrival-ordered envelopes then flush exactly once. |
  | `Console.switchTo` | Split canonical candidates across takeover and replay; processed cutoffs and notification spans keep Console control bytes outside every envelope. |

  Run: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchtty -run 'Test(OutputBatchFocusOrder|SplitNotificationAcrossTakeover|StripReplayNotifications|CrossActorNotificationDeferral)' -count=1`

  Expected: FAIL because Console has no focus-stamped notification batches and replay still includes Pair envelopes.

- [x] **Step 5: Implement Console batch routing and replay filtering, then pass focused tests**

  Implement the four named contracts above, keeping replay/filtering pure and Console as the thin serialized host-IO owner. Then rerun: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchtty -run 'Test(OutputBatchFocusOrder|SplitNotificationAcrossTakeover|StripReplayNotifications|CrossActorNotificationDeferral)' -count=1`.

  Expected: PASS.

- [x] **Step 6: Add allocation and throughput benchmark coverage**

  Benchmark `Screen.Feed` with sustained malformed 4 KiB chunks; guard bounded pending memory and zero steady skip-mode allocation, while the target-only protocol enforces `>=10 MiB/s`.

- [x] **Step 7: Run all sink consumers and benchmark smoke**

  Run: `go test -p 20 ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/termcmd ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1 && go test -p 20 ./cmd/internal/ptychild -run '^$' -bench 'NotificationSkip' -benchtime=100x`

  Expected: PASS; benchmark reports throughput and allocations without a CI wall-clock gate.

- [x] **Step 8: Commit terminal observation**

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
- Verify unchanged: `cmd/internal/couchtty/console_menu_operation_test.go` (switch acknowledgement regressions live with notification integration tests)
- Modify: `cmd/internal/couchtty/notice.go`
- Modify: `cmd/internal/couchtty/notice_test.go`

- [x] **Step 1: Write failing pure ledger transition tests**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `AttentionLedger.Mark` | Generate repeated/distinct messages across actors and near sequence overflow; enforce three-message bounds, exact deduplication, stable newest ordering, and atomic retained/capture rebase. |
  | `AttentionLedger.Capture` / `Acknowledge` / `Cancel` | Permute dispatch, rebase, later arrival/deduplication, success, and failure; only identities present at capture are clearable. |
  | `AttentionLedger.DropActor` / `Projection` / `NewestActor` | Mutate actor lifecycles and caller-owned projections; removed/fresh actors expose no stale or aliased state. |

- [x] **Step 2: Run ledger tests and verify they fail**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestAttentionLedger' -count=1`

  Expected: FAIL because the ledger does not exist.

- [x] **Step 3: Implement the pure ledger with immutable projections**

  Implement the named pure transitions with `couchcore.ThreadAddress` identity, owned projections, three messages per actor, and message-less legacy BEL compatibility.

- [x] **Step 4: Write failing deterministic Console integration and race tests**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `Console.onChunk` | Drive focused/inactive/malformed observations through the stateful child/host fake; forwarding and ledger ownership remain singular with no persistence, refresh, or auxiliary work. |
  | `Console.finishOperation` | Interleave switch result with later attention and actor park/exit; success acknowledges captured identities, failure preserves them, and lifecycle teardown drops only ephemeral state. |

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(Console.*Attention|Switch.*Attention|Expected.*Exit|AttentionHandlingStartsNoAuxiliaryWork)' -count=1`

  Expected: FAIL because Console still owns parallel bell state and does not route typed notification parts.

- [x] **Step 5: Replace parallel bell authority, capture the switch watermark, and pass integration tests**

  Replace `pane.bell`, `MenuState.Bells`, and `MenuEventBell` with the sole Console-owned ledger, wiring the named capture/acknowledge/cancel/drop transitions at existing operation and lifecycle boundaries.

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(Console.*Attention|Switch.*Attention|Expected.*Exit|AttentionHandlingStartsNoAuxiliaryWork)' -count=1`

  Expected: PASS.

- [x] **Step 6: Run ledger and menu reducer suites together**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(AttentionLedger|Menu|ConsoleMenuOperation)' -count=1`

  Expected: PASS with no `Bells` or `pane.bell` authority remaining (`rg -n 'Bells|\.bell' cmd/internal/couchtty` finds only intentional terminal-bell compatibility test prose, if any).

- [x] **Step 7: Commit attention ownership**

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

- [x] **Step 1: Write failing status-row presentation tests**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `RenderStatusRow` | Fuzz widths and actor projections; attention changes only ANSI style around inactive labels and never changes measured cells, roster order, or focused treatment. |

- [x] **Step 2: Write failing switcher hierarchy and selection tests**

  | Function | Adversarial strategy and mechanical guard |
  |----------|-------------------------------------------|
  | `RenderMenu` | Fuzz terminal dimensions and hostile/wide message text across the 100-actor/300-child bound; notification children stay clipped, indented, and display-only. |
  | `VisibleMenuThreads` / `reduceKey` | Generate filters and navigation over actors with children; only actor fields admit rows and only actor identities participate in order/selection. |
  | `Console.showMenu` | Vary unread sequences without changing inventory order; initial selection is the ledger's newest unread actor from resident state. |

- [x] **Step 3: Run render tests and verify the old star/bell UI fails**

  Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderStatus|RenderMenu|Menu.*Attention)' -count=1`

  Expected: FAIL because current renderers use bell stars and have no notification children.

- [x] **Step 4: Implement read-only attention projections in both render paths**

  Implement the three named rendering/selection contracts from immutable ledger projections; the menu reducer gains no parallel attention authority.

- [x] **Step 5: Extend the 100-row performance fixture and target protocol**

  Exercise `RenderMenu`, `reduceKey`, and `Console.showMenu` at the declared 100-actor/300-message target using the existing 20-warmup/200-sample, four-worker protocol and its p95 budgets.

- [x] **Step 6: Run render, Console, and portable performance tests**

  Run: `go test -p 20 ./cmd/internal/couchtty -count=1`

  Expected: PASS, including `TestMenu100Bounds`; target-only latency test skips unless explicitly enabled.

- [x] **Step 7: Commit the user-facing attention UI**

  Run:

  ```bash
  git add cmd/internal/couchtty
  git commit -m "#158: surface actor attention in Couch UI"
  ```

### Task 6: Prove the complete PTY path, package it, and update the map

**Files:**
- Test: `cmd/internal/couchtty/notification_pty_test.go`
- Verify derived unchanged: `cmd/internal/runtimebundle/assets/runtime/files/bin/pair-notify`, `cmd/internal/runtimebundle/assets/runtime/manifest.json` (ignored generated outputs already matched tracked inputs)
- Verify unchanged: `cmd/internal/runtimebundle/embed_test.go`
- Modify: `cmd/internal/artifactpath/coverage_test.go`
- Modify: `atlas/architecture.md`
- Verify unchanged: `atlas/index.md` (already links `atlas/architecture.md`)
- Modify: `workshop/issues/000158-couch-actor-notifications.md`

- [x] **Step 1: Write a real-PTY conformance test over the completed production path**

  Start a controlled child on a real PTY that invokes the production canonical notification command against its recorded outer TTY. Prove the host receives the exact envelope once and Console attributes the decoded message to the inactive actor. Repeat focused to prove exact once-forwarding with no retained attention. Use bounded deadlines only as safety guards and always join/reap the child.

- [x] **Step 2: Run the conformance test as independent verification**

  Run: `go test -p 20 ./cmd/internal/couchtty -run '^TestNotificationPTYConformance$' -count=1 -v`

  Expected: PASS. If it fails, return to the owning earlier task and add a focused failing regression before changing production code; do not add test-only injection or an unspecified wiring patch here.

- [x] **Step 3: Regenerate and verify the runtime bundle from tracked inputs**

  Run: `make runtimebundle-generate && go test -p 20 ./cmd/internal/runtimebundle ./cmd/internal/artifactpath -count=1`

  Expected: PASS; generated manifest and `pair-notify` mirror match tracked sources in both directions.

- [x] **Step 4: Update architecture and issue log**

  In `atlas/architecture.md`, replace the old independent OSC 9/777 helper description with the singular canonical Pair envelope, native replacement path, Couch tee behavior, bounded ephemeral ledger, and successful-switch acknowledgement. Ensure `atlas/index.md` still links the map. Append implementation discoveries and verification commands to the issue `## Log`; do not rewrite prior revisions.

- [x] **Step 5: Run the complete bounded verification suite**

  Run:

  ```bash
  go test -p 20 ./cmd/internal/notifyosc ./cmd/internal/notifycmd ./cmd/internal/wrapcmd ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/termcmd ./cmd/internal/dispatcher ./cmd/internal/entrypoint ./cmd/internal/runtimebundle ./cmd/internal/artifactpath -count=1
  go test -p 20 -race ./cmd/internal/notifyosc ./cmd/internal/wrapcmd ./cmd/internal/ptychild ./cmd/internal/couchtty -count=1
  go test -p 20 ./cmd/internal/ptychild -run '^$' -bench 'NotificationSkip' -benchtime=100x
  bash -n bin/pair-notify
  git diff --check
  ```

  Expected: all tests PASS, benchmark reports bounded allocations/throughput, shell syntax is clean, and `git diff --check` prints nothing. Never exceed 20 Go test workers.

- [x] **Step 6: Run the target M2 Max performance protocol**

  Run on the operator's otherwise normal development machine:

  ```bash
  PAIR_MENU_PERF_TARGET=m2-max go test -p 20 ./cmd/internal/couchtty ./cmd/internal/ptychild -run 'Test(MenuTargetPerformance|NotificationSkipTarget)$' -count=1 -v
  ```

  Expected: PASS with the 100-actor/300-message UI p95 limits and malformed-stream throughput `>=10 MiB/s` under the documented four-worker co-tenancy load.

- [x] **Step 7: Commit the conformance evidence and documentation**

  Run:

  ```bash
  git add cmd/internal/couchtty/notification_pty_test.go cmd/internal/runtimebundle cmd/internal/artifactpath atlas workshop/issues/000158-couch-actor-notifications.md
  git commit -m "#158: verify actor notification routing"
  ```

- [x] **Step 8: Stop at the SDLC close boundary**

  Review `git status --short`, `git diff main...HEAD --stat`, and the issue checklist. Then run `sdlc close --issue 158 --verified '<exact commands and observed results>'` only after every task above is checked and the implementation is smoke-testable. The close command owns the mandatory fresh-context boundary review; do not dispatch a redundant review manually.

## Revisions

### 2026-09-01T00:05:00-07:00 — separate observation from framing-only feeds

**Reason:** close review BR-5 found that adding retained notification output to
`Screen.Feed` changed the contract for every consumer. The Console's
framing-only `hostScan` never drained those parts, so focused actor output grew
without bound.

**Delta:** `Screen.FeedFraming` is the explicit non-retaining seam for callers
that only need terminal state and `MidSequence`; observation-owning child pumps
continue to use `Feed` and drain `TakeOutputParts`. A Console-level sustained
ordinary-output regression asserts that the framing-only scanner retains no
output parts (`ARCH-PURE`, `ARCH-CONSTRAINTS`).

### 2026-08-31T23:51:00-07:00 — classify every declared path against the committed diff

**Reason:** close review BR-4 found that the integration table bundled one
tracked atlas modification with an unchanged index and ignored generated
runtime outputs. This repeated the `plan-code-traceability` family from BR-1.

**Delta:** every Core concepts row was checked against `git diff --name-status
f93bd568..HEAD`; only real symbols at tracked changed paths retain `new` or
`modified`. The architecture row now names only `atlas/architecture.md`.
Prospective Task file lists that remained unchanged are explicitly classified
`Verify unchanged`, and runtime assets are `Verify derived unchanged` because
they are ignored generator output that already matched tracked inputs. The
rule is: a boundary table describes committed entities; verification-only and
derived surfaces are stated separately, never labeled modified
(`ARCH-PURPOSE`).

### 2026-08-31T23:42:00-07:00 — align replay concepts with delivered ownership

**Reason:** close review BR-1 found that the Core concepts table named a
planned `ptychild.ReplayWindow` in `replay.go`, but implementation placed the
two responsibilities on their existing lifecycle owners.

**Delta:** the table and prose now name `Screen` as replay-safe stream-offset
owner and `Child.ReplayThrough` plus Child's retained spans as replay-filtering
owner. No standalone ReplayWindow entity or `replay.go` modification exists;
the revised model matches the tested implementation (`ARCH-PURPOSE`).

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

### 2026-08-31T22:28:00-07:00 — express verification at named function boundaries

**Reason:** the `change-code` plan-quality gate rejected enumerated prose cases
and procedural diff restatements as a lossy duplicate of the tests and code.

**Delta:** Tasks 1–5 now name every risky production function and give one
adversarial-input/mechanical-guard strategy line per function. Red/green
commands, integration seams, public contracts, and acceptance evidence remain;
case enumeration and implementation choreography were removed (`ARCH-PURE`,
`ARCH-PURPOSE`).
