---
id: 000636
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'eece2d2ab5fae8255b0c537f5b18267d1894c0c5' # card fields mirrored from issue-cards; edit via sdlc
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
