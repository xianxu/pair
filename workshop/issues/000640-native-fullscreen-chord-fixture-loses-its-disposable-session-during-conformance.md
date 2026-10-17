---
id: 000640
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5ce80c5ae0b92f2b93053dae03deae2f27acf9b5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T09:08:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Native fullscreen chord fixture loses its disposable session during conformance

## Problem

Native TestFullscreenChordZellijLive intermittently loses its disposable Zellij session: list-panes --json --all returns exit 1, "There is no active session!". Seen while qualifying #639, before any toggle on the candidate attempt; reproduced later during the original three-state scenario with the original layoutcmd code and test body. Root cause is unknown. This is a session-loss symptom, distinct from #636's ErrZellijNoReply.

## Spec

Investigate the native fixture/server exit and make the actual-chord conformance stable without sleeps, longer deadlines or hiding failed queries. Preserve all input, geometry and focus assertions.

## Done when

- A deterministic reproduction identifies why the disposable session disappears.
- The fixture/product cause is fixed with a regression and native conformance proof.

## Plan

- [ ] Capture an isolated native failure and server/client diagnostics; identify a deterministic trigger.
- [ ] Add a deterministic regression and minimal cause-level fix while preserving every chord/input/geometry/focus assertion.
- [ ] Verify native conformance and the recorded pre-close suite, then close and let the landing turn integrate main.

## Log

### 2026-10-17 — #639 qualification controls

Candidate first run: PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run '^(TestFullscreenZellijLive|TestFullscreenChordZellijLive)$' -count=1 -v (the first named test does not exist; only chord ran). Failed after 11.18s at fullscreen_chord_live_test.go:149, before toggle input, list-panes reports no active session.

Original control source: 1ac101a17404a392927a4dac83676b43e6030b9a. All tracked layoutcmd Go files restored via go test/go build overlay from this Git commit. Build original-pair with go build -overlay ORIGINAL.json -o original-pair ./cmd/pair-go, then PAIR_TEST_BINARY=original-pair PAIR_LIVE_ZELLIJ=1 go test -overlay ORIGINAL.json ./cmd/internal/layoutcmd -run '^TestFullscreenChordZellijLive$' -count=1 -v. This passed draft-nvim and agent-wrapper scenarios, then failed after 4.22s during terminal-shell-bottom at list-panes (line308), same no-active-session diagnostic; restore-tiling current-tab-info exit1 also logged. The precise failing phase differs, so this establishes only the pre-existing session-loss family. Workbench help text on disk was newer; original planner, runtime, test body and supplied binary's layout code came from the overlay. No live tests overlapped these controls.

Local evidence root /tmp/pair639-7xyqf5rg; original.json and original-*.go preserve the exact source; original-build.log, environment.json and original-chord-correct-binary.log retained. Original binary SHA256 6b99f66417c5204867841b112751d920e7754b21d24a482d17dda0dc7bb736bf; control log SHA256 225feb439e35694e73558be8dc20488af695c9437f8a63fac3473229b8b214f4.

An earlier original-test-only run used a candidate Pair binary, so it is excluded as a baseline. A separate direct-conformance invocation inherited Couch's long ZELLIJ_SOCKET_DIR and was refused before render (session name must be less than 0 characters); short private socket roots are required for that fixture and this is not the session-loss reproduction. Candidate Focus-return errors (already-focused exit2 and skipped same-pane return-focus) are #639's defects and are being corrected there, not assigned to this issue.

### 2026-10-17 — resumed diagnosis

Fresh dispatch claimed on main 30e3f624ccf13df378f2dac5c5c036ebdc0015ce; current native Zellij is 0.45.1. The retained original-pair control log still contains both no-active-session and restore-tiling/current-tab-info failure. Root cause remains unknown. A bounded read-only upstream audit confirms ordinary CLI actions send ClientExited after their response; query completion does not prove server-side disconnect cleanup completed. Cached screen.rs also has a last-selectable-tiled-pane close path, but neither that path nor the startup RemoveClient race is yet established as the cause. Server routing/lib.rs and native fatal diagnostics are missing from the existing evidence.

The chord fixture clears every inherited ZELLIJ-prefixed variable (including an externally supplied socket root), then resets TMPDIR. Diagnostic controls will exec the unchanged real binary through a private-environment wrapper, giving native processes short owned socket/log roots without masking exit codes or failed queries. Preserve native logs outside testing.TempDir cleanup. ARCH-ORDER: observe the exact failure ordering before choosing a readiness change. ARCH-SECURE/FUNERAL: diagnostic sessions and files belong to the private fixture; no operator session or :0 binary is touched.

### 2026-10-17 — isolated diagnostic controls

Evidence root /tmp/pair640-mi9l7uyn holds current Pair/test binaries, a sterile native-env.json produced by tests/lib/pair-test-env.sh, and an exec-only zellij wrapper fixing short private native TMPDIR/socket roots. Tests run from cmd/internal/layoutcmd. An initial invocation from the repo root failed locating zellij/config.kdl before session startup; harness-wrong-cwd.log is excluded.

baseline-1.log: unchanged chord test fails after 16.03 s restoring the terminal-shell-top geometry; current-tab-info exit1 was logged, but the private native log shows a live initialized server until fixture cleanup, without a native panic. This is distinct from proven session disappearance. A logging-only overlay of execZellijContext preserves each invocation/output/error and records argv, effective native executable/session/socket root and ExitError stderr. baseline-trace-1.log fails after 7.57 s in terminal-shell-top on unexpected end of JSON input. query-trace.log identifies a list-panes --json --command --state --geometry command returning success with empty stdout; all traced calls resolve the intended wrapper and private socket/session, excluding wrapper bypass. No traced command has an exit error in this run. Neither run yet reproduces the original no-active-session diagnostic; root cause remains open. No production/test implementation changed and no failed query was retried or hidden.

Exact native v0.45.1 source audit (copies retained here: server-lib.rs, server-route.rs, commands.rs, sessions.rs) distinguishes discovery failure from server exit: no-active-session is emitted before attachment when the session inventory is empty. Inventory filters sockets through ConnStatus probes; refused connections unlink their socket paths. Startup RemoveClient can unwrap absent metadata, but neither that race nor a socket unlink has been observed here. A possible ConnStatus duplicate-removal/client-ID-reuse ordering is an unproven hypothesis, not a fix justification. ARCH-ORDER: preserve separate failure observations until a deterministic mechanism connects them.

### 2026-10-17 — false absence and protocol controls

baseline-trace-2.log reproduces Pair current-tab-info exit1 with exact stderr "There is no active session!" at 09:25:44.494; the observer subsequently succeeds on list-panes through the same session/socket root until teardown. This disproves treating the message as server death for this run. Its final failure is restored geometry (13.11 s). The observer additionally received two successful empty list-panes replies and an empty list-clients reply. All failures remain retained; nothing was retried within an observation.

A transparent frame-logging socket proxy (wire-proxy.py, diagnostic test overlay wire-overlay.json) changes scheduling: wire-run-1.log and wire-run-2.log each pass, followed by all ten in wire-count10.log (3.30–3.68 s). These are diagnostic perturbation controls, not fixes or acceptance evidence. wire-trace.jsonl retains native frames; none of those passing runs captured the bad probe ordering. Proxy/client/server resources were private and joined on fixture cleanup.

protocol-control.py drives the installed, unchanged Zellij 0.45.1 CLI against an owned live Unix-socket protocol fake, using separate length-prefixed protobuf frames. Connected then a JSON Log yields exit0 / [] JSON. UnblockInputThread before Connected makes the real CLI return exit1 / "There is no active session!" despite the listener remaining alive. Connected followed by query UnblockInputThread before the JSON Log yields exit0 / empty stdout. Exact requests/results are in protocol-control.jsonl. An initial fake incorrectly put two oneof variants inside one frame, then timed out; that setup failure is excluded and the corrected fake uses distinct frames. This deterministically establishes both native client failure modes, but does not prove the real server's triggering schedule.

Version-pinned server source has a concrete stale-ID candidate: ConnStatus both removes the client in its handler and queues a second RemoveClient on router exit; IDs reuse the lowest free integer. ServerOsInputOutput clones share one numeric-ID-to-sender map; send_to_client resolves the current sender, and remove_client unconditionally deletes by number. Thus delayed cleanup/response can target a replacement connection. This remains a candidate until the real bad frame/interleaving is captured. No sleeps, deadlines, native installation, Pair production code or conformance assertions changed. The present evidence exceeds a simple fixture-readiness diagnosis; request hub scope guidance before any native dependency change or new transport/serialization mechanism.

## Revisions

### 2026-10-17 09:30 — approved read-only native-trigger investigation

TL approved bounded investigation in the existing Zellij checkout, read-only, time-boxed to approximately two hours (stop around 11:30 local and report). No upstream issue/PR, dependency edits/installation, retries/sleeps/transport machinery in Pair tests or known-cause exemption is authorized. If stale-ID race is confirmed, retain minimal reproduction in this issue and bring any upstream publication to the TL first. This replaces the assumption that the next step is necessarily a quick fixture-only implementation. Production and tests remain unchanged. Pair:6 now owns #636 and has been sent the shared empty-reply findings; native runs are serialized between slots.
