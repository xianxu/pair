---
id: 000319
status: working
deps: [pair#317]
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
started: 2026-09-24T14:13:32-07:00
flow: {kind: quick, provenance: inferred, spec: "a79ba2df", done: "e5aff577"}
---

# Slot glyph shows behind and diverged resting branches

## Problem

The #317 slot glyph marks a resting branch that is *ahead* of its upstream
(`+`) but says nothing when it is *behind*. A slot that has fallen behind
origin/main is a poor place to start work or move a branch into without a pull
first, and one that has diverged (as `main-slot1` did during #317: ahead 3,
behind 34) needs a rebase or a publish before either direction is safe. Today
both look clean.

## Spec

Extend the resting-branch divergence glyph from one state to three. It still
compares the checked-out resting branch against its configured upstream
(`origin/main` today), as in #317:

| glyph | meaning | example |
|---|---|---|
| `+` | ahead only (unpublished commits) | `pair+` |
| `-` | behind only (upstream has commits you don't) | `pair-` |
| `±` | both ahead and behind (diverged) | `pair±` |

- **Data:** porcelain v2's `# branch.ab +A -B` already carries the behind
  count, and `ParseSlotGitStatus` currently discards it. Add `Behind` to
  `couchcore.SlotGitStatus`; no extra git call.
- **Precedence:** `SlotGlyph` keeps a single glyph per slot. Proposed order:
  off-resting `` > dirty `*` > `±` > `+` > `-` > none. Dirty wins because
  uncommitted work is the most urgent fact about a slot; `±` beats `+` because it
  is the strictly stronger claim.
- **Staleness:** "behind" is only as fresh as the last `git fetch` of that
  checkout. The glyph reports the local remote-tracking ref and must not fetch
  (no network on a 10s background tick, ARCH-CONSTRAINTS). Document this in
  the README glyph table.
- **No upstream:** unchanged. No evidence means no glyph.

Open questions for design:
- Should dirty combine with divergence instead of hiding it (e.g. `*±`)? That
  would make it two glyphs per slot; the proposal keeps one.
- Is `±` (U+00B1, one column) safe across the operator's terminals? It is not a
  Nerd Font glyph, so it should be. Confirm `textwidth` measures it as one
  column.

## Done when

- `SlotGlyph` table tests cover ahead-only `+`, behind-only `-`, diverged `±`,
  dirty-over-divergence, and no-upstream; `ParseSlotGitStatus` keeps `Behind`
  and still rejects a malformed `branch.ab`.
- Golden fixtures (`couchtty/testdata/slots_grouped_glyph_*`) add behind and
  diverged scenarios; the switcher and the tab bar agree.
- README glyph table and `atlas/couch.md` list the new glyphs and the
  fetch-staleness caveat.
- Live check: a slot whose resting branch is behind shows `-`; commit on it
  without pulling and it shows `±` within the refresh interval.

## Plan

- [x] `couchcore/slotgit.go`: `SlotGitStatus.Behind` from `branch.ab`; `SlotGlyph` precedence `` > `*` > `±` > `+` > `-`; table tests (TDD)
- [x] Goldens: add `glyph_behind` and `glyph_diverged` scenarios to `TestGroupedRenderedFixtures`; eyeball switcher/tab agreement
- [x] README glyph table + `atlas/couch.md` (#317 section): new glyphs + fetch-staleness caveat
- [ ] Mutation-check the new precedence cases; `make test` (unsandboxed); operator live check

## Revisions

- 2026-09-24 — open questions resolved at start-plan: one glyph per slot
  (dirty hides divergence, as proposed; a combined `*±` stays out of scope);
  `±` (U+00B1) measures one column in `textwidth` (not in its wide ranges).

## Log

### 2026-09-24

- Filed from the #317 session, after `main-slot1` diverged from origin/main
  (ahead 3, behind 34) while showing no glyph. Builds on #317's
  `couchcore/slotgit.go`; small, likely quick-flow.
- Implemented (fd1bb165). `SlotGitStatus.Behind` from `branch.ab`;
  `SlotGlyph` checks `HasUpstream` once, then `±` > `+` > `-`. New goldens
  `glyph_behind` / `glyph_diverged`; the other ten fixtures are byte-unchanged.
  No in-app legend lists the glyphs (README + atlas only), so no help text moved.
- Mutation checks (restored by `cp`, `cmp`-verified): drop the `±` case →
  precedence + fixture fail; parser drops `Behind` → parse test fails; drop the
  `-` case → precedence + fixture fail.

