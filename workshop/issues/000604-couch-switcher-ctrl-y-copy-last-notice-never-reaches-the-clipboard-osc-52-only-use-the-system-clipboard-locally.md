---
id: 000604
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '22b7204910faa5d41e6b1facc4ede86de5fe925c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T15:06:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "bfa43c96", done: "10f5608e"}
---

# Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally

## Problem

The operator (2026-10-15): "the ctrl+y to copy last notice in switcher never worked for me." Couch's console
(`couchtty/console_menu.go` `copyLastNotice`) copies through the terminal's OSC 52 clipboard escape
(`ESC ]52;c;<base64> ESC \`) and then says "Full notice copy requested; your terminal may not support it." On the
operator's setup (Ghostty, with Couch's console possibly behind a multiplexer layer) the clipboard never receives it, and
the UI can't tell. Tests only assert the escape bytes were written, never that a clipboard got them.

## Spec

- On macOS with a local console, copy with the system clipboard directly (`pbcopy`, or NSPasteboard via a tiny
  helper), and report success only when that returns OK; keep OSC 52 as the fallback for remote or unknown setups,
  labelled as unverifiable.
- Find out why OSC 52 doesn't reach Ghostty here (a multiplexer not passing it through, or Ghostty's
  `clipboard-write` setting), and note it in the atlas.
- Diagnostic scope amendment (2026-10-16; approved Revision below): the clause above is superseded by the completed read-only investigation. Record that Ghostty defaults allow OSC 52, the current Couch launch has no outer multiplexer, and the original loss layer remains unknown. Further investigation is required only if the clipboard smoke fails after the next natural restart; successful local copying does not depend on OSC 52, so this uncertainty primarily concerns the SSH fallback.

## Done when

- A test with a fake clipboard seam asserts the notice text reaches it; the operator confirms Ctrl+Y works live.
- Acceptance amendment (2026-10-16; Revisions below): the live confirmation above is deferred until the next natural Couch restart. Landing requires passing recorded verification and SHIP; do not stage on pair:0, and fix forward if the later smoke fails.

- Diagnostic acceptance amendment (2026-10-16): the bounded investigation and its limits are recorded in the atlas; a proven original loss layer is not a landing prerequisite under the approved scope.

## Plan

- [x] Red first: drive Ctrl+Y against a private fake clipboard executable; assert complete multiline retained text and repeat copies, with no OSC 52 on confirmed success.
- [x] Reuse clipcmd's command list (ARCH-DRY) via a context-aware local-copy method that distinguishes unavailable tools; preserve existing ClipboardCopy callers. Explicit SSH skips local copying. Bound the notice attempt to one second plus bounded pipe cleanup; failure falls back to Presenter.Copy without retry.
- [x] Cover command failure, unavailable tool, cancellation/timeout, remote fallback, empty history and focused-child Ctrl+Y; no tests touch the actual clipboard. Keep copy feedback out of retained history.
- [x] Update the existing atlas account with confirmed local versus unverifiable OSC results and the read-only transport/config evidence. Run the recorded full suite and close review; coordinate operator Ctrl+Y acceptance through the hub before landing.


The checked investigation and acceptance-coordination step follows the approved Revisions: bounded read-only diagnosis is complete; live smoke is deferred to the next natural restart, with further diagnosis only if that smoke fails.

## Log

### 2026-10-15

### 2026-10-16 — bounded design
- 2026-10-16: closed — Red-green full-notice clipboard regression; clipcmd and couchtty focused tests and race checks passed. Complete make -k test passed (79a1e5cb433c); recorded suite on fe4eb6d6 passed in 14m03s (46dbb4bf499d); earlier independent failures retained in unassigned #620/#623. Later changes docs only. BR-2 README corrected; BR-1 diagnostic scope explicitly narrowed by TL delivery f7d15982: bounded read-only investigation complete, original OSC loss layer unknown. Live clipboard smoke deferred by TL to next natural restart, no pair:0 staging, fix-forward if it fails.; review verdict: SHIP

Plan posted to the hub; proceeding under the bounded quick-flow dispatch. The existing method only writes OSC 52. A private executable on PATH supplies the clipboard seam, avoiding a second tool registry or test-only production API (ARCH-PURE/ARCH-DRY). One serial user action creates no durable state or background work; the subprocess is joined before returning. Timeout/failure may be uncertain, so fallback is an explicitly requested second transport, not proof that a local copy never happened; feedback claims confirmed copying only on successful exit.

Read-only investigation found installed Ghostty defaults allow clipboard-write and no local permission override; Zellij documents OSC 52 writes. The affected live chain/loss layer is not proven, so atlas must not blame either without measurement. Live acceptance remains required.

### 2026-10-16 — red/green and transport behavior

The regression TestSwitcherCopiesFullNoticeToSystemClipboard failed on the old implementation: the private clipboard file did not exist after the semantic Ctrl+Y action (0.04s). After the fix the same test passes and asserts exact multiline/Unicode bytes, two actual tool invocations, confirmed feedback and no OSC 52. Focused console success/fallback/empty/child tests pass (1.494s); the complete clipcmd package passes (0.392s), including successful stdin transfer, exit 7, missing tool, legacy optional behavior, each SSH marker, pre-cancellation and deadline termination. Tests use private executables/files and never the operator clipboard. The long-standing fallback regression explicitly hides system tools from PATH so it continues testing OSC 52 rather than modifying the real clipboard.

Targeted race verification also passed: clipcmd (1.495s) and couchtty (2.964s), covering the notice-copy and local-clipboard regressions.

Read-only current process ancestry was Couch (71982) → zsh (35159) → login (35158) → Ghostty (80193); no outer multiplexer was observed. This rules out an outer Zellij layer for the current launch, not for every historical occurrence. No OSC or real-clipboard probe was sent.

### 2026-10-16 — first recorded suite and independent fixture failure

Recorded suite on fe4eb6d6e83bd53b9d42fcf27811a383c348b222 stopped before Go tests in tests/review-branch-restore-test.sh: data/landed.json was absent after its five-second handoff wait. Failed run retained at /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log (exit 2, 5m38s, SHA256 prefix eee92d770af5; file timestamp 2026-10-16T22:18:18Z). The unchanged targeted repeat passed, logged at /tmp/pair604-review-branch-restore-repeat.log. Filed unassigned #620; no root cause or load attribution is claimed and no fixture/timeout change belongs to #604.

Running make -k PAIR_GO_TRIMPATH=-trimpath test to complete coverage rather than skip the remaining recipes; log /tmp/pair604-keepgoing-suite.log. This run uses the same code with only the later atlas/issue prose commit. Its branch-restore check has passed and Go packages are still running. The exact pushed head's issue-backed recorded rerun remains outstanding. The hub received this status (receipt 93c64a3f-fcd4-4527-aa23-32267900716b submitted). Live staging permission/acceptance remains pending; pair:0 is untouched.

The complete make -k run finished exit 0: all shell/editor and Go checks passed, including the previously failing branch-restore case, couchcore (328.985s), couchtty (48.517s), launcher (36.033s) and clipcmd (0.430s). Log /tmp/pair604-keepgoing-suite.log, SHA256 79a1e5cb433c82e9af9207831d7deb750f9b8280a826238059423d03fc5b8624. Started the one issue-backed recorded rerun explicitly on the original pushed head fe4eb6d6e83bd53b9d42fcf27811a383c348b222; no tests/production code changed. No second unrelated failure was observed.

### 2026-10-16 — second unrelated verification failure; stop for hub decision

The single issue-backed recorded rerun on fe4eb6d6e83bd53b9d42fcf27811a383c348b222 acquired the machine gate after 1239s, then failed exit 2 in 6m59s. Branch-restore (#620) passed. A different case failed: tests/review-observation-diagnostics-test.py::ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order; its nested fixture asserted `matching branch external edit did not reload; snapshot lost no-final-newline state`. Eight diagnostic cases ran, one failed. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/ffdc28f65f4aad6e6fdf5f09c7feeb01.log, SHA256 prefix 7c703e6585f2. The log retains bounded observation ordering details; root cause is not established here.

The preceding complete make -k suite passed all checks, including this diagnostic command and Go packages. Per the fleet's second-unrelated-failure rule, no further rerun, independent fixture fix or timeout change is attempted. Reported to hub with receipt cd372a9e-60f4-42b8-8070-d9cf36343f96; waiting for the hub's verification decision and live-staging authorization. No stage, real clipboard probe, close or landing has occurred.

### 2026-10-16 — authorized recorded rerun passed

The explicitly authorized additional sdlc local-check run on fe4eb6d6e83bd53b9d42fcf27811a383c348b222 completed exit 0 in 14m03s after 746s machine-queue wait. All shell/editor and Go checks passed, including branch-restore (#620), all eight observation diagnostic cases (#623), couchcore (559.722s), couchtty (65.305s), launcher (42.425s), and wrapper (82.858s). Recorded as sdlc-local/suite; log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/fe4eb6d6e83bd53b9d42fcf27811a383c348b222/suite.log, SHA256 prefix 46dbb4bf499d. All later branch changes are atlas/issue prose only. Both failed recorded attempts remain documented above and in their unclaimed issues.

Implementation/verification and acceptance coordination are complete for the revised contract; the binary-owned close review runs next. Per the hub's explicit arrangement there is no staging and no claimed live clipboard success: the paste-full-notice smoke follows the next natural Couch restart, with fix-forward.

### 2026-10-16 — close review round 1: documentation and diagnostic scope

The binary-owned quick review read cd7beba6916499e28d64600662660ecd000ea3d9 and returned REWORK, no code defect or Critical findings. BR-1 (spec-deliverable-traceability) requires either a proven OSC 52 loss diagnosis or an explicit scope revision across the Spec, checked Plan and atlas; smoke deferral alone does not waive that separate deliverable. BR-2 (user-surface-documentation) requires README to explain confirmed local versus unconfirmed OSC fallback. Updated the existing README paragraph with tool success, SSH/tool failure fallback and the one-second budget; no wording-presence test is appropriate. Added both general lessons to workshop/lessons.md. The reviewer independently passed the complete affected packages and focused race tests, and verified the recorded-suite log hash.

Requested explicit approval to narrow diagnosis to the completed bounded read-only investigation, retain the unknown loss layer, and investigate further only if the deferred natural-restart smoke fails. Message to hub was refused with missing-registration diagnostic; fresh actors showed both slots live, so used the documented exact sender fallback ops:0. Receipt b327b1c9-3e48-47eb-a41d-a688a81cc43a is submitted. Scope reduction is only a proposal until the hub answers; no new code, tests, full-suite retry, staging or landing while it is pending.

### 2026-10-16 — approved scope resolves BR-1

Canonical TL delivery f7d15982-4387-40dd-ad41-3da44432c15c approved narrowing the diagnostic deliverable to the completed read-only investigation and keeping the loss layer unknown. The Spec, Done when, checked Plan and atlas now explicitly follow that scope. BR-2 was addressed by the existing README update. No code or tests changed after the passing recorded suite; running the ordinary close again because the first REWORK did not finalize acceptance.

## Revisions

### 2026-10-16 16:02 PDT — acceptance arrangement from the hub

TL delivery 2f40f35c-5099-49e9-a8f3-c3221de005e5 (accepted 15:40, delivered after the rerun failure) defers the original live-confirmation criterion and the Plan's live-before-landing step: do not stage on pair:0. Once recorded verification passes and close review is SHIP, land normally. The operator's paste-the-full-retained-notice smoke rides the next natural Couch restart, with fix-forward if needed. This is an explicit acceptance arrangement, not evidence that live smoke passed, and it does not waive the recorded-suite gate. #619 starts only after #604 lands.

At receipt the recorded rerun had already failed on the second unrelated review fixture, as logged above. Thus its passing condition is not met; retain the stop for the hub's verification decision, with no new test rerun or gate bypass. No production or test code changes accompany this revision.

### 2026-10-16 16:03 PDT — one additional recorded attempt authorized

Canonical hub decision efadcd45-6fb6-487e-9392-7fc79782168e authorizes filing the second fixture failure and exactly one additional recorded rerun on the same head; any failure stops and is reported. Filed #623 unclaimed with its log path. The hub links likely load pressure to #618 and reported load average 140; that remains a hypothesis for the specific fixture failure. No production/test changes or timeout increases are authorized by this decision. The no-staging/natural-restart acceptance arrangement remains in force, now reflected by an appended Done-when amendment.

The first rerun invocation was refused before testing because fe4eb6d6e83bd53b9d42fcf27811a383c348b222 was no longer a remote tip after documentation-only checkpoints. Following the named push precondition, published temporary verification ref verify-604-suite-fe4eb6d6 at that exact commit, without rewriting the issue branch. The single test attempt then starts through sdlc local-check on the same commit. Retire the temporary ref once verification/landing evidence no longer needs it; it carries no independent development.

### 2026-10-16 16:37 PDT — approved bounded diagnostic scope

TL approval f7d15982-4387-40dd-ad41-3da44432c15c supersedes the original Spec demand to identify the OSC 52 loss layer and qualifies the checked Plan investigation step. The completed read-only investigation establishes Ghostty defaults permit OSC 52 and the observed Couch launch has no outer multiplexer; the actual loss layer remains unknown. Record these limits in the atlas. Further investigation is required only if the deferred clipboard smoke fails after the next natural restart. Local pbcopy success bypasses OSC 52, so the remaining transport uncertainty primarily affects SSH fallback. This explicitly changes the diagnostic deliverable; it does not claim a live smoke pass or waive recorded verification and SHIP.
