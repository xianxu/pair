---
collaboration-mode: discussion
---

# Term wrapper resilience implementation plan

> **For agentic workers:** Follow AGENTS.md Section 3. Use bounded delegation for independent tests; the SDLC close owns the fresh review. This is one atomic delivery, with no milestone tags.

**Goal:** A malformed backend cell or transient zero-sized pane cannot destroy the operator's terminal, and a lost right pane has a working recovery key.

**Architecture:** Repair backend-derived screen/history cells at their shared publication boundary, retaining strict validators and owned snapshots. Keep existing host/presenter/PTY ownership, wait for valid launch geometry, and recover unexpected wrapper failures only after complete teardown. Reuse shared shortcut routing to recreate a terminal on confirmed absence.

**Tech stack:** Go, existing VT emulator and terminal presenter, hostty.FakeHost/ttyio.Fake, real disposable PTYs and /bin/sh, stateful zellijtest fixture.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Backend cell repair and bounded first-fault evidence | cmd/internal/terminal/cell_degrade.go | new |
| Session ending/recovery classification | cmd/internal/termcmd/recovery.go | new |
| Right terminal launch command | cmd/internal/workbenchshortcut/terminal_command.go | new shared owner; delete termcmd duplicate |

Cell repair walks each owned row once using validateCell, replacing an invalid cell with
U+FFFD of width one, removing its hyperlink and retaining safe style. Wide-cell neighbors
must remain valid and preserve their screen columns; never shift the remaining row. Handle
control/invalid UTF-8/oversized clusters, invalid links, unsupported widths, mismatched
clusters and broken wide pairs. The strict Frame/History validators and Render refusal
remain intact: trusted chrome/panel mistakes still refuse. The repair is only for cells
coming from the backend, including retained primary history, before any composition/tap.
Return bounded first-fault evidence and degradation counts; do not retain untrusted strings.

The recovery classifier distinguishes ordinary child exit, intentional host termination,
recoverable wrapper/setup failure, and a gone terminal. Only wrapper failure with usable
stdio requests one native-shell exec. It must preserve a host-termination marker through
joined cleanup errors; there is no automatic wrapper restart or recovery loop.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Endpoint publication | cmd/internal/terminal/endpoint.go | modified | owned emulator screen/history snapshots |
| Cell diagnostic observation | terminal/observation.go, termcmd/diagnostics.go, ptychild.Options.Observer | modified/new | existing nonblocking observation and diagnosticlog.OpenBuffered |
| Initial geometry wait and session teardown | termcmd/run.go, termcmd/recovery.go | modified/new | existing Host.Resized/Terminated and shell exec |
| Missing terminal creation | cmd/internal/layoutcmd/layoutcmd.go | modified | confirmed native current-tab inventory and existing Zellij effect admission |

No backend mutation or raw PTY replay. Observations borrow only bounded encoded evidence;
existing capture consumers remain compatible. Term diagnostics use a registered managed
JSONL under the selected Pair data directory, opened outside endpoint locks; the existing
buffered writer owns its worker/timer, closes after all children, and obeys diagnosticlog's
size/retention policy (64 MiB generations, daily rotation, seven-day retention). At most one
fault observation per endpoint lifetime; emit no whole frame, raw URL or full path. Encode
up to 64 content codepoints plus up to 32 invalid-byte hex values, width, cell position,
reason and original byte lengths; clipping is explicit. Logging failure cannot stop paint.
Tests select a private directory; no real user diagnostics or configuration are used.

## Visible states and recovery

Actions: continue editing; resize/reveal pane; use the plain shell; press Alt+Shift+T;
report terminal transport failure.

| Internal state/failure | Operator sees | One recovery action | Possible loss |
|---|---|---|---|
| Backend cell invalid | One replacement glyph, app remains interactive | Continue editing | That cell's displayed glyph/link only; backend content unchanged |
| Launch size zero, within 5 s | Waiting for pane size | Resize/reveal pane | None; no child launched yet |
| Size becomes valid | Normal terminal/app | Continue editing | None |
| Size remains zero or wrapper setup/paint fails, terminal usable | Plain interactive shell with bounded reason | Use the plain shell | Terminated child may have unsaved work; no invented restoration |
| Mid-session zero resize | Last valid display | Resize/reveal pane | None; input and child retained |
| Right terminal absent, surviving Pair pane | Alt+Shift+T creates a right terminal | Press Alt+Shift+T | Closed app state not restored |
| Native pane inventory unknown | Existing refusal; no new pane | Press Alt+Shift+T after the native inventory recovers | None from this action; uncertain absence is never authority |
| Host terminated/disconnected | Existing session ends | Report terminal transport failure | No usable terminal exists; do not spawn into a dead host |

The recovery-key contract applies to the existing Pair-aware draft/agent panes, whose
current binding already owns this key. Do not add a new global chord or change plain nvim's
configuration. Operator acceptance must cover loss and recreation from a surviving pane.

## Ordering, budgets and trust

- Child terminal output is untrusted. Raw controls never become parent cell text or
  diagnostic control sequences. Only repaired owned publications reach renderer/taps.
- Geometry: read once, then wait on coalesced Resized with one 5 s deadline, checking
  termination/closed channels; no sleeps, filesystem polls, busy retry or guessed 80x24.
  A valid size starts exactly one child. Timeout is wrapper failure and native-shell recovery.
- Existing zero-resize ignore remains before chrome/endpoint work. No new worker for size.
- Unexpected wrapper failure joins input/resize producers, closes children, releases
  presenter modes and restores raw/transport ownership before fallback exec. Restore
  failures are visible; no recursive wrapper fallback. Native shell inherits original
  environment/stdio, not virtual TERM overrides. Normal shell exit stays normal.
- New-terminal action positively confirms current-tab inventory for this chord, bypassing
  a stale registry-only fast path. Existing terminal: deliver once. Confirmed missing:
  one create. Unknown/cancelled/missing caller: no effect. No retry on uncertain write.
- Hot path: C screen cells + H retained history cells per publication => O(C+H), bounded
  by existing cell/history caps; no per-cell IO or logging allocations on healthy rows.
  Count repair visits/degradations at 8/16 rows and assert one row pass and at most one
  observation per endpoint after repeated publications. No endpoint-sized dedup map.
- Sidecar/catalog formats are unchanged. Diagnostic records are additive facts, not
  authority. Unknown observation kinds must not break existing optional capture readers.
- No machine changes, production pane driving or pair:0 staging without hub coordination.

## Function-level adversarial strategy

| Risky production function | Strategy and mechanical guard |
|---|---|
| repairBackendRow(cells) | Generated arbitrary Cell rows (including arbitrary byte strings/width/link fields); independent strict validation after repair, fixed row length/column positions, unchanged originally valid cells unless part of a broken wide pair. Count visits at 8/16 rows. |
| describeCellFault(cell, position, reason) | Arbitrary large/malformed cell strings; bound encoded bytes, codepoint/hex element counts and verify no terminal controls or raw link addresses in JSON output. |
| Endpoint.capture / captureHistory | Inject raw backend screen/history cells, publish through the real presenter and emulator oracle; confirm valid projection, unchanged backend, later input delivery, one observation across repeated snapshots. |
| openTermDiagnostics / observeCellFault | Actual managed JSONL writer in private directory; close/drain ownership and sink failure cannot affect presentation; verify codepoint evidence through encoding. |
| waitInitialGeometry(host, deadline) | Stateful FakeHost with controlled resize/termination/closed-channel sequences; one size read per wake, no child effect before success, fixed deadline and no spin. |
| RecoveryTransition(state,event) | Exhaust all state/event pairs and controlled overlapping-event sequences; termination dominates every pre-exec state, normal exit never requests shell, ExecShell requires completed cleanup. |
| runShellOnHost / runShellRecovery | Real disposable PTY plus held teardown and actual shell command acknowledgment; no shell handoff before restore/join, observed termination during cleanup prevents exec, one exec effect maximum. |
| SwitchRightTerminalTabContext | Stateful zellijtest inventory mutations and adversarial replies through actual native admission; only confirmed same-tab absence authorizes new-pane, existing terminal authorizes one targeted chord, unknown/cancelled permits no effects. |
| shared RightTerminalPaneShell | Existing layout/split conformance plus recovery invocation must all derive the same command. |

## Recovery transition authority (ARCH-ORDER)

Production RecoveryTransition in termcmd/recovery.go owns the ending; the IO owner cannot
set independent recovery flags. States are Running, CleanupNormal, CleanupFault,
CleanupTerminated, ShellReady, Finished. Events are ChildExit, WrapperFailure,
HostTermination, CleanupComplete, CleanupFailed, Handoff. Effects are Cleanup, ExecShell,
Finish; a rejected event produces no effect. This is process-local and creates no durable state.

| Current | Event | Next / effect |
|---|---|---|
| Running | ChildExit | CleanupNormal / Cleanup |
| Running | WrapperFailure | CleanupFault / Cleanup |
| Running or any cleanup or ShellReady | HostTermination | CleanupTerminated (if cleanup still owed) or Finished / never ExecShell |
| CleanupNormal or CleanupTerminated | CleanupComplete or CleanupFailed | Finished / Finish |
| CleanupFault | CleanupComplete (stdio usable) | ShellReady / none |
| CleanupFault | CleanupFailed or dead stdio | Finished / report, no exec into unusable terminal |
| ShellReady | Handoff | Finished / ExecShell exactly once |
| Finished | any event | Finished / no effect |
| all other pairs | any event | unchanged / no effect |

A normally completed child keeps normal-exit authority even if cleanup adds an error;
report cleanup errors without restarting a shell the operator just exited. Host termination
always dominates observed failure or completion before handoff. Cleanup itself remains owed
until existing input/resize producers, children, presenter and host have been joined/restored.

runShellOnHost records a real termination value as a typed termination outcome. The outer
runShellRecovery retains the same host.Terminated channel across the call. OSHost.Close
stops signal.Notify, restores normal process signal delivery, then closes that channel;
buffered values survive closure. After teardown returns, drain buffered real signal values
and pass HostTermination events before CleanupComplete/Handoff. A closed channel with no
value is cleanup completion, not evidence of termination. From signal.Stop onward the
process's normal SIGTERM/SIGHUP behavior remains active through syscall.Exec, so a later
signal cannot be silently consumed by an abandoned wrapper signal watcher. No check-then-
rearm window, second signal subscription or polling goroutine is introduced.

Controlled tests inject termination before completion, while child/presenter cleanup is held,
and at the post-close drain; they also close the channel without a signal and verify recovery
is still allowed. A native self-exec/PTY fixture verifies the handoff shell can accept a command
and termination is not transformed into a new shell. Dead terminal EIO/EPIPE and missing
usable stdio are explicit no-exec outcomes; the pane-survival contract presumes a live parent.

## Execution

### 1. Backend cells and evidence

Files: create terminal/cell_degrade.go and cell_degrade_test.go; modify endpoint.go and
observation.go; create termcmd/diagnostics.go and diagnostics_test.go; wire existing
ptychild.Options.Observer from terminalMux; update artifactpath's diagnostic-source manifest.

- [ ] Write and run red tests for the named functions above; retain semantic failure evidence.
- [ ] Implement pure repair and bounded first-fault evidence at owned backend snapshots.
- [ ] Wire existing nonblocking observation to the registered buffered diagnostic sink.
- [ ] Run terminal/termcmd/ptychild/terminalcapture/source-guard tests and commit.

### 2. Geometry and native-shell recovery

Files: termcmd/run.go, recovery.go, recovery_test.go, presentation_test.go.

- [ ] Write and run red geometry/recovery transition and real-PTY tests from the strategy table.
- [ ] Implement waitInitialGeometry and RecoveryTransition with the signal/cleanup handoff above.
- [ ] Retain the existing mid-session zero-size ignore and qualify it through real input/output.
- [ ] Run targeted terminal/termcmd/hostty/ptychild checks and commit.

### 3. Missing terminal shortcut and qualification

Files: layoutcmd/layoutcmd.go and tests; shared workbenchshortcut command; termcmd/run.go;
README.md, atlas/terminal.md and the relevant atlas shortcut reference.

- [ ] Write and run red stateful-fake recovery routing tests from the strategy table.
- [ ] Implement confirmed-absence creation and shared pane command; qualify existing nvim/agent wiring.
- [ ] Update README/atlas for diagnostics, waiting, shell fallback and recovery key.
- [ ] Run affected checks, commit/push, open PR and recorded full suite; close through SDLC review.
- [ ] Land on deterministic tests per hub approval, then operator's next natural relaunch owns live acceptance.
- [ ] On SHIP, sdlc merge --yes; verify merge/clean slot, report hub and stop.

## Alternatives

Weakening validateCell or forwarding raw bytes risks terminal controls and is rejected.
Blanket presenter failure hides a recoverable glyph error and destroys the app. Restarting
the wrapper recursively can loop on the same trigger; one plain-shell handoff is simpler.
Changing Zellij's global binding is unnecessary: shared layout routing already owns the
key and contains the explicit no-terminal no-op. A minimal default geometry could start
an app with fabricated dimensions; bounded wait followed by a usable shell keeps uncertainty visible.

## Revisions

### 2026-10-20 — hub-approved rollout

No pair:0 staging: it holds #722. The final execution smoke step is superseded by
landing on deterministic tests, a full recorded suite and SHIP review. The operator's
next natural relaunch on rebuilt main owns live acceptance. Do not drive their pane.
Pin full flow before code; the combined implementation is expected to exceed 100
added production lines despite the compact design.

### 2026-10-20 — PQ-1/PQ-2 refinements

Replaced repetitive prose test inventories with named-function adversarial strategies,
including generated row safety/column invariants. Added one production recovery transition
owner and exact termination precedence through teardown, closed-channel/value distinction,
post-close signal drain and normal signal delivery across exec. These refine the approved
cleanup-before-shell contract; no new visible state, retry, watcher or authority store.
