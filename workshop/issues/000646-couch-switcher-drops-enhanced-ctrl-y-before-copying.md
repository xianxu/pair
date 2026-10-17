---
id: 000646
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4b9f6a48bd8279af0477f438b3088b7d591e6930' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:59:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "672aa21a", done: "97d498a1"}
---

# Couch switcher drops enhanced Ctrl+Y before copying

## Problem

After the operator's 2026-10-17 10:48 Couch restart onto 15d84ef7 (including #638), Ctrl+Y in the switcher still did not copy, and supervision.jsonl contained no clipboard_copy event. #638 instruments the copy tool, but a key dropped before its handler produces nothing.

Read-only reproduction proves a decoder mismatch: Presenter requests Kitty disambiguation/event types (ESC[>3u), but DecodePanelKeys recognizes copy only from legacy 0x19. terminal.Decoder canonicalizes Kitty Ctrl+Y press forms CSI 121;5u and CSI 121;5:1u to CSI 121;5u, which decodeCSIu rejects as a modified printable. No clipboard tool is reached.

## Spec

Recognize Ctrl+Y consistently in the switcher for legacy and enhanced input, without broadening other modified printables or copying on key release. Keep the existing copy operation, persistent feedback and native/OSC policy unchanged. Add a narrow content-free Ctrl+Y arrival observation before product routing: encoding class (legacy/enhanced), press/release and whether the switcher owns focus, through the existing bounded supervision writer. Never log raw input, arbitrary keys, notice text or clipboard contents.

No clipboard probes, live key injection or pair:0 staging. Historical bytes are unknown; this fixes a proven decoder defect consistent with the historical symptoms, and diagnostics distinguish future key-arrival failures from copy-tool failures.

## Done when

- Deterministic regressions show enhanced Ctrl+Y currently drops before the handler, then legacy and enhanced press forms reach copying exactly once; key releases and unrelated modified keys do not copy.
- A fake-backed complete Console input-path test covers enhanced Ctrl+Y and the content-free arrival plus existing copy-outcome records, without using the system clipboard.
- Ctrl+Y delivered outside the switcher is observed as such and still reaches the child without invoking the copy handler; ordinary keys produce no arrival diagnostic.
- Docs explain the two diagnostic boundaries; targeted tests and recorded full suite pass before close. Live verification belongs to the operator's next natural restart; no staging.

## Plan

- [x] Qualify the read-only key path and retain a red regression for the Kitty canonical input; no runtime behavior edits before that evidence.
- [x] Extend the existing strict panel decoder for enhanced Ctrl+Y only. Add a narrow CopyKeyObservation callback at decoded Console input arrival and wire it to clipboard_key facts via the existing supervision writer.
- [x] Test whole Console routing with fake clipboard tools, split enhanced input, release/lock-modifier/extra-modifier controls, child forwarding and metadata privacy. Update README/atlas; recorded suite, close, normal landing.

ARCH-DRY/PURE: reuse terminal.Decoder, existing panel grammar, Console and bounded supervision writer; recognition remains pure over decoded input. ARCH-PURPOSE: fix the pre-handler loss and separately expose arrival, rather than adding more tool diagnostics. ARCH-MOCK/SECURE: only fake tools in private headless fixtures; a diagnostic contains fixed classifications, never key bytes or copied text. ARCH-CONSTRAINTS: one bounded fact per Ctrl+Y event, existing queue/size cap, no IO or clocks in the recognizer. ARCH-ORDER: no new runtime state or timers; release remains child-only and grants no copy authority. ARCH-FUNERAL: no new file family or lifecycle; existing diagnostic retention and fixture cleanup own the added data.

## Log

#621 is parked cleanly on 000621-rename-prefix-test (f04a2de1c), with targeted evidence recorded and no suite/merge runner active.

Read-only Go overlay probe log /tmp/pair-ctrl-y-readonly-probe.log: 0x19 passes, CSI 121;5u and CSI 121;5:1u fail because canonical CSI 121;5u yields no PanelKey. No repository mutation, live input or clipboard tool in the probe. Read-only config/process investigation: Ghostty user/effective/default bindings do not intercept Ctrl+Y; ancestry is Ghostty -> login -> zsh -> Couch, with zellij and nvim downstream. Pair zellij config has no Ctrl+Y binding. Thus Parley's C-y chord cannot intercept Couch's outer switcher. These observations establish the defect and topology, not the historical received bytes.

### 2026-10-17 — approved implementation and red evidence
- 2026-10-17: closed — Read-only overlay reproduced dropped Kitty Ctrl+Y before code; checked-in every-split decoder regressions red/green. Fake-only complete Console tests cover legacy/enhanced presses, releases, child forwarding and content-free arrival/outcome facts. Removal of arrival wiring makes the regression fail. Targeted race and full couchtty/couchcmd packages passed. First suite 81bf0bd9 failed only the new-source inventory omission; fixed and targeted guard passed. Corrected recorded suite 3ec7888ac81895610ef2e3110edb186aa1aa0ff3 passed 13m43s, log b4782ace19fa. README/atlas updated. No clipboard probes or staging; TL approved live acceptance on the next natural restart.; review verdict: SHIP

TL approved the quick plan (86b6ec05-260e-4ae6-bc7c-6c5a4cc41db7), including arrival facts and fake-only tests; live verification is on the next natural restart, no staging. Resume parked #621 afterwards. Checked-in TestCopyNoticeThroughTerminalDecoder failed before the fix for plain enhanced, explicit press, repeat and Caps/Num-lock forms; legacy passed. It covers every byte split, key release, extra modifiers, unrelated control letters and ordinary y. The parser change is one explicit canonical Ctrl+Y case; no general modified-printable fallback.

### 2026-10-17 — targeted verification

Targeted copy/decoder/panel tests passed under race: go test -race ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'CopyNotice|CopyKey|ClipboardCopy|Decode|Panel' -count=1 (3.905s / 1.938s). Full packages passed, log /tmp/pair646-packages.log. Separate metadata-shape/lifetime test also passes. Complete Console fixtures use private fake pbcopy only; each legacy/enhanced press produces one arrival record followed by one outcome, and a release produces arrival without copying. Actor-focus tests preserve Ctrl+Y child delivery; unrelated input and paste produce no observation. No existing copy tool policy changed.

An overlay that removes only observeCopyKey from routeInputEvent makes all three complete Console record tests fail at missing arrival (retained /tmp/pair646-arrival-negative.log); production files were unchanged by the control. This demonstrates the diagnostic wiring, not merely its projection helper. Decoder red evidence preceded its fix; the original read-only probe remains /tmp/pair-ctrl-y-readonly-probe.log. Both no-clipboard/no-staging constraints remain in force.

### 2026-10-17 — first recorded suite found a branch inventory omission

The recorded suite on 81bf0bd99 found TestProductionArtifactReferencesAreExactlyClassified: copy_key.go was absent from the exhaustive source inventory. This is a reproducible branch omission, not a flake. Classified the new pure diagnostic source in NonArtifactSources (it reads no artifact paths); the exact inventory guard and child-forwarding regression now pass, exit 0. Also isolated the child-forwarding test behind a private fake clipboard executable as a precaution. Existing lesson #526 already requires this guard when adding a production file; it was missed in the initial targeted run. The original suite is allowed to finish all remaining packages before its outcome is recorded and the corrected head is run.

The first recorded suite completed all packages: exit 2 in 13m48s on 81bf0bd9, sole failure TestProductionArtifactReferencesAreExactlyClassified (the corrected inventory omission), all remaining checks passed. Attempt 6ef3533721711c196dcb0c1b0dbb2f71, saved .git/sdlc/test-stats/failed/6ef3533721711c196dcb0c1b0dbb2f71.log, log hash ca2861909bcc. The corrected guard and child-forwarding test pass. A fresh recorded suite on the corrected pushed head follows; this is verification after a branch fix, not a flake retry.

### 2026-10-17 — corrected full suite passed

Recorded suite on 3ec7888ac81895610ef2e3110edb186aa1aa0ff3 passed exit 0 in 13m43s after 501s queued; attempt 01225ed931448120455af9eefdeba116. Saved .git/sdlc/local-checks/3ec7888ac81895610ef2e3110edb186aa1aa0ff3/suite.log, hash b4782ace19fa. No flakes or additional failures. Earlier failed attempt 6ef3533721711c196dcb0c1b0dbb2f71 remains retained with the branch fix identified. Live Ctrl+Y acceptance remains with the TL/operator on the next natural restart, per approved plan; no clipboard probes or pair:0 staging were performed.
