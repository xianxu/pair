---
id: 000876
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'a1be60a9d53f2ec79de362323c63cc890c9737b5' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast Note tool is missing or undiscoverable in the live tool column

## Problem

During the combined live check of #856, the operator said: “works. there's no annotation tool though, is that by design?” The accepted mock included a Note tool in the column. Relayed by TL delivery35862e4f. The live submission itself worked; Cancel is separately tracked as #875.

Source inspection finds an explicit `▤ Note` button: `web/annotate.js` creates it, `web/cast-tools.js` places it in the note group below Pen/colors, and clicking it enters annotation box/tap mode. Pointer is a separate pointing tool, not the annotation entry point. The tool was not deliberately removed. Native tests already exercise this actual button, but its presence alone does not explain the operator's missing visible control.

## Spec

Diagnose why the dedicated Note action was not visible or discoverable in the operator's actual cast tool column. Check the built/staged asset revision, viewport and column scroll position, group/button hidden state and annotation capability/readiness. The column has vertical overflow and Note follows the palette; below-fold placement is a candidate, not an established cause.

Restore a clear, reachable Note action matching the accepted tool set. Preserve the distinct Pointer, Pen and Note modes and existing capability/page-load guards. If annotation is unavailable, retain an understandable disabled reason rather than inventing an alternate Pointer workflow or weakening admission.

## Done when

- Root cause is recorded from the actual column state or a faithful disposable native reproduction; no unsupported claim that Note was absent from implementation.
- A live ready cast visibly offers a reachable labelled Note action, including at the reported viewport and with Pen controls expanded; existing Pointer and Pen behavior remains intact.
- Selecting Note permits the existing anchored-box/tap note flow; capability-off, paused and pending-load states preserve their existing guards and explanation.
- A regression pins the observed visibility/reachability failure, including overflow/scroll geometry if that is the cause; operator recheck confirms the tool is easy to find.

## Plan

Unclaimed follow-up; coordinate sequencing with the hub. Keep separate from #857's page-load scope and #875's Cancel action.

## Log

Filed from the operator's accepted-layout live feedback. No private capture is copied. Production button creation/adoption and native test references were inspected read-only; root cause of the operator's missing visible action remains unconfirmed.
