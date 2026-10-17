---
id: 000641
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a288eb746296ca4da58a8d7dd195a6abfb81ac9a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T09:59:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d5226d48", done: "492cfe48"}
actual_hours: 0.46
---

# Qualify Claude PR footer background work

## Problem

#632 preserves unknown for unfamiliar native footer fields. A real Claude slot with a free composer and background shells includes a native PR field, so this valid variant currently remains unknown and keeps the previous fading behavior.

## Spec

Qualify the native Claude PR footer variant from the exact captured input below, retaining #632's strict complete-footer grammar and independent admission policy. Do not accept arbitrary fields or infer work from body/completion prose. Obtain complete native terminal-frame/geometry evidence before expanding the production recognizer; an unfamiliar or malformed field remains unknown. No protocol, Settled, admission or timer change is needed.

Inline text fixture, captured read-only with Couch peek from pair:2 on 2026-10-17 at about02:13 PDT (tag1-pair-8, agent Claude; free composer, two background shells). This is the exact rendered footer text, not original VT bytes; native CLI version and original synchronized VT frame were not retained. Preserve that provenance when turning it into a recorded test fixture:

```json
{"fixture":"claude-pr-background-footer","source":"couch peek pair:2, live rendered terminal","footer":"⏵⏵ bypass permissions on · PR #387 · 2 shells · ← for agents · ↓ to manage"}
```

The same read-only observation showed an empty composer above the footer and the native completion line reporting two shells still running. Those corroborate the observed state; the footer is the fixture input. Repository policy is text-only, so evidence is preserved inline rather than as a file attachment. No unrelated transcript content is copied.

## Done when

- The exact captured PR/footer variant is preserved as a regression fixture with its provenance and qualified composer geometry; complete native capture evidence is recorded before parser expansion.
- A behavioral red test demonstrates the valid PR footer currently returns unknown, then passes with active background evidence while the composer remains free for a successful peer receipt.
- Malformed/truncated PR fields, arbitrary prefix/suffix fields and counters outside the qualified footer stay unknown. Existing no-shell, zero, completion/lifetime and older-transport behavior remains covered.
- Qualification and native coverage limits are documented; no loose substring fallback or admission change is introduced.

## Plan

- [x] Qualify the preserved native archive before parser changes: native version, 94×39 geometry, synchronized frame boundaries, exact PR hyperlink/footer and free-composer cursor. Preserve a minimal recorded composer/footer fixture plus source hashes/offsets and the original inline observation's weaker provenance.
- [x] Red/green: recorded PR footer reports active work while its composer stays free through confirmed peer submission. Extend the existing strict complete-footer parser only for canonical positive-decimal PR #N at the observed position; reject duplicates, malformed/truncated numbers, arbitrary fields and unqualified PR-only absence.
- [x] Keep body/draft/partial-frame, zero/no-shell, completion/lifetime and older-transport regressions; update qualification docs and run targeted/race plus recorded full suite before close.

ARCH-DRY/PURE/PURPOSE: reuse backgroundWorkState, terminal replay and peer-delivery fixtures; one strict grammar, no substring fallback. This is display evidence only: no protocol, admission, Settled or timer change. ARCH-SECURE: capture only composer/footer evidence from the native archive, excluding unrelated transcript and credentials; incomplete evidence stays unknown. ARCH-MOCK: actual native evidence qualifies the existing fake/replay seam. ARCH-CONSTRAINTS: existing bounded frame dimensions and footer scan remain; no runtime IO. ARCH-ORDER/FUNERAL: no new cross-event state, artifacts or resource owners; existing terminal/frame lifetime rules remain authoritative.

## Log

2026-10-17: follow-up from #632 close/landing. Report52b2dbea-d7a5-404e-ada3-5e31438cefac submitted to the hub. Verified submitted TL74bd9c72-9451-4c36-9963-38eba67e0491 directs landing #632 as reviewed and filing this PR #N variant separately with the capture as fixture; do not loosen the parser without a capture. Filed unclaimed for later dispatch; no implementation started.

### 2026-10-17 — capture investigation
- 2026-10-17: closed — Archived native Claude 2.1.301 prefix qualified before parser expansion; 94x39 PR footer fixture preserves native cells/cursor, hash f009b62d4ab8. Red/green native PR and erased-space regressions; successful Submitted peer delivery with active background hint. All composer/settle/recorded native fixture tests pass without changed existing expectations; broad race run passed (410.916s). Recorded full suite on 367a42ab81623cc367531576f2030c208402007d passed exit 0 in 10m39s, log b4893d7fc185. README/atlas updated. Archived replay acceptance; no pair:0 staging. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Claimed pair:7 and entered planning. Pair:6 confirms only the original rendered peek was retained there (dd478cee-34e0-4348-869a-4f25b58a45d8). Read-only discovery found the original tag's parked native recording at repos/990458a6956a804b/parked-scrollback-1-pair-8-20261017T024328.raw with capture/events sidecars. Preliminary archive evidence: Claude Code v2.1.301, 94×39, synchronized frames; PR number is OSC8-linked. Current live pair:2 is idle without the PR footer and now reports v2.1.302, so it cannot substitute for archived provenance. Parser unchanged while the exact qualifying frame is located.

### 2026-10-17 — complete native capture qualified before parser expansion

Read-only replay recovered the exact original idle footer from the preserved native archive (not synthesized from peek): Claude Code v2.1.301, 94 columns × 39 rows, visible cursor zero-based (2,34), borders on rows 34/36, prompt row 35 and footer row 37. Event markers bound this idle screen to 2026-10-17 00:12:56–02:42:27 PDT, consistent with the reported ~02:13 observation. Original PR #387 is OSC8-linked. The final idle screen requires replay of the prefix through byte 32210830; its last synchronized frame is half-open [32207890,32210763), SHA256 f9ebc9efb5a667ecc9290235508baf9740c686056fc5a774d3bcfdc06edb1a53. Subsequent bytes through the selected offset are notification/control sequences only. Native source has 237570 balanced synchronized opens/closes and only one geometry event, at offset 0.

Source base: /Users/xianxu/.local/share/pair/repos/990458a6956a804b/parked-scrollback-1-pair-8-20261017T024328. Raw 32285046 bytes, SHA256 900257bb7be631dea64469939d864c297b3f1b20dc1aebc1dfd84df561eaaa3e; capture metadata SHA256 551abe760a835726e30e267c203dbd85ce9e4e50aa0724a4ddb331c99f499be9; events SHA256 27969c1621449b9611614994d65b13d1b86d53a450c6afb273c2f20c6af1c533. Capture witness v2, launch ordinal67, generation be0b6a7cd7ee87d0de3e0dfe87b70c25, scope990458a6956a804b, tag1-pair-8, captured 2026-10-17T09:43:28.805915Z. Pair:6's inline peek remains accurately labelled weaker rendered-text evidence. The regression fixture will be re-rendered from native composer/footer cells after full-prefix replay, as existing Claude fixtures are, rather than mislabelling an incremental frame as self-contained. Production parser remains unchanged at this checkpoint.

## Revisions

### 2026-10-17 — preserve erased spaces in the shared row projection

Native replay exposed empty Content/Width 1 cells between footer words. terminalRowText dropped these erased spaces, producing PR#387 and 2shells even with the PR grammar fixed. Hub approved the shared rendering correction (1461743d-daf1-4478-9a84-07b5394bb3ff): preserve erased spaces and skip width-zero wide-glyph continuation cells, without changing admission policy. Add a direct red/green row regression; run all composer/settle recognizers and recorded native fixtures. Any changed expectation must be explained as dependence on the old projection bug, not silently updated.

### 2026-10-17 — implementation and targeted evidence

Minimal fixture SHA256 f009b62d4ab8d404a7a9017b44300457a3773255aaae7ca8da70362437c1c7e6 (2047 bytes). Its README distinguishes original prefix evidence from the privacy-trimmed cell re-rendering. Same pinned x/vt verified all 376 selected native cells (content/width/style) and cursor. Fixture preserves erased cells rather than replacing them with printed spaces, which exposed the shared projection defect.

Observed red tests before production changes: native PR footer returned unknown and the free-composer delivery test had no active background hint; canonical PR positive cases returned unknown. Direct erased-space/wide-continuation regression returned 界AB instead of 界 A B. After both fixes, go test ./cmd/internal/wrapcmd -run 'BackgroundWork|Composer|Settle|Recorded|TerminalRowText' -count=1 passes; no pre-existing expectations changed. PR delivery reaches Submitted, with independent Empty composer and active background work. The fake explicitly negotiates bracketed paste because the minimal native fixture preserves cells/cursor, not unrelated terminal modes; simulated paste paint remains labelled synthetic.

Full wrapcmd package run passed (64.672s), log /tmp/pair641-wrapcmd.log, covering the existing native fixture conformance and recognizers unchanged. Subsequent test-only additions assert exact native borders/footer and reject PR-only cycle chrome, missing PR delimiter and truncation; their targeted runs also pass. Broader race verification and recorded full suite follow before close.

Race verification passed: go test -race ./cmd/internal/wrapcmd -run 'BackgroundWork|Composer|Settle|Recorded|TerminalRowText|HarnessTTY' -count=1 (410.916s). This includes all HarnessTTY fixture conformance tests as well as composer/settle-recognizer families; no existing expectation was adjusted. Recorded suite runs on pushed code head 367a42ab8 via PR #405; no manual main integration before close, per current worker contract.

Recorded full suite passed on 367a42ab81623cc367531576f2030c208402007d: sdlc local-check run --name suite, exit 0 in 10m39s, log SHA256 prefix b4893d7fc185, saved .git/sdlc/local-checks/367a42ab81623cc367531576f2030c208402007d/suite.log. No flakes or failures. All subsequent branch changes are issue-log/plan ticks only; production code remains the tested head's code. Acceptance is archived native-frame replay and independent peer delivery, without a visible window or pair:0 staging.
