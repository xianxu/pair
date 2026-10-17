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
flow: {kind: quick, provenance: inferred, spec: "6c10da25", done: "4de037cf"}
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

Approved boundary (TL19e11f39, 2026-10-16): this issue adds the admitted wrapper's existing work state, per-session completion time and a bounded authenticated LastActivity seed on reconnect. A free-composer background-only command must not change Settled or message admission. Its independent display signal requires new state and is deferred to unclaimed follow-up #632; the broad original background-only requirement above is not claimed by #625.

## Done when

- Tests: a slot whose only turns come from peer messages stays bright; a slot idle for over a day fades.
- Live: a working, TL-driven slot is not faded. Per TL approval, this acceptance runs after landing in the coordinated restart; record executable steps before close and leave the live result explicitly pending.
- Background-only waits after the turn ends are tracked in #632 under TL approval; #625 preserves all admission behavior and verifies silent waits with a turn still open.

## Plan

- [x] Retain last-work time on the existing admitted-wrapper session plus the approved bounded reconnect seed; combine it with file activity, treating known-unsettled as current work. No new probe/timer, protocol or admission change; legacy remains unchanged.
- [x] Red then green: peer-only busy/completion, silent open-turn wait beyond a day, settled idle/refresh, replacement/disconnect and exact scope; run affected tests and recorded full suite.
- [x] Record live restart check for TL/operator: on the new Couch, send work only through a peer message to an unfocused slot with old file activity; confirm its tab and switcher stay bright while busy, including a silent command wait, and after completion. Compare an idle-over-a-day control. Live result is post-landing, owned by TL/operator.

Normal close review and PR/merge follow the verified implementation steps above; they are workflow gates, not prerequisites claimed complete before invoking close. Live checking uses an open turn awaiting its command; background-only work after the turn ends is #632.

## Log

### 2026-10-16

Claimed and started from main. `weave refresh` catch-up reported unsupported --defer-settings; start-plan explicitly allowed current checkouts (missing changes recent/docs-only). No substrate edits. ARCH-DRY: activity currently uses threadactivity.Latest (Pair-log sends, pane birth, established native activity); the UI never reads wrapper Settled. Reuse the existing authenticated admitted session and its Settled signal used by relaunch. It includes open turns, occupied composers/pickers and recent IO, not a distinct background-command count. Keep the current file sources and legacy behavior; completed work remains recent through a session-scoped timestamp, while idle refreshes cannot advance it. Unknown/replaced/disconnected sessions supply no borrowed activity.

TL approved this quick plan in delivery f7100a95-50db-4297-917a-9362a2c5b506: no pair:0 staging, land on suite and review; TL/operator own the post-landing restart and live check above. The issue's live criterion remains outstanding until their result; no pre-landing live claim.

Red evidence: with only the new API fields/scaffold present, registry test failed because peer-only last work was zero and the fade adapter returned the old four-day timestamp for busy/completed turns. After implementation, the Console regression passes; a disposable Go overlay disabling only the wrapper merge fails specifically waiting for the peer-only chip to brighten (10.02s under a 20s test deadline). Evidence overlay: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair625-red.7xg4my/overlay.json. An initial mutation invocation used the default 10m deadline and was deliberately interrupted; that interrupted attempt is not the RED proof. No worktree source was reverted or stashed.

Live steps for TL/operator after restart (pending): verify the running Couch build includes #625; use an unfocused existing slot whose file activity is older than a day (the reported ariadne:2 is a candidate), keeping its connected wrapper so launch evidence does not make this a vacuous fresh-slot check. Send a peer-only request to run a silent 90s command and then report completion. Observe its tab and switcher within the existing 60s refresh while busy and after completion; both must be unfaded. Compare an actually idle-over-a-day slot, which must remain faded. Do not touch live artifact mtimes to manufacture age. If no old-file control exists after restart, report that live gap rather than calling a fresh-slot check conclusive.

Affected package run PASS: couchmessage 11.123s, couchcmd 48.119s, couchtty 37.909s, threadactivity 0.621s, artifactpath 6.220s. Focused race checks PASS for registry work/session lifetime, wrapper activity/transport, Console activity and switcher fade. Positive Console regression includes a failed native lookup; obsolete/disconnected/legacy/scoped foreign evidence cannot brighten the idle control. README and atlas describe the broad Settled semantics and unchanged 60s schedule. Full suite and normal close remain pending.

Final-candidate full suite PASS: `sdlc local-check run --name suite` ran `make PAIR_GO_TRIMPATH=-trimpath test` on pushed9d6f1e0d39dbe6ff7ed054a05062362491e4360c, exit0 in11m20s after2681s queue wait. Saved log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/9d6f1e0d39dbe6ff7ed054a05062362491e4360c/suite.log, SHA256 c67152c6cd1018632c2a942c518268b65101071ca3d93863c01192cda35b04a8. Couchcmd50.005s,couchmessage11.353s,couchtty46.905s,threadactivity1.569s,wrapcmd54.213s; all suite commands passed. Descendants so far change README, lessons and this issue only. Normal close review and landing are next; no live acceptance result is claimed.

Final-candidate suite recovery checkpoint: pushed code9d6f1e0d39dbe6ff7ed054a05062362491e4360c includes the seed and main's #632 details. `sdlc local-check run --name suite` is running against that pinned commit in disposable worktree /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-1921126583; inflight log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/5eb417847ef220e84e6cd323c9ed6aa0/run.log. The final saved result belongs under .git/sdlc/local-checks/9d6f1e0d39dbe6ff7ed054a05062362491e4360c/suite.log. At this checkpoint it has waited over30minutes and reached position4; no tests executed/result claimed yet. Later README/checkpoint commits are docs-only. Wait for this run rather than launching a duplicate; normal close and landing remain outstanding. Post-seed focused race checks PASS: couchmessage1.271s,couchcmd1.588s,couchtty2.343s.

Read-only live trace before review found an independent acceptance gap: ariadne:2 (Claude 2.1.296, tag 1-ariadne-9) has an empty composer and a visible "1 shell still running" / "1 shell" footer while waiting on its recorded suite. The authenticated resource snapshot at 2026-10-16T19:33:16.792142-07:00 has idle_known=true, idle=true; resources_inventory requires the wrapper's Settled=true for that. CPU/rate evidence was partial, not used to infer work. Thus Settled currently misses detached background commands; the approved implementation alone cannot claim that Spec case. Submitted hub question8882b139-071b-444f-b1ce-54f853d4f9ed proposes the revision below. Code changes pause pending approval; the original pushed candidate's suite remains queued (position6 at last observation), no passing suite claimed.

## Revisions

### 2026-10-16 19:20 PDT — live acceptance timing

TL reserves pair:0 as the restart binary. Suite plus normal review gates landing; TL/operator run the recorded live fade check after the combined restart. This changes timing, not the required visible behavior.

### 2026-10-16 19:33 PDT — proposed background-wait correction (resolved below)

Reuse the already-owned terminal snapshot in the existing settle recognizer: a qualified Claude composer plus its recorded positive background-shell footer must keep Settled=false, with zero counts, body/composer quotations and unrelated footer text as negative controls. This also makes relaunch refuse a running background command. No process scan, new probe/timer or protocol. The checked-in peer/claude/2.1.296 shell-footer.raw and idle-suggestion.raw both contain a real positive shell count, so recorded-frame tests can precede code. Separately, seed the first settled report's LastWorkAt from its authenticated LastActivity (bounded by supervisor receipt time) so a supervisor reconnect retains recently completed work without inventing activity at restart. Keep session replacement isolation and settled refresh non-renewal. This changes busy admission semantics; do not implement until TL approves.

### 2026-10-16 19:36 PDT — preserve admission; split independent background display

Verified submitted TL19e11f39-5e0e-46cf-8fe8-a1fbc76d223b approves the LastActivity seed and rejects changing Settled for background commands: a free composer must still receive messages. The existing admitted snapshot has no background-work field, and detached threads need TailReader.Tail IO for their agent footer. Filed #632 unclaimed with dependency #625 for the independent background display signal and message-receipt conformance. Current work keeps admission untouched and lands the original projection plus seed. This supersedes the proposed settle-recognizer change above; no such code was written. The original queued candidate was cancelled before execution (exit130); no remaining process mentions its disposable worktree. Final candidate gets a new recorded suite. Seed tests failed on the missing recent/old/bounded-future timestamp before implementation.

Seed implementation GREEN: first settled report retains its own authenticated LastActivity capped at server receipt time; settled refreshes and client attachment cannot renew it, unknown remains unknown, and busy/completion transitions still use monotonic server time. Focused registry, service transport and Console regressions pass again. Re-ran change-code for the approved contract revision: quick flow retained. No wrapper/admission/protocol source changed. Reported the split and #632 to the hub as26feaaef-c103-4ad4-969a-9d53c68ddd56.

### 2026-10-16 — checklist boundary clarification

Separate the mandatory close/landing workflow from implementation checkboxes so invoking close does not require falsely claiming that close/landing already happened. Tests and recorded suite are complete; the actual workflow gates follow normally. The TL-approved live timing and #632 scope split are unchanged.
