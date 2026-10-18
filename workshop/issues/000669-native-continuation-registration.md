---
id: 000669
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1ef404e0372beefde8aefe2f01da8aa19e790834' # card fields mirrored from issue-cards; edit via sdlc
---

# Native continuation fixture times out during registration

## Problem

TestContinuationZellijSeedTransportLive fails its original 25-second registration context on unchanged main as well as #665 M2. This blocks native acceptance independently of the M2 readiness-sleep repairs. Retain the distinction between fresh ready-record admission and later ownership admission; the two retained failures do not prove identical internal causes.

## Spec

Diagnose the native continuation fixture's complete registration contract with installed Zellij 0.45.1 and disposable roots. Preserve the existing context/deadline and deterministic readiness/ownership semantics. Do not use retries, sleeps or operator sessions as substitutes for evidence. Obtain the hub's native-lane permission before live runs.

Evidence:
- Unchanged origin/main db5ddb4e2bf5af6ec85c3c916e51b17523c12085, clean detached temporary worktree, same outer private-root script: FAIL package25.902s, registration24.235s, no later ownership-poll diagnostic. Log /tmp/pair665-main-native-TestContinuationZellijSeedTransportLive.log; head /tmp/pair665-main-native-head.txt.
- #665 M2 native fixture source at64b2c6873d10bf3a0baafd725a71cffaaf3c52ff: FAIL package25.857s, ownership polling23.952s/2385polls, last check not owned. Log /tmp/pair665-r11-native-TestContinuationZellijSeedTransportLive.log; head /tmp/pair665-r11-native-head.txt.
- Both top-level failures: launch failed during registration, context deadline exceeded at original25s; source's separate2m registration allowance is capped by this parent context. Both one-shot, no retry; no operator sessions/focus/build touched.
- M2's generated-script regression first reproduced a definite older env-i gap: ZELLIJ_SESSION_NAME was erased although ReadyRecord requires it. M2 now carries the owned target explicitly; harmless actual-helper subprocess normal/race pass. That fix did not establish complete native registration.
- Source launch_existing.go runs awaitFreshRegistration followed by awaitResumeRegistration for a Fresh launch. Candidate ownership-poll text comes from the second phase, proving its fresh reader returned success; baseline's earlier phase remains unclassified from retained output alone. The continuation fixture supplies real OSReadyReader.Registered but retains its fake Couch.Artifacts checker. Do not describe the NO Pair session is live suffix as a native session query.
- Normal fixture cleanup removed child directories; retained output lacks child environment/pane logs. Do not infer child behavior from that loss.

## Done when

- Reproduce with retained diagnostics that distinguish the actual native child readiness, exact process/session ownership and fixture artifact view.
- Fix the confirmed cause with a deterministic regression and unchanged bounds, or document any separate fixture defects accurately.
- Qualify the complete continuation parent once under a newly granted private native lane; normal and race regression evidence retained.

## Plan

- [ ]

## Log

2026-10-18: filed from #665 M2 under hub3df07941 and7f5795c6: baseline failure is pre-existing, M2 proceeds on deterministic evidence, skips this native continuation with this follow-up cited. Native lanes released after each completed attempt; no fourth mechanism patch in M2.
