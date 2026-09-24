---
id: 000317
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T23:22:18-07:00
flow: {kind: quick, provenance: inferred, spec: "11433814", done: "fa95a3c0"}
---

# Slot quick-status glyph in Couch tab bar and switcher

## Problem

The switcher and tab bar (#307) show each slot's address (`pair:1`, `:2`) and
liveness, but not whether the slot has work in it. Deciding where to start
something, or which slot can take a moved branch ("move this branch to :N",
ariadne#248), means opening each slot or running git by hand.

## Spec

Show one quick-status glyph per slot, taking the first rule that matches:

| glyph | meaning |
|---|---|
| `` (U+E0A0, Powerline branch) | slot is **not** on its resting branch, so it has issue work |
| `*` | on its resting branch, but the working tree is dirty |
| `+` | on its resting branch and clean, with commits not on origin/main |
| none | on its resting branch, clean, nothing unpublished |

- **Where:** the switcher's slot rows (next to `pair:1`) and the tab bar
  entries (`:1`, `:2`). Both views come from the shared presentation path
  (`couchtty/thread_presentation.go` `PresentThreads`), so derive the glyph
  once there and have both consume it (ARCH-DRY, same shape as #197/#236).
- **Data:** `branch` and `resting_branch` already arrive through
  `couchcore.WorkspaceIdentity` (`sdlc workspace`). Dirtiness and
  ahead-of-origin need two more facts per slot: porcelain status (tracked and
  non-ignored untracked, matching ariadne's workspace-branching preflight) and
  `rev-list --count origin/main..<resting>`, using the resting branch's
  configured upstream rather than a hard-coded `origin`.
- **Pure core (ARCH-PURE):** `slotGlyph(branch, resting, dirty, ahead) string`
  holds the precedence and gets table tests. Git reads sit behind the existing
  workspace-identity IO seam.
- **Cost envelope (ARCH-CONSTRAINTS):** git status in a large tree takes
  tens to hundreds of ms, so never run it on the render/keystroke path.
  Refresh on a timer and on focus/switch, cache per slot, and render the
  cached value. A stale glyph is acceptable; a blocked keypress is not. Bound
  the refresh and keep the last value on failure (or show no glyph), never an
  error in chrome.

Open questions for design:
- Does `:0` (resting `main`) get the same glyph? Proposed: yes, same rules.
- Is a Nerd Font glyph acceptable everywhere Couch runs, or is an ASCII
  fallback needed (e.g. `^`)? The operator's terminal renders it today.
- "Not on origin/main": count only the resting branch, or any local branch?
  Proposed: resting branch only, since an off-resting slot already shows ``.

## Done when

- Primary :0 and numbered slots use the shared Nerd Font branch glyph; on the resting branch, ahead is measured against its configured upstream (no upstream means no ahead glyph). One bounded porcelain-v2 status read supplies the facts.
- The switcher and tab bar show the glyph by the precedence above, from one
  derivation; golden testdata covers each of the four states and a narrow
  layout.
- `slotGlyph` is table-tested; the refresh path has a stateful fake for slow
  and failed git reads that shows the render never blocks.
- Live check: move a branch into :1, dirty :2, commit unpublished work on :0's
  `main`. The glyphs update within the refresh interval.

## Plan

Durable plan: `workshop/plans/000317-slot-quick-status-glyph-plan.md`.

- [x] Pure core: `SlotGitStatus`, `ParseSlotGitStatus`, `SlotGlyph`, `RestingBranch` (couchcore/slotgit.go), table-tested; `validate()` uses `RestingBranch`
- [x] One derivation: `PresentThreads(rows, MenuState.SlotGit)` sets `Glyph`; switcher + tab bar render it; glyph goldens (four states + narrow)
- [x] Background refresh: single-flight `RefreshSchedule` owner on Console.Run (10s ticker + switcher open + switch + inventory landed); stateful fake probe proves no render blocking, failure keeps last value, shutdown joins
- [x] Atlas/README and build verified; operator live smoke passed

## Revisions

- 2026-09-23 — data source. The Spec assumed `branch`/`resting_branch` reach
  presentation via `WorkspaceIdentity`; they don't (sdlc workspace runs only on
  explicit operations, uncached). Delta: one
  `git --no-optional-locks status --porcelain=v2 --branch` per checkout supplies
  branch, upstream ahead and dirtiness in one read; resting branch comes from
  `couchcore.RestingBranch(n)`, the same helper `WorkspaceIdentity.validate()`
  uses. Open questions resolved: `:0` same rules; Nerd Font glyph, no ASCII
  fallback (one constant); `+` counts the checked-out resting branch against its
  configured upstream, none shown without an upstream.

## Log

### 2026-09-23

- Filed from the #316 session. Related: pair#307 (grouped slot display),
  #197/#236 (single-derivation labels), ariadne#248 ("move this branch to :N"
  needs a quick view of which slots are free), couch-slots-v2 project.

### 2026-09-24

- Implemented per plan. Triggers deviate slightly (logged in atlas): the switcher
  opening reaches the slot-git pass through the landed inventory
  (`finishMenuRefresh` → `requestSlotGit`), not a separate `onHotkey` hook; the
  switch trigger lives in `switchTo`. No probe installed ⇒ no pass (else empty
  passes polluted `COUCH_TRACE`, caught by `TestTheConsoleTracesStartup…`).
- Probe set reads `menuRows(c.menu)`, per the inventory-lookup guard test.
- Mutation checks (each restored byte-identically, `cmp`-verified): tab-bar
  `label += a.Glyph` removed → glyph fixtures fail; `:0` glyph branch removed →
  fixture + scope test fail; probe run under `c.mu` → render test hangs (timeout
  FAIL); failed probe overwrites → reducer + console failure tests fail; each
  of inventory / switch / ticker triggers removed → its trigger test fails.
- Process slip: a mutation was reverted with `git checkout <file>`, which also
  discarded uncommitted edits; restored from the pre-mutation copy. Lesson added.


### 2026-09-24 — landing verification

Operator confirmed glyphs appeared in live smoke testing and requested landing.
Published origin/main merged cleanly before verification. Focused glyph/parser/
refresh race tests pass for couchtty and couchcore; make build passes. Initial
headless checks failed without diagnostic output; after removing inherited
PAIR_/COUCH_ environment variables, the continuing make -k test run passed all
shell/editor targets and reached the full Go suite. Final result follows.

### 2026-09-24 — acceptance wording aligned

Reason: the implemented and previously approved data-source revision should be
explicit in Done when. Delta: record :0 parity, the configured-upstream rule,
the Nerd Font decision and the single porcelain-v2 read. No scope change.

Final verification: clean-environment `make -k test` exited 0, including the full Go suite and shell/editor checks; `make build` and focused couchtty/couchcore race tests exited 0.

### 2026-09-24 — review parser correction

BR-1: reject empty/duplicate branch.head and upstream headers, duplicate branch.ab,
trailing tokens, signed or malformed counts and overflow. ParseUint validates the
complete decimal fields. Added regression cases failed before the fix and now pass.
Focused couchcore/couchtty glyph tests pass under race (3.787s/4.715s); make build
passes. The preceding full make -k test and independent reviewer full Go suite
passed. ARCH-SECURE: malformed observations remain probe failures, retaining the
last display value through the existing reducer. No new lifecycle state.

### 2026-09-24 — final documentation correction

Second review confirmed BR-1 is addressed in prose, but omitted the structured
findings disposition required by SDLC. Its only new finding is corrected: the
README, atlas and issue legend now show the literal branch glyph `` (U+E0A0).
No production code changed after the verified parser correction.
