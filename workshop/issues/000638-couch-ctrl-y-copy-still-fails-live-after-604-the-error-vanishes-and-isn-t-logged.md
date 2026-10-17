---
id: 000638
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ffdfcaf1ec73a85e833a3953e3bf34c8c14251c9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T02:01:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch Ctrl+Y copy still fails live after #604; the error vanishes and isn't logged

## Problem

## Spec

pair#604 (PR #381) landed Ctrl+Y "copy last notice" in the Couch switcher using the local system clipboard. The operator's live check on 2026-10-16 at about 23:15, on Couch built at 712a6478 and running since 22:22, failed:
- pressing Ctrl+Y showed an error;
- nothing reached the clipboard;
- by the time they went back to screenshot it, the error was gone.

Nothing about it is in supervision.jsonl, so the error text is lost too.

Two defects:
1. The copy fails in the operator's real environment, even though the tests pass with a fake clipboard. Find which tool Couch picks (pbcopy?), and how it fails from the Couch process: PATH, sandboxing or the launch environment, a timeout, stdin handling.
2. A failure has to stay visible and be recorded: the notice persists until it is dismissed, and the error, with the tool and its exit status and stderr, goes to the supervision log.

The original OSC 52 loss layer is still undiagnosed (#604's close review).

## Done when

- In the operator's Couch, Ctrl+Y puts the last notice on the clipboard. Live check: paste it.
- A forced failure, such as a missing tool, shows a notice that persists and logs the tool, exit status and stderr; a test covers it.
- A live conformance check runs the real clipboard tool from a Couch-launched process environment.

## Plan

- [ ] Deliver typed clipboard diagnostics and a pure retained-copy feedback model, with red/green tests.
- [ ] Wire bounded copy outcomes to supervision.jsonl; exercise production Console wiring with stateful clipboard doubles and retained-error sequence tests.
- [ ] Obtain the actual Couch-path failure evidence through live conformance, correct the measured cause, and verify a real Ctrl+Y paste with the operator.
- [ ] Run the complete recorded suite, update README/atlas and close through SDLC.

Durable design: workshop/plans/000638-couch-clipboard-copy-plan.md.

## Log

### 2026-10-16

### 2026-10-17 — initial investigation and authorized real-tool probe

#624 landed as PR390 before this claim. start-plan refreshed the slot's ariadne copy from 585ae4f7 to 0a1f6a45 and required a rerun; the rerun created this issue branch from main. Brew reported existing go/cue/uv; no new machine packages were chosen.

Source proves three evidence defects: copyLastNotice discards the ClipboardCopyLocal error and shadows it with Presenter.Copy's error; copyClipboard discards stderr and returns no tool identity; copy feedback bypasses the supervision writer. The menu copy message has no expiry timer: ordinary operator events clear it, and inventory/bookkeeping can replace its ownerless notice. The historical event that removed the operator's error remains unknown. Presenter.Copy only emits OSC52, with no clipboard acknowledgment; a stalled-parent drop may return nil. Neither fallback success nor a new fake-backed pass proves actual clipboard delivery.

TL approved exactly one real pbcopy/pbpaste probe in receipt e82fa054-d8cb-4cf5-8b21-970e0c91884e: original bytes only in memory, no clipboard contents printed/logged, restore only while the clipboard still contains the marker. From this Couch-launched worker's non-interactive environment, /usr/bin/pbcopy and /usr/bin/pbpaste resolve and SSH markers are unset. The exit-only probe completed exit 0 in 0.077s, verified its marker and restored the previous bytes. This confirms that one worker-launched path works; it does not reproduce or explain the operator's original Couch failure. No further clipboard write, primary build, visible window or slot drive was performed.

A bounded read-only lsof check found no outer terminal capture for PID99807. A fresh-context read-only agent independently traced the lost-error and notice-clearing paths; no implementation edits were delegated. Plan approval and real Couch-path evidence are required before inventing a timeout/PATH/bootstrap correction.
