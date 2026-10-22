---
id: 000784
status: open
deps: [pair#752]
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '53a2861d80ee330664e1cc4d358a61a66da87bec' # card fields mirrored from issue-cards; edit via sdlc
---

# Capture cast pixels without console flicker

## Problem

Periodic fidelity screenshot capture hides the trusted console UI host through privacy suppression, causing visible flashes and removing focused controls from the focus tree. #752 defers background captures while a sketch/label UI is open; outside that bounded fallback the underlying capture mechanism still toggles visibility.

## Spec

Find a capture method that excludes private operator controls without changing their visible display or focus: qualify underlying layer/region capture or host rendering outside the captured scene. Preserve page pixels/geometry and fail-closed privacy, including cross-process frames, navigation and unknown CDP outcomes. Do not replace suppression with an unproved compositor assumption.

## Done when

- Background and explicit captures contain no operator controls and never visibly flicker or interrupt focus/caret.
- Capture fidelity, UI privacy and navigation/uncertainty behavior pass native Chrome conformance with source frames and private overlays.

## Plan

- [ ]

## Log

2026-10-21: filed unclaimed at TL72ca720e direction. defaultShotEvery is5s; recorder/side.go calls recorder/ui_capture.go CaptureScreenshot, whose inject.js __couchcastUI suppression currently sets host.hidden and display:none. #752's scoped fallback must retain suppression for captures actually taken.
