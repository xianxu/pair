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
