---
id: 000550
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'f9c0150640714b62f16bc9b17e386b35bd7fa273' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-14
