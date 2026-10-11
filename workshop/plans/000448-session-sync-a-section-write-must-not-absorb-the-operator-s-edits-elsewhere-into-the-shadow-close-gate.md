---
gate: boundary-review
issue: 448
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T12:42:46-07:00"
      dispatched: "2026-10-11T12:40:39-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Agent SKILL.md still says diff is "since your last write" and does not warn that read --all now empties diff
          detail: 'cmd/internal/couchcmd/skills/couch/SKILL.md:267. An agent that runs read --all and then diff sees nothing, which is the symptom this issue fixes. Reword the line and add "run diff before read --all". Same stale statement in comments: file.go:24, sync.go:301 (DiffText doc), sync.go:111 (hasShadow comment).'
          family: shadow-meaning-restated-stale
          round: 1
        - id: BR-2
          severity: Important
          title: shadowAfter's no-shadow branch (!f.hasShadow) is untested; removing it keeps the suite green
          detail: 'With the mutation, the first section write with no shadow stores an empty shadow and diff shows the whole file. Add a test: remove the shadow, write a section, assert shadow == file and diff is empty.'
          family: untested-branch
          round: 1
        - id: BR-3
          severity: Minor
          title: Code comments file.go:24, sync.go:301, sync.go:111 describe the shadow as "the agent's last write"
          detail: The shadow now also reflects read --all, and a section write updates only that section. Update the three comments together with SKILL.md.
          family: shadow-meaning-restated-stale
          round: 1
        - id: BR-4
          severity: Minor
          title: 'Two shadow write paths: saveShadow (Read) and a direct writeAtomic in write (sync.go:401)'
          detail: write does not update f.shadow and f.hasShadow the way saveShadow does. Harmless today; one helper would keep the shadow's invariants in one place.
          family: single-shadow-writer
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T12:45:03-07:00"
      dispatched: "2026-10-11T12:44:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: SKILL.md:264,267 reworded and the Unseen edits bullet (284-287) added; DiffText, Write and hasShadow comments updated at sync.go:110,301,320.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestASectionWriteWithoutAShadowStoresTheWholeFile added; removing the hasShadow branch makes it FAIL (verified by mutation), restored makes it pass.
          round: 2
        - id: BR-3
          disposition: addressed
          note: file.go:24 says the shadow is the file as the agent last saw it; sync.go:110-111 and the DiffText doc agree.
          round: 2
        - id: BR-4
          disposition: addressed
          note: write now calls f.saveShadow (sync.go:401); saveShadow is the only writeAtomic to paths.Shadow and keeps f.shadow and f.hasShadow in step.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: b26fec8c997d7ec790d11b10630416a80b517c0c
      blocked: false
---

# Gate ledger — pair#448 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T12:42:46-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `shadow-meaning-restated-stale` Agent SKILL.md still says diff is "since your last write" and does not warn that read --all now empties diff
  cmd/internal/couchcmd/skills/couch/SKILL.md:267. An agent that runs read --all and then diff sees nothing, which is the symptom this issue fixes. Reword the line and add "run diff before read --all". Same stale statement in comments: file.go:24, sync.go:301 (DiffText doc), sync.go:111 (hasShadow comment).
- **BR-2** [Important] `untested-branch` shadowAfter's no-shadow branch (!f.hasShadow) is untested; removing it keeps the suite green
  With the mutation, the first section write with no shadow stores an empty shadow and diff shows the whole file. Add a test: remove the shadow, write a section, assert shadow == file and diff is empty.
- **BR-3** [Minor] `shadow-meaning-restated-stale` Code comments file.go:24, sync.go:301, sync.go:111 describe the shadow as "the agent's last write"
  The shadow now also reflects read --all, and a section write updates only that section. Update the three comments together with SKILL.md.
- **BR-4** [Minor] `single-shadow-writer` Two shadow write paths: saveShadow (Read) and a direct writeAtomic in write (sync.go:401)
  write does not update f.shadow and f.hasShadow the way saveShadow does. Harmless today; one helper would keep the shadow's invariants in one place.

## Round 2 — 2026-10-11T12:45:03-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — SKILL.md:264,267 reworded and the Unseen edits bullet (284-287) added; DiffText, Write and hasShadow comments updated at sync.go:110,301,320.
- BR-2 — addressed — TestASectionWriteWithoutAShadowStoresTheWholeFile added; removing the hasShadow branch makes it FAIL (verified by mutation), restored makes it pass.
- BR-3 — addressed — file.go:24 says the shadow is the file as the agent last saw it; sync.go:110-111 and the DiffText doc agree.
- BR-4 — addressed — write now calls f.saveShadow (sync.go:401); saveShadow is the only writeAtomic to paths.Shadow and keeps f.shadow and f.hasShadow in step.

## Open findings

(none — every finding has been disposed)
