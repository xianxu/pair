---
id: 000727
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '990ab260516b760fc7ca72dfb649d90f6256c2f1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T08:47:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "4bfa0616", done: "07e5e5d8"}
---

# term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died

## Problem

10-20 ~09:00, the operator's ops:0 right pane (nvim on the TL session file, launched as `v teams/devinfra/tl-status-xian-xu.md`
through pair's `term` wrapper, supervisor on the staged pair:0 build f3c597f8) crashed, and relaunching it failed:

- `term: terminal: unsafe cell 12` then `terminal: presenter unavailable` (screenshot: ops
  `teams/devinfra/assets/tl-status-xian-xu/2026-10-20-term-unsafe-cell.png`);
- `term: terminal: invalid geometry 0x0 (maximum 262144 cells)` (screenshot `…/2026-10-20-term-geometry-0x0.png`).

`unsafe cell` is `validateCell` (cmd/internal/terminal/frame.go:119): a cell whose content isn't plain text (a control
rune or invalid UTF-8), exceeds MaxClusterBytes, or has an invalid link. The session file itself has no control
characters (checked), so the offending cell comes from nvim's rendered screen (sign column, virtual text, an OSC 8
link, or a grapheme cluster). One bad cell ends the whole presenter, so the operator loses their pane; and a 0x0
geometry at launch (pane measured before it had a size) refuses outright instead of waiting for a real size.

## Spec

- An unsafe cell degrades that cell (render a replacement character, log once with the codepoints), never kills the
  presenter. The safety intent (no control sequences reach the parent terminal) is kept by the replacement.
- A 0x0 or transient geometry at launch waits briefly for a real size (or uses a minimal default) instead of exiting.
- Capture the offending cell's content in the diagnostic so the trigger is identifiable.

## Done when

- A frame with a control rune in one cell renders with a replacement and the presenter stays up (test).
- Launch with a 0x0 geometry that becomes 80x24 shortly after succeeds (test).
- The operator's `v teams/devinfra/tl-status-xian-xu.md` opens again (live).

- A mid-session resize with either dimension zero preserves the last good geometry, child, presenter and input (test).
- An unexpected wrapper error restores terminal ownership and leaves an interactive plain shell in the same pane; normal exit and host termination do not restart a shell (tests).
- Alt+Shift+T from a surviving Pair-aware pane recreates a missing right terminal after confirmed same-tab inventory; unknown inventory never creates a pane (tests and live).

## Plan

- [ ] Execute [the single-pass plan](../plans/000727-term-wrapper-one-unsafe-cell-kills-the-presenter-unsafe-cell-12-and-a-0x0-launch-geometry-exits-the-operator-s-right-pane-died-plan.md): safe backend-cell projection and bounded diagnostics; geometry/recovery lifecycle; missing-terminal shortcut recovery.
- [ ] Run the recorded full suite, close review, coordinated operator acceptance, then land through SDLC.

## Log

### 2026-10-20


## Revisions

### 2026-10-20 — include pane survival and recovery (hub updates)

The operator reported that wrapper exit removed the Zellij pane, leaving no shell, and
Alt+Shift+T could not recreate it. Add the three Done-when rows above. Preserve the
last valid geometry during zero-sized resizes; wait up to five seconds for a nonzero
launch size, then fall back to a plain shell rather than kill the pane. Unexpected
wrapper failure restores/joins all wrapper ownership before replacing this process
with the configured interactive shell. Normal child exit and terminal termination
remain exits. Alt+Shift+T confirms same-tab inventory and creates one right terminal
only on positively observed absence; malformed/empty replies remain refusal.
The original cell's precise trigger is unknown: retain bounded codepoints/invalid-byte
hex from the first degradation per endpoint, not whole frames or raw link addresses.

### 2026-10-20 — investigation checkpoint

- Claimed/start-plan in pair:7. start-plan's refresh generated the two pending #699 CI
  workflows; inspected and reverted only those generated changes before retrying.
- Git comparison e712d8335f3e77a0694d6bd8869f5e3bf69b033f to
  f3c597f8dab1955b8b81c304e0cf2e99b935071b has no changes under terminal,
  termcmd, hostty, ptychild or layoutcmd. #722 changes Couch drain/status rows.
  Restart timing is not evidence that #722 introduced this renderer problem.
- runShellOnHost samples launch size once. inheritSize already ignores zero dimensions
  mid-session. Presenter/render validation is strict and a failed publication ends the
  wrapper, whose deferred closeAll closes the child. Both pane launch commands exec
  pair term, so returning from it leaves no shell. Screenshots confirm these messages,
  but contain no offending-cell payload.
- Shared SwitchRightTerminalTabContext explicitly returns nil when no terminal exists.
  New-terminal recovery must use confirmed inventory, including when stale pane IDs
  remain in sidecars, rather than interpret a missing registration as absence.
- A read-only internal subagent independently traced geometry, teardown and shortcut
  paths; no live process, native pane or pair:0 was touched.

### 2026-10-20 — approved rollout and full-flow checkpoint

Hub approved the full design (delivery 42487af6-bd3c-4a38-91f3-2ef1cdf804d0).
No staging: pair:0 retains #722 for its live drain check. Land on deterministic
tests/suite/review, then the operator's next relaunch on rebuilt main is live
acceptance, with diagnostic evidence to identify the still-unknown original cell.
This supersedes the plan's coordinated-staging step. Initial change-code inferred
quick from design length alone; pin full before code because the planned combined
production changes exceed the 100-line shell. Canonicalized the durable plan filename.
