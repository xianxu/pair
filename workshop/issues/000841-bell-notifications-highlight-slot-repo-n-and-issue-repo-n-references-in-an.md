---
id: 000841
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3c5de2807ca73fb85a6d634a7c4bae9a00ed9f5b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T16:55:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "38603a9c", done: "622e9fdd"}
---

# Bell notifications: highlight slot (repo:N) and issue (repo#N) references in an accent color

## Problem

References blend into notification prose, making the operator scan every word.

## Spec

**Highlight slot and issue references in the bell's notification list (operator, 10-23).** In each notification,
slot addresses (`repo:N`, e.g. `pair:3`) and issue references (`repo#N`, e.g. `ariadne#575`, and bare `#N`) render in
an obvious accent color so they're easy to spot. Display only: the text, copy (Ctrl+Y) and selection behaviour are
unchanged. Use an existing palette accent that reads in both light and dark themes; match whole tokens only (not
inside paths, URLs or longer words).

## Done when

- Rendered notification rows color `repo:N`, `repo#N` and `#N` tokens, and only those (render test with edge cases:
  inside a URL or path, punctuation-adjacent, wrapped rows).
- Copying a notification yields plain text (test).

## Plan

- [x] Add RED-first terminal-cell controls for bell references, punctuation, paths/URLs, longer words, Unicode, wrapping/clipping and hover; retain default notes-menu geometry/style controls and a real clipboard plain-text assertion.
- [x] Opt bell items into pure source-span detection before existing wrapping. Project accents onto cells using existing notesMenuLink; preserve text, IDs, chips and selection. Keep other menus unchanged.
- [x] Run targeted/race tests and reader inventory; publish the PR, start recorded affected verification and prepare close.

Downstream release gates remain outstanding until their own result: recorded affected PASS on the pushed executable head, SDLC close review, TL stage disposition and SDLC landing.

Quick-flow design: source references are display hints, never resolved addresses or navigation authority. Scan whitespace-delimited source chunks; reject path/URL chunks and recognize whole reference tokens adjacent to prose punctuation. Reuse canonical slot validation for repo:N. Accent spans refer to sanitized source bytes; forward projection through unchanged wrapNote output keeps hard-wrapped fragments colored without recognizing a path fragment as a new reference. No settings, format changes or new IO.

ARCH-DRY: reuse wrapNote, notesMenuLink, canonical slot validation, existing menu cells and copy source. ARCH-PURE: detector/projection are pure; Console only opts bell items into styling. ARCH-PURPOSE: cover all bell item kinds and label/title/body while excluding other menus. ARCH-CONSTRAINTS: per rendered item O(B+C+R) for source bytes B, displayed cells C and reference spans R, no repeated source scan per line; count detector visits at N/2N. ARCH-SECURE: sanitize before matching; colors never interpret escapes or change copy payload. ARCH-MOCK: no new external dependency; clipboard uses existing isolated fake executable. ARCH-ORDER: holds no state across events because spans exist only in one menu layout/draw; selection/recovery are unchanged. ARCH-FUNERAL: creates nothing durable; spans die with the existing frame. ARCH-EVOLVE: persisted formats unchanged.

## Log

### 2026-10-23

Claimed after #840 landed and cleanup returned a clean resting branch. TL plan posted; quick-flow proceeding-unless-objection rule applies. Existing notes renderer provides fixed dark background and existing actionable-link blue; no new palette is needed. Alternatives: matching each wrapped line risks false positives inside split paths; storing decorated strings risks leaking styles to copy. Use source spans projected to cells instead.

Implementation qualification: reference-cell regression RED before the change (0.368s; every expected token lacked accent), then GREEN including notify bodies, prompt/question titles, slot labels, punctuation, Unicode, hard wrapping and clipped paths. A separate zero-width-source regression exposed byte-mask drift, RED before its correction; advance the source byte offset even for skipped zero-width graphemes. Final targeted bell/notes/toggle/notification controls PASS 1.838s and targeted race PASS 2.697s. Clipboard test drives Ctrl+Y while the bell overlays the switcher and reads the isolated fake pbcopy output: exact original plain body. N=8/16 classifier counts are exactly 2N, with mask byte size equal to source bytes. Atlas updated in place. Reader inventory PASS; new pure helper classified as non-artifact after the exhaustive inventory failed on its missing entry. No artifact paths are read or constructed by that helper. Lint/godoc controls PASS 4.658s/0.429s.

Final exhaustive artifact guard PASS 4.005s after the one audited non-artifact classification; final renderer/clipboard/notes subset PASS 1.111s. Production additions total 100 lines, within the quick-flow shell. Implementation checkpoint precedes recorded affected verification.

PR #571 opened on implementation f2257a6dc2532c77257969cd90e99851fee33f09. Recorded affected runner started on that pushed head; no passing recorded result claimed yet. Close may overlap verification under tl-worker rules, and landing remains gated on the pass. Asked TL to explicitly confirm proposed render-only/no-stage acceptance; no slot :0 or live fleet touched.

## Revisions

### 2026-10-23 — Make release-gate sequencing explicit

Split the final Plan checkbox into completed pre-close preparation and downstream release gates in prose. Close review and landing cannot truthfully be pre-ticked before running their own verbs; scope and Done when are unchanged.
