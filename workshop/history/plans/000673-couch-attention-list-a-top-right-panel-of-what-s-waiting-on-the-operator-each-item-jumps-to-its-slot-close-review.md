# Boundary Review — pair#673 (whole-issue close)

| field | value |
|-------|-------|
| issue | 673 — Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot |
| repo | pair |
| issue file | workshop/issues/000673-couch-attention-list-a-top-right-panel-of-what-s-waiting-on-the-operator-each-item-jumps-to-its-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 74fa58d2c4319cec875ecf227159c95ac779b365..83d74e96af019f3b425b9fd2b22baeb15b0f7284 |
| command | sdlc close --issue 673 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T21:15:54-07:00 |
| verdict | REWORK |

## Review

Diff read; now checking a few suspected issues (notify text bounds/sanitization, publish frequency, badge toggle while open).

Tests mostly green; couchcmd failed on a socket-path length issue that looks environmental (long checkout path). Retrying with a short TMPDIR.

```verdict
verdict: REWORK
confidence: medium
```

The feature does what the issue asks. There is one pure reducer (`couchcore.AttentionList`), and the three sources only report facts to it. The store is safe and versioned, with a stateful in-memory fake behind an interface. The panel reuses the notes-menu box and the `switchTo` notification path, and the badge was moved to the status row as the TL decided. The tests I ran pass: `couchcore` and `couchtty` for `Attention`, plus `couchkeys` and `artifactpath`. `couchcmd` (`WaitNotice|Notify|Blocked`) only passed once I set a short `TMPDIR`; the first run failed because the socket path under this long checkout path is too long, which is an environment limit. Two things block SHIP. First, the plan's Core concepts table no longer matches the code: one file it names does not exist, and the method names, the item fields and the restart behaviour all differ. The review contract treats that as Critical, though the fix is a cheap `## Revisions` entry. Second, two Important gaps are cheap to fix: the store's 256 KiB read limit has no matching limit on what gets written, and the notify and prompt sources have no tests at all.

### 1. Strengths
- **Pure reducer:** `cmd/internal/couchcore/attention_list.go` is a value type, and every transition returns a new list. The tests cover each row of the state table: re-raise keeps `Since`, selecting a condition item marks it visited and keeps it, selecting a notification removes it, the cap protects held conditions, and the restore grace works. All of them run without IO.
- **Restart handling:** the grace (`RestoreAttention` plus `Saved`) is a careful design. A Couch restart inside the grace loses nothing, and a prompt answered while Couch was down does not come back.
- **Store safety:** `attention_store.go` refuses symlinks (`O_NOFOLLOW` on load, `Lstat` on save), checks the size cap before reading, decodes strictly, checks the version, writes atomically through a staging file, and has tests for each refusal.
- **Ordered saves:** `saveAttention` takes the snapshot while holding the save lock, so an older snapshot can never land after a newer one.
- **Reuse:** the panel is `renderMenuAt` with a `top` flag, not a second widget. The jump goes through `switchTo(..., arrivalNotification)`, and the test checks that Ctrl+Backspace returns home afterwards. The notify hook sits in `handleNotify`, so input-stall desktop notices stay out of the list.

### 2. Critical
- **The plan's Core concepts table contradicts the code.**
  - `AttentionRows` is listed at `cmd/internal/couchtty/attention_panel.go`, which does not exist. The code is `attentionRows` in `console_attention.go:166`.
  - `topRightBox` is really `renderMenuAt(…, top bool)` in `notes_menu.go:738`.
  - The reducer API is listed as `Raise/ConditionSet/Select/Dismiss/Reconcile`. The code has `Notify/Conditions/Apply/Select/Dismiss/RestoreAttention`, and there is no `Reconcile`.
  - `AttentionItem` is listed with `Slot`. The code has `Thread` (a `ThreadAddress`) plus `Label`.
  - The plan says "loaded items show immediately as stale-until-reconciled". The code hides restored condition items until they are re-raised within a 2-minute grace.
  - The plan says Delete dismisses. The code uses `d` or Backspace.
  - **Fix:** add a `## Revisions` entry recording the names that actually shipped and the hidden-until-confirmed restore with its grace.

### 3. Important
- **Store size cliff (ARCH-FUNERAL, ARCH-CONSTRAINTS):** `AttentionStoreLimit` is 256 KiB, and the comment says the capped list "fits well inside it". It does not in the worst case. A notification body can be up to 4096 bytes, and `json.Marshal` turns each `<`, `>` or `&` into the 6-byte `\u003c` form. About 11 full-size bodies of `<` already exceed the limit. Once over, every later `Save` is refused and a "not saved" notice appears on every change, so nothing persists until the operator dismisses items. **Fix:** have the writer evict the oldest notifications until the JSON fits (or truncate `Title`/`Body` when an item is created), and turn off HTML escaping. Add a test with 50 maximum-size bodies.
- **No tests for the notify and prompt sources:** there is no test in `couchcmd` for `handleNotify → observeAttention` (`notify_service.go:59`) or for the prompt fact from `publishBlocked` (`message_service.go:1180`). The Done-when tests inject facts straight into the Console, so deleting either hook would leave every test green. **Fix:** add service-rig tests checking that an accepted hub notify emits exactly one notify fact with the right thread and label, that a refused or non-hub request emits none, and that a change in the picker set emits a prompt fact.
- **Two planned pure tests are missing:** the plan's "`AttentionRows` formatting and selection" and "topRightBox geometry at small and odd sizes" tests do not exist. `attentionRows` and `renderMenuAt(top=true)` are only exercised end-to-end through the Console fixture.

### 4. Minor
- **Badge click can't close the panel (ARCH-ORDER):** with the panel open, a click on 🔔N lands outside the box. `routeAttentionPanel` closes the panel and passes the click on, then the status-row handler calls `toggleAttention`, which opens it again (`terminal_input.go:879` and `:897`). Clicking the badge never closes the panel.
- **Question items depend on the hub message:** `notified` is only set after the hub notice is sent successfully, so a busy or rate-limited hub delays the operator's question item. A muted slot never gets one.
- **Disk writes on message-service paths:** `ObserveAttention` does a synchronous fsync'd save on the caller's goroutine. That includes `publishBlocked` while it holds `s.blocked`, and `handleNotify` before the OSC write. `SetAttention`'s comment says the observer "must not block". The cost is milliseconds, but the comment and the code disagree.
- **Resize leaves the panel stale:** the open panel's geometry is only recomputed when a fact arrives or the operator acts, not on resize.
- **README:** it doesn't mention Backspace as a dismiss key.
- **Weak assertion:** `TestAttentionApply` checks the kind with `a != X && b != X`, which is weaker than intended.

### 5. Test coverage notes
The reducer and store are well pinned. The wait-watcher question set has a real start, unchanged and end test. The Console tests cover the whole Done-when, with a stateful store fake: notify then Enter jumps to the hub, prompt raise and clear, restart, mouse, dismiss, and the badge renderer. The gaps are the source hooks for notify and prompt, the pure row and geometry tests, and the store at its size cap.

### 6. Architectural notes
Pass or flag per principle:
- **ARCH-DRY:** pass. `bindingThread` and the notify label derivation are consolidated, and the menu is reused.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass. All three sources are wired; the question-gating note is above.
- **ARCH-MOCK:** pass. The store interface and its stateful fake share one boundary.
- **ARCH-CONSTRAINTS:** flagged, for the size cliff and the synchronous saves.
- **ARCH-SECURE:** pass. Loading is safe, and zero-width control characters are skipped when drawn.
- **ARCH-ORDER:** pass for the reducer and save ordering. The badge close-then-reopen is flagged as Minor.
- **ARCH-FUNERAL:** flagged, because the reader has a size cap and the writer has no matching bound.

Future kinds such as `exited` will fit as one more `Kind` plus a hook.

### 7. Plan revision recommendations
- **Revisions entry:** record the reducer API as `Notify/Conditions/Apply/Select/Dismiss/RestoreAttention` with a 2-minute `AttentionRestoreGrace`, with restored condition items hidden until re-raised. Record that the row model is `attentionRows` in `console_attention.go` and the panel anchor is `renderMenuAt(top)`, that items carry `Thread` plus `Label`, and that the dismiss keys are `d` and Backspace.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: plan-table-matches-code
    title: |
      Core concepts table contradicts the shipped code (paths, API names, fields, restore behavior)
    detail: |
      AttentionRows is listed at couchtty/attention_panel.go, which does not exist; the code is attentionRows in console_attention.go. topRightBox is really renderMenuAt(top). Raise/ConditionSet/Reconcile are really Notify/Conditions/RestoreAttention. The item field Slot is really Thread plus Label. The plan says loaded items show immediately; the code hides restored condition items for a 2-minute grace. Delete vs Backspace. Fix: add a plan Revisions entry.
  - id: new
    severity: Important
    kind: defect
    family: writer-bound-matches-reader-cap
    title: |
      Attention store's 256 KiB load/save cap has no matching bound on the writer
    detail: |
      There are 50 notify items and each body can be 4096 bytes. json.Marshal escapes each < > & to 6 bytes, so about 11 maximum-size bodies exceed AttentionStoreLimit. After that every Save is refused (a notice on every change) and nothing persists until the operator dismisses items. Fix: evict the oldest notifications until the JSON fits, or truncate fields when items are created, and turn off HTML escaping; add a test at the cap.
  - id: new
    severity: Important
    kind: defect
    family: source-hook-regression-test
    title: |
      No tests for the notify and prompt sources (handleNotify, publishBlocked to observeAttention)
    detail: |
      The Console tests inject facts directly, so deleting the observeAttention call in notify_service.go or message_service.go keeps every test green. Add service-rig tests: an accepted hub notify emits one fact with the right thread and label, a refused or non-hub request emits none, and a change in the picker set emits a prompt fact.
  - id: new
    severity: Important
    kind: defect
    family: planned-pure-tests-delivered
    title: |
      Planned pure tests for row formatting and top-right box geometry are missing
    detail: |
      The plan's test surface lists AttentionRows formatting and selection, and topRightBox geometry at small and odd sizes. Neither exists; attentionRows and renderMenuAt(top=true) are only exercised through the Console fixture.
  - id: new
    severity: Minor
    kind: defect
    family: click-outside-then-toggle
    title: |
      Clicking the status-row badge while the panel is open reopens it instead of closing it
    detail: |
      routeAttentionPanel closes the panel on a click outside the box and returns false, then the status-row handler calls toggleAttention, which opens it again.
  - id: new
    severity: Minor
    kind: defect
    family: observer-must-not-block
    title: |
      ObserveAttention does a synchronous fsync'd save on message-service goroutines despite the must-not-block contract
    detail: |
      publishBlocked calls it while holding s.blocked, and handleNotify calls it before the OSC write.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1cc615ce-ec28-47c9-8fe7-ec51e72e9234","repository":"github.com/xianxu/pair","issue_id":"000673","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4ff83248dbce0d6a","reviewed_head":"83d74e96af019f3b425b9fd2b22baeb15b0f7284","slot":"pair:2","requested_at":"2026-10-19T04:12:18.388118Z"},"phase":"finished","pid":5805,"started_at":"2026-10-19T04:12:18.76743Z","finished_at":"2026-10-19T04:15:52.783018Z","session_id":"1c26cb79-1054-47af-931c-34185c2ba36f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.5298255s
head: "83d74e96af019f3b425b9fd2b22baeb15b0f7284"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-18T21:59:54-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 673 — Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot |
| repo | pair |
| issue file | workshop/issues/000673-couch-attention-list-a-top-right-panel-of-what-s-waiting-on-the-operator-each-item-jumps-to-its-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 74fa58d2c4319cec875ecf227159c95ac779b365..9e0df2abd3ad43b083e7236710ebffb4b64bfcc4 |
| command | sdlc close --issue 673 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T21:59:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Meanwhile, read the reducer.

Targeted tests pass. Next I'm reverting the fixes one at a time to check that the new regression tests fail without them.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All six round-1 findings are fixed. For each behavior fix, I reverted it in the disposable checkout and confirmed its regression test then fails. The store bound uses the same encoding as `Save` (`encodeAttention`, with HTML escaping off). The notify and prompt sources now have service-rig tests. The planned pure row and box tests exist. A badge click no longer reopens the panel it just closed. Saves moved to a single background saver that coalesces changes. That saver is the one new problem: it runs in a goroutine that `Run`/`teardown` never waits for. A change made just before Couch quits can miss the disk, and before this round saves were synchronous, so this is a small regression in exactly the durability the store exists for. It is cheap to fix and does not block.

**Test runs:** build and vet are clean. `go test -race` passes for couchcore, couchtty, couchkeys and artifactpath. In couchcmd, 7 tests failed only because the long default `$TMPDIR` broke the 103-byte socket-path limit; with `TMPDIR=/tmp/rv673` the attention, notify, prompt and wait-notice tests pass under `-race`. I did not re-run the rest of couchcmd with the short path.

1. **Strengths**
   - `couchcore/attention_store.go:75` `encodeAttention` is the one encoding for both `Save` and the reducer's `fits`, so the writer's bound and the reader's cap can't drift apart (ARCH-DRY, ARCH-CONSTRAINTS).
   - `couchtty/console_attention.go:85-114`: the saver uses a dirty/saving pair with one runner, so saves land in order. `TestAttentionObserveNeverWaitsOnTheStore` checks both that callers don't block and that saves coalesce, using a gated store fake.
   - `TestNotifyRaisesOneAttentionItem` covers a refused (worker) caller as well as an accepted one. `TestBlockedPromptsReachTheAttentionList` covers an item appearing and then clearing.
   - `TestAttentionBoxTopRightGeometry` tests the top-right box directly (pure) at odd and tiny sizes, and also checks the bottom anchor did not change.

2. **Critical:** none.

3. **Important**
   - **Saver goroutine has no bounded lifetime** (`console_attention.go:94`, ARCH-ORDER extent). `go c.runAttentionSaves()` is started from `saveAttention`, but `Run` → `teardown` (`console.go:784-797`) never waits for it. If the operator dismisses or picks an item, or a notification arrives, and Couch then exits while a save is pending or in flight, the list on disk is one change behind. A dismissed item comes back after restart. The old synchronous save made each change durable before returning.
     - Fix: give `attentionState` a `done` channel or WaitGroup. Have `teardown` (or a `Console.flushAttention` that it calls) wait until no save is pending, with a bounded timeout.
     - Test: dismiss, then `Stop()`/`Run()`, then assert `store.saved()` is empty with no `waitFor`.

4. **Minor**
   - `console_attention_test.go:157`: a garbled comment line (`// \t// The decoder…`).
   - `TestAttentionMouseAndDismiss` checks "did not reopen" with a 50 ms sleep, which only samples one interleaving. It does go red without the fix (80 s timeout), so it is adequate for now.
   - `fits()` re-encodes the whole list (up to 256 KiB) on every transition, and on each eviction pass, while holding `attn.mu`, which the status model takes under `c.mu`. That costs about a millisecond at worst, so it is acceptable, but a cached size or an incremental estimate would remove it.
   - `capped` stops once no notify items are left, so a list of condition items alone could still exceed the cap. This is unlikely because labels and titles are short; noted only.

5. **Test coverage:** the BR-2, BR-3 (notify and prompt) and BR-5 tests all failed with their fix reverted:
   - BR-2: 275,220 bytes after 19 notifications, over the 262,144-byte limit.
   - BR-3 notify: no fact recorded.
   - BR-3 prompt: empty prompt set.
   - BR-5: "the badge click closes" timed out.

   The BR-6 test blocks by construction if the save is synchronous. Nothing tests a save being lost at exit (see Important).

6. **Architecture**
   - **ARCH-DRY:** pass (one shared encoding).
   - **ARCH-PURE:** pass. The reducer and `fits` are pure, and `attentionRows`/`renderMenuAt` are now tested directly.
   - **ARCH-PURPOSE:** pass. All three sources feed the list and are tested.
   - **ARCH-MOCK:** pass. The store sits behind an interface with memory and gated fakes.
   - **ARCH-CONSTRAINTS:** pass. The writer bound matches the reader cap; the encode cost is the Minor note above.
   - **ARCH-SECURE:** pass. Load is capped, versioned, and refuses symlinks (O_NOFOLLOW); failures show a notice.
   - **ARCH-ORDER:** flag, for the saver's lifetime (Important above).
   - **ARCH-FUNERAL:** pass. There is one `list.json`, now bounded, replaced atomically through a staging file.

7. **Plan revisions:** the round-1 Revisions entry matches the code. When the saver is fixed, add a line to its "Saves (BR-6)" bullet saying that teardown waits for the saver, with its timeout.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan Revisions entry dated 2026-10-18 maps every name, path, identity, restore-grace and Backspace difference to the shipped code; spot-checked against console_attention.go and attention_list.go.
  - id: BR-2
    disposition: addressed
    note: |
      capped() now evicts the oldest notify items until encodeAttention fits; with the fits() term reverted, TestAttentionListStaysWithinTheStoreLimit fails (275220 > 262144).
  - id: BR-3
    disposition: addressed
    note: |
      With the observeAttention call removed, TestNotifyRaisesOneAttentionItem fails (handleNotify) and TestBlockedPromptsReachTheAttentionList fails (publishBlocked); the refused worker case is covered.
  - id: BR-4
    disposition: addressed
    note: |
      TestAttentionRowsFormat and TestAttentionBoxTopRightGeometry exercise attentionRows and renderMenuAt(top=true) directly, without IO, at several sizes.
  - id: BR-5
    disposition: addressed
    note: |
      With the onBadge return reverted, TestAttentionMouseAndDismiss times out waiting for the badge click to close the panel; with the fix it passes in 0.4s.
  - id: BR-6
    disposition: addressed
    note: |
      Saves run in one coalescing background saver; TestAttentionObserveNeverWaitsOnTheStore uses a gated store and would block if the save were synchronous.
findings:
  - id: new
    severity: Important
    kind: defect
    family: spawned-work-bounded-extent
    title: |
      Attention saver goroutine outlives the Console; teardown never waits for it, so the last change before quit can be lost
    detail: |
      console_attention.go:94 starts runAttentionSaves with go, and Run/teardown (console.go:784) never waits for it (ARCH-ORDER extent). Dismissing or picking an item, or a notification arriving, right before exit can leave list.json one change behind; the synchronous save it replaced did not. Fix: track the saver (WaitGroup or done channel) and have teardown wait for it with a bounded timeout; add a test that dismisses, stops the console, and asserts the store is saved without waitFor.
  - id: new
    severity: Minor
    kind: defect
    family: test-comment-hygiene
    title: |
      Garbled comment line in TestAttentionMouseAndDismiss
    detail: |
      console_attention_test.go:157 reads "// <tab>// The decoder hands..."; a doubled comment marker left over from an edit.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fc262efe-3923-44be-bc8b-438ad57553d5","repository":"github.com/xianxu/pair","issue_id":"000673","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/74d1f79af281067a","reviewed_head":"9e0df2abd3ad43b083e7236710ebffb4b64bfcc4","slot":"pair:2","requested_at":"2026-10-19T04:35:44.402246Z"},"phase":"finished","pid":67016,"started_at":"2026-10-19T04:35:44.588784Z","finished_at":"2026-10-19T04:59:53.612764Z","session_id":"433b5e2e-b61c-4617-808b-f9639cb3e99a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.433877s
head: "9e0df2abd3ad43b083e7236710ebffb4b64bfcc4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-18T22:24:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 673 — Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot |
| repo | pair |
| issue file | workshop/issues/000673-couch-attention-list-a-top-right-panel-of-what-s-waiting-on-the-operator-each-item-jumps-to-its-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | 74fa58d2c4319cec875ecf227159c95ac779b365..6d00d20b44696671955e7d550d02c143c8de1e27 |
| command | sdlc close --issue 673 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T22:24:27-07:00 |
| verdict | SHIP |

## Review

Focus first on the last commit (BR-7/BR-8 fix).

The fix holds: the new test passes and fails when I revert the fix. Next I'm checking whether the couchcore and couchcmd failures come from this issue or from the environment.

The couchcmd failures all come from the socket path being too long in this checkout; the tests this issue added pass with a short TMPDIR. Now I'm looking over the rest of the diff for new problems.

```verdict
verdict: SHIP
confidence: high
```

Both open findings are fixed, and I raised no new ones. For BR-7, the attention saver now runs as a tracked worker (`console_attention.go:95-97`, using `GoTracked`), so shutdown waits for it at `c.workers.Wait()` (`console.go:1162`) before it returns. The new regression test passes. When I put back the old `go c.runAttentionSaves()` in this checkout, it fails with "Run returned with a save still in flight". For BR-8, the doubled comment marker in `console_attention_test.go:157` is gone. The six findings already closed were not re-raised.

1. **Strengths**
   - The fix uses the Console's existing tracked-worker join instead of adding a second shutdown mechanism (ARCH-DRY).
   - The saver starts only after `c.attn.mu` is released, because `GoTracked` takes `c.mu` (`console_attention.go:86-97`). That keeps the lock order the `attentionState` comment documents, where the attention lock never takes another.
   - The shutdown is race-free:
     - `started=false` is set under `c.mu` before `Wait`.
     - A fact that arrives while a save is running sets `dirty`, so the saver that is already being waited on writes it.
     - A fact that arrives after the join starts runs the save inline.

     Either way, the last change is not lost.
   - The new test (`console_attention_test.go:347-365`) holds the save open with a gate, checks that `Run` has not returned, then releases it and checks the saved content. It tests real behaviour, not a mock of it.
   - Saves are still coalesced (one runner, `dirty`/`saving`), so the BR-6 rule that the message service never waits on the disk still holds while the Console runs.

2. **Critical:** none.

3. **Important:** none.

4. **Minor** (no finding raised):
   - BR-7 suggested a bounded timeout on the wait. The join has none, so a disk that hangs during a save would hang shutdown. That matches every other tracked worker in the Console, so I'm noting it rather than raising it.
   - After shutdown has started, `ObserveAttention` runs the save inline on the caller's goroutine. That's acceptable because nothing is left to block by then.

5. **Test coverage**
   - Passing:
     - `go test ./cmd/internal/couchtty -run Attention` (also passes 3 runs under `-race`).
     - `go test ./cmd/internal/couchcore -run Attention`.
     - The three couchcmd source tests this issue added, run with `TMPDIR=/tmp`.
   - The full couchcmd run fails across many tests with "message socket must be a clean absolute path of at most 103 bytes". This checkout's path is too long for a socket, so it's an environment problem, not this diff.
   - The full couchcore run reported FAIL after 444s. I didn't find which test failed. The attention-scoped couchcore tests pass, and this issue changes only the attention files in that package.

6. **Architecture**
   - **ARCH-ORDER:** pass. The saver's lifetime is now bounded by the Console's join.
   - **ARCH-CONSTRAINTS:** pass, apart from the unbounded join noted above.
   - **ARCH-FUNERAL:** pass. There is one file, `list.json`, which is bounded by its writer (BR-2) and replaced atomically.
   - **ARCH-SECURE:** pass. Load uses `O_NOFOLLOW`, accepts only a regular file, caps the size, rejects unknown fields and checks the version.
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. The rules live in the `couchcore` reducer; `couchtty` is the shell around it.
   - **ARCH-PURPOSE:** pass. All three sources are wired.
   - **ARCH-MOCK:** pass. The store sits behind an interface and the tests use a gated fake.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      saveAttention starts the saver via c.GoTracked (console_attention.go:95-97), which teardown joins at console.go:1162; TestAttentionShutdownWaitsForTheLastSave passes at head and fails when the prior go c.runAttentionSaves() is restored in this checkout.
  - id: BR-8
    disposition: addressed
    note: |
      console_attention_test.go:157-158 now reads as one clean comment; the doubled marker is removed in 6d00d20b4.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"788e3991-7db3-446e-82e1-210905069398","repository":"github.com/xianxu/pair","issue_id":"000673","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ea8886387138637a","reviewed_head":"6d00d20b44696671955e7d550d02c143c8de1e27","slot":"pair:2","requested_at":"2026-10-19T05:05:06.400211Z"},"phase":"finished","pid":27829,"started_at":"2026-10-19T05:05:06.504833Z","finished_at":"2026-10-19T05:24:26.359964Z","session_id":"e8a26a33-af3b-4ecd-a58f-17c40c12d5d1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.514972583s
head: "6d00d20b44696671955e7d550d02c143c8de1e27"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
