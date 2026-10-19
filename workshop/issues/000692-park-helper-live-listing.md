---
id: 000692
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: '71272f7b3bd2a9c95e110c4e4a658403ec0b29be' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T05:14:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6e340b9c", done: "fa34bba1"}
actual_hours: 0.21
---

# Native park helper refuses a live session before attachment

## Problem

macOS conformance run39201615767, job117895590257, attempt1 on f479a6e80c2849d687539c6c4377d31b0fdc12c5 (#687 PR447) failed TestParkLifecycleLiveIntentOnlyMutation before attachment: park_lifecycle_live_test.go:255 reports helper exit3; its PTY says park helper attach returned code=1 error=terminal "pair-park-5220-mut" is not listed live. The test ended after0.62s, before its750ms intent-only observation phase. Runner loadavg16.55/26.44/14.88,3CPUs. No timeout increase or product retry proposed.

CI: https://github.com/xianxu/pair/actions/runs/39201615767/job/117895590257
Retained complete failed job log: /Users/xianxu/workspace/pair/.git/pair-687-evidence/ci-39201615767-attempt1.log
SHA2562245c9b554b27c8e95fc208d69f5252ef603d20c8715d090a7963f762d0fa088. No uploaded artifacts in that run.

Retained recovery capture has12records: park session start returns nil at04:54:30.017792Z. Parent list-clients repeatedly returns client1 on terminal_1 between04:54:30.077899Z and04:54:30.259Z; one intervening query at04:54:30.209798Z exits0 with empty output and is correctly reported as no reply. Child helper's own list-sessions fence output is not captured, so absence versus exited state in that listing remains unknown. The matching refusal comes from launcher.requireListedLiveTerminal (session_handoff_os.go), not retention or the test's completion timer.

Same production code passed this CI family in earlier run39201203710 (3m11s) on230a100ed9601fbb70209de6ad677f19ea8af004; the intervening #687 change only replaces a pure predicate test's fixture and updates process docs. Earlier run39200274828 also passed. Classification: intermittent native park setup/live-session observation refusal, cause not yet proved. This differs from #601's explicit client-disconnect exit4, and from #660's direct reattach symptom; do not assume the upstream ID-reuse race without evidence. #636 retained recovery-query diagnostics provide the next failure seam.

## Spec

## Done when

- Attribute the helper's failed list-sessions observation while parent client evidence is live; reproduce or retain enough child query facts to choose a fix.
- Fix the proven setup/observation class with a deterministic regression, preserving the positive live-session fence.
- Retain CI attempts and distinguish pre-attachment admission failure from intent-only completion observation.

## Plan

### Attribution (2026-10-19, pair:2)

The helper attaches an existing session (`launcher.AttachExistingSession`, no Couch intent). Its only native query is the handoff Check, `requireListedLiveTerminal`, one `list-sessions --no-formatting` before the client starts; after Start the attach path issues none. Native `list-sessions` keeps a session only if its ConnStatus probe gets Connected back (#640 source audit). Zellij 0.45.1's client-ID reuse race (zellij-org/zellij#5270; fix PR 5630 merged, unreleased) misdelivers replies between connections to one server; #640 measured it natively (Connected and Exit frames going to the other request).

The retained capture shows the complementary pair at one instant: the parent's list-clients at 04:54:30.209798 to .233491 returned exit 0 with empty stdout (NoReply: it received a frame that was not its table, such as Connected), and the helper failed by .2626 with "not listed live". Our ConnStatus probe losing its Connected omits the live session. With no other session in the private socket root, native prints "No active zellij sessions found." and exits 1. `ZellijSource.runContext` normalizes exactly that to an observed empty inventory, so `sessionListingState` says absent and the fence reports "not listed live".

The concurrency comes from the fixture: `waitParkHelperAttachmentWithQuery` polls list-clients every 25 ms from the moment the helper starts, so a parent connection is almost always being opened or torn down while the helper's single fence probe runs. In production the listing poller and inventory queries can overlap an attach the same way, at a lower rate, and the same omission becomes the same misleading absence.

A local native probe (0.45.1, private socket root, serial list-clients loop against serial list-sessions, with an attached client and 3x CPU burners, 30 s) saw no omission, matching #640's low synthetic rate (about 3%) under raw parallel requests. So there is no native red test here; the deterministic reds use the stateful fakes.

### Fix

- [x] Production, classification (ARCH-ORDER: uncertainty must not collapse into absence; ARCH-DRY: the rule `OSRuntime.SessionBlocksReuse` already applies, "native discovery may silently omit a still-live exact server"). In `requireListedLiveTerminal`, a listing that omits the name while independent process evidence shows a live exact server for it (`SessionServers`, an OS process scan, not a native query) is unknown. It refuses as "terminal listing omitted a live server; attach unverified", wrapping a sentinel, never "not listed live". A failed server scan is also unknown. The positive fence is unchanged: only a listing that names the session live admits; there is no retry and no new native query. Deterministic red: fake listing empty plus fake live server gives the unknown error (today "not listed live"); listing empty plus no server keeps "not listed live"; a listed-live session still admits.
- [x] Fixture ordering. The readiness loop must not open native connections while the helper's admission fence runs. It first waits on OS evidence that the helper passed Check and started its client (a `zellij` child process of the helper, through the process table), and only then confirms the client count. Deterministic red: with injected process and query seams, no client query is issued before the attach child is observed (today the first query is immediate).
- [x] Diagnostics: the helper failure keeps the PTY line; a refusal now names which class it is (absent, or listing omitted a live server), so a CI rerun distinguishes pre-attachment admission from intent-only completion observation (Done when 3).
- [x] Verify: launcher and couchcore targeted tests; the live park tests locally with PAIR_LIVE_COUCH=1; the full suite (sdlc-local/suite green on 3e204bec). The PR CI conformance job is the CI attempt; merge waits for it.

What this does not do: no retry, no timeout change, no native build. The remaining exposure is the upstream race against the attach itself (#5546-style eviction) during the post-start count confirmation, now a few queries rather than a 25 ms stream. PR 5630's release retires the class.

## Log


- 2026-10-19: closed — sdlc-local/suite green on 3e204bec (11m45s). TestListedTerminalOmissionOfALiveServerIsUnknown (red with the classification disabled) and TestParkReadinessQueriesOnlyAfterTheHelperClientStarts (red with the gate removed). Live local: TestParkLifecycleLiveIntentOnlyMutation 5/5 with real zellij 0.45.1. TestParkLifecycleLive fails locally 5/5 on "quit intent does not match observed session owner" and identically on origin/main in a clean worktree (3/3): environmental, unrelated, passes in CI conformance. Attribution from the retained capture plus upstream zellij#5270; a 30 s native probe under load saw no omission (low synthetic rate). No retry, no timeout change.; review verdict: SHIP
Filed unclaimed independently from #687; no lifecycle/session behavior fix is folded into supervisor retention registration. The issue backs one unchanged failed-job rerun under the worker flake rule. A second unrelated failure on this head stops and goes to the hub; no bypass or blind retry.

2026-10-19 (pair:2) — Built. Hub approved the plan; pair:1 adopted the classification for #695 Teardown and #684 Observe (no second one).
- `requireListedLiveTerminal` takes the server evidence (`OSRuntime.liveSessionServers`, now shared with `SessionBlocksReuse`): an omission with a live exact server refuses with `ErrListingOmittedLiveServer`, a failed scan refuses as unknown, and only a true absence keeps "not listed live". TestListedTerminalOmissionOfALiveServerIsUnknown goes red when the classification is disabled.
- The fixture gates every client-count query on `parkHelperClientStarted` (a live zellij child of the helper in the process table). TestParkReadinessQueriesOnlyAfterTheHelperClientStarts goes red with the gate removed.
- Live, local: TestParkLifecycleLiveIntentOnlyMutation passed 5/5 (`-count=5`), so the process-table evidence matches a real helper client. TestParkLifecycleLive fails locally 5/5 with "quit intent does not match observed session owner", and fails identically on origin/main in a clean worktree (3/3), so it is environmental here and unrelated; CI's conformance job has passed it.
