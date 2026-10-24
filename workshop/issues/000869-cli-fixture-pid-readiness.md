---
id: 000869
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '0f6a21711083ed9eba135acca40ec1abeab69ea2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T08:28:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ccb7b222", done: "55fd417f"}
---

# Publish cancellation fixture PID readiness atomically

## Problem

TestCLILiveLeaderCancellationJoinsBeforeCompletion observes final marker existence between shell redirection creation and printf PID write, then parses an empty value. This races the cancellation test oracle rather than production cancellation.

TestCLILiveLeaderCancellationJoinsBeforeCompletion intermittently parses an empty PID marker. First observed in #863 recorded suite attempt0c7a9818a957d392fc43e9fb44889545 on d00c828e28d41b317b924f3bde4f2f0a622076f7 (2026-10-24; suite exit2 in18m38s). cli_group_test.go:84: strconv.Atoi: parsing "": invalid syntax. Package duration20.261s. Six other failures in that attempt are #863 stale fixtures, not this bug.

The shell uses printf redirected directly to the final marker path. Its open publishes an empty file before printf writes the PID. waitFixture waits only for os.Stat success, then the test reads once and parses it, so a valid scheduling gap reaches this exact empty-string error. This is a fixture readiness race, not evidence of a cancellation-owner defect. #863 changes none of this package; Git diff from the failed head to the repeated head is empty for cmd/internal/testcouch/agentfixture.

One unchanged focused repeat passed (package0.130s, exit0). That is a targeted repeat, not a passing whole suite on the same commit; no known-flake registration is claimed.

## Spec

Publish the complete fixture leader PID through one adjacent temporary file and rename before exec, shared by both live-leader cancellation fixtures. Keep production lifetime/group cleanup unchanged. A held-writer conformance control exposes the creation/write interval and requires final readiness absent until complete publication. ARCH-DRY shares the producer; ARCH-ORDER makes the final name the publication barrier. No runtime state or product surface changes.

Publish the PID atomically or make the readiness observation require the complete valid expected PID before cancellation. Preserve the real leader-death/join assertions and the existing bounded timeout; never fix this by sleeping or increasing a budget. Add an injected hold between marker creation and content publication so the original test fails deterministically.

## Done when

- A deterministic held-write control rejects early final-marker visibility and then observes a complete positive PID.
- Both cancellation fixtures use the same complete PID producer and still prove child exit/join before acknowledgment.
- Repeated targeted/race checks, audited reader inventory and required affected check pass; mandatory review and landing complete.

- An injected marker-open/write gap cannot cause empty or partial PID parsing.
- The test still cancels a genuinely live leader and proves joined completion.
- Original timeout and process ownership/cleanup assertions remain.

## Plan

- [x] Qualify the creation/write gap with a held writer, fix atomic publication and repeat cancellation controls.
- [x] Audit readers and launch the recorded pre-close check; submit the verified patch to SDLC review and landing gates.

- [ ]

## Log

2026-10-24: TL dispatch8443c3a7-e008-4e83-be76-433b6d4e37ff; #865 parked/claimed on separate branch. Claim/start-plan completed from main. Source census found the direct producer in cli_group_test.go and request_lifetime_test.go; sibling consumer already checks parseability. Quick plan posted to TL, proceeding unless objected. Asked TL for explicit exception to cross-repo generic lint growth for this test-only publication fix; existing held-write regression will guard the actual race.

Failed full log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/0c7a9818a957d392fc43e9fb44889545.log, SHA256 prefix13f839805c08. Exact failure: github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCLILiveLeaderCancellationJoinsBeforeCompletion. Focused repeat command: go test ./cmd/internal/testcouch/agentfixture -run '^TestCLILiveLeaderCancellationJoinsBeforeCompletion$' -count=1 -timeout=60s. Its log is retained in #863 evidence/independent-cli-marker-focused-repeat.log. No production or fixture change folded into #863; unclaimed follow-up.

### 2026-10-24 — atomic producer and deterministic evidence
- 2026-10-24: closed — BR-1 held post-rename RED fails0.220s; final builtin ready marker follows helper reaping in all3 consumers. Held writer/helper and cancellation count100 pass21.398s, racecount20 pass5.935s, agentfixture13.165s and repolint15.102s pass. Individually re-audited reader inventory and diff checks pass. No architectural surface for test-only fixture (--no-atlas). New recorded pre-close check on pushed be270eeb7 is active; pass required before landing, review may overlap per tl-worker. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

TL460aae5a-7711-4ad1-b064-8d1f864405c4 approved the shared test-only producer/held-writer control and explicit lint exception: lesson entry, no golint rule. No production code changed. A shell printf override acknowledges the opened output and waits on a release marker, exposing the redirection/create-to-write interval without scheduler assumptions; runCLI still owns child/group cancellation and joined cleanup on every test path.

Held-write RED failed deterministically: final PID readiness visible before write completed (/tmp/pair869-red.log,0.185s). Shared producer now writes adjacent private .tmp and mv publishes only after printf success, before exec. Both cancellation consumers use it. Focused three-test count30 PASS2.734s; CLI/cancellation race count10 PASS12.634s; full agentfixture package PASS13.245s. Repo lint PASS8.542s. Existing ESRCH and cleanup-uncertainty assertions still prove the owned process is joined before completion.

Reader audit before repair: cli_group_test reads only its own temporary leader/writing/release files plus existing OS process identity; added shell mv reads/writes only literal paths passed beneath t.TempDir. request_lifetime_test reads only its existing generated fixture root/command/mailbox markers; no new artifact or external repository inputs. Both remain local_or_literal_fixture. Inventory first rejected exactly those two stale certificates, then each was individually recorded and staged; make test-affected-inventory PASS. git diff --check PASS. No atlas change applies to this private test-only fixture ordering correction (ARCH-ORDER/DRY); close will acknowledge no-atlas explicitly. Next: push PR and run recorded pre-close check on exact head.

## Revisions

- 2026-10-24 — Clarify the final procedural Plan row as readiness for the binary-owned close/landing gates rather than circularly requiring those gates complete before close can run. Done-when still requires their completion. No scope or design change.

### 2026-10-24 — pushed-head check and review readiness

PR591 opened from2269764c21de3c734ef9f3a5137cea34dc06eaa7. Recorded sdlc local-check run --pre-close resolved affected, then selected the full suite conservatively for the reader certificate delta (known #853 policy); lease-3370177446 is queued in machine FIFO, preserved in this terminal. GitHub local-gates-guard/verdict and merge-check pass. Mandatory close review may overlap the queued recorded check per tl-worker; a final check pass still precedes landing, and any code correction requires a fresh check. No atlas change applies to this private test-fixture publication ordering fix.

- 2026-10-24 — BR-1/TL ab6e849c-3b4c-4553-951f-3abb463e16d6: complete PID publication must also follow helper quiescence. Keep temp+rename, then shell builtin creates payload-free .ready after mv returns/reaped; all three PID readers wait .ready before reading the complete PID. Add held post-rename helper control, preserve production cleanup strictness and original held-writer control. Existing Done-when includes reliable joined cancellation, so remains current. Family census: TestCLILiveLeaderCancellationJoinsBeforeCompletion, TestRequestCancellationAcknowledgesJoinedChild and TestCLIPIDReadinessWaitsForCompletePublication. The external mv in the first implementation introduced the race; reviewer reproduced it repeatedly.

### 2026-10-24 — BR-1 reproduction and owner cancellation

Boundary review REWORK/high raised Critical BR-1 readiness-requires-helper-quiescence. Existing three-test count30 and racecount10 host passes were insufficient. New held-helper RED failed0.220s because PID readiness was visible before the helper completed (/tmp/pair869-helper-red.log). The held subprocess creates the final PID then waits for a release marker; no guessed process signals or timing-based readiness. Shared readiness predicate now checks the payload-free builtin marker and all three consumers reuse it. Explicit PID parse failures are retained in request cancellation too. Obsolete queued suite lease-3370177446 was cancelled through sdlc local-check cancel and its owner returned interrupted; no suite actually started. Final candidate needs a new recorded check and close review.

### 2026-10-24 — BR-1 corrected verification

Focused cancellation/publication controls count100 PASS21.398s; race count20 PASS5.935s; full agentfixture package PASS13.165s and repolint PASS15.102s. Both held writer and held post-rename helper cases run in the same shared-producer conformance test. Reader re-audit confirms the only added inputs are the generated adjacent payload-free .ready marker and generated held-helper gate under t.TempDir; request consumer still reads only generated fixture inputs. Inventory rejected exactly the two changed readers; each certificate was individually re-recorded as local_or_literal_fixture, staged, and inventory passed. No new artifact authority or production process behavior. BR-1 is addressed in all three enumerated consumers and will be checked by the next mandatory close review.
