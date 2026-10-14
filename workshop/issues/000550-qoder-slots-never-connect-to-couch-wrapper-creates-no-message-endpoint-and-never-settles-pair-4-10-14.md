---
id: 000550
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '2747cb29a99f362dd03d0df8d5e2cfc2e8a796cc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T08:52:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Qoder slots never connect to Couch: wrapper creates no message endpoint and never settles (pair:4, 10-14)

## Problem

First Qoder trial, 2026-10-14. The operator switched idle pair:4 (tag 1-pair-6) to Qoder (`qoder --dangerously-skip-permissions --session-id …`, "Auto Model", YOLO). Qoder runs and its screen renders (`couch --peek` shows source `recording`), but the slot never joins Couch:
- `couch --actors` doesn't list pair:4; `--peek` says "live tail: unavailable: no wrapper is connected for this thread"; `--send-to` can't reach it.
- The wrapper (pid 14097, started 07:53) created **no** `wrapper-<pid>-*.sock` in `/tmp/pair-message-501/`, while every Codex/Claude wrapper has one.
- After a Couch restart (08:42, main 895c7868) the slot still didn't connect; `couch --relaunch pair:4 --confirm --same-binary --force-unknown` restarted it (new session 📁1-243) but failed "unready: … did not settle within 2m0s", still not connected.
- `agent-ready-1-pair-6-qoder.json` exists (launch_ordinal 63, pid 14123); wrap events show `start`, `child-start`, `agent-restart-request`, `proxy-defer-close`, `winsize`, no idle/settled.
Qoder's first-run folder-trust dialog also appeared (accepted at the operator's direction).

## Spec

Find why a Qoder wrapper never brings up its message endpoint / registration and never reports Settled: likely the Qoder adapter's readiness or idle detection (screen markers differ: "Type your message or @path/to/file", footer "Auto Model · ctx …"), gating endpoint creation. Fix in the Qoder adapter, with a recorded-screen test, and check the trust-dialog case (first run in an untrusted folder).

## Done when

- A Qoder slot appears in `couch --actors`, settles when idle, and receives `couch --send-to` messages; `--peek` shows source `live`.
- Recorded-screen tests cover Qoder's idle prompt and its trust dialog.

## Plan

- [ ] Freeze bounded Qoder 1.1.71 idle and 1.1.66 folder-trust repaint fixtures from pair:4's existing raw recordings, with source metadata; exclude conversation bodies and credentials.
- [ ] Add failing recorded-screen and generated-input tests for Qoder peer registration, empty versus occupied multiline composer, placeholders, trust/permission/question dialogs, slash commands, settling, and exactly one bracketed paste plus bare-CR submit.
- [ ] Admit Qoder to the receiver profiles; share its ruled-box specification between Return recognition and peer parsing (ARCH-DRY). Derive its text origin and empty cursor position from captured geometry; recognize only the captured placeholder presentation, keeping drafts occupied (ARCH-PURE).
- [ ] Exercise Qoder through the existing stateful harness fake and broker/socket delivery seam, including trust -> idle -> paste -> submit -> clear and interruption by operator input/picker (ARCH-MOCK, ARCH-ORDER). Reuse existing delivery transitions and runtime teardown; add no new lifecycle or artifacts (ARCH-FUNERAL).
- [ ] Update the Couch atlas support statement. Run targeted wrapper tests, make -k build, and the suite through sdlc local-check run --name suite against the pushed final head. Terminal parsing stays bounded by the existing snapshot dimensions and existing delivery deadlines (ARCH-CONSTRAINTS); captured external screen bytes fail closed on unknown shape and fixtures carry no secrets (ARCH-SECURE).
- [ ] With hub authorization only, relaunch pair:4 on the built selected head; prove actor registration, Settled, live peek and one harmless peer receipt. Close via the mandatory SDLC review and report the outcome to ops:0 (ARCH-PURPOSE).

## Log

### 2026-10-14

- Refresh completed on main-slot3; design base from git rev-parse HEAD: 41861430868ce99ad949ce01b1154b61ec584077. Claimed and started #550 in pair:3.
- Root cause: peer_runtime.go explicitly admits only codex and claude, returning before endpoint creation for qoder. peer_composer.go likewise has no qoder case and assumes column 2 for all input origins. Endpoint startup is not gated by observed screen readiness; Qoder lacks a receiver profile entirely.
- Live inspection: couch peek pair:4 still reports source recording, no connected wrapper, and an idle YOLO composer. Current raw stream identifies Qoder 1.1.71; parked 20261014T084549 stream preserves the first-run folder trust dialog from 1.1.66. No relaunch or input sent.

## Revisions

### 2026-10-14 — root cause established

The initial readiness-gate hypothesis resolves to missing Qoder receiver admission and peer-composer support. Add the receiver profile while retaining unknown-screen refusal and proving trust-dialog blocking; reuse the existing Return geometry and delivery lifecycle. Implementation awaits hub plan approval.
