---
id: 000031
status: done
estimate_hours: 1.0
deps: []
created: 2026-05-31
updated: 2026-05-31
actual_hours: 0.5
---

# codex ak-user-question flow need alt+return to confirm

for ask-user-question flow, normal return should work. this likely is caused by our tinkering around how return, alt+return works in agent pane. I remember we have code to detect the presence of ask-user-question (probably per agent) and remap temporarily to allow return to pass through. probably that's not working for codex.

## Done when

- Codex ask-user-question overlays accept normal Return to confirm.
- Normal Codex textarea Return still inserts a newline.

## Spec

`pair-wrap` already suspends Codex's textarea Return remap while it believes a
blocking overlay is open. The bug is narrower: the Codex detector recognizes
resume-CWD picker labels and `Press enter to continue`, but not the
ask-user-question choice picker. Those prompts render option labels that include
`(Recommended)` per the request-user-input contract, so use that as the overlay
marker.

## Plan

- [x] Add ask-user-question marker coverage to the Codex overlay detector.
- [x] Verify unit tests for overlay detection and Return remapping.

## Log


- 2026-05-31: closed — env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap -count=1; env GOCACHE=/private/tmp/pair-go-cache make test
### 2026-05-31

- Marked working. Existing Codex overlay detector only matches resume-CWD labels
  and `Press enter to continue`, so ask-user-question choices can leave
  `pickerActive` false and cause plain Return to be translated to `\n`.
- Added `(Recommended)` as the Codex ask-user-question marker and covered it
  with an overlay detector regression test.
- Verified with `env GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap
  -count=1` and `env GOCACHE=/private/tmp/pair-go-cache make test`.
