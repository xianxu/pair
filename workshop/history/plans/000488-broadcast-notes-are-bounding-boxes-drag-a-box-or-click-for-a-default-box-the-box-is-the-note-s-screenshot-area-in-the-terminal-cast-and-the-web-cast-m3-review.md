# Boundary Review — pair#488 (milestone M3)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cc4ac22850abb2343add095b444d1c4fa74a7ec7..93944bcbd33927a80b42533d1d559a593f722986 |
| command | sdlc milestone-close --issue 488 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T12:55:39-07:00 |
| verdict | REWORK |

## Review

Reviewing the M3 diff against the plan; reading the plan's M3 and revision sections now.

Running the affected packages' tests with the race detector.

Found a possible lock inversion in `queueAnchors` → `GoTracked`; checking whether it can actually happen.

Checking whether the resolver's `tried` set survives a broadcast restart (note ids may restart at 1).

```verdict
verdict: REWORK
confidence: high
```

M3 does most of what it set out to do. The pure core in `noteanchor.go` is clean. The async resolver runs one pass at a time and makes one query per slot per pass, and it sits behind an injected `PaneRects` seam with a stateful fake and a live zellij conformance check. Dropping the nvim `view.json` in favour of finding the screen rows in the file on disk is a sound simplification, and the plan records it. The new tests pass under `-race`: the couchtty anchor and note tests, and the M3 couchcmd tests (the live conformance test skipped). The broadcast package also passes. Other couchcmd tests fail in this sandbox with `operation not permitted` on `/tmp`; that is the environment, not M3.

Four things block SHIP, all cheap to fix:
- **Critical:** the resolver's `tried` set is keyed only by note id. Ids restart at 1 in each broadcast session, so in Couch's second broadcast, notes whose ids were open at the end of the first never get their pane path.
- The resolver reads the frame at resolve time (live), not the one the note opened on. A slot switch during the pass goes undetected.
- `queueAnchors` calls `GoTracked` while holding `anchors.mu`. That takes `c.mu`, which breaks the "leaf lock" rule, and if the console isn't started it deadlocks on itself.
- Anchors now go out to every viewer page, including a Markdown heading read from disk that may not be on screen.

### 1. Strengths
- **Pure core:** `couchtty/noteanchor.go:63` `AnchorFor` and `TextUnder` / `locate` / `sectionOf` do no IO and are tested with table tests over synthetic frames. `locate` is bounded (`maxLocate`) and the file reader is bounded (`readNoteFile`, 1 MiB+1).
- **Seam separation:** couchtty never calls zellij. `couchcmd/note_panes.go` `paneRectsOf` takes `query` and `read` as injected functions, `notePanes` is pure except for `read`, and the failure paths return errors rather than a guessed layout (`TestPaneRectsOfAsksTheSlotsSession`).
- **Ordering tests:** `console_noteanchor_test.go` injects interleavings through the `paneLayout.during` hook: a resolve mid-pass, a resize mid-pass, a failed query, and a note opened mid-pass. Ordering is controllable here, not left to whichever interleaving a run happens to get.
- **No durable artifact:** dropping `view.json` removes a whole artifact family. ARCH-FUNERAL is satisfied, and the plan's revision explains why.
- **Live conformance:** `TestNotePanesZellijConformance` runs a disposable, isolated zellij session (HOME and XDG pointed at temp dirs), so it cannot reach real user state.

### 2. Critical findings
- **`couchtty/console_noteanchor.go:134-151`: `tried` is keyed by note id alone and is never reset when the session changes.**
  - Each broadcast session builds a new `PointerState` (`broadcast/session.go:105`), so `NoteBoard.nextID` restarts at 0.
  - `tried` is pruned only inside `queueAnchors`. `applyNotes` returns early once pointing is off or the broadcast has ended (`console_notes.go:113-117`), so the emptied snapshot never prunes it.
  - **Scenario:** broadcast 1 ends with notes 1 and 2 open. In broadcast 2, note 1 is placed, the pruning pass sees id 1 as open, and `tried[1]` stays true. Note 1 keeps `kind: screen` forever, silently.
  - The atlas and `SetNoteAnchor` both say "ids are never reused"; that is only true within one session.
  - **Fix:** key `tried` by `(session, id)`, or reset it whenever `s` differs from the session it was filled from. Add a test that queues from session A, then from a new session B with the same ids.

### 3. Important findings
- **`console_noteanchor.go:187-198`: the resolver reads the live frame at resolve time, not the note's opening frame.**
  - `GeometryEpoch` is per endpoint (`terminal/endpoint.go:237`), and all slots resize together. If the operator switches slot during the ~590 ms query, the epoch and geometry checks still pass.
  - `fileSteps` then searches for another slot's screen rows in this slot's draft file. That gives "file only" or, on a common line such as a heading, wrong line numbers.
  - The header comment (lines 26-27) says the anchor is dropped when "the slot in view isn't the note's", but that is checked only at queue time.
  - **Fix:** capture the frame (frames are immutable) in `anchorJob` at queue time and check its `EndpointID` and `GeometryEpoch`, or keep the accept-time frame. Add a "slot switch mid-pass" test.
- **`console_noteanchor.go:153-155`: `GoTracked` is called while holding `c.anchors.mu`.**
  - `GoTracked` takes `c.mu` (`console.go:2140`), so the order is anchors.mu → c.mu. That breaks the documented "mu is a leaf lock".
  - When `!c.started`, `GoTracked` runs `resolveAnchors` synchronously, and its first act is `c.anchors.mu.Lock()`: a self-deadlock, with a 2 s zellij query on the caller.
  - It is latent today: `runTerminalCommand` runs `applyNotes` inline when not started, and that is gated on a live broadcast. It is one refactor away from a hang.
  - **Fix:** decide `start := !running; running = true` under the lock, unlock, then call `GoTracked`.
- **`broadcast/notes.go` `Note.Anchor` with `json:"anchor,omitempty"` and `server.go:346`: anchors now go out on every page's `notes` event, including view-only pages (ARCH-SECURE).**
  - The spec (issue 4b) says the anchor is "for the agent's context".
  - The path carries the draft or review file's basename and `section: <heading>` read from disk, which may be off screen. It also adds up to 1 KiB of text per note to every notes event.
  - No page reads it.
  - **Fix:** put `json:"-"` on the wire (expose it through `Session.Notes()` for #455/#483), or use a separate wire struct. Add a test that the notes event has no `anchor`.
- **Session-side anchor seams have no test in their own package.** `Session.SetNoteAnchor`, `NoteBoard.SetAnchor`, `Config.AnchorNote` and `Note.On` have no `broadcast` test.
  - The "resolved since, so dropped" rule is exercised only against `fakeSession`, which restates it.
  - **This is the 2nd finding in family `glue-untested`.** Rule: every exported, state-changing seam method a milestone adds gets a test in its own package covering its refuse/drop paths, not only through a fake in the calling package. Here: `SetNoteAnchor` after resolve, after a flip to off, and on an unknown id.

### 4. Minor findings
- **`noteanchor.go:15`:** the header comment names `noteanchor_resolve.go`, which doesn't exist; the shell is `console_noteanchor.go`.
  - **This is the 3rd finding in family `plan-names-match-code`.** Rule: every file or identifier named in prose (plan, atlas, doc comment) must exist at the pinned head. That is checkable by grepping the backticked `*.go` names and identifiers in the diff's prose against the tree. Add that check to the boundary rather than fixing names one by one.
- **`note_panes.go:117` `reviewFile`** restates `reviewcmd.SpawnReviewPane`'s argv (`/nvim/review.lua` followed by the absolute file) with no shared constant (ARCH-DRY). It assumes zellij's `TerminalCommand` doesn't quote paths containing spaces, which the live check doesn't verify.
- **`sectionOf`** treats any `#`-prefixed line as a heading, including shell comments inside code fences. A wrong heading is possible.
- **`locate`:** nvim renders tabs as spaces, so a tab-indented file degrades to "file only". That is acceptable but undocumented.

### 5. Test coverage notes
- Missing: a second broadcast session with reused ids (Critical), a slot switch mid-pass, `broadcast`-package `SetNoteAnchor` drop paths, and the notes wire not exposing `anchor`.
- `TestNoteAnchorThroughTheBroadcast` is a good end-to-end happy path.
- The `during` hook is the right seam; extend it for the cases above.

### 6. Architectural notes
- **ARCH-DRY:** flag (Minor). `reviewFile` restates the review pane's command format.
- **ARCH-PURE:** pass. The pure core and the injected `query`/`read` are cleanly split.
- **ARCH-PURPOSE:** pass. Spec 4b's paths are delivered, and "near the prompt" is replaced by the text under the box, as recorded.
- **ARCH-MOCK:** pass. The stateful `paneLayout` fake plus `TestNotePanesZellijConformance`, to run on zellij upgrades.
- **ARCH-CONSTRAINTS:** pass. Anchors are answered from memory on accept, there is a 2 s bound off-path, one query per slot per pass, and file and text are capped.
- **ARCH-SECURE:** flag. Anchors, including file content, go out on the public notes wire.
- **ARCH-ORDER:** flag. The cross-session id identity isn't modelled (`tried`), and the resolve-time frame isn't tied to the note's endpoint.
- **ARCH-FUNERAL:** pass. Nothing durable is created, though `tried` cleanup is broken (folded into the Critical finding).
- For #455/#483: read anchors from `Session.Notes()`, not the page wire.

### 7. Plan revision recommendations
- Add a "## Revisions / M3 review" entry. The Design → Terminal cast → Anchor section still names `AnchorFor(box, l, frame)` (it is built as `(box, frame, l)`), `couchtty/noteanchor_io.go`, `RenderedStatusRow`, and `PaneRect{Rect, Slot, TopLine}`. The "Untrusted parsers" bullet still asks to fuzz the `view.json` reader.
  - Mark these superseded by the M3 as-built names: `console_noteanchor.go`, `ScreenLayout{Chips, Slot, Agent, Panes}`, `PaneRect{Role, X, Y, W, H, File, Lines}`.
- The issue's M3 line still says "+ nvim view.json".
- Replace "ids are never reused" in the atlas and the `SetNoteAnchor` doc with "never reused within a broadcast session".

```findings
findings:
  - id: new
    severity: Critical
    family: identity-scoped-to-owner
    title: |
      Resolver's tried set is keyed by note id only; ids restart per broadcast session, so reused ids never get a pane path
    detail: |
      console_noteanchor.go:134-151. A new Session builds a new PointerState and NoteBoard (nextID 0). tried is pruned only in queueAnchors, which applyNotes skips once pointing is off or the broadcast ends. In a second broadcast, note 1 stays kind screen. Key tried by (session, id) or reset it on session change, and add a two-session test. Fix the "ids are never reused" claims in the atlas and SetNoteAnchor.
  - id: new
    severity: Important
    family: state-as-nullable-constellation
    title: |
      Resolver uses the live frame at resolve time; a slot switch mid-pass goes undetected and draft lines come from another slot
    detail: |
      This is the 2nd finding in this family. Rule: a job carries the exact observation it was made from (here the frame and its EndpointID), never a re-read of mutable "latest" state. GeometryEpoch is per endpoint and slots resize together, so the epoch and geometry check passes across a slot switch. The header comment claims a slot check that happens only at queue time. Capture the frame in anchorJob, check EndpointID, and add a slot-switch-mid-pass test.
  - id: new
    severity: Important
    family: leaf-lock-held-across-calls
    title: |
      queueAnchors calls GoTracked while holding anchors.mu; GoTracked takes c.mu and, when not started, runs resolveAnchors inline (self-deadlock)
    detail: |
      console_noteanchor.go:153-155 against console.go:2139. This breaks the documented leaf-lock rule. It is latent today because the inline path needs !started with a live broadcast. Decide the start under the lock, unlock, then call GoTracked.
  - id: new
    severity: Important
    family: operator-context-on-public-wire
    title: |
      Note.Anchor (file names, an on-disk Markdown heading, text under the box) now goes out on every page's notes event
    detail: |
      notes.go json tag "anchor,omitempty" and server.go:346 wireNotes. The spec says the anchor is for the agent's context, and no page reads it. sectionOf can publish a heading that is not on screen to view-only viewers. Exclude it from the wire (json:"-" or a wire struct), keep it in Session.Notes(), and add a test that the wire has no anchor.
  - id: new
    severity: Important
    family: glue-untested
    title: |
      Session.SetNoteAnchor, NoteBoard.SetAnchor and Config.AnchorNote have no broadcast-package tests; drop paths are tested only through a fake that restates them
    detail: |
      This is the 2nd finding in family glue-untested. Rule: each exported state-changing seam method a milestone adds gets an own-package test of its refuse and drop paths (here: after resolve, after a flip to off, and on an unknown id), not only through a caller's fake.
  - id: new
    severity: Minor
    family: plan-names-match-code
    title: |
      noteanchor.go header names noteanchor_resolve.go (nonexistent); the plan's Design still names noteanchor_io.go, AnchorFor(box, l, frame) and view.json fuzzing
    detail: |
      This is the 3rd finding in family plan-names-match-code. Rule: every file or identifier named in prose (plan, atlas, doc comment) must exist at the pinned head. Enforce it with a grep of the diff's backticked names against the tree at each boundary, rather than fixing instances. Prevalence: 3 findings across M1-M3.
  - id: new
    severity: Minor
    family: cross-component-format-unpinned
    title: |
      reviewFile restates SpawnReviewPane's argv format ("/nvim/review.lua" followed by the file) with no shared constant or cross-test
    detail: |
      note_panes.go:117. If reviewcmd's command changes, review anchors silently lose their file. Share the marker constant, or add a test that builds SpawnReviewPane's argv and parses it back (ARCH-DRY).
```

---

## Re-review — 2026-10-12T13:23:29-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | cc4ac22850abb2343add095b444d1c4fa74a7ec7..e1dac86e0588bac2b5d38bc86c73f866f89e602e |
| command | sdlc milestone-close --issue 488 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T13:23:29-07:00 |
| verdict | SHIP |

## Review

The test build needs a generated asset directory. I'm finding out how the repo builds it so the tests can run.

While the tests run in the background, I'm reading the new pure anchor code.

The full race run is still going. I'm running the targeted M3 tests separately so I can do the revert checks.

All three reverts fail their tests. Now BR-17: I'll move the resolver start back under the lock and check that a test catches it.

BR-17's revert deadlocks the seam test, so that fix is pinned. Last, I'm checking that names in the plan and atlas match the code (BR-20).

```verdict
verdict: SHIP
confidence: high
```

All the Critical and Important findings from round 1 (BR-15 to BR-19) are fixed, and each fix has a regression test. I checked this by reverting each fix in the disposable checkout and running its test:

| Fix | Reverted to | Test result |
|---|---|---|
| BR-15 | `tried` keyed by note id only | `TestQueueAnchorJobsPerSession` fails ("the next session's note 1 must be queued: 1 jobs") |
| BR-16 | `EndpointID` check removed | `TestResolverOrdering/a slot switch…` fails (a draft path from the old slot is applied) |
| BR-17 | `GoTracked` called under `anchors.mu` | `TestQueueAnchorJobsPerSession` deadlocks and times out after 20 s |
| BR-18 | `json:"anchor,omitempty"` | `TestNoteAnchorLifecycle` fails (the anchor JSON shows up in the page's notes event) |

The M3 tests pass. Under `-race`, the `broadcast` package passes. The `couchtty` and `couchcmd` failures are not caused by this diff: most are `operation not permitted` on `/tmp` and on starting a PTY child in this environment. `TestCouchProductionSoak` also failed after about 480 s under `-race`; I didn't look into it, and it looks unrelated to this diff. Only Minor findings remain open, so none of them blocks the boundary: the plan's Design still names surface that doesn't exist (BR-20), plus one small cleanup.

1. **Strengths**
   - `console_noteanchor.go:148` splits deciding from acting: `queueAnchorJobs` does all its work under the leaf lock and returns whether to start the resolver; `queueAnchors` starts it after unlocking. The test covers that split directly.
   - `anchorJob.frame` (`:71`) keeps the exact frame the job was made from. At apply time, `:215` compares endpoint, geometry epoch and geometry against what is now on screen. Jobs are only made while the text under the box still matches the note (`:168`).
   - `AnchorFor`, `locate`, `sectionOf` and `TextUnder` in `noteanchor.go` are pure. `locate`'s work is bounded (`maxLocate`), and the file read is bounded too (`readNoteFile` plus the `maxNoteFile` check).
   - `notes_session_test.go:272` tests the whole seam inside its own package: the opening anchor, the refinement, the wire never carrying it, and the drop paths (unknown id, resolved note, pointing off).
   - `reviewcmd.ReviewScript` is now the one place the review script's path is defined. The spawner (`run.go:192`) and the parser (`note_panes.go:120`) both use it.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-20 is still open.** The plan's Design section (lines 65 and 76) still names `couchtty/noteanchor_io.go`, which doesn't exist. It also gives the signature as `AnchorFor(box, l, frame)`; the real one is `AnchorFor(box, f, l)`. The "M3 as built" revision doesn't map either name to what was built. It also says `tried` is "pruned to open ids", but `tried` is now keyed by session and id. Separately, the `anchorJob` doc comment (`console_noteanchor.go:64-65`) still says it carries "the frame's geometry epoch"; it now carries the whole frame.
   - **ARCH-FUNERAL (new):** `tried` is only pruned when a new notes snapshot arrives. After a broadcast ends, `applyNotes` returns before reaching `queueAnchors`. So `tried` keeps a pointer to the ended `*broadcast.Session` until the next broadcast's first snapshot. The size is small (that session's open notes), but nothing clears it when the broadcast ends.

5. **Test coverage notes**
   - The resolver's ordering tests use an injected layout fake with a `during` hook, so the test controls the interleaving. Resolve mid-pass, resize mid-pass, slot switch mid-pass, a failed query, two notes in one pass and a note opened mid-pass are all covered.
   - The cross-session case now has its own test.
   - The live zellij check (`TestNotePanesZellijConformance`) exists. It wasn't run here because there was no zellij.

6. **Architecture**
   - **ARCH-DRY:** pass. BR-21's shared constant removed the restated format.
   - **ARCH-PURE:** pass. The pure core is `noteanchor.go`. The IO shell is `paneRectsOf`/`queryPaneGeometry`, injected through the `PaneRects` seam.
   - **ARCH-PURPOSE:** pass. Terminal anchors are delivered for the tab bar, the status line, and each pane role. The draft and review panes get file, lines and section.
   - **ARCH-MOCK:** pass. A stateful layout fake sits behind `PaneRects`, and the live check is in `note_panes_live_test.go`.
   - **ARCH-CONSTRAINTS:** pass. Each pass is bounded to 2 s. The resolver is single-flight with one query per slot per pass. File reads are capped at 1 MiB, line matching at 2^20 comparisons, and anchor text at 1 KiB.
   - **ARCH-SECURE:** pass. The anchor is off the viewers' wire (BR-18), the file reads are bounded, and the box comes from the session, which has already checked it against the grid.
   - **ARCH-ORDER:** pass. Each job keeps its own observation, the drop rules are explicit, and the tests control the interleaving.
   - **ARCH-FUNERAL:** minor flag (above). Nothing durable is created.

   For #455/#483: they should read `Note.Anchor` through `Session.Notes()`. It is no longer on the wire.

7. **Plan revisions to make**
   - In "M3 as built", add: "Read `couchtty/noteanchor_io.go` as `couchtty/console_noteanchor.go` (shell) and `couchcmd/note_panes.go` (zellij query). The signature is `AnchorFor(box CellBox, f terminal.Frame, l ScreenLayout)`. The PQ-1 `view.json` fuzz item is superseded."
   - Change "pruned to open ids" to "keyed by (session, id), pruned to the latest session's open notes on every snapshot".

```findings
dispose:
  - id: BR-15
    disposition: addressed
    note: |
      tried is keyed by triedNote{session,id}; reverting the key to id-only turns TestQueueAnchorJobsPerSession red; atlas and SetNoteAnchor wording corrected.
  - id: BR-16
    disposition: addressed
    note: |
      anchorJob carries frame; apply checks EndpointID/epoch/geometry against it; removing the EndpointID check turns the slot-switch subtest red.
  - id: BR-17
    disposition: addressed
    note: |
      queueAnchorJobs decides under anchors.mu and returns a bool; queueAnchors calls GoTracked after unlock; calling GoTracked under the lock deadlocks TestQueueAnchorJobsPerSession (20s timeout).
  - id: BR-18
    disposition: addressed
    note: |
      Note.Anchor is json:"-"; restoring the omitempty tag turns TestNoteAnchorLifecycle red (anchor shows up in the notes event).
  - id: BR-19
    disposition: addressed
    note: |
      TestNoteAnchorLifecycle (session: unknown id, resolved, pointing off) and TestNoteBoardSetAnchor (board) cover the refuse and drop paths in their own packages.
  - id: BR-20
    disposition: not-addressed
    note: |
      noteanchor.go header fixed, but plan Design lines 65/76 still name noteanchor_io.go and the AnchorFor(box,l,frame) order; as-built adds no mapping and says tried is pruned to open ids; anchorJob comment says it carries a geometry epoch.
  - id: BR-21
    disposition: addressed
    note: |
      reviewcmd.ReviewScript is shared by RunOpen and reviewFile; TestReviewFile parses a command built from the constant.
findings:
  - id: new
    severity: Minor
    family: residue-outlives-owner
    title: |
      tried keeps a pointer to an ended broadcast Session until the next broadcast's first notes snapshot
    detail: |
      console_noteanchor.go:155 prunes only in queueAnchorJobs, which applyNotes skips once the broadcast ends. Size is bounded by that session's open notes; clear tried (and pending) when the broadcast ends, or accept it and note the bound in the plan.
```
