---
id: 000841
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c5c87e071520a3c10acc197cd976bae201c30737' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T16:55:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e66040f8", done: "622e9fdd"}
actual_hours: 0.41
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
- 2026-10-23: closed — TL-approved conservative contract (submitted f36581c3-b946-416d-8e79-500a08ff5601, recorded in Revisions): false negatives allowed, no source projection; standalone final fitted-row words only, bracket/link/path/URL chunks and wrapped edges stay plain, generated time excluded. Revised regressions RED on previous code 0.459s then GREEN 1.274s. Exact clipped-row accents for all 3 forms, fixed minute 09/10/59 matrix, actual Ctrl+Y plain clipboard, hover/default notes controls; broad targets PASS 1.853s, race PASS 3.933s, timezone PASS 0.585s, exact artifact guard PASS 3.506s, N=8/16 classifier 4N, inventory PASS immediately before close, diff check clean. Atlas/lessons updated. Fresh recorded affected started on pushed d30f7c517f25; required PASS before landing. TL no stage confirmed; no staging performed.; review verdict: SHIP

Claimed after #840 landed and cleanup returned a clean resting branch. TL plan posted; quick-flow proceeding-unless-objection rule applies. Existing notes renderer provides fixed dark background and existing actionable-link blue; no new palette is needed. Alternatives: matching each wrapped line risks false positives inside split paths; storing decorated strings risks leaking styles to copy. Use source spans projected to cells instead.

Implementation qualification: reference-cell regression RED before the change (0.368s; every expected token lacked accent), then GREEN including notify bodies, prompt/question titles, slot labels, punctuation, Unicode, hard wrapping and clipped paths. A separate zero-width-source regression exposed byte-mask drift, RED before its correction; advance the source byte offset even for skipped zero-width graphemes. Final targeted bell/notes/toggle/notification controls PASS 1.838s and targeted race PASS 2.697s. Clipboard test drives Ctrl+Y while the bell overlays the switcher and reads the isolated fake pbcopy output: exact original plain body. N=8/16 classifier counts are exactly 2N, with mask byte size equal to source bytes. Atlas updated in place. Reader inventory PASS; new pure helper classified as non-artifact after the exhaustive inventory failed on its missing entry. No artifact paths are read or constructed by that helper. Lint/godoc controls PASS 4.658s/0.429s.

Final exhaustive artifact guard PASS 4.005s after the one audited non-artifact classification; final renderer/clipboard/notes subset PASS 1.111s. Production additions total 100 lines, within the quick-flow shell. Implementation checkpoint precedes recorded affected verification.

PR #571 opened on implementation f2257a6dc2532c77257969cd90e99851fee33f09. Recorded affected runner started on that pushed head; no passing recorded result claimed yet. Close may overlap verification under tl-worker rules, and landing remains gated on the pass. Asked TL to explicitly confirm proposed render-only/no-stage acceptance; no slot :0 or live fleet touched.

Corrected final verification: targeted attention/notes/toggle/notification controls PASS 2.964s; targeted race PASS 6.408s; original reviewer timezone reproduction (TZ=Asia/Kolkata) with fixed boundary times PASS 0.591s; exhaustive artifact guard PASS 2.294s; lint/godoc PASS 4.779s/0.442s; reader inventory PASS and diff whitespace clean. Branch patch now adds 99 production lines, so it remains within the quick shell. No recorded pass yet on the corrected executable head; commit and push precede its fresh affected run and second close review.

TL-approved conservative implementation: source projector, source sanitization helper and ordered path lexer deleted. Final fitted-row whole-word matcher leaves ambiguous bracket/link/path/URL chunks and wrapped edge fragments plain. Revised tests RED on former implementation (0.459s): bracket/split highlights violate the narrowed contract and exact clipped-row reference loses its accent. Simplified implementation GREEN 1.274s on reference/notes/clipboard controls. Production additions now 75 lines, within quick flow. Atlas and learned rule revised to state the false-negative tradeoff.

Conservative final verification: attention/notes/toggle/notification tests PASS 1.853s, targeted race PASS 3.933s, timezone/fixed-minute matrix PASS 0.585s, exhaustive artifact guard PASS 3.506s, reader inventory PASS, diff whitespace clean. Exact clipboard remains covered; candidate counts are 4N at N=8/16 (one match per rendered word), with fixed-size row masks. No recorded check is active yet on this new executable state; commit/push, fresh affected run and close follow.

### 2026-10-23 — Landing blocked by main park regression #852

Recorded affected attempt e5a7db4c973ffa5a5ff24dae9232b02b on executable d30f7c517f25996e15b8bf190d2ad6327e7bd4a5 failed TestLeaveParkRetriesProvenRollbackWithinOneQuit/15 (14 parked, 1 unresolved, context deadline). Other selected Go packages passed; script groups did not run. Terminal escape bytes before the failure header hid the named test from SDLC attribution (independent ariadne#582 filed by TL); immutable failed log SHA256 c3bacd32bd9aa1fd007370840ad07ec85903f8c8b8040d922ed1e362cff3592d.

At TL direction, ran the identical focused command once on separate untouched diagnostic trees: go test -json ./cmd/internal/couchcore -run '^TestLeaveParkRetriesProvenRollbackWithinOneQuit$' -count=1 -timeout=90s. Issue executable d30f7c51 failed both /15 and /30 with an unexpected third recovery drive (65.102s). Pure origin/main f109640a6ad2477df1629ea9b1238d0b90a3f634 also failed both: /15 14 parked/1 unresolved (20.72s), /30 28 parked/2 unresolved (39.97s), context deadline after one drive; package61.055s. Filed unclaimed high-severity #852 with both reproductions for TL dispatch. Logs and trees retained under /Volumes/TestScratch/pair841-leave-repro-gub_n4rh. No park source changed, no flake registered, no check retried and no recorded runner remains active. #841 remains accepted SHIP/high, not landed; await #852 disposition, then obtain required affected pass before boarding. #825 remains queued after landing.

### 2026-10-23 — Authorized overlapping claims

TL directive 295d506f-33e6-4f3b-a09a-89a60e9a0537 authorizes starting #825 from fresh main while retaining #841's queued check, then returning to board #841 on its pass. Both #841 (accepted SHIP, waiting for check/landing) and #825 (claimed, beginning root-cause work) remain owned by pair:9; ariadne#580 tracks overlapping-time attribution. Verified lease-1223841125's detached checkout HEAD954e3caea7078371c91c7523e86a70bdad751658 at /Volumes/TestScratch/local-checks/sdlc-local-1464763110; caller branch changes cannot alter that source. Dependency declarations, affected policy and Makefile.local match main. Retain runner81109, no restart/cancel or duplicate check. Scheduling only; no design revision.

### 2026-10-23 — Competing-writer flake classification and fresh verification

Repaired-head affected attempt01392f0a6d0a73ce07338d4fa6c68104 on954e3caea7078371c91c7523e86a70bdad751658 failed5m43s only in TestRepoConfigurationPublicationBoundsCompetingWriter/store (line417, busy-store warning instead of deadline cause); parser coarsened attribution to the whole top-level. Raw log SHA prefix37b0a76723f5. All other selected Go packages passed, including park, couchtty and agentfixture; build/doctor/Lua/review/shell groups did not run after Go failure. No passing status claimed.

Focused whole-test count1 on an unchanged exact-head detached tree PASS10.562s, both store5.07s and coordinator5.22s. JSON evidence retained in /Volumes/TestScratch/pair841-leave-repro-gub_n4rh/841-repoprefs-repro.jsonl. Filed/registered independent unclaimed #854 at the whole top-level, matching proven attribution, under the established flake rule. This first intermittent failure on the repaired executable head is unrelated to bell rendering. No repopreferences/store source changed. SDLC resume with name affected explicitly refused: only suite is supported; original immutable failure remains. Advance only this Log and run one fresh pre-close check, retaining #825's independent full suite. Both open claims remain authorized by295d506f-33e6-4f3b-a09a-89a60e9a0537; #825 is SHIP/high waiting for its running suite, #841 remains SHIP/high waiting for a complete affected pass and boarding.

## Revisions

### 2026-10-23 — Resume landing after #852

#852 landed in PR573 (12fd2e56acc65bb9e34a02f9d472e50e6918f0af); its controlled fixture budgets restore the park test, with production unchanged. TL's latest directive 738c7185-dcc5-4f7b-9fa8-4e9a9ddd5b26 resumes #841 before #825. #825 is reserved by this slot but not started. Re-entered merge; it refused because #841's original affected attempt had failed, so no passing boarding record existed. Merged current main without conflicts to include the landed repair before verification. Own accepted bell code is unchanged; no passing check is being discarded or repeated. Run the new pre-close selector on the pushed repaired head, then re-enter landing. Existing SHIP/high review and no-stage disposition remain; no re-close unless a gate requires it.

### 2026-10-23 — Make release-gate sequencing explicit

Split the final Plan checkbox into completed pre-close preparation and downstream release gates in prose. Close review and landing cannot truthfully be pre-ticked before running their own verbs; scope and Done when are unchanged.

### 2026-10-23 — Boundary round 1 correction

BR-1 and BR-2 verified, not disputed. The generated HH:MM prefix was incorrectly eligible as a numeric repository reference, and a whitespace-wide exclusion suppressed prose references beside URLs/paths/email. Their common boundary is source role, not the appearance of a substring. Regression matrix now fixes the clock at minutes 09, 10 and 59 for each notification kind and hover state; adds numeric repository content, all three reference forms as Markdown link labels, comma-adjacent slash/backslash/email siblings, and internal URL/path bracket exclusions. New controls RED before fixes (0.452s), with timestamps and missing references observed independently. BR-1-only prose/numeric controls GREEN 0.356s after excluding generated prefix. Complete reference/clipboard matrix GREEN 0.775s after ordered lexical spans replaced chunk rejection. Full tests follow below.

TL confirmed no stage (submitted receipt 85abae89-0703-4e22-940a-ae634fdc02c9): cell-level rendering plus actual Ctrl+Y clipboard control are acceptance; operator sees the result after next restart. Queued affected lease-2367206354 on the obsolete executable head was explicitly canceled through SDLC, joined with cancellation, and recorded no pass; a fresh run on corrected pushed code is required.

### 2026-10-23 — Correct source-role boundaries

Supersedes the Plan's whitespace-chunk exclusion sentence: generated timestamp metadata is outside matching. Ordered lexical spans retain path/URL internal brackets while allowing separate punctuation-delimited prose references, including Markdown labels. Whole-token classifier, source masks, existing wrap/clip projection and cell-only styling remain the mechanism. Done when and feature scope are unchanged. No new state or IO.

### 2026-10-23 — TL narrows acceptance to a conservative visual aid

Authoritative submitted TL directive f36581c3-b946-416d-8e79-500a08ff5601 supersedes the earlier source-projection design and the earlier all-reference completeness expectation in Spec/Done when: missing a highlight is acceptable; wrong highlights are not. Match rendered row text after wrapping/fitting. Color only a standalone whole word bounded by whitespace or plain punctuation. Anything touching URL/path/bracket/Markdown syntax remains plain. Split references may remain plain. Keep generated timestamp exclusion. No balanced-delimiter lexer or source-to-cell projection.

Effective Done when: ordinary standalone repo:N, repo#N and #N examples use existing blue; rendered path/URL/bracket/link/time false positives are excluded, including wrapped fragments; actual Ctrl+Y is exact plain input; hover, chip colors, selection and default notes rendering remain unchanged. False-negative controls explicitly allow adjacent Markdown references to stay plain; this is the TL's chosen tradeoff, not deferred work.

Revised quick plan: delete source mask projection and path-token lexer. Scan only final display-row whitespace chunks, trim allowed prose punctuation, apply existing whole-token classifier, and color matching byte ranges in the cell renderer. Skip generated timestamp bytes on the first row. For wrapped items conservatively skip each row's edge chunks so split path/reference fragments cannot masquerade as complete tokens. This is a one-pass per-row pure display helper, O(B+C), with no source search or retained offsets. Keep the existing isolated clipboard seam. Add RED-first cases for the old bracket/link false positives and fixed timestamps, a few standalone positives, wrapped paths and split references, and exact clipped-row checks.

Round 2 retained BR-2 and raised BR-3 (substring projection fails when fitting removes zero-width clusters). Both disappear structurally under the approved narrower contract; the new tests must demonstrate that rather than relying on the design claim. Own obsolete affected lease-2706042998 explicitly canceled and joined, with no recorded result; no owned background runner remains. Quick flow remains intended if the final patch fits; no durable full-flow plan is needed for the TL-approved simplification.
