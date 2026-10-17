---
id: 000636
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '44344bdc20af013f8214e635730ad318c253276d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:14:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Warm recovery live conformance intermittently receives no Zellij query reply

## Problem

Native Zellij 0.45.1 occasionally returns no reply to a recovery fixture query while the exact owned session remains alive. The pre-existing warm live recovery test fails before fixture cleanup. This is separate from #627's daemon/cache cleanup join.

## Spec

Diagnose empty native list-clients/list-panes replies during warm recovery and make the fixture or product respect the observed readiness contract. Establish whether Zellij, the fixture's detach transition, or the query adapter causes the gap. Preserve recovery and ownership assertions and keep unknown observations distinct from absence; no blind retries or longer deadlines.

## Done when

- Reproduce the no-reply seam deterministically, classify its cause, and add a regression at that seam.
- Original observed failure cases and repeated isolated warm recovery conformance pass with every existing assertion retained.

## Plan

## Log

2026-10-16: Found during #627 qualification on Git-verified code head 8847a4c0501a8ea19daf142445f555c29a433ff2, Darwin, /opt/homebrew/bin/zellij 0.45.1. Private HOME/config/data/cache/runtime/socket roots from native-private-env.json in the scratch directory below, no overlapping live runs. Initial test-couch-zellij-live target passed all four launcher tests and park test, then warm recovery failed: observe live session panes: action list-panes --json --command: zellij query returned no reply.

The original helper (byte-for-byte verified against fc899cd5d40530707ffe2577a2e0d392c8e7dd31) and original shortcut body were restored with the original-fixture-overlay.json Go overlay. Warm-only count=10 PASS (34.375s), but the exact core park+warm selection count=10 FAIL (47.705s), one warm failure: recovery detached state could not be checked; retry: action list-clients: zellij query returned no reply. Immediate diagnostic list-panes succeeded and showed live wrapper/source panes. This proves the no-reply query family predates #627; it does not claim identical failing CLI actions. Same selection on candidate unchanged head count=10 PASS (48.288s). All controls use the same private environment and run sequentially.

Reproduction selection: PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore -run '^(TestParkLifecycleLiveIntentOnlyMutation|TestRecoveryRealHelperAndSessionConformanceLive)$' -skip '^TestRecoveryRealHelperAndSessionConformanceLive$/^checkpoint$' -count=10 -v; original adds -overlay original-fixture-overlay.json. Native empty replies map to launcher.ErrZellijNoReply (cmd/internal/launcher/zellij.go). No production/fixture diagnosis claimed yet.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-live-target.log: SHA256 89d52bd636f2c6694c47f0e3e279ae05bed9efe23d2bc49504513a69e9bd19cf
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-recovery-original.log: SHA256 362b4eeaed6676e8839b52930f1e72fab8bf21aac26932e91cf7925b37d09676
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-original.log: SHA256 a4700a0f4aada38e80878549dfd9b03be429ba55509cc1194a3e665d562ec7e8
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-candidate.log: SHA256 8c28e75af317e1a7f9064f98d35515133c65fa617252227710052a9a47cbdc46
- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-shortcut-race.log: SHA256 54f57c0bd67c9fbe90e48cda2164ca39859a0f5740e2f9ed871a10d2eb979ec9

### 2026-10-17 — Coordinated read-only audit with #640

Authenticated TL dispatch5aa2c335; claimed in pair:6. Before any implementation/native run, coordinated with pair:1 (submitted request06ba1299; authenticated replyfe007ec6). Read its committed #640 Log at Git-verified2c42749dc01932629eaa2f9d5f1d33f82041e0d4. Pair:1 owns the bounded native-trigger diagnostic and requested all other native runs held; TL's scope there is read-only investigation until about11:30, with no upstream publication or Pair transport/retries. No source/fixture code, native environment, timeout or assertions were changed for #636.

Read-only source audit finds no demonstrated #636-specific cause. ControlledZellij.KillClient (pairlifecycletest/live_zellij.go) joins the local client process but has no server detach/RemoveClient acknowledgement. Warm recovery then proves exact live ownership through list-panes and detached state through separate listings/list-clients (couchcore/recovery_conformance_live_test.go, recovery_execute.go, artifactcollision.go). Local calls are sequential; queued native work can overlap later CLI clients. This is compatible with #640's candidate race, not independent proof of its trigger (ARCH-ORDER).

Production launcher.ZellijSource rejects empty/whitespace successful stdout as ErrZellijNoReply before parsing. Header-only list-clients is a received zero-client response; exact no-sessions exit1 diagnostic is the separate absence proof. SessionOwnerProbe preserves unknown on empty pane responses and checks exact server identity around ownership evidence. Recovery observer errors return before mutation/launch; existing empty-reply stateful-fake and recovery refusal regressions cover those boundaries. No safe cause-level Pair change is justified by the present evidence (ARCH-PURPOSE).

Retained original failure log still exists and its SHA256 matches the filed value: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-original.log, a4700a0f4aada38e80878549dfd9b03be429ba55509cc1194a3e665d562ec7e8. Its failed warm list-clients is followed by a successful same-session pane diagnostic. That later observation proves the session subsequently answers, not what bytes the failed query received.

Pair:1's /tmp/pair640-mi9l7uyn/protocol-control.jsonl was inspected read-only (SHA256e8de734728e0dabe1e7dfd59ab1ad2cf1dd18cad4edd9135d12dd4edf348acef). It records installed0.45.1 CLI: Connected then JSON Log -> exit0/[]; Unblock before Connected -> exit1/no-active-session; query Unblock before JSON Log -> exit0/empty, with the private listener alive in all cases. These deterministically establish the client failure shapes; the native stale-ID/double-RemoveClient trigger remains unproven. Transparent proxy runs perturb scheduling, so their passes are not acceptance proof. Pair:1's raw native diagnostic is the shared blocker; do not duplicate its native work or weaken unknown handling.

Independent source-level fixture weakness filed unclaimed as #645: park_lifecycle_live_test.go zellijClientCount accepts empty stdout as0 and counts the header; baseline0 followed by header-only count1 can falsely prove helper attachment. No causal connection to #636/#640 is claimed and no fixture edits were made.

If more #636 evidence is needed after native coordination, use the existing WithZellijQueryObserver for bounded in-memory argv/exit/stdout/stderr/timing capture, retaining fixture client PID/start and PTY-close/signal/wait timestamps. Flush after the run: synchronous observer IO or extra queries/proxies can alter scheduling. Do not add retries or infer detachment/absence from silence. Implementation planning remains pending shared native-cause classification and hub scope guidance.
