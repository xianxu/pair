---
id: 000640
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8557cf3ee5b2be58b892328c231cadf24fdbf528' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-17 — #639 qualification controls

Candidate first run: PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run '^(TestFullscreenZellijLive|TestFullscreenChordZellijLive)$' -count=1 -v (the first named test does not exist; only chord ran). Failed after 11.18s at fullscreen_chord_live_test.go:149, before toggle input, list-panes reports no active session.

Original control source: 1ac101a17404a392927a4dac83676b43e6030b9a. All tracked layoutcmd Go files restored via go test/go build overlay from this Git commit. Build original-pair with go build -overlay ORIGINAL.json -o original-pair ./cmd/pair-go, then PAIR_TEST_BINARY=original-pair PAIR_LIVE_ZELLIJ=1 go test -overlay ORIGINAL.json ./cmd/internal/layoutcmd -run '^TestFullscreenChordZellijLive$' -count=1 -v. This passed draft-nvim and agent-wrapper scenarios, then failed after 4.22s during terminal-shell-bottom at list-panes (line308), same no-active-session diagnostic; restore-tiling current-tab-info exit1 also logged. The precise failing phase differs, so this establishes only the pre-existing session-loss family. Workbench help text on disk was newer; original planner, runtime, test body and supplied binary's layout code came from the overlay. No live tests overlapped these controls.

Local evidence root /tmp/pair639-7xyqf5rg; original.json and original-*.go preserve the exact source; original-build.log, environment.json and original-chord-correct-binary.log retained. Original binary SHA256 6b99f66417c5204867841b112751d920e7754b21d24a482d17dda0dc7bb736bf; control log SHA256 225feb439e35694e73558be8dc20488af695c9437f8a63fac3473229b8b214f4.

An earlier original-test-only run used a candidate Pair binary, so it is excluded as a baseline. A separate direct-conformance invocation inherited Couch's long ZELLIJ_SOCKET_DIR and was refused before render (session name must be less than 0 characters); short private socket roots are required for that fixture and this is not the session-loss reproduction. Candidate Focus-return errors (already-focused exit2 and skipped same-pane return-focus) are #639's defects and are being corrected there, not assigned to this issue.
