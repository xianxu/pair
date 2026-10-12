# Boundary Review — pair#461 (whole-issue close)

| field | value |
|-------|-------|
| issue | 461 — broadcast-web: notes model — anchors, focused context, structure-snapped crops |
| repo | pair |
| issue file | workshop/issues/000461-broadcast-web-notes-model-anchors-focused-context-structure-snapped-crops.md |
| boundary | whole-issue close |
| milestone | — |
| window | f1fbac36ac8eb920629962f2152fb781634f37e5..608a42e167951799114603000ef2d6ac94d274d6 |
| command | sdlc close --issue 461 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:53:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Notes-model tests pass. Now I'm checking whether the artifactpath failures are environmental (generated assets that aren't present in this checkout).

I've read the tests. Now I'm writing a quick probe for two edge cases: template content and a test attribute whose value contains the separator.

The probe confirms that a test attribute containing ` > ` produces a selector that matches nothing; I've removed the probe file. Next I'm checking the crop and landmark tests and the atlas section.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The notes model delivers what the Spec asks for. `notes.Structural` is pure over `recording.State`, implements #457's `Model` without changing the seam, runs `RunModelContract`, and is tested on all fixtures, including the blog cases for header snapping, the tall `li`, and clustering. `go test ./cmd/internal/webcast/...` is green. The two `artifactpath` failures (`TestProductionArtifactReferencesAreExactlyClassified`, `TestCompositeBindingsIsolateGoShellNeovimAndBothLayouts`) are environmental. They come from `cmd/internal/runtimebundle/assets/`, which is gitignored and hasn't been built in this checkout. Neither failure mentions the four new notes sources, which are listed in `NonArtifactSources`. One cheap correctness fix is worth making before close. A probe confirmed that the selector generator can emit selectors its own matcher can't parse, which breaks the "made and matched in one file so they can't drift" contract.

### 1. Strengths
- **Selector generation and matching in one file.** `cmd/internal/webcast/notes/selector.go` keeps both together. Its round-trip test, `selector_test.go` `TestSelectorRoundTripOnFixtures`, checks every element across shadow roots and frames, and fails if no deep element was checked.
- **Crops match the #455 pane rule.** Snapping is per axis, and a tall landmark widens only horizontally (`crop.go:65-71`). Clipping widens the page box to each note's own box (`crop.go:25`), so a crop never cuts its own note.
- **Shared live-notes rule.** `liveNotes` is extracted in `fake.go`, so "notes on the current document" has one definition across the fake and the real model (ARCH-DRY).
- **Re-rendered elements are handled.** `Crops` and `Context` go through `Resolve`, so a re-rendered element still finds its landmark. This is pinned by `TestResolveAfterARerender`.
- **Seam problems were reported, not forked.** The region note and `Context` JSON tags went to the TL as seam questions instead of being changed locally.

### 2. Critical findings
None.

### 3. Important findings
- **Test-attribute values that contain the separator produce unmatchable selectors** (`selector.go:72-78`). `uniqueStep` rejects values containing `"` but not ones containing ` > ` or ` >>> `. A probe gave `[data-testid="a > b"]` for a uniquely tested button, and `matchSelector` returned 0 matches, because the selector string is split on the separators before steps are parsed.
  - Effect: after a re-render, such an anchor can't resolve, even though its element is still uniquely named.
  - Fix: in `uniqueStep`, skip any value containing `deepSep`/`childSep` (or just `>`), the same way `"` is skipped, and add a builder case to `TestSelectorSteps`.
  - The rule behind this is: never emit what the matcher can't match. Ids are already safe because `identRe` excludes spaces.

### 4. Minor findings
- **Template content round trip.** `selectorFor` produces a `host >>> path` selector for nodes inside template content (its comment says so at `selector.go:55`), but `matchSelector` only steps into `Shadow`/`Frame`, not `Content`. The round-trip test's `walkElements` skips `Content` too. Either stop `Anchor` from accepting template-content nodes, or have the matcher step into `Content`.
- **`Context.Landmark` doc mismatch.** The doc in #457's `model.go` says "Node itself when none is closer", but `landmark()` returns the parent element (the atlas and plan say parent). Report this to the TL so the seam doc matches.
- **Atlas wording vs code.** The atlas says crops "less than a margin apart merge", but `cluster` uses `<= margin`.
- **`anchorBox` ignores the resolved node.** With `at.Box == nil` and `at.Node` gone, `Crops` errors even when `Resolve` would find the replacement. This matches the plan as written, and `Anchor` always fills `Box` when layout is known, so it's low impact.
- **Plan checkboxes.** The task checkboxes in the plan file are still `[ ]`, while the issue's Plan list is all `[x]`.

### 5. Test coverage notes
- Contract, fixture resolve/context/crop, landmark rule table, the 500-character window, clustering, clipping, the given box and hints are all covered.
- The log's mutation checks are believable, given that the assertions are specific (for example, the crop's right edge at 700 and full width at 1280).
- Gaps:
  - selector values containing the separator (the Important finding above);
  - the template-content round trip;
  - `Crops` with an empty `ats` slice. It returns `nil, nil`, which is fine but not pinned.

### 6. Architectural notes
- **ARCH-DRY: pass.** `liveNotes` and `trim` are shared with the fake, and there is no duplicated logic.
- **ARCH-PURE: pass.** Every file is pure over `State`, and the tests use builders and fixtures without mocks.
- **ARCH-PURPOSE: pass.** All the Spec bullets are delivered. Region-note crops are explicitly a seam gap that was escalated, not a hidden deferral of the main point.
- **ARCH-MOCK: N/A.** There are no external dependencies. The real model runs the same contract as `FakeModel`.
- **ARCH-CONSTRAINTS: pass.** Cost is O(nodes) per call. `uniqueStep` walks the whole tree for each ancestor, so `selectorFor` can be O(depth·nodes). That's fine at per-note frequency, but it would need indexing if anchors were computed per pointer-move later.
- **ARCH-SECURE: flag (Minor).** `at.Sel` from a recording is treated defensively: anything outside the subset matches nothing. The one gap is the generator-side asymmetry in the Important finding.
- **ARCH-ORDER: N/A.** It holds no state between events; every call is a function of `State`.
- **ARCH-FUNERAL: N/A.** It creates nothing durable.

### 7. Plan revision recommendations
- Add a Revisions entry: "Selector steps skip test-attribute values containing `>` (the combinator), so a generated selector always matches."
- Either note that template content isn't anchorable, or that the matcher steps into it.
- Tick Tasks 1-4 in the plan file.

```findings
findings:
  - id: new
    severity: Important
    family: selector-generator-matcher-symmetry
    title: |
      selectorFor emits [data-testid="..."] steps whose value contains " > " or " >>> ", which matchSelector can never match
    detail: |
      selector.go:72-78 rejects only a quote in the value; matchSelector splits on the separators before parsing steps. Probe: data-testid="a > b" gives a selector matching 0 elements, so re-render resolve fails. Skip values containing ">" and add a TestSelectorSteps case.
  - id: new
    severity: Minor
    family: selector-generator-matcher-symmetry
    title: |
      Template-content nodes get a host >>> path selector, but matchSelector never steps into Content
    detail: |
      selector.go:55 handles template content when generating, but the matcher (and the walkElements round-trip helper) covers only Shadow and Frame. Either refuse to anchor such nodes or match into Content.
  - id: new
    severity: Minor
    family: seam-doc-drift
    title: |
      model.go's Context.Landmark doc says Node itself when none is closer; Structural returns the parent element
    detail: |
      The atlas and plan describe the parent element (body at most). Report to the TL so the doc for #457's seam matches.
  - id: new
    severity: Minor
    family: seam-doc-drift
    title: |
      Atlas says crops less than a margin apart merge; cluster merges at a gap <= margin
  - id: new
    severity: Minor
    family: plan-bookkeeping
    title: |
      Plan file task checkboxes remain unchecked although the issue marks all four tasks done
```

---

## Re-review — 2026-10-12T01:56:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 461 — broadcast-web: notes model — anchors, focused context, structure-snapped crops |
| repo | pair |
| issue file | workshop/issues/000461-broadcast-web-notes-model-anchors-focused-context-structure-snapped-crops.md |
| boundary | whole-issue close |
| milestone | — |
| window | f1fbac36ac8eb920629962f2152fb781634f37e5..62293ee2868491c4e9e7b401407ab777a0cd5779 |
| command | sdlc close --issue 461 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:56:28-07:00 |
| verdict | SHIP |

## Review

Quoting trouble; I'll use a Python helper for the edits.

```verdict
verdict: SHIP
confidence: high
```

All five findings from round 1 are fixed, and I checked each one in the code, not from the commit messages. For BR-1 the value check now skips test-attribute values containing a quote or `>` (`selector.go:68`). There is also a general safeguard: `Anchor` drops any selector that doesn't find its own element (`structural.go:27-29`), so the rule holds even for inputs the value check didn't anticipate. That fixes the class, not just the reported case. I undid each behaviour fix in the throwaway checkout, one at a time, and each time its regression test failed:
- **Value check:** `TestSelectorSteps` fails on the `"a > b"` case.
- **Template content in the matcher:** `TestSelectorSteps` fails on `#row >>> b`.
- **`Anchor` safeguard:** `TestAnchorDropsASelectorThatCantRoundTrip` fails.
- **`anchorBox` using the element the anchor resolves to now:** `TestResolveAfterARerender` fails at line 95.

`go test ./cmd/internal/webcast/...` passes with everything restored, and `git status` shows only the issue file that was already modified. One new Minor finding: two code comments still describe the old behaviour that this round changed. That is the third finding in family `seam-doc-drift`. It doesn't block.

1. **Strengths**
   - `structural.go:27-29`: the safeguard checks the matcher's actual result, so the rule "a stored selector always round-trips" doesn't depend on predicting every bad value.
   - `selector.go` keeps the selector generator and matcher in one file, sharing `stepRe`, `matchStep` and `typeIndex`. `TestSelectorRoundTripOnFixtures` checks every element of the real fixtures, including those inside shadow roots and frames.
   - `crop.go:21-22`: the crop code resolves the anchor once and uses that result for both the box and the landmark, so the re-render case is consistent.
   - The new tests check observable round-trip behaviour (whether a selector finds its element); they don't restate the code.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `crop.go:86`: the `cluster` comment still says "less than margin apart", but the code merges at `<= margin`. That is the same mismatch BR-4 fixed in the atlas.
   - `selector.go:85`: the comment says "shadow root or frame document" but the code now also steps into template content.
   - Both are the 3rd finding in `seam-doc-drift`. The rule that covers all three: a commit that changes behaviour must grep and update every prose restatement of it in the same commit (doc comments in the touched files, the atlas, the plan). Rather than patching these two lines one by one, sweep the `notes/*.go` doc comments against the current atlas section.
   - `landmark.go:36-39`: when there is no container and `n` is `body`, `landmark` returns `n` itself, so the new `model.go` doc ("with none, the parent element") has a one-node edge case. Not worth a finding.

5. **Test coverage:** every fix in this round has a regression test, and each one fails without its fix. The pure entities are tested on builder fixtures with no IO.

6. **Architecture**
   - **ARCH-DRY: pass.** The generator and matcher share their step helpers.
   - **ARCH-PURE: pass.** Everything is a pure function over `recording.State`.
   - **ARCH-PURPOSE: pass.** The BR-1 fix covers the whole class through the round-trip safeguard. The template sibling (BR-2) was fixed in the same round.
   - **ARCH-MOCK: N/A.** There are no external dependencies.
   - **ARCH-CONSTRAINTS: pass.** `uniqueStep` walks the tree once per ancestor step, and `Anchor` adds one more match per note. That runs once per note, not on a hot path.
   - **ARCH-SECURE: pass.** A stored selector is re-read through an anchored regex. Anything outside the subset matches nothing, and the note falls back to resolving by node id.
   - **ARCH-ORDER: N/A.** The code holds no state between events; every call computes from the `State` it is given.
   - **ARCH-FUNERAL: N/A.** Nothing durable is created; every value is computed per call.

7. **Plan revisions:** none needed. The 2026-10-12 Revisions entry matches the code, and all task checkboxes are now ticked.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      selector.go:68 skips quote/`>` values and structural.go:27 drops any selector that doesn't round-trip; reverting either turns TestSelectorSteps / TestAnchorDropsASelectorThatCantRoundTrip red.
  - id: BR-2
    disposition: addressed
    note: |
      matchSelector steps into h.Content (selector.go:88) and walkElements covers Content; the `#row >>> b` case fails when the change is reverted.
  - id: BR-3
    disposition: addressed
    note: |
      model.go Context.Landmark doc now says the parent element when there is no landmark, matching landmark.go:36-39 and the atlas.
  - id: BR-4
    disposition: addressed
    note: |
      The atlas now says "at most a margin apart"; the sibling code comment crop.go:86 is still stale and is raised as a new seam-doc-drift finding.
  - id: BR-5
    disposition: addressed
    note: |
      All plan Task 1-4 checkboxes are ticked in 62293ee2.
findings:
  - id: new
    severity: Minor
    family: seam-doc-drift
    title: |
      Code comments still describe the old behaviour this round changed (crop.go:86 "less than margin", selector.go:85 omits template content)
    detail: |
      3rd finding in seam-doc-drift. Rule: a behaviour-changing commit must grep and update every prose restatement (doc comments, atlas, plan) in the same commit. Sweep notes/*.go comments against the atlas notes-model section, not just these two lines.
```

---

## Re-review — 2026-10-12T01:59:11-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 461 — broadcast-web: notes model — anchors, focused context, structure-snapped crops |
| repo | pair |
| issue file | workshop/issues/000461-broadcast-web-notes-model-anchors-focused-context-structure-snapped-crops.md |
| boundary | whole-issue close |
| milestone | — |
| window | e5caaf7de5f8eb33c4d0f0785f1986cdb357d9e4..e3ca1a7bfeb789cc14ca1c42c46f036325e9152b |
| command | sdlc close --issue 461 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T01:59:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking whether the issue file's Spec or Done-when changed (scope check), then writing the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix-then-ship, with nothing blocking.** The changes since the close review are small: two corrected code comments, a new lesson, two review/gate records, and the merge of main into `atlas/index.md` and `atlas/webcast-format.md`. The comments that BR-6 named are now correct. `crop.go:86` says "at most margin apart", which matches `gap <= margin`. `selector.go:85` and `selector.go:132-133` now mention template content, which matches the `h.Content` step and `treeRoot`.

The merge keeps both sides. In `atlas/index.md` the #461 notes-model text and the #466 remote-driving text are joined into one line. In `atlas/webcast-format.md` the "notes model (#461)" section and the "Attribution and control (#466)" section are both there, nothing is lost, no conflict markers remain, and the "(Remote driving, below)" pointer still finds `## Remote driving (#466)`.

The issue file has no Spec or Done-when changes, only status, checkboxes and Log entries, so the scope has not grown. `go test ./cmd/internal/webcast/notes/` passes.

BR-6 is only partly fixed. It asked for a sweep of every comment in `notes/*.go` against the atlas section, and one comment that disagrees with the atlas survived. It is Minor, so it does not block.

1. **Strengths**
   - The merge of the atlas files is correct and keeps all of both sections (`atlas/webcast-format.md:185-261`, `atlas/index.md:116`).
   - The new lesson in `workshop/lessons.md` states the rule behind the `seam-doc-drift` and `selector-generator-matcher-symmetry` findings, not just the one instance.
   - The comments in `selector.go` now match the code that steps into shadow roots, frames and template content.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-6 is not fully fixed (`seam-doc-drift`):** `model.go:35-37` says `Anchor` produces "its id, a stable selector, its text…". Since BR-1, `Structural.Anchor` (`structural.go:27-29`) sets `Sel` to `""` when the selector doesn't find its own element, and the atlas says so. The interface comment still promises a selector every time.
     - The header comment at `selector.go:16` also leaves out that a test-attribute value containing a quote or `>` is never used (that skip is at `selector.go:68`).
   - **Merge formatting:** `atlas/webcast-format.md:228-229` has no blank line between the "Region notes…" paragraph and the `## Attribution and control (#466)` heading. CommonMark still renders it, but it differs from every other section break in the file.

5. **Test coverage:** the window only touches comments and docs, so no new tests are needed. The notes package tests pass at the head commit.

6. **Architecture**
   - **ARCH-DRY: pass.** Nothing is duplicated.
   - **ARCH-PURE: pass.** No code logic changed.
   - **ARCH-PURPOSE: flag.** The BR-6 fix covered the two cited lines but missed others from the same sweep. That is the "fixed the instance, not the class" pattern, and it is recorded on BR-6 below.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-6
    disposition: not-addressed
    note: |
      crop.go:86 and selector.go:85/132 fixed, but the sweep is partial: model.go:35-37 still says Anchor yields "a stable selector" though Structural.Anchor blanks Sel when it does not round-trip (the atlas says so), and selector.go:16 omits the quote and > exclusion. Rule: a behaviour change must also update every comment that describes it, in the same commit.
findings:
  - id: new
    severity: Minor
    family: seam-doc-drift
    title: |
      Conflict resolution in atlas/webcast-format.md leaves no blank line before the Attribution and control heading
    detail: |
      This is the 4th finding in family seam-doc-drift; it is a formatting slip from the merge, not a new defect class. Line 228 "Region notes ... no region." runs straight into "## Attribution and control (#466)". It renders under CommonMark but differs from every other section break; add a blank line.
```
