---
id: 000865
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'f68efb83794ea258d8453e551dfb4ea666d683f4' # card fields mirrored from issue-cards; edit via sdlc
---

# Live Couch burns ~1.2 cores continuously in the Chrome cast recorder (PNG decode, LowRes) while a cast records

## Problem

## Spec

**The live Couch spends about 1.2 cores continuously, mostly in the Chrome cast recorder (ops:0, 10-24 02:15).**
Measured on the operator's Couch (`~/workspace/pair/bin/couch`, up 5.5 h): 115–140% CPU sustained over several
samples, while `fseventsd` also ran at 120–140%. The machine's load average was ~22 on 12 cores, and ariadne's
suite shards ran ~1100 s instead of ~500 s, so Couch's share directly slows every recorded suite.

A 3-second `sample` of Couch, top of stack (Couch frames): `webcast/recorder.LowRes`, `image/png` decoding
(`readImagePass`, `filterPaeth`), `image/draw.DrawMask`, `image.(*NRGBA).SetRGBA64`, plus terminal emulation
(`vt.(*Emulator).handleGrapheme`, `terminal.plainText`, `ptychild.ingest`) and GC (`scanObject`). The Chrome cast of
echo's board was left recording; that page repaints every 100–200 ms (pair#849's finding).

Find the cause and bound it: e.g. decoding and downsampling every captured frame even when nothing changed, or
recording at full rate with no viewer. Also check whether Couch's writes drive the `fseventsd` load.

## Done when

- Couch's CPU while a cast records a busy page with no viewer is measured before and after, with the fix bounding it
  (a stated budget), and a test pins the mechanism (e.g. unchanged frames skip decode).
- The `fseventsd` question is answered with evidence.

## Plan

- [ ]

## Log

### 2026-10-24
