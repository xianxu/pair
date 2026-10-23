---
id: 000802
status: done
deps: [789]
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: 'db235ceba36211a31f15629d23f3ae589de51927' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T08:16:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
actual_hours: 1.29
---

# Couch: shorten common statuses

## Problem

Common Couch chrome status sentences are longer than needed. The 40-cell display cap in #789 already protects layout, so cosmetic wording does not need an exhaustive global proof.

## Spec

Low-priority follow-up authorized by TL receipt58326574-1d4c-4767-b222-cfb4d5adddce. Shorten a handful of frequently seen built-in messages, starting with wait notices and lifecycle labels; choose the concrete set when taking the issue. Keep the action and essential meaning clear, retain dynamic diagnostic details and existing full-copy recovery. No exhaustive producer catalog, transitive sweep, or total-catalog proof is required. The display cap remains the layout invariant.

Useful starting points: couchtty common wait/broadcast/cast feedback; couchcmd supervision and lifecycle notices. Earlier exploratory wording changes and runtime variant experiments are visible in #789's branch history (797d86d9108817b9a8d5374460a96a53f96133be), but are deliberately removed from its final display-path diff. Do not carry that broad catalog scope into this issue.

### Selected set (2026-10-23)

Seven strings in three production files, picked for frequency and for static frame the surrounding UI already implies. Every dynamic part stays — slot, minutes, screen summary, mute path, unverified count, cast kind and link — and so does every action: peek, mute, hover, rotate.

| Site | Now | New |
|---|---|---|
| wait_notice.go:201 | `slot %s waiting %d min %s: %s (couch --peek %s; mute: add %s to %s)` | `slot %s waiting %dm %s: %s (couch --peek %s; mute: add %s to %s)` |
| wait_notice.go:36 `waitIdle` | `idle off its resting branch` | `idle off resting branch` |
| lifecycle_notices.go:179 | `Couch lifecycle: %d observations lacked verified slot metadata at restart drain` | `Lifecycle: %d slots' metadata unverified at restart drain` |
| lifecycle_notices.go:232 | `Couch lifecycle: hub projection capacity reached; inspect Slots` | `Lifecycle: hub projection full; inspect Slots` |
| console_castcodes.go:397 | `New %s %s; its viewers were disconnected. Hover %s to see it.` | `New %s %s; viewers disconnected. Hover %s for code.` |
| console_castcodes.go:485 (capped) | `Too many wrong codes on the %s %s link: its code changed; the link is locked until you rotate its code.` | `%s %s link: too many wrong codes; code changed, link locked until you rotate its code.` |
| console_castcodes.go:487 (uncapped) | `Too many wrong codes on the %s %s link: its code changed. Hover %s to see it.` | `%s %s link: too many wrong codes; code changed. Hover %s for code.` |

Not taken: the repeated "Viewers need its code: hover … to see it" suffix (five sites in three more files) and the webcast link `grants` sentences. That is the catalog scope #789 deliberately removed; this issue stays with the seven above.

## Done when

- A small, explicitly named set of frequent statuses has concise, actionable wording.
- Focused tests pin those selected messages and preserve relevant dynamic diagnostics; no total-catalog assertion is added.
- The existing cap and full-text copy behavior are unchanged.

## Plan

- [x] When #789 lands, merge origin/main into this branch and re-run the cap and full-copy tests before any wording edit; #789 also edits console_castcodes_test.go, so the merge comes first.
- [x] Apply the seven selected strings; no other producer is touched.
- [x] Update the focused pins: wait_notice_test.go (body prefix and suffix), lifecycle_notices_test.go (drain diagnostic; pin the capacity diagnostic if it has none), console_castcodes_test.go (three notices). No total-catalog assertion.
- [x] go vet and go test for cmd/internal/couchcmd and cmd/internal/couchtty.
- [x] make test-affected-inventory; the changed test readers' dispositions are recorded.
- [x] sdlc close with the package runs as evidence; the cap and full-copy behavior is unchanged.

## Log

### 2026-10-22

Split from #789 at the TL's direction; low priority, unclaimed. #789 lands cap plus full-copy only. Pair:4 takes #796 after #789, not this cleanup.

### 2026-10-23
- 2026-10-23: closed — go test ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1 green (106s/48s); go vet clean; make test-affected-inventory green; boundary-review Important finding (embedded SKILL.md old wording) fixed in 4c0239406 with a repo-wide restatement sweep; cap and full-copy behavior untouched.; review verdict: SHIP

Claimed and planned as rung 3 at the hub's direction. #789 is still unlanded (PR 543 open), so this branch carries design only until it lands; code starts on the merged tree. The seven strings cover the issue's three starting points: wait notice, lifecycle labels, cast feedback. The repeated viewers-code suffix was rejected as the catalog scope #789 removed. No estimate: this is inside the quick-flow shell.

### 2026-10-23 (implementation)

#789 landed 10:56 (PR 543); merged origin/main into the branch (c013b0c36) and re-ran the cap and full-copy pins green before editing. Applied the seven strings verbatim, then found a fourth reader the plan missed: `picker_notice_test.go` pinned the wait notice's `1 min` prefix, caught by the full package run, not the focused pins. The inventory flags only `wait_notice_test.go` (the other changed tests read no repository paths); audited its sole input, the literal `../wrapcmd/testdata/drive/` fixture, unchanged, and recorded it as `fixed_scanner`.

Evidence: `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1` green (couchtty 48s, couchcmd 106s after the picker fix); `go vet` both packages clean; `make test-affected-inventory` green. Atlas `couch.md` wait-notice body shape updated to `Mm`. The drain diagnostic renders `1 slots'` at N=1 as approved; flagged to the TL.

Boundary review verdict FIX-THEN-SHIP with one Important finding: the embedded `skills/couch/SKILL.md` still quoted `waiting M min` and `idle off its resting branch` — a missed restatement of the same format, the class the atlas edit should have swept. Fixed in 4c0239406 after a repo-wide sweep found no other live copy (the remaining hits are archived issues and a descriptive test comment). The review's Minor formatting note on the reader certificate stands: the inventory tool itself writes one-line JSON, and siblings are mixed, so no change.

## Revisions

- **2026-10-23 08:20 — plan review steers.** Reason: the TL approved the set with two corrections. Delta: the wait-notice body keeps `mute: add %s to %s`, because "mute: X in PATH" reads as a state rather than an action, and shortens only `min` to `m`; the drain diagnostic becomes `Lifecycle: %d slots' metadata unverified at restart drain`, because "N slot metadata" does not parse. The other five strings stand as written, and the rejected scope stays out.
