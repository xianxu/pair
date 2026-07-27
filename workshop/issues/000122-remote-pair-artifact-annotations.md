---
id: 000122
status: open
deps: ["#119"]
github_issue:
created: 2026-07-26
updated: 2026-07-26
estimate_hours:
---

# Remote Pair artifact annotations

## Problem

Once artifacts can be viewed remotely, a tablet should be able to comment and
draw on them. Pair and Ariadne already use `🤖[]`-style inline markers for
human-agent document review, but that model is text-centric. A web/tablet surface
also needs anchored comments on rendered documents, SVG-specific comments, and
freeform screenshot annotations without corrupting source artifacts.

## Spec

- Add annotation as a layer on top of the read-only artifact browser from #119.
- For markdown/text artifacts, support structured comments that can compile into
  the existing `🤖[]` marker family using the shared marker codec (`ARCH-DRY`).
- For rendered SVG and similar structured visuals, support comments anchored to
  stable element IDs where available, falling back to geometry plus content hash
  when needed.
- For freeform tablet drawing, capture annotations against a screenshot/render
  snapshot and store them as sidecar records rather than destructive source-file
  edits.
- Provide an agent handoff path: selected annotations can become Pair review
  requests or draft prompts, but arbitrary file mutation remains gated through
  existing review/agent workflows.
- Include conflict/staleness detection: when the underlying artifact changes,
  annotations must show whether their anchor still matches, drifted, or needs
  manual reconciliation.

## Done when

- A browser/tablet can add comments to markdown/text artifact renders and export
  them into valid `🤖[]`-style markers.
- SVG or rendered visual artifacts can receive anchored comments without editing
  the source file.
- Freeform screenshot annotations are stored and displayed as sidecars tied to a
  render/content identity.
- Changed artifacts surface stale/drifted annotation state instead of silently
  applying comments to the wrong content.
- Tests cover marker compilation/escaping, sidecar serialization, anchor
  resolution, stale-anchor detection, and the agent handoff path.

## Plan

- [ ] Define annotation sidecar schema and artifact identity model.
- [ ] Implement markdown/text comment capture and `🤖[]` marker compilation.
- [ ] Implement SVG/visual anchor capture and resolution.
- [ ] Implement screenshot/freeform annotation sidecars.
- [ ] Add stale-anchor detection and agent handoff integration.

## Log

### 2026-07-26

Seeded as the tablet-native layer after artifact viewing. It deliberately builds
on existing marker semantics for text while keeping visual/freeform notes as
sidecars.
