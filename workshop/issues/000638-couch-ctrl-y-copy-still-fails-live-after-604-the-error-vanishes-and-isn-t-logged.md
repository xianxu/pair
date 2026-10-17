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

- Typed native tool/exit/bounded-stderr evidence survives through the production Console path and one content-free clipboard_copy supervision event.
- Failed or unconfirmed CopyFeedback persists until explicit dismissal or successful retry; stateful fake tests prove retry, payload retention and menu transitions.
- An opt-in real Console clipboard conformance harness is available; the normal suite never touches the real clipboard.
- Recorded full suite and close review pass. No pair:0 staging. The historical copy failure is not yet fixed: after landing, the next natural Couch restart carries the operator's Ctrl+Y check (real paste, or measured logged cause filed as a follow-up), owned by the hub.

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

## Revisions

### 2026-10-17 — TL-approved instrumentation-first acceptance

Receipts dd9c7097-9be3-4fca-9cc3-c7046cde424a and 00b6f3ff-12f2-4f3e-8407-9da4c629a989 approve the full-flow single-pass plan and explicitly change acceptance sequencing. Close and land on tested instrumentation, persistent CopyFeedback and the opt-in conformance harness. Do not stage on pair:0. The historical clipboard failure remains unknown and is not claimed fixed. The hub holds the actual Ctrl+Y result for the next natural restart: either a real paste, or the logged measured cause filed as a follow-up. This supersedes the pre-landing live-check steps above without guessing a mechanism or changing the 1s/OSC policy.
