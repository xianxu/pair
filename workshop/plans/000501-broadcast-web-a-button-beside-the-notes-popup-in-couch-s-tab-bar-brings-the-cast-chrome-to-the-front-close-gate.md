---
gate: boundary-review
issue: 501
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T19:00:57-07:00"
      dispatched: "2026-10-12T18:48:34-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: reserve.go button closure duplicates the capability closure
          detail: reserve.go:280 button() repeats capability() at reserve.go:248 (space, start, append, empty-span check); hoist one span(label, sgr) helper (ARCH-DRY).
          family: duplicated-span-helper
          round: 1
        - id: BR-2
          severity: Minor
          title: focusWebcast shows a raw ErrEnded notice if the cast ends mid-click
          detail: 'console_webcast.go:255 checks the phase, then calls cast.Focus() in a goroutine; an end in between shows "Chrome cast: ...ended". Could ignore errors.Is(err, webcast.ErrEnded).'
          family: race-surfaces-raw-error
          round: 1
        - id: BR-3
          severity: Minor
          title: Revision 3 still names broadcast.ControlLabel for the Chrome group's alien glyph
          detail: '#503 removed ControlLabel; the code uses couchtty.WebcastDriveLinkLabel. Amend the Revisions entry so it matches the code.'
          family: plan-revision-stale
          round: 1
      recipe: milestone-review
      isolation: fenced
      reviewed: 0331d3da8df4bcbaa46d322de2c0473be5de8f8d
      blocked: false
---

# Gate ledger — pair#501 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T19:00:57-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `duplicated-span-helper` reserve.go button closure duplicates the capability closure
  reserve.go:280 button() repeats capability() at reserve.go:248 (space, start, append, empty-span check); hoist one span(label, sgr) helper (ARCH-DRY).
- **BR-2** [Minor] `race-surfaces-raw-error` focusWebcast shows a raw ErrEnded notice if the cast ends mid-click
  console_webcast.go:255 checks the phase, then calls cast.Focus() in a goroutine; an end in between shows "Chrome cast: ...ended". Could ignore errors.Is(err, webcast.ErrEnded).
- **BR-3** [Minor] `plan-revision-stale` Revision 3 still names broadcast.ControlLabel for the Chrome group's alien glyph
  #503 removed ControlLabel; the code uses couchtty.WebcastDriveLinkLabel. Amend the Revisions entry so it matches the code.

## Open findings

- **BR-1** [Minor] `duplicated-span-helper` reserve.go button closure duplicates the capability closure
- **BR-2** [Minor] `race-surfaces-raw-error` focusWebcast shows a raw ErrEnded notice if the cast ends mid-click
- **BR-3** [Minor] `plan-revision-stale` Revision 3 still names broadcast.ControlLabel for the Chrome group's alien glyph
