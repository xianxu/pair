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

## Execution

### 1. Backend cells and evidence

Files: create terminal/cell_degrade.go and cell_degrade_test.go; modify endpoint.go and
observation.go; create termcmd/diagnostics.go and diagnostics_test.go; wire existing
ptychild.Options.Observer from terminalMux; update artifactpath's diagnostic-source manifest.

- [ ] Write red tests injecting bad backend cells into a real Endpoint, then use the real
  presenter and emulator oracle. Assert U+FFFD, neighboring cells/columns preserved,
  valid publication/history, presenter Ready and later keyboard delivery to child.
  Cover C0/C1/ESC/invalid bytes, overlong content/link, widths/wide pairs and primary
  history. Keep strict Frame.Validate/Render negative controls on directly supplied bad frames.
- [ ] Run targeted tests and retain semantic failures before implementation.
- [ ] Implement pure row repair, call it on owned screen/history snapshots; add first-fault
  observation and bounded managed term log. Never log while holding a file lock on a paint path.
- [ ] Verify no raw control payload/link reaches parent; fault evidence survives the actual
  JSONL encoding, is bounded and emitted once; closed/unavailable diagnostics do not kill paint.
- [ ] Run terminal, termcmd, ptychild, terminalcapture and source-guard tests; commit.

### 2. Geometry and native-shell recovery

Files: termcmd/run.go, recovery.go, recovery_test.go, presentation_test.go.

- [ ] Red: zero initial size has no child effect before a valid resize; valid resize starts
  once; repeated zero wakes do not spin; termination/closed resize channel exits and restores
  ownership. Use FakeHost plus channel readiness, never elapsed-sleep readiness.
- [ ] Pin valid→zero rows/cols→valid sequence with same endpoint identity, last good
  geometry, presenter Ready, later output and key delivered. Existing ignore should pass.
- [ ] Red: real disposable wrapper failure returns only after producer/child cleanup and
  raw restoration, then native shell accepts a command on original PTY; normal child exit,
  host termination and dead stdio never invoke recovery. Test exec failure once, no recursion.
- [ ] Implement bounded geometry wait and minimal ending classifier/native-shell exec seam.
  Session cleanup still owns every child, watcher, descriptor and presenter.
- [ ] Run targeted terminal/termcmd/hostty/ptychild checks and commit.

### 3. Missing terminal shortcut and qualification

Files: layoutcmd/layoutcmd.go and tests; shared workbenchshortcut command; termcmd/run.go;
README.md and atlas/terminal.md plus relevant atlas shortcut reference.

- [ ] Red stateful-fake tests: missing pane/new action creates one in caller's confirmed tab;
  prev/next do nothing; existing/stale-registry-but-present terminal gets one chord and no
  creation; terminal only in another tab is not selected. Empty/malformed/cancelled/missing
  caller observations produce no creation. Preserve exact-target effect guards.
- [ ] Implement the new-action absence branch using shared pane command. Keep one source
  for layout/split/recovery command, checked by existing layout conformance tests.
- [ ] Verify nvim CLI and agent shortcut paths reach that same shared action. Update README
  and atlas with degraded-cell diagnostic location, startup wait, plain-shell fallback and key.
- [ ] Run all affected packages, headless editor/shortcut and source guard checks, commit/push,
  open PR and run recorded full suite. Use SDLC close's mandatory fresh review; fix findings.
- [ ] Ask hub to coordinate staging/operator smoke: open the original status file through v,
  verify text and input survive; exercise disposable failure/zero resize and Alt+Shift+T
  recreation. Do not claim the original unknown cell has been identified from a screenshot.
- [ ] On SHIP and acceptance, sdlc merge --yes; verify merged ancestry/clean slot, report hub.

## Alternatives

Weakening validateCell or forwarding raw bytes risks terminal controls and is rejected.
Blanket presenter failure hides a recoverable glyph error and destroys the app. Restarting
the wrapper recursively can loop on the same trigger; one plain-shell handoff is simpler.
Changing Zellij's global binding is unnecessary: shared layout routing already owns the
key and contains the explicit no-terminal no-op. A minimal default geometry could start
an app with fabricated dimensions; bounded wait followed by a usable shell keeps uncertainty visible.
