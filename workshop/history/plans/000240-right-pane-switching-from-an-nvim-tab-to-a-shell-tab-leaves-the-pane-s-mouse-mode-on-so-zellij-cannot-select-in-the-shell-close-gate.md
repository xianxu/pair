---
gate: boundary-review
issue: 240
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-12T19:39:00-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: hostScan comments predating the mouse prefix are now false
          detail: |-
            run.go:796 says hostScan is fed child bytes only; run.go:1053-1054 says the composed prefix is HomeAndClear alone and "changes no behaviour". Both are contradicted by the reconcile prefix; the block at run.go:1046-1050 already explains the suspension, so point the two stale sentences at it.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: comment-matches-code
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-12T19:39:00-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-12T23:07:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: run.go:796-803 and run.go:1053-1058 now describe the takeover prefix as the one suspension and point at applyTakeover.
          round: 3
      findings:
        - id: BR-2
          severity: Important
          title: Done-when's live operator selection check is not recorded anywhere in the Log
          detail: The Plan row defers it to "the operator's step after install" while Done-when lists it as a close criterion; record the live outcome in the Log or --verified, or move the bullet to the follow-up via a Revisions entry.
          family: done-when-evidenced
          round: 3
        - id: BR-3
          severity: Important
          title: Spec claims the pane's modes equal the active child's, but only mouse modes are reconciled; measured siblings 1004, 1, 2004 stay on the pane
          detail: Widening the probe's regex to every DEC private mode shows the switch to the shell writes only ?1002l ?1006l, leaving nvim's ?1004h (zellij then forwards focus in/out to the shell), ?1h and ?2004h. Write the enumeration (which modes reconcile, which are excluded and why, 1049 per repaint.go:55) and track the sweep as a follow-up; narrow Spec bullet 1 to mouse. ARCH-PURPOSE.
          family: pane-modes-follow-active-child
          round: 3
        - id: BR-4
          severity: Minor
          title: removeTab with a rename open skips the takeover, so a dead child's mouse modes outlive it until the next switch
          detail: run.go:1471-1486 takes the paintStripInline branch and finishRename (run.go:1373-1376) only repaints the strip. This is the 2nd finding in family pane-modes-follow-active-child; do not patch the site, dispose it through the events axis of the same enumeration.
          family: pane-modes-follow-active-child
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-12T23:12:51-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: Log "2026-09-12 (close)" records the operator's live confirmation of selection in the right-pane shell on this build.
          round: 4
        - id: BR-3
          disposition: addressed
          note: 'Spec header narrowed to MOUSE modes; the reconcile/exclude enumeration (1004, 2004, 1 reconcile; 1049/1047/47/1048 excluded per repaint.go:55) and the sweep are tracked in #241.'
          round: 4
        - id: BR-4
          disposition: addressed
          note: 'Disposed through #241''s paths axis, which names removeTab-with-rename-open explicitly; the site was not patched, as asked.'
          round: 4
      findings:
        - id: BR-5
          severity: Minor
          title: Close-review Spec narrowing was applied in place with the delta in the Log, not as a Revisions entry
          detail: AGENTS.md section 1 asks for an appended Revisions entry (timestamp, reason, delta) when a plan artifact changes mid-stream; the Spec paragraph was rewritten and only the Log's close entry records why. Add a second Revisions entry naming BR-3/BR-4 and the move to issue 241.
          family: revisions-appended-not-overwritten
          round: 4
        - id: BR-6
          severity: Minor
          title: workshop/lessons.md carries no rule from this review cycle
          detail: AGENTS.md section 4 asks for a lessons rule after each code review; the prior round proposed one (run the probe with the family-wide DEC-mode regex before closing a one-family fix and record the siblings). Land it as a post-close follow-up commit.
          family: lessons-recorded-after-review
          round: 4
      blocked: false
---

# Gate ledger — pair#240 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T19:39:00-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `comment-matches-code` hostScan comments predating the mouse prefix are now false
  run.go:796 says hostScan is fed child bytes only; run.go:1053-1054 says the composed prefix is HomeAndClear alone and "changes no behaviour". Both are contradicted by the reconcile prefix; the block at run.go:1046-1050 already explains the suspension, so point the two stale sentences at it.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-12T19:39:00-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-12T23:07:38-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — run.go:796-803 and run.go:1053-1058 now describe the takeover prefix as the one suspension and point at applyTakeover.

### Raised

- **BR-2** [Important] `done-when-evidenced` Done-when's live operator selection check is not recorded anywhere in the Log
  The Plan row defers it to "the operator's step after install" while Done-when lists it as a close criterion; record the live outcome in the Log or --verified, or move the bullet to the follow-up via a Revisions entry.
- **BR-3** [Important] `pane-modes-follow-active-child` Spec claims the pane's modes equal the active child's, but only mouse modes are reconciled; measured siblings 1004, 1, 2004 stay on the pane
  Widening the probe's regex to every DEC private mode shows the switch to the shell writes only ?1002l ?1006l, leaving nvim's ?1004h (zellij then forwards focus in/out to the shell), ?1h and ?2004h. Write the enumeration (which modes reconcile, which are excluded and why, 1049 per repaint.go:55) and track the sweep as a follow-up; narrow Spec bullet 1 to mouse. ARCH-PURPOSE.
- **BR-4** [Minor] `pane-modes-follow-active-child` removeTab with a rename open skips the takeover, so a dead child's mouse modes outlive it until the next switch
  run.go:1471-1486 takes the paintStripInline branch and finishRename (run.go:1373-1376) only repaints the strip. This is the 2nd finding in family pane-modes-follow-active-child; do not patch the site, dispose it through the events axis of the same enumeration.

## Round 4 — 2026-09-12T23:12:51-07:00 (claude) — passed

### Disposed

- BR-2 — addressed — Log "2026-09-12 (close)" records the operator's live confirmation of selection in the right-pane shell on this build.
- BR-3 — addressed — Spec header narrowed to MOUSE modes; the reconcile/exclude enumeration (1004, 2004, 1 reconcile; 1049/1047/47/1048 excluded per repaint.go:55) and the sweep are tracked in #241.
- BR-4 — addressed — Disposed through #241's paths axis, which names removeTab-with-rename-open explicitly; the site was not patched, as asked.

### Raised

- **BR-5** [Minor] `revisions-appended-not-overwritten` Close-review Spec narrowing was applied in place with the delta in the Log, not as a Revisions entry
  AGENTS.md section 1 asks for an appended Revisions entry (timestamp, reason, delta) when a plan artifact changes mid-stream; the Spec paragraph was rewritten and only the Log's close entry records why. Add a second Revisions entry naming BR-3/BR-4 and the move to issue 241.
- **BR-6** [Minor] `lessons-recorded-after-review` workshop/lessons.md carries no rule from this review cycle
  AGENTS.md section 4 asks for a lessons rule after each code review; the prior round proposed one (run the probe with the family-wide DEC-mode regex before closing a one-family fix and record the siblings). Land it as a post-close follow-up commit.

## Open findings

- **BR-5** [Minor] `revisions-appended-not-overwritten` Close-review Spec narrowing was applied in place with the delta in the Log, not as a Revisions entry
- **BR-6** [Minor] `lessons-recorded-after-review` workshop/lessons.md carries no rule from this review cycle
