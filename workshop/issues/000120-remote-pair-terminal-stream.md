---
id: 000120
status: open
deps: ["#121"]
github_issue:
created: 2026-07-26
updated: 2026-07-26
estimate_hours:
---

# Remote Pair terminal stream

## Problem

After remote lifecycle control works, Pair still cannot be operated remotely as a
live coding surface. The local terminal UI is visible only inside zellij, so a
tablet or browser cannot watch agent output stream, send prompts, or recover
orientation without attaching to the local terminal.

Pair already captures PTY output through `pair-wrap` and can replay scrollback
from raw bytes plus resize events. The next step is to expose that stream to a
browser without changing the local zellij workflow.

## Spec

- Extend the remote daemon/control-plane protocol from #121 with live terminal
  stream subscriptions and input messages for a selected session.
- Reuse `pair-wrap` as the PTY observation point and existing scrollback raw/event
  artifacts for replay (`ARCH-DRY`). Do not add a second terminal capture path.
- Render terminal output in the browser with a proven terminal renderer such as
  xterm.js, fed by Pair's PTY byte stream and resize metadata.
- Support a read-only stream mode first, then a restricted input path for sending
  prompt text and the submit gesture to the active agent.
- Preserve Pair key semantics deliberately: newline vs submit, overlay/picker
  behavior, paste handling, and interrupt keys must either work through the same
  mapping tables as `pair-wrap` or be explicitly out of scope for the first
  interactive pass.
- Keep arbitrary shell access out of scope. The browser controls only the
  Pair-managed agent session selected through the authenticated control plane.
- Keep rendering and routing logic factored so terminal replay/serialization is
  testable without a live relay, browser, or zellij (`ARCH-PURE`).

## Done when

- A browser can attach to a selected Pair session and see live agent output with
  usable ANSI rendering.
- The browser can reconnect and replay recent scrollback from existing Pair
  artifacts before following the live tail.
- A minimal input box can send text to the selected agent through Pair's existing
  input path.
- Tests cover stream framing, resize/replay behavior, reconnect behavior, and
  input authorization/routing.

## Plan

- [ ] Define stream/replay protocol messages on top of #121.
- [ ] Add daemon-side adapters for live `pair-wrap` output and existing
  scrollback raw/events.
- [ ] Build a browser terminal view using a terminal renderer.
- [ ] Add restricted input routing for prompt text and submit.
- [ ] Verify reconnect/replay and resize behavior locally.

## Log

### 2026-07-26

Seeded as the second remote Pair layer. This depends on #121 so terminal access
inherits the same pairing, authorization, and relay trust boundary.
