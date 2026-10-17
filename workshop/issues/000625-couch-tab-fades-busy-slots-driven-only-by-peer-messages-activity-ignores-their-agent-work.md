---
id: 000625
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '7efb2347b777a7845e6355c95db0dc8acadd736f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T19:17:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "fb81ef39", done: "61ec02af"}
---

# Couch tab fades busy slots driven only by peer messages (activity ignores their agent work)

## Problem

Couch's tab bar fades a slot's chip when the slot looks idle for a day or more (`couchtty/reserve.go:370-390`,
`idle_shade.go:16-39`), from `MenuState.Activity[thread]` (`console_presentation.go:37-39`, merged at `menu.go:461`):
"each thread's last operator input or agent work". On 2026-10-16 the operator saw `ariadne:2` faded while it was
actively working (running ariadne#412's recorded suite after a day of peer-message-driven turns). Slots driven only by
the TL's peer messages apparently never register "agent work", so they fade even when busy: the same blind spot pair#618
found for codex binding confirmation (only Pair-log turns written by the draft pane count).

## Spec

- Find what feeds `Activity` today and confirm the hypothesis (peer-message turns, background shells and agent output
  don't count).
- Activity must reflect agent work however the turn started: a turn in progress, a completed turn (peer message, draft
  pane, launch prompt), or a running background command the agent is waiting on. Reuse the signal Couch already has for
  "turn open" (the busy check behind relaunch's refusal) rather than adding a new probe.
- Fading then means what the operator reads it as: nobody (operator or agent) has done anything there for a day.

## Done when

- Tests: a slot whose only turns come from peer messages stays bright; a slot idle for over a day fades.
- Live: a working, TL-driven slot is not faded. Per TL approval, this acceptance runs after landing in the coordinated restart; record executable steps before close and leave the live result explicitly pending.

## Plan

- [x] Retain last-work time on the existing admitted-wrapper session; combine it with file activity, treating known-unsettled as current work. No new probe/timer or protocol; legacy remains unchanged.
- [ ] Red then green: peer-only busy/completion, silent command wait beyond a day, settled idle/refresh, replacement/disconnect and exact scope; run affected tests and recorded full suite, then normal close/landing.
- [x] Record live restart check for TL/operator: on the new Couch, send work only through a peer message to an unfocused slot with old file activity; confirm its tab and switcher stay bright while busy, including a silent command wait, and after completion. Compare an idle-over-a-day control. Live result is post-landing, owned by TL/operator.

## Log

### 2026-10-16

Claimed and started from main. `weave refresh` catch-up reported unsupported --defer-settings; start-plan explicitly allowed current checkouts (missing changes recent/docs-only). No substrate edits. ARCH-DRY: activity currently uses threadactivity.Latest (Pair-log sends, pane birth, established native activity); the UI never reads wrapper Settled. Reuse the existing authenticated admitted session and its Settled signal used by relaunch. It includes open turns, occupied composers/pickers and recent IO, not a distinct background-command count. Keep the current file sources and legacy behavior; completed work remains recent through a session-scoped timestamp, while idle refreshes cannot advance it. Unknown/replaced/disconnected sessions supply no borrowed activity.

TL approved this quick plan in delivery f7100a95-50db-4297-917a-9362a2c5b506: no pair:0 staging, land on suite and review; TL/operator own the post-landing restart and live check above. The issue's live criterion remains outstanding until their result; no pre-landing live claim.

Red evidence: with only the new API fields/scaffold present, registry test failed because peer-only last work was zero and the fade adapter returned the old four-day timestamp for busy/completed turns. After implementation, the Console regression passes; a disposable Go overlay disabling only the wrapper merge fails specifically waiting for the peer-only chip to brighten (10.02s under a 20s test deadline). Evidence overlay: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair625-red.7xg4my/overlay.json. An initial mutation invocation used the default 10m deadline and was deliberately interrupted; that interrupted attempt is not the RED proof. No worktree source was reverted or stashed.

Live steps for TL/operator after restart (pending): verify the running Couch build includes #625; use an unfocused existing slot whose file activity is older than a day (the reported ariadne:2 is a candidate), keeping its connected wrapper so launch evidence does not make this a vacuous fresh-slot check. Send a peer-only request to run a silent 90s command and then report completion. Observe its tab and switcher within the existing 60s refresh while busy and after completion; both must be unfaded. Compare an actually idle-over-a-day slot, which must remain faded. Do not touch live artifact mtimes to manufacture age. If no old-file control exists after restart, report that live gap rather than calling a fresh-slot check conclusive.

Affected package run PASS: couchmessage 11.123s, couchcmd 48.119s, couchtty 37.909s, threadactivity 0.621s, artifactpath 6.220s. Focused race checks PASS for registry work/session lifetime, wrapper activity/transport, Console activity and switcher fade. Positive Console regression includes a failed native lookup; obsolete/disconnected/legacy/scoped foreign evidence cannot brighten the idle control. README and atlas describe the broad Settled semantics and unchanged 60s schedule. Full suite and normal close remain pending.

## Revisions

### 2026-10-16 19:20 PDT — live acceptance timing

TL reserves pair:0 as the restart binary. Suite plus normal review gates landing; TL/operator run the recorded live fade check after the combined restart. This changes timing, not the required visible behavior.
