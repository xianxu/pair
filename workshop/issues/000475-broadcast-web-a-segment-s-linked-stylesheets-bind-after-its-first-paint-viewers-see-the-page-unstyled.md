---
id: 000475
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3df007bf91d2ccfd620a73349c2b94355794719c' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: a segment's linked stylesheets bind after its first paint (viewers see the page unstyled)

## Problem

Found by pair#472 (real recorder fixtures). At a segment's start the recorder emits the `doc` snapshot at once, but a
linked stylesheet's `asset` record only when its body has been fetched, tens of ms later (`rec-stress`: seg at 0,
first frame at 84, `stress.css` bound at 103, photo at 152). The tab was already styled; a viewer, live or replaying,
draws the page unstyled until the record arrives. `rec-stress` frame 0 replays 24% off its screenshot (tolerance 3%).
Every navigation and tab switch repeats it, briefly.

`replay_test.go` lists the frame as a known gap (`knownGaps`), and fails if it starts passing, so the fix removes the
entry.

## Spec

To decide (TL): the recorder holding back the segment's first records until the snapshot's stylesheet bodies are
bound or a short deadline passes, or the viewer holding the previous paint until a new document's stylesheets are
bound. Either way a viewer never draws an unstyled page that the tab never showed.

## Done when

- `rec-stress` frame 0 replays within tolerance at its own time and `knownGaps` is empty.

## Plan

- [ ] TL picks the side (recorder holds back, or viewer holds the previous paint).
- [ ] Test first: rec-stress frame 0 within tolerance; drop its `knownGaps` entry.
- [ ] Implement, with a recorder or viewer unit test for a stylesheet bound after the segment's snapshot.

## Log

### 2026-10-12
