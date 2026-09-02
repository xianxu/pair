# Rescope couch to couch-lite Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Narrow couch to a switcher over a group of pair sessions: one switch rule that keeps a notification hop from costing the operator their place, a warm `alt+d` detach whose detached sessions are listed and reattachable, startup that resumes whatever is already in the tree, and the deletion of the machinery that only defended multi-owner/multi-host cases.

**Architecture:** Four boundaries, each independently operable. M1 is pure UI model plus the key layer — a `SwitchTracker` value with one rule, and `ctrl-space`/`ctrl+backspace`/`alt+d` added to the one exact-string interceptor table couch already owns. M2 makes *detached* a derived actionable state rather than a persisted one, proved from `launcher`'s existing 0-client zellij classification, so detach reuses the process-group teardown couch already has minus the session deletion. M3 widens interactive startup's unique-root selector from parked-only to parked-or-detached. M4 deletes admission/policy, start grants, legacy migration, the never-instantiated actor loop, and the dead registry-era surface — and states, for each subsystem the issue named a candidate, why it survives.

**Tech Stack:** Go; `cmd/internal/couchtty` (console + pure menu), `cmd/internal/couchcore` (domain + seams), `cmd/internal/couchcmd` (CLI composition root), `cmd/internal/launcher` (zellij session authority), `cmd/internal/workbenchshortcut` (canonical chord encodings); existing fake Runner/ThreadStore/Path/Git/artifact seams; Go `testing`.

---

## Decisions taken before this plan

Recorded here because each closes an ambiguity in the issue Spec, and the plan below depends on them.

1. **`ctrl-space` from an actor opens the switcher; `ctrl-space` inside the switcher still opens the start form.** The ladder that dies is child → root-actor → panel. The panel's own `ctrl-space` is not a rung of that ladder, and it is the only route to starting a thread (`menu.go:318`). Operator decision, 2026-09-02.
2. **`alt+x` on the panel means `leave couch`.** With the root-actor concept gone, `leave` loses its only trigger (`console.go:1101`, `isRoot`). Deriving it from focus — `alt+x` quits what you are looking at, an actor or couch itself — needs no new key and reuses the existing typed `leave` confirmation. Operator decision, 2026-09-02.
3. **`leave couch` detaches every thread instead of parking them.** Operator decision, 2026-09-02, from the observation that `alt+d` is categorically safe and `alt+x` is not: park kills the agent process, so parking on the way out kills every agent including ones mid-turn. Detach kills nothing. This makes *detached* the normal resting state rather than an edge case, which is what makes M3's startup reattach worth having.
4. **couch keeps intercepting `alt+x`.** Interception does not add the risk the operator raised — `pair quit` kills the agent identically whether couch intercepts or pair's own `PairConfirmDetach`/`PairConfirmQuit` handles it. What interception buys is the durable park transaction; without it couch sees only a child exit, the incarnation becomes conservative `unknown`, and `ProjectActionableThreads` hides the row (`actionableinventory.go:106-126`) — the thread disappears from the switcher. Not intercepting is strictly worse.

## Chunk 1: The switch rule and the key layer (M1)

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SwitchTracker` | `cmd/internal/couchtty/switchrule.go` | new |
| `Focus` | `cmd/internal/couchtty/focus.go` | modified |
| `Up` | `cmd/internal/couchtty/focus.go` | deleted |
| `sequenceAt` / `knownSequences` | `cmd/internal/couchtty/keys.go` | modified |
| `DecodePanelKeys` CSI-u branch | `cmd/internal/couchtty/panelkeys.go` | modified |

- **`SwitchTracker`** — the whole of §"The switch rule" as one value: `{current, previous ThreadAddress; currentViaNotification bool}` with one method, `Switch(target ThreadAddress, viaNotification bool)`, and one reader, `Previous() (ThreadAddress, bool)`.
  - **Relationships:** 1:1 with a Console (one operator, one terminal). Keyed by `couchcore.ThreadAddress`, not by console-local actor id, because a park/resume cycle mints a new actor id for the same durable thread and `previous` must survive that. N:1 to threads.
  - **DRY rationale:** one authority for what `previous` is. `Console.switchTo` (`console.go:398`) is the funnel for every *subsequent* landing, but it is **not** the only writer of `c.active`/`c.focus`: `installObservedThreadActor` (`console.go:334-338`) lands the **first** attached actor directly, and `onExit` (`console.go:693-700`) re-points `c.active` on a fallback. So the tracker is seeded at the first attach and updated at `switchTo`, and the exit fallback is a landing too. Missing the first attach is not cosmetic — it breaks the Spec's leading consequence, "first hop from working actor A pins A", because A would never have been recorded as `current`.
  - **Future extensions:** a stack instead of one slot is a deliberate non-extension (the Spec rejects it: a stack the operator cannot see is one they lose track of). The natural widening is a second boolean axis if another arrival mode ever needs to be non-pinning; `Switch` already takes the mode as a parameter rather than inferring it.
  - **Deliberate consequence, asserted by test, not special-cased:** `ctrl+backspace` out of a notification actor lands on A with `previous == A`, so the next `ctrl+backspace` is a no-op. The operator is home and there is nowhere to bounce to.
- **`Focus`** — loses nothing structurally; `Up` and the `rootActor` parameter go. `Console.root` (`console.go:71`) is deleted with them, along with its maintenance in `onExit` (`console.go`, the `event.id == c.root` block) and the `isRoot` branch in `onParkHotkey` (`console.go:1101,1112`).
- **`knownSequences`** — gains `ctrl+backspace` in both encodings and `alt+d` from the canonical table. Legacy `ctrl+backspace` is the bare byte `0x08`, so it needs a bare-byte branch beside `hotkeyByte` in `FeedHit` (`keys.go:130`), not a `knownSequences` row; the Kitty form `\x1b[127;5u` is an ordinary exact-string row. `alt+d` comes from `workbenchshortcut.ChordEncodings(ChordAltD)` — the same construction `ChordAltX`/`seqPark` already uses (`keys.go:71`) — and yields exactly one encoding, `\x1b[100;3u`, because the chord table declares no legacy `\x1bd` form (`shortcut.go:300`). Alt+d is therefore uninterceptable with the Kitty protocol off; zellij pushes it, so this is a documented edge, not a gap.
- **`DecodePanelKeys`** — `case 127, 8` (`panelkeys.go:198`) ignores the `modified` flag computed at `:191`, so `\x1b[127;5u` decodes as plain backspace. This is a latent correctness bug independent of interception and is fixed here; the interceptor takes both encodings first, so the panel is defence in depth (it still matters inside a bracketed paste, where the interceptor forwards content verbatim).

**Architecture and operating envelope (M1)**

- `ARCH-PURE`: the rule is a value type with no clock, IO or console reference; `Console.switchTo` is the thin seam that calls it. Its tests need no fakes.
- `ARCH-DRY`: `alt+d` derives its bytes from the canonical chord table rather than re-spelling protocol literals; `ctrl+backspace` reuses the existing dual-encoding shape rather than introducing a parser.
- `ARCH-CONSTRAINTS`: keystroke-critical path. The switcher's committed envelope (100 rows at 120x40; 50 ms open, 16 ms filter/navigation/render, 100 ms first progress, `BenchmarkMenu100`) is the budget this must not regress. `SwitchTracker.Switch` is O(1) with no allocation; `sequenceAt` gains one row, so matching stays O(len(knownSequences)) over exact strings with no timing window. No new split-read latency: `\x1b[1` is *already* a partial prefix via `ChordAltX`'s `\x1b[120;3u` (`shortcut.go:302`), so `\x1b[127;5u` adds none, and the existing 35 ms `escapeAmbiguity` flush already covers that case.
- **Accepted cost, deliberate:** in legacy encoding `0x08` *is* `^H`, so intercepting `ctrl+backspace` takes `ctrl-h` from the child (readline and nvim insert-mode treat it as backspace). Under the Kitty protocol they separate (`\x1b[104;5u` vs `\x1b[127;5u`) and zellij pushes the protocol, so this bites only with the protocol off.

### Task 1: Pin the switch rule

**Files:**
- Create: `cmd/internal/couchtty/switchrule.go`
- Create: `cmd/internal/couchtty/switchrule_test.go`

- [x] **Step 1: Write failing tests for `SwitchTracker`.** Strategy: drive sequences of `Switch(target, viaNotification)` calls and assert `Previous()` after each. The adversarial classes are: first hop from a working actor; notification-to-notification chains (N1 → N2 → N3); a manual detour out of a notification actor; `ctrl+backspace` out of a notification actor and immediately again; a switch to the actor that is already current; and a switch whose target equals `previous`. The mechanical guard is the single rule — `previous` advances if and only if the actor being *left* was not entered via notification.

```go
func TestSwitchTrackerKeepsWorkingActorPinnedAcrossNotificationHops(t *testing.T) {
	a := couchcore.ThreadAddress{RepoScope: "s", Tag: "a"}
	n1 := couchcore.ThreadAddress{RepoScope: "s", Tag: "n1"}
	n2 := couchcore.ThreadAddress{RepoScope: "s", Tag: "n2"}
	c := couchcore.ThreadAddress{RepoScope: "s", Tag: "c"}

	var tracker SwitchTracker
	tracker.Switch(a, false)  // working in A
	tracker.Switch(n1, true)  // ctrl-space + Return onto a paged actor
	tracker.Switch(n2, true)  // chase a second page
	tracker.Switch(c, false)  // manual detour to spot-check C

	got, ok := tracker.Previous()
	if !ok || got != a {
		t.Fatalf("Previous() = (%v, %v), want (%v, true) -- A must stay pinned", got, ok, a)
	}
}
```

- [x] **Step 2: Run `go test ./cmd/internal/couchtty -run '^TestSwitchTracker' -count=1` and confirm it fails because the type is absent.**

- [x] **Step 3: Implement the minimal rule.** Nothing beyond the Spec's four lines:

```go
// SwitchTracker is the whole of couch's `previous` behaviour: one slot, and
// one boolean that keeps a notification hop from spending it.
//
// Keyed by ThreadAddress rather than console-local actor id: a park/resume
// cycle mints a new actor id for the same durable thread, and `previous` has
// to survive that.
type SwitchTracker struct {
	current              couchcore.ThreadAddress
	previous             couchcore.ThreadAddress
	currentViaNotification bool
}

// Switch records a landing. viaNotification is true only when the operator
// arrived by ctrl-space + Return on an actor that had a pending notification;
// such an actor never becomes `previous`, so chasing pages never costs the
// operator the actor they were actually working in.
func (t *SwitchTracker) Switch(target couchcore.ThreadAddress, viaNotification bool) {
	if !t.currentViaNotification {
		t.previous = t.current
	}
	t.current = target
	t.currentViaNotification = viaNotification
}

func (t *SwitchTracker) Previous() (couchcore.ThreadAddress, bool) {
	return t.previous, t.previous != couchcore.ThreadAddress{}
}
```

- [x] **Step 4: Run the focused test and confirm it passes.**
- [x] **Step 5: Register the new file in the production source inventory.** Add `cmd/internal/couchtty/switchrule.go` to `NonArtifactSources` in `cmd/internal/artifactpath/manifest.go`. `productionSourceInventoryViolations` (`artifactpath/coverage_test.go:194`, implementation at `:1148-1180`) requires **every** production `.go` file to be classified, and new files have no implicit default — so skipping this fails `make test` at Task 6, far from its cause. Run `go test ./cmd/internal/artifactpath -count=1` to confirm.
- [x] **Step 6: Commit.** `git commit -m "#170 M1: couch: pin the previous-actor switch rule"`

### Task 2: Retire the focus ladder

**Files:**
- Modify: `cmd/internal/couchtty/focus.go` (delete `Up`)
- Modify: `cmd/internal/couchtty/console.go` (delete `root`; rewrite `onHotkey`; rewrite `onParkHotkey`'s root branch; drop root maintenance in `onExit`)
- Modify: `cmd/internal/couchtty/focus_test.go`, `console_test.go`, `console_panel_regression_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: drive `ctrl-space` through the production interceptor into a console with a non-root actor focused, and `alt+x` with the panel focused. Mechanical guards: `ctrl-space` from *any* actor lands on the panel in one press (never on another actor), and `alt+x` with the panel focused dispatches the typed `leave` confirmation while `alt+x` on an actor dispatches `park` for that actor. Assert absence too — `grep`-style source absence is not enough, so assert that no exported `Up` symbol and no `Console.root` field remain by compiling a test that would reference them only if present (delete their existing tests; a replacement is incomplete while its superseded surface still compiles — `workshop/lessons.md`).
- [x] **Step 2: Run `go test ./cmd/internal/couchtty -count=1` and confirm the new expectations fail.**
- [x] **Step 3: Delete `Up` and `Console.root`.** `onHotkey` becomes: if the focus is already the panel, forward `KeyCtrlSpace` to the menu (the start form, unchanged); otherwise set the focus to the panel, apply notification focus, refresh, and show the menu. The `next == cur` early return and the `actorAlive` liveness consult go with `Up` — there is no landing actor left to be dead. `onParkHotkey` drops `isRoot` and always dispatches `park` for the active actor; the panel case is Task 3.
  **`onExit` needs a replacement, not just a deletion.** `c.active = c.root` (`console.go:697`) is a *use*, not root maintenance — the comment above it names the invariant it defends ("Panel actions address the active actor, not merely the highlighted durable row"). Replace it with `c.order[0]` (or `""` when `c.order` is empty), which is behaviour-preserving — but not for the reason the line above suggests: `c.root` is re-pointed to `c.order[0]` **only when the exiting actor was the root** (`console.go:687-692`). The equivalence holds because of the invariant `root == order[0]`, not because of that assignment, so verify the invariant rather than the line. Assert the invariant directly — after the active actor exits with others remaining, a panel action still addresses a live actor.
- [x] **Step 3b: Delete `actorAlive` (`console.go:1119`) too.** It exists only as `Up`'s liveness predicate. Go does not report an unused *method*, so the compiler will not find it — this one has to be deleted deliberately.
- [x] **Step 3c: Confirm the deletion is complete by compilation, not by grep.** `go build ./... && GOOS=linux go build ./...`. A replacement is incomplete while its superseded surface still compiles (`workshop/lessons.md`).
- [x] **Step 4: Run `go test ./cmd/internal/couchtty -count=1` and confirm it passes.**
- [x] **Step 5: Commit.** `git commit -m "#170 M1: couch: retire the focus ladder and root actor"`

### Task 3: `alt+x` on the panel means leave

**Files:**
- Modify: `cmd/internal/couchtty/console.go` (`onParkHotkey`)
- Modify: `cmd/internal/couchtty/console_test.go`

- [x] **Step 1: Write the failing test.** Strategy: `alt+x` with the panel focused and with an actor focused, over consoles with zero, one and several live panes. Mechanical guard: exactly one typed operation is dispatched per press — `leave` for the panel, `park` with the active actor's address otherwise — the existing confirmation frame is rendered before any durable work, and **the zero-live-pane case still reaches the leave confirmation**. That case is not exotic: Decision 3 makes an all-detached couch the normal resting state, so it is the state the operator will most often quit from.
- [x] **Step 2: Run the focused test and confirm it fails.**
- [x] **Step 3: Implement — `leave`'s confirmation must stop being thread-bound.** Today `leave` rides the root actor's live address, so every thread-bound check passes by accident. With no root and possibly no live pane, a zero `Address` fails at **five** sites, not the two that are obvious:
      1. `onParkHotkey` returns early with `"park: no active thread"` when `c.panes[c.active] == nil` (`console.go:1104-1107`).
      2. `reduceParkHotkey` (`menu.go:407-416`) refuses unless `findMenuThread` locates a live thread at `event.Address`.
      3. `reduceConfirmationKey` (`menu.go:475-478`) discards the frame on the **first keypress** when `findMenuThread` misses.
      4. `reduceConfirmationKey`'s Enter arm (`menu.go:502`) requires `thread.Live()`, so the confirm could never dispatch.
      5. `reconcileMenuFrames` (`menu.go:1016-1033`) drops the frame **asynchronously** on the next inventory refresh, for the same two reasons.
      Patching each is five special cases for one concept. Instead make the leave confirmation a **global frame**, the shape the menu already has for the start form ("a global start frame overlays the preserved originating stack" — `atlas/couch.md`): a confirmation frame that is not bound to a thread, so none of the five thread lookups applies to it. Sites 2–5 then need one predicate change each — "thread-bound frames resolve a thread; global frames do not" — rather than a `leave`-shaped exception, and `confirmationMenuItems` (`menu.go:870`) already renders `leave` without touching the thread. `ARCH-DRY`: one frame-scope concept, reused, instead of a second confirmation path.
- [x] **Step 3b: Assert the async site explicitly.** Site 5 fires on the next refresh, not on a keystroke, so a test that only presses keys will pass while the real console drops the confirmation a second later. Drive a refresh between opening the leave confirmation and confirming it.
- [x] **Step 4: Run `go test ./cmd/internal/couchtty -count=1`; confirm pass.**
- [x] **Step 5: Commit.** `git commit -m "#170 M1: couch: leave couch from the panel's alt+x"`

### Task 4: Recognise `ctrl+backspace` in both encodings

**Files:**
- Modify: `cmd/internal/couchtty/keys.go`
- Modify: `cmd/internal/couchtty/panelkeys.go:191-203`
- Modify: `cmd/internal/couchtty/keys_test.go`, `panelkeys_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: feed both encodings through `Interceptor.FeedHit`, split across every read boundary, and inside a bracketed paste; and feed `\x1b[127;5u` plus plain `\x7f`/`\x08` through `DecodePanelKeys`. Mechanical guards: both encodings yield exactly one `HitPrevious` with a correct before/rest split; neither fires inside a paste; a split prefix is held and resolved rather than forwarded; plain backspace still yields `KeyBackspace` in the panel while the modified CSI-u form does not.
- [x] **Step 2: Run `go test ./cmd/internal/couchtty -run 'Interceptor|PanelKeys' -count=1` and confirm the new cases fail.**
- [x] **Step 3: Implement.** Add `HitPrevious` to `InterceptorHit` and `seqPrevious` to `seqKind`; add `{[]byte("\x1b[127;5u"), seqPrevious}` to `knownSequences`; add a bare-byte branch for `0x08` beside the `hotkeyByte` check in `FeedHit`, guarded by `!i.inPaste` exactly as the NUL branch is. In `panelkeys.go`, split `case 127, 8` (`:198`) so it returns `KeyBackspace` only when `!modified`.
      **Leave `panelkeys.go:98` alone**, despite the Spec naming both sites. All operator input passes `Interceptor.FeedHit` before `onMenuInput` (`console.go:527-559`), so the panel never sees a legacy ctrl+backspace once the interceptor claims `0x08` — and inside a bracketed paste, where the interceptor deliberately forwards content verbatim, `0x08` *should* stay a plain backspace. Changing `:98` would break the paste case to fix a case that cannot occur. Say this in the commit message; an implementer reading only the Spec will otherwise change it.
- [x] **Step 4: Run the focused tests, then `go test ./cmd/internal/couchtty -count=1`; confirm pass.**
- [x] **Step 5: Commit.** `git commit -m "#170 M1: couch: intercept ctrl+backspace in both encodings"`

### Task 5: Wire previous and notification focus

**Files:**
- Modify: `cmd/internal/couchtty/console.go` (`switchTo`, `onHotkey`, new `onPreviousHotkey`, `Run`'s `processInput`)
- Modify: `cmd/internal/couchtty/console_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: drive the full N1 → N2 → manual-detour sequence through the *production* input path (raw bytes into `processInput`, not direct method calls — reducer support is not user reachability, `workshop/lessons.md`), in both terminal encodings, and assert where `ctrl+backspace` lands. Mechanical guards: every focus landing updates the tracker exactly once regardless of mechanism (hotkey, menu `switch` operation, post-start attach, exit fallback); `viaNotification` is true only when the arrival was a menu `switch` whose target had a non-empty `AttentionLedger.Projection`; `ctrl+backspace` with no previous is a no-op notice, not a blank screen.
- [x] **Step 2: Run `go test ./cmd/internal/couchtty -run 'TestConsolePrevious|TestConsoleNotificationFocus' -count=1`; confirm failure.**
- [x] **Step 3: Implement — one funnel carrying two per-landing rules, and three landing sites.** Give `switchTo` an `arrival` argument (`arrivalMenuEnter`, `arrivalPrevious`, `arrivalFirstAttach`, `arrivalExit`) and make it the single place that applies **both** rules every landing owes:
      1. `tracker.Switch(thread, arrival == arrivalMenuEnter && targetHadPendingNotification)`.
      2. **Acknowledge the landed actor's pending notifications**, whatever the arrival was.
      Rule 2 is not optional bookkeeping — the Spec states "an actor does not notify while the operator is attached to it", and today the only site that honours it is `onHotkey`'s home-landing block (`console.go:1085-1090`), which Task 2 deletes. `switchTo` (`console.go:398-421`) never acknowledges, and Task 5's `onPreviousHotkey` calls `forceSwitch` directly, bypassing `runMenuOperation` entirely. So without this, `ctrl+backspace` home to A leaves A still marked notifying, `NewestActor()` then names the actor the operator is *sitting in*, and the next `ctrl-space` opens the switcher on it instead of on whoever actually paged — the exact behaviour the milestone exists to deliver, inverted.
      Acknowledging in `switchTo` also preserves the property `runMenuOperation`'s capture/acknowledge dance was protecting: a *failed* switch never lands, so it never acknowledges. Keep the capture (it is the record of what was pending at dispatch, which feeds rule 1); move the acknowledgement to the funnel.
      The three landing sites: `switchTo` (`console.go:398`) for ordinary switches, `installObservedThreadActor` (`console.go:334-338`) for the **first** attached actor, and `onExit` (`console.go:684-701`).
      `onExit` is the subtle one, because the operator lands on the **panel**, not on an actor (`console.go:699-701`). So it is not a `Switch` at all: recording `Switch(order[0].thread, false)` there would set `previous` to the thread that just died, which is the one place the operator can never go back to. Instead **drop the exited thread from the tracker** — if it was `current`, clear `current` without advancing `previous`; if it was `previous`, clear `previous` so `ctrl+backspace` reports "nothing to return to" rather than failing to resolve a dead pane. An empty `c.order` clears both. Test the exited-thread-was-previous case explicitly; it is the one a naive implementation gets wrong. Seeding only at `switchTo` leaves the operator's starting actor unrecorded, so the first notification hop would set `previous` to the zero address instead of A — the Spec's leading consequence, silently broken.
      Take `targetHadPendingNotification` from the dispatch path rather than recomputing it at landing: `runMenuOperation` already captures the pending set before dispatch (`console.go:1130`, `origin.AttentionCapture != 0` is exactly "the target had a pending notification"), so reuse that capture — it is the value that was true when the operator chose the row, not after. `arrivalPrevious`, `arrivalFirstAttach` and `arrivalExit` are never notification arrivals, so rule 1 sees `false` for them regardless of what is pending; **rule 2 still fires**, which is the whole point of separating them.
      `onPreviousHotkey` resolves `Previous()` to a live pane via the existing `switchTargetForAddressLocked` and calls `forceSwitch`; an address with no live pane sets a notice rather than blanking the screen.
- [x] **Step 3a: Assert rule 2 independently of rule 1.** A test that only checks `previous` will pass while the bell stays lit. Guard: after landing on an actor by **each** arrival kind, that actor's `AttentionLedger.Projection` is empty and `NewestActor()` does not name it; and after a *failed* switch, its pending notifications survive.
- [x] **Step 3b: Implement the "defined default" the Done-when requires — it does not exist today.** In `onHotkey`, lift the existing newest-notification focus block (`console.go:1072`) out of the `next.IsPanel()` conditional so it runs on every `ctrl-space` from an actor. With **no** notification pending, nothing currently selects the current thread: `NewMenuState` (`menu.go:196-201`) defaults `Frames[0].SelectedAddress` to `owned[0].Address`, and `reconcileRootSelection` (`menu.go:953-966`) keeps the prior selection or falls to `visible[0]`. `switchTo` maintains `menu.ActiveAddress` (`console.go:406`) but never `SelectedAddress`. So add the fallback explicitly: no pending notification → truncate to `Frames[:1]` and clear the filter (as the notification branch already does at `console.go:1073`), then route the selection through `reconcileRootSelection(&state, menu.ActiveAddress)` (`menu.go:953-966`) rather than assigning `SelectedAddress` raw. Raw assignment can open the switcher with **no selection**: `onExit` never updates `c.menu.ActiveAddress` (`console.go:684-701`), so it can name a dead thread that is absent from the inventory. `reconcileRootSelection` already means "preferred if present, else `visible[0]`", which is exactly the fallback wanted. Test both branches — with and without a pending notification, and with a stale `ActiveAddress` — since only the notification branch has any coverage today.
- [x] **Step 4: Run `go test ./cmd/internal/couchtty -count=1` and `go test ./cmd/... -count=1`; confirm pass.**
- [x] **Step 5: Update the help row and the README.** `menuControls` (`menu.go:18-28`) is the **panel's** help row, so its `{Keys: "Ctrl-Space", Action: "start"}` entry stays correct under Decision 1 — do **not** rewrite it to "switcher", which would document the opposite of what ships. Add `Ctrl-Backspace previous` only. **Not `Alt+d detach`** — M2 wires that key, and `TestREADMEDocumentsEveryPanelControl` (`couchcmd/readme_test.go:61`) would pin a control that does nothing for a whole milestone; Task 11 adds the row alongside the implementation. Leave `Alt+x park/leave` as is. Then update `README.md` unconditionally — it describes the ladder explicitly at `:319`, `:336`, `:342`, `:351` ("Focus has three levels. From a non-home actor, `ctrl-space` returns to the first…"), which is exactly what Task 2 deleted. The enforcing test is `TestREADMEDocumentsEveryPanelControl` (`couchcmd/readme_test.go:61`), which pins the `menuControls` keys — so the new rows *are* audited. The ladder prose at `README.md:336-351` is **not** audited by anything; delete it because it is now false, not because a test will catch it.
- [x] **Step 6: Run `make test`; confirm pass. Commit.** `git commit -m "#170 M1: couch: ctrl+backspace returns to the previous actor"`

### Task 6: Close M1

- [x] **Step 1: Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`** (the session env leaks into `review-target`/`changelog` tests when run inside a pair session) **and `git diff --check`.** Record the exact commands in the issue `## Log`.
- [x] **Step 2: Update `atlas/couch.md`** — replace the "Navigation" section's focus-ladder paragraph with the switch rule, and the `ctrl-space` row of the key table.
- [x] **Step 3: `sdlc milestone-close --issue 170 --milestone M1`.** Fix Critical/Important findings before crossing; log the `Review-Verdict:` outcome.

---

## Chunk 2: Detach (M2)

**Detached-ness is derived; the incarnation retirement is not.** `launcher` already classifies a live zellij session with zero clients as `SessionDetached` (`launcher/session.go:10`, `launcher/list.go:23`, `launcher/zellij.go:30`), and `pair resume` onto such a session already reattaches rather than creating (`launcher/decision.go:33-37`, via `sessionBlocksReuse` → `ActionAttach`). couch's own rule is that liveness is recomputed, never stored (`atlas/couch.md`, "Liveness is recomputed, never stored"). So detach adds **no `ThreadRecord` field**: the row's detached-ness is proved on each inventory refresh by observing the session.

**But it does need one durable transition, and an earlier draft of this plan was wrong to claim otherwise.** `FinalizePark` (`threadstore.go:391`) is the only path that removes a **live** incarnation from a record (`DeleteStart` also clears one, but only a `creating` one it owns), and `reconcileInterruptedStarts` (`couch.go:539`) only touches records with an *open start transaction* — a fully-started thread is never reconciled. So if detach merely killed the process group, the record would keep `Incarnations: [{State: IncarnationLive, PID: <dead>}]` forever, and three things break at once:

- `actionableThreadState` (`actionableinventory.go:106-126`) emits nothing for one incarnation with zero TTY observations, so the row stays **hidden**. Making the detached rule tolerate a stale live incarnation would weaken the fail-closed projector — the wrong fix.
- `DecideResume` refuses **any** record carrying a Live/Creating/Unknown incarnation (`resume.go:73-86`) *before* it ever reaches the `VerifiedPark` check at `:88`. That refusal — not the verified-park precondition — is the actual blocker for "detached sessions reattach".
- `hasActiveIncarnation` (`park.go:147`) would report a detached thread as active, so `Leave` re-detaches it; `soleParkableIncarnation` (`park.go:690`) would still see it; and M4's `CommitStartClaim` would append a second incarnation beside the stale one, breaking the `len(record.Incarnations) != 1` invariants throughout `threadstore.go`.

Retiring the incarnation fixes all three at the source and keeps the projector fail-closed. So detach is: **SIGTERM the pair client, wait for exit, prove the zellij session survives, then retire the incarnation by CAS.** That is one revision-checked write, not park's state machine — park's attempt history, `ParkUnknown` phase and Retry/Recover/Abandon modes exist to survive a two-process handshake with a deadline, and detach has no handshake.

**Detach must not reuse `handleCleanup`.** That path (`couch.go:450-479`) sends SIGTERM and then an *unconditional* `os.Kill` to the process group, and its own comment says "this path is rollback, not graceful actor shutdown". SIGKILL would skip pair's client-side cleanup on the everyday gesture — and, under Decision 3, on every thread each time the operator leaves couch. Detach sends SIGTERM and waits under the operation deadline; **if the client does not exit, detach fails and the thread stays live**. Nothing was destroyed, so failing is the safe outcome, and it needs no recovery mode. This is not a new philosophy: `TermSignal` (`procops.go:121-124`) already carries the comment "Nothing escalates to SIGKILL automatically — an agent mid-write is worse to truncate than to leave running." Detach is that rule applied to the everyday gesture. What dies: the pair client and the zellij client it hosts, plus the session-watcher and title-poller sidecars sharing its process group. What survives: the zellij *server* session and the agent running inside it — which is the whole point, and Task 9 asserts it rather than assuming it.

**Two gates beyond `DecideResume` also stand between a detached row and reattachment, and both must be handled in this milestone or M2 is not independently operable.**

- **`ReconcileResumeAdmission` is a second `VerifiedPark == nil` refusal** (`admission.go:183`), reached from `ResumeContext` (`resume.go:219`) *after* `DecideResume` returns. Widening only `DecideResume` ships a detached row whose Enter fails with "is not verified parked". Task 10 widens both. Admission is deleted at M4, so this edit is short-lived — that is a one-line cost, and it is cheaper than reordering the milestones to avoid it. **M4's `CommitStartClaim` must carry the same widened precondition forward**, or M4 silently re-breaks what M2 fixed; Task 13 names it.
- **Resuming a record with no `VerifiedPark` currently *deletes the record* on rollback.** `DeleteStart` (`threadstore.go:724-756`) branches on the verified park: with one set it clears the incarnation and keeps the record; with none it falls through to `deleteThreadIf`, whose accept predicate guards only revision, reservation, `threadHasMetadata` and incarnation count. `threadHasMetadata` (`threadmetadata_model.go:28-30`) does already refuse a record carrying a name, description or published summary — so a *named* thread is safe today. The unprotected case is an **unnamed** detached thread, whose `LatestLaunchProfile` is not guarded by anything: any post-claim failure on a detached resume — `resume.go:233,236,242`, or `StartRollback` at the next `New()` — deletes it while the zellij session keeps running, and with it the agent+argv needed to reattach. `starttransaction.go:83-86` states the premise plainly: "Until this transition the verified park remains the rollback authority." A detached thread needs its own rollback authority, and it has one — **the surviving session**: the safe state to roll back to is "still detached", not "gone".
  The fix is the class, not the instance (`ARCH-PURPOSE`): **a record carrying a `LatestLaunchProfile` is durable history and is never deleted.** Add that to `DeleteStart`'s accept predicate so the no-verified-park branch clears the start claim instead of deleting. This is safe for every existing caller — a thread that never started successfully has no profile, so pristine-reservation and unstarted rollbacks are unaffected.

**Why couch intercepts `alt+d` at all**, the counterpart of Decision 4: pair binds `ChordAltD` to `PairConfirmDetach` (`shortcut.go:120`), which runs `zellij action detach` from inside the session. Un-intercepted, that leaves couch with a dead child and a stale live incarnation — the invisible-thread state above, reached by the operator's most common gesture. Intercepting costs the hosted pair its own detach chord (as `alt+x` already costs it its quit chord) and buys the durable retirement that makes the thread reattachable.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ActionableThreadState` (`ThreadDetached`) | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `DetachedSessionObservation` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `ProjectActionableThreads` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `DecideResume` | `cmd/internal/couchcore/resume.go` | modified |
| `menuActionItems` / `reduceRootFrame` | `cmd/internal/couchtty/menu.go` | modified |

- **`ThreadDetached`** — a third actionable state beside `ThreadLive` and `ThreadParked`. Emitted only when the record is not reserved, has no active park transaction, has **zero incarnations** and no verified park, carries a `LatestLaunchProfile`, and exactly one `DetachedSessionObservation` matches its address. The zero-incarnation requirement is what keeps the projector fail-closed: a record with a stale `IncarnationLive` stays hidden exactly as today, so a couch that died without detaching cannot masquerade as a clean detach.
  - **Relationships:** 1:1 with a `ThreadRecord` at a time; mutually exclusive with `ThreadLive` and `ThreadParked` by construction (a live thread has a TTY observation; a parked thread has no zellij session).
  - **DRY rationale:** consumes `launcher`'s existing session classification rather than teaching couch a second way to ask whether a zellij session is alive. The observation seam is the one `ScopedThreadArtifactCollisionChecker.PairSession` already uses (`artifactcollision.go:126`) — `ARCH-DRY`.
  - **Future extensions:** the natural widening is a client count on the observation if "attached elsewhere" ever needs to be a distinct row; the projector's rule would gain a branch, not a new authority.
- **`DetachedSessionObservation`** — `{Address ThreadAddress; SessionName string}`, one per live zellij session couch can bind to a thread address. Shaped exactly like the existing `LiveTTYObservation` and `ParkedResumeObservation` (`actionableinventory.go:22,29`) so the projector keeps one argument style.
- **`DecideResume`** — gains a second admissible precondition: a proved-detached record with a `LatestLaunchProfile` and no verified park. Both paths converge on the same effect — spawn `pair resume <tag> --layout2` — because zellij reattaches a detached session by itself (`launcher/decision.go:33-37`). The difference is only which precondition was proved and whether a verified park is cleared afterwards. **The occupied-incarnation refusal at `resume.go:73-86` stays exactly as it is** — detach retires the incarnation, so a detached record passes that gate on its own merits rather than by relaxing it. Only the `VerifiedPark == nil` refusal at `:88-95` gains the detached branch.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `DetachedSessionResolver` | `cmd/internal/couchcore/artifactcollision.go` | new | `launcher` session classification |
| `ProcOps.SignalGroup` | `cmd/internal/couchcore/procops.go` | new | `syscall.Kill(-pid, sig)` |
| `ThreadStore.RetireIncarnation` | `cmd/internal/couchcore/threadstore.go` | new | journaled record CAS |
| `ThreadStore.DeleteStart` | `cmd/internal/couchcore/threadstore.go` | modified | journaled record CAS / delete |
| `Couch.Detach` | `cmd/internal/couchcore/detach.go` | new | SIGTERM + bounded wait + session observation |
| `detach` operation | `cmd/internal/couchcore/ops.go`, `operationdispatch.go` | new | typed operation surface |
| `Couch.Leave` | `cmd/internal/couchcore/park.go` | modified | now detaches rather than parks |
| `Couch.ActionableThreadInventoryContext` | `cmd/internal/couchcore/actionableinventory.go` | modified | adds one session-list query per refresh |
| `Console.onDetachHotkey` | `cmd/internal/couchtty/console.go` | new | `alt+d` → typed `detach` |

- **`DetachedSessionResolver`** — one method, **`DetachedSessions(ctx context.Context, addresses []ThreadAddress) ([]DetachedSessionObservation, error)`**, satisfied by `ScopedThreadArtifactCollisionChecker` alongside the `NativeBindingResolver` it already satisfies (`artifactcollision.go:205`). Obtained by type assertion on `Couch.Artifacts`, the pattern `actionableinventory.go:155` and `resume.go:192` already use.
  - **Why it takes addresses rather than returning the whole set:** the session-name index is **per repo scope** — `artifactpath.Resolve` puts it at `<dataDir>/repos/<RepoScope>/session-names.jsonl` (`paths.go:353,500`), and `PairSession` (`artifactcollision.go:130-167`) reads exactly one scope's index. A no-argument whole-set method would need a `<dataDir>/repos/*` enumeration the checker does not have and this plan should not add. Taking addresses mirrors the existing `PairSession(address)` / `ResolveEstablished(scope, tag, agent)` shape (`ARCH-DRY`), and the caller already holds the record set.
  - **Injected into:** `ActionableThreadInventoryContext`, which passes only **candidate** addresses — records with zero incarnations, no verified park and a `LatestLaunchProfile`, i.e. the only records that could *be* detached — and hands the result to the pure projector. Bounding the query to candidates is what keeps the refresh cost proportional to detached threads rather than to all threads. The existing `FakeThreadArtifactCollisionChecker` (`artifactcollision_fake.go`) gains a `detachedSessions` field and hook, so every projection test stays fake-driven — `ARCH-MOCK`.
  - **Future extensions:** client counts, or an "attached by another client" state.
- **`ProcOps.SignalGroup(pid int, sig os.Signal) error`** — signals the whole process group. `Couch.Detach` runs from `CouchLiveOwnerExecutor` (`operationdispatch.go:176`) with no `Handle`, and the only seam it has today is `ProcOps.Signal` (`procops.go:110-119`), which is `os.FindProcess(pid).Signal(sig)` — a **single PID**. Under couch, `launcher/osruntime.go:399-408` deliberately returns nil `SysProcAttr` (no `Setsid`) when `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG` are set, so the session-watcher and title-poller share the actor's group; single-PID signalling would orphan them on every `alt+d` and on every thread of every `leave`.
  - **DRY rationale:** the implementation is the existing `signalOwnedProcessGroup` (`runner.go:180-190`, `syscall.Kill(-pid, sig)` with `ESRCH` treated as success), lifted from `execHandle` so both the Handle-holding and Handle-less callers share one group-signal implementation rather than two.
  - **Injected into:** `Couch.Detach`. `FakeProcOps` records group signals in its existing signal log, so Task 9's "no SIGKILL is ever sent" guard is assertable.
- **`ThreadStore.DeleteStart`** — its no-verified-park branch stops deleting records that carry a `LatestLaunchProfile`, clearing the start claim instead. See the rollback-authority paragraph above; this is the data-loss fix, and it is a precondition for detached resume rather than an optional hardening.
- **`ThreadStore.RetireIncarnation(address, expectedRevision, identity ProcessIdentity) (ThreadRecord, error)`** — removes the one incarnation whose exact `{PID, Identity}` matches, leaving the record with zero incarnations, no verified park, and its `LatestLaunchProfile` intact. Refuses if the record has an open park or start transaction, if no incarnation matches exactly, or if the revision moved.
  - **Injected into:** `Couch.Detach` only. It is deliberately *not* a general "clear incarnations" verb: exact process identity is the authorization, the same rule `observeExactProcess` (`couch.go:587-602`) and `MarkIncarnationUnknown` already use, so a recycled PID cannot retire a live thread.
  - **DRY rationale:** it is `FinalizePark`'s incarnation-removal half (`threadstore.go:391`) without the park transaction, `VerifiedPark` write or `ParkHistory` append, reusing the same `UpdateExistingThread` revision-checked write path rather than opening a second one.
  - **Future extensions:** a couch-crash reconciler that retires incarnations proved dead at startup would use this same transition. Deliberately out of scope here — see "Known gap" below.
- **`Couch.Detach(ctx, address)`** — `ProcOps.SignalGroup(pid, TermSignal)`, then a bounded wait for exit, then two proofs before any durable write: the `{scope, tag}` zellij session is still observable, and the exact process is gone. Only then `RetireIncarnation`. It **never** calls `Artifacts.Quiesce` or `DeleteSession`, and it does **not** reuse `handleCleanup` (see above — that path SIGKILLs unconditionally). A client that will not exit, a vanished session, or a failed CAS leaves the thread live and occupied.
  - **The bounded wait is a poll, and the plan should say so:** `Couch` has no wait seam — `Wait` belongs to `PairLifecycleController` (`park.go:97`) — so detach polls `observeExactProcess` (`couch.go:587-602`) on `c.sleep`/`Clock` under the existing operation deadline, the same shape `awaitThreadRegistration` (`couch.go:514-537`) already uses. No new goroutine, no new timer source.
  - **It refuses a non-live incarnation.** `Leave` accepts `IncarnationUnknown` as active (`park.go:145-153`), but detach must not retire one: an `unknown` incarnation is precisely the state the fail-closed projector exists to keep out of the switcher, and retiring it would let an unproven thread present as cleanly detached. Detach requires exactly one `IncarnationLive` with matching process identity.
  - **`Leave` SKIPS an unknown-incarnation thread and reports it; it does not park it.** Parking is the destructive option, and taking it on state couch cannot prove is the opposite of what Decision 3 is for — an `unknown` incarnation may be an agent mid-write. Skipping leaves the thread occupied, which the next couch startup already handles, and `LeaveResult` names it so the operator learns about it rather than discovering it later. Leave still exits: an unprovable thread is not a reason to trap the operator in couch.
  - **Injected into:** the typed operation table; the console dispatches it like every other operation.
  - **Future extensions:** none — a detach that needed options would be a different operation.
- **`Couch.Leave`** — the serial loop stays; the per-thread work becomes `Detach` instead of `Park`. Three things in it are park-shaped and must change with it: the `record.Park != nil` → `Recover` branch (`park.go:118-145`) has no detach counterpart, so a mid-park thread is *parked* to completion rather than detached (it is already being shut down; interrupting that is worse); the `parkResult.Thread.VerifiedPark != nil` success assertion becomes "the incarnation is retired and the session survives"; and `LeaveResult.Parked` is renamed — it now names threads that were detached, and leaving the old name is the kind of lie the next reader pays for.

**Known gap, deliberately out of scope.** A couch that dies *without* leaving cleanly (crash, SIGKILL, power loss) still leaves stale `IncarnationLive` records whose threads are invisible and unresumable. That is **pre-existing** behaviour — `reconcileInterruptedStarts` only reconciles open start transactions — not something this milestone introduces, and closing it means a startup reconciler that retires incarnations proved dead by `observeExactProcess`. `RetireIncarnation` is the transition such a reconciler would use, so this plan builds toward it without building it. File it as a follow-up issue at M2 close rather than absorbing it here.

**Architecture and operating envelope (M2)**

- `ARCH-PURE`: the detached rule is one branch in the existing pure projector; process teardown and session observation stay in the `Couch` shell.
- `ARCH-DRY`: no new zellij observation path, no new resume effect, no new confirmation machinery; `Leave`'s loop is reused with a different per-thread verb.
- `ARCH-MOCK`: the zellij dependency is already behind `ScopedThreadArtifactCollisionChecker` with a stateful fake and a live conformance target (`make test-couch-zellij-live`, `TestSessionQuiescenceLive`). Detach is the *inverse* assertion of quiescence — the session must still be there — so the live check gains a detach case rather than a new harness.
- `ARCH-PURPOSE`: the deliverable is the class, not the instance. `alt+d`, the detached row, reattach, startup selection (M3) and `leave` all derive from the one detached state; none is deferred.
- `ARCH-CONSTRAINTS`: the inventory refresh is **not** on the keystroke path — it runs on the existing single-flight refresh worker with one dirty follow-up (`console_menu.go:88-113`), and the switcher renders from the last-good projection meanwhile. **The honest cost is 2 + N subprocess spawns per refresh, not one.** `ZellijSource.Snapshot` (`launcher/zellij.go:15-40`) runs `zellij list-sessions --short`, `zellij list-sessions --no-formatting`, and one `zellij --session <name> action list-clients` **per pair session** for the client count (`zellij.go:30,41`). N is bounded to *candidate* records — zero incarnations, no verified park, a launch profile — not to all threads, which is the mitigation; a couch with no detached threads pays 2 spawns and no per-session calls. The 100 ms first-progress budget applies to the *open*, which reads the in-memory projection, so it is unaffected. Measure the refresh against the committed `BenchmarkMenu100` fixture before M2 closes; if candidate-bounded refresh still regresses the 16 ms refresh-apply budget, the answer is to cache the snapshot across refreshes, not to move the query onto the keystroke path. Detach itself is a bounded lifecycle operation on the existing capacity-one queue, under the same operation deadline park uses; a hung teardown blocks that queue, not input or repaint.

### Task 7: Observe detached sessions

**Files:**
- Modify: `cmd/internal/couchcore/artifactcollision.go`, `artifactcollision_fake.go`
- Modify: `cmd/internal/couchcore/artifactcollision_test.go`

- [ ] **Step 1: Write failing tests for `DetachedSessions`.** Strategy: drive the scoped checker over session-index and zellij-listing states — session present with clients, present with zero clients, exited, absent, malformed index, listing error, and a session name that binds to no known thread. Mechanical guard: an observation is emitted only for a live zero-client session whose name binds to an exact `{scope, tag}`; every ambiguous or unreadable state emits nothing and a listing error propagates rather than becoming an empty set.
- [ ] **Step 2: Run `go test ./cmd/internal/couchcore -run 'DetachedSession' -count=1`; confirm failure.**
- [ ] **Step 3: Implement `DetachedSessions` on `ScopedThreadArtifactCollisionChecker`,** reusing the session-name index read that `PairSession` (`artifactcollision.go:126`) already performs and `launcher`'s existing state classification. Add `var _ DetachedSessionResolver = ScopedThreadArtifactCollisionChecker{}`. Extend the fake with the same state model.
- [ ] **Step 4: Run the focused tests; confirm pass. Commit.**

### Task 8: Project the detached row

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go`
- Modify: `cmd/internal/couchcore/actionableinventory_test.go`

- [ ] **Step 1: Write failing tests for `ProjectActionableThreads` with detached observations.** Strategy: cross the new observation against every existing record class — reserved, mid-park, verified-park, live-with-matching-TTY, **live-without-TTY (the stale-incarnation case)**, multi-incarnation, zero-incarnation without a launch profile, two observations for one address, one observation for an unknown address. Mechanical guards: `ThreadDetached` is emitted only for a record with **zero incarnations**, no verified park, a `LatestLaunchProfile`, and exactly one matching observation; a record still carrying an incarnation stays **hidden** even when a detached observation matches it (this is the fail-closed property — a crashed couch must not look like a clean detach); and `ThreadLive`/`ThreadParked` results are byte-identical to today for every input containing no detached observation (a regression fixture pins this).
- [ ] **Step 2: Run `go test ./cmd/internal/couchcore -run 'ProjectActionableThreads' -count=1`; confirm failure.**
- [ ] **Step 3: Implement.** Add `ThreadDetached` to the state enum and one branch to `actionableThreadState` (`actionableinventory.go:104`); add the parameter to `ProjectActionableThreads` and the single query to `ActionableThreadInventoryContext`. Add `.Detached()` beside `.Live()`.
- [ ] **Step 4: Run `go test ./cmd/internal/couchcore -count=1`; confirm pass. Commit.**

### Task 9: The detach operation

**Files:**
- Create: `cmd/internal/couchcore/detach.go`, `detach_test.go`
- Modify: `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `park.go` (`Leave`)
- Modify: `cmd/internal/couchcore/ops_test.go`, `operationdispatch_test.go`, `park_test.go`

- [ ] **Step 1: Write failing tests for `ThreadStore.RetireIncarnation` first.** Strategy: cross record shape (zero / one / several incarnations; open park transaction; open start transaction; verified park present) against the supplied identity (exact match, recycled PID with a different start token, no match) against revision (current, stale). Mechanical guards: exactly the exactly-matching incarnation is removed and nothing else on the record changes; `LatestLaunchProfile` survives; every non-matching or transaction-bearing case refuses without a write; a stale revision refuses.
- [ ] **Step 2: Run `go test ./cmd/internal/couchcore -run 'RetireIncarnation' -count=1`; confirm failure. Implement it as `FinalizePark`'s removal half over `UpdateExistingThread`; re-run and confirm pass.**
- [ ] **Step 3: Write failing tests for `Couch.Detach` and the retargeted `Leave`.** Strategy: drive teardown and post-teardown observation through the existing fake Runner/Proc/artifact seams across: clean detach; the client ignoring SIGTERM until the deadline; session absent after exit; session still attached (non-zero clients) after exit; the process still alive after the wait; context cancellation mid-wait; detach of an address with no live incarnation; a `RetireIncarnation` CAS failure after a successful teardown; and `Leave` over zero, one and several threads, with a mid-park thread among them and with a failure in the middle. Mechanical guards, each asserted rather than assumed:
      1. `Quiesce`/`DeleteSession` is **never** called on the detach path (assert the fake's call log — this is the entire difference from park).
      2. **No SIGKILL is ever sent** on the detach path (assert the fake `ProcOps` signal log) — this is what makes Decision 3 safe to apply to every thread on the way out.
      3. `RetireIncarnation` is called only after *both* proofs (session survives, process gone); a failed proof leaves the record untouched and the thread live.
      4. After a successful detach the record has zero incarnations, no verified park, and an intact `LatestLaunchProfile`, so `hasActiveIncarnation` is false and `DecideResume`'s occupied-incarnation gate passes.
      5. `Leave` parks a mid-park thread to completion rather than detaching it, and reports partial progress on failure without closing the console.
- [ ] **Step 4: Run `go test ./cmd/internal/couchcore -run 'Detach|Leave' -count=1`; confirm failure.**
- [ ] **Step 5: Implement `Couch.Detach` in `cmd/internal/couchcore/detach.go`;** declare `detach` in `Operations()` with a confirmation-free presentation and the same execution owner `park` uses, and route it in `DispatchOperation`. Retarget `Leave`'s per-thread call and rename `LeaveResult.Parked` (no consumer outside `park.go:143` and `park_test.go:281`).
- [ ] **Step 5b: Retire the superseded "couch has no detach" invariant — in the same step that violates it, not later.** Three places assert it and one of them is a test that fails the moment `detach` is declared:
      - `cmd/internal/couchcore/ops_declarations_test.go:55`, `TestParkLeaveResumeAndNoCouchDetachSurface`, contains `case "detach": t.Fatal("Couch exposes a detach operation")`. **Invert it** — assert `detach` is declared with the expected effect and owner — rather than deleting it; it encodes a decision that is being reversed, and an inverted test records the reversal where the next reader will find it. Rename it accordingly.
      - `README.md:302` — "Alt+d remains Pair-local detach; Couch exposes no detach operation."
      - `atlas/couch.md:333` — the same claim in prose.
      Leaving these to Task 11 means Task 10 Step 4's `go test ./cmd/...` fails for a reason two tasks away from its cause.
- [ ] **Step 5c: Add `"detach"` and `"leave"` to `consumeExpectedParkExitLocked`** (`console.go:1234-1240`), which today matches only `origin.Operation == "park"`. Without it every detach — and every thread on `leave` — pushes a spurious `ExitNotice` into the operator's status row, which is exactly the noise the notification design is trying to keep meaningful.
- [ ] **Step 5d: Fix `couch --list`'s zero-incarnation rendering** (`couchcmd/run.go:596-598`), which prints "(no agent running)". For a detached thread the agent *is* running — only the client is gone — so the CLI would contradict the switcher.
- [ ] **Step 6: Register `cmd/internal/couchcore/detach.go` in `NonArtifactSources`** (`cmd/internal/artifactpath/manifest.go`) — same requirement as Task 1 Step 5; without it `make test` fails at Task 11.
- [ ] **Step 7: Run `go test ./cmd/internal/couchcore ./cmd/internal/artifactpath -count=1`; confirm pass. Commit.**

### Task 10: Reattach a detached thread

**Files:**
- Modify: `cmd/internal/couchcore/resume.go`, `resume_test.go`, `resume_launch_test.go`
- Modify: `cmd/internal/couchcore/admission.go:183` (the second verified-park gate; deleted at M4)
- Modify: `cmd/internal/couchcore/threadstore.go` (`DeleteStart`'s accept predicate), `threadstore_test.go`
- Modify: `cmd/internal/couchtty/menu.go` (`menuActionItems`, the `KeyEnter` branch at `menu.go:369-379`)
- Modify: `cmd/internal/couchtty/menu_test.go`

- [ ] **Step 1: Write failing tests.** Strategy: for `DecideResume`, cross verified-park presence against detached-proof presence against launch-profile presence against incarnation state (zero / live / creating / unknown) **against `ParkHistory` tombstone presence**, including both-proofs-present and neither-present. The tombstoned-history × detached cross is the one that would otherwise ship a permanently unreattachable class. For the menu, drive `KeyEnter` and `KeyTab` over live/parked/detached rows. Mechanical guards: a detached record resumes without a verified park and without clearing one; **a record carrying any occupied incarnation still refuses even with a detached proof** (the `resume.go:73-86` gate is unchanged — retirement is what a real detach relies on, not a relaxed check); a record with neither proof refuses; Enter on a detached row dispatches `resume`; its action list offers `detach` for live rows and `resume` for detached ones.
- [ ] **Step 1b: Add the rollback test that pins the data-loss fix.** Strategy: drive a detached resume to failure at each post-claim site (`resume.go:233,236,242`) and through `StartRollback` at a fresh `New()`. Mechanical guard: **the thread record still exists** afterwards with its name, description and `LatestLaunchProfile` intact, and the row returns to `ThreadDetached` on the next refresh. Write this before the fix; on today's code it fails by deleting the record, which is the behaviour being removed.
- [ ] **Step 2: Run the focused tests; confirm failure.**
- [ ] **Step 3: Implement — three edits, not one.**
      1. `resume.go:88-95`: add the detached branch to the `VerifiedPark == nil` refusal, carrying the detached proof in a new `ResumeEligibilityInput` field (the proof has to reach `DecideResume` somehow, and a field keeps the function pure). Leave the occupied-incarnation loop at `:73-86` untouched — retirement is what a real detach relies on, not a relaxed check.
         **The detached proof is checked BEFORE the `ParkHistory` tombstone scan**, and the ordering is load-bearing. That scan (`resume.go:89-92`) refuses on *any* tombstoned entry, has no `break`, and `AbandonPark` appends tombstones permanently (`threadstore.go:414-415`). So a thread that was once abandoned mid-park, then started again and detached, would be **permanently unreattachable** if the detached branch sat after it. The tombstone rule means "there is no valid park to resume from"; a detached thread is not resuming from a park at all, so the rule does not apply to it. Order the branch accordingly and say so in the code comment.
      2. `admission.go:183`: widen `ReconcileResumeAdmission`'s `candidate.VerifiedPark == nil` refusal the same way. Without this the row still fails with "is not verified parked" after `DecideResume` passes.
      3. `threadstore.go`, `DeleteStart`'s no-verified-park accept predicate: refuse deletion when `LatestLaunchProfile != nil` and clear the start claim instead.
      Then in `menu.go` treat `ThreadDetached` like `ThreadParked` for Enter (`operation = "resume"`, the arm at `menu.go:369-379`) and give `menuActionItems` (`menu.go:862`) a detach entry for live rows.
- [ ] **Step 3b: Assert the native binding still resolves for a 0-client session.** A detached record must reach `BindingEstablished` — if `bindingResumeDiagnostic` (`resume.go:107`) refuses because the session has no clients, the whole reattach path is dead and every test above would still pass on mocked bindings.
- [ ] **Step 4: Run `go test ./cmd/... -count=1`; confirm pass. Commit.**

### Task 11: `alt+d` and the M2 close

**Files:**
- Modify: `cmd/internal/couchtty/keys.go`, `console.go`, `keys_test.go`, `console_test.go`
- Modify: `atlas/couch.md`

- [ ] **Step 1: Write failing tests.** Strategy: feed `ChordEncodings(ChordAltD)` through `FeedHit` split across read boundaries and inside a paste, and through a console with an actor focused and with the panel focused. Mechanical guards: exactly one `HitDetach` per press with a correct split; no fire inside a paste; `alt+d` on an actor dispatches typed `detach` for that actor; `alt+d` on the panel is a no-op notice (there is no actor to detach).
- [ ] **Step 2: Run the focused tests; confirm failure.**
- [ ] **Step 3: Implement.** Add `seqDetach`/`HitDetach` built from `workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltD)` exactly as `seqPark` is (`keys.go:69-75`); add the `processInput` branch and `onDetachHotkey`.
- [ ] **Step 4: Add the live conformance case** to `TestSessionQuiescenceLive`'s neighbourhood: detach an ephemeral real session and assert the session survives with zero clients (the inverse of the quiescence assertion). Gate it behind the existing `make test-couch-zellij-live`.
- [ ] **Step 4b: Land M2's documentation with M2's behavior.** Add the `Alt+d detach` row to `menuControls` here (Task 5 deliberately left it out so no milestone ships a documented key that does nothing), and rewrite `README.md:350-352` — "parks every active actor sequentially, and returns to the parent shell only after all parks are verified" becomes false the moment `Leave` detaches. No other task claims that sentence.
- [ ] **Step 4c: Measure the refresh, do not assert it.** Run the committed `BenchmarkMenu100` fixture plus a refresh-apply timing with N detached candidates, and record the numbers in the issue `## Log`. The envelope paragraph claims 2 + N subprocess spawns bounded to candidates and a 16 ms refresh-apply budget; without this step that claim ships unverified. If it regresses, cache the snapshot across refreshes — do not move the query onto the keystroke path.
- [ ] **Step 5: Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`; confirm pass. Update `atlas/couch.md`** — the detached state and its two proofs, `RetireIncarnation` as the second incarnation-removal path beside `FinalizePark`, the `Leave`-detaches change, and the `alt+d` interception (which contradicts the current "Alt+d remains Pair-local detach and is not a Couch operation" sentence).
- [ ] **Step 6: File the follow-up issue** for startup reconciliation of stale `IncarnationLive` records left by a crashed couch (`sdlc issue new`), referencing `RetireIncarnation` as the transition it would use. This is the pre-existing gap named above; recording it is what keeps it from being silently absorbed.
- [ ] **Step 7: `sdlc milestone-close --issue 170 --milestone M2`.**

---

## Chunk 3: Start or resume in a folder (M3)

**What the Spec's "live" maps to, stated rather than assumed.** Behavior #1 and the first Done-when bullet say `couch` resumes the session already there, "live or parked". couch is a singleton holding a supervisor lease for the whole console run, so at *startup* there is no other couch hosting a session: a thread whose zellij session is still up but has no pair client is precisely what M2 calls **detached**. "Live or parked" therefore means "detached or parked", and that is what the selector covers. Two neighbouring states are deliberately excluded:

- **Attached elsewhere** — a zellij session with non-zero clients (`SessionAttached`) that couch did not start. It yields no `DetachedSessionObservation`, so it is never selected, and stealing it would be wrong.
- **Stale `IncarnationLive` from a crashed couch** — invisible to the projection, therefore never selected. This is the pre-existing gap named at M2; the follow-up issue filed there owns it. Startup creating a new root in that case is the same behaviour as today, not a regression.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SelectUniqueResumableRoot` | `cmd/internal/couchcore/startup.go` | modified |
| `SelectUniqueParkedRoot` | `cmd/internal/couchcore/startup.go` | deleted |

- **`SelectUniqueResumableRoot`** — renames and widens `SelectUniqueParkedRoot` (`startup.go:11`) from `State == ThreadParked` to `State == ThreadParked || State == ThreadDetached`, with the exact scope + physical path match and the exact-cardinality-one rule unchanged. Deliberately no ranking and no prompt: a parked and a detached row at one path is *two* matches and therefore creates a new root, as two parked rows do today.
  - **DRY rationale:** one selector, not a parked one and a detached one — `ARCH-DRY`. The rename is load-bearing: leaving the name `Parked` while it selects detached rows is a lie the next reader pays for.
  - **Future extensions:** preferring detached over parked (warm over cold) is a ranking policy and is deliberately **not** added here; the Spec's rule is exactness, and a preference would have to be designed as policy.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.StartInteractive` | `cmd/internal/couchcore/startup.go` | modified | path/git resolution, actionable inventory, `ResumeContext`, the resolved start path |

- **`Couch.StartInteractive`** — unchanged in shape; it swaps which selector it calls. It remains the thin IO shell around the pure selector, and every dependency is already in the Couch composition root, so its tests keep using the existing stateful fakes (`ARCH-PURE`, `ARCH-MOCK`). No new seam is introduced by M3 — the detached observation it now depends on was added at M2 and reaches it through the same `ActionableThreadInventoryContext` call it already makes.

**Architecture and operating envelope (M3):** unchanged from `#167` — one target resolution, one local actionable snapshot, O(n) pure selection, no fleet scan, no prompt, no goroutine fan-out. Inventory failure or a resume refusal still stops startup without creating a fallback actor.

### Task 12: Widen the startup selector

**Files:**
- Modify: `cmd/internal/couchcore/startup.go`, `startup_test.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [ ] **Step 1: Write failing tests.** Strategy: cross row state (parked / detached / live / hidden) against scope and path identity against cardinality, including one-parked-one-detached-at-one-path and alias paths. Mechanical guard: exactly one resumable exact match resumes; zero or several create a new root; a live row is never selected (couch is a singleton — a live row means this couch already hosts it).
- [ ] **Step 2: Run `go test ./cmd/internal/couchcore -run 'SelectUnique|StartInteractive' -count=1`; confirm failure.**
- [ ] **Step 3: Rename and widen the selector; update `StartInteractive`'s call.**
- [ ] **Step 3b: Make detached rows carry a physical `WorkingPath`, like parked ones.** `ActionableThreadInventoryContext` re-resolves the path to physical **only** inside the parked-candidate loop (`actionableinventory.go:169-173`); detached rows would keep the stored path. Since the selector compares paths by exact string, parked and detached rows at one alias path would compare asymmetrically — one matching, one not — which is a selection bug that only shows up on a symlinked tree. Extend the physicalization to detached candidates. Step 1's alias-path cross is what proves it.
- [ ] **Step 4: Add a restart-level acceptance test** in `cmd/internal/couchcmd/run_test.go` using the existing `newRT` temp-namespace runtime: detach a thread, construct a fresh `Couch`, and assert interactive launch reattaches that exact address through production routing to initial Console attach (not below it — `workshop/lessons.md`: reducer support is not user reachability, and a prior close review already caught a test that stopped above `dispatchInitialAttach`).
- [ ] **Step 5: Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`; confirm pass. Update `atlas/couch.md`'s startup paragraph.**
- [ ] **Step 6: `sdlc milestone-close --issue 170 --milestone M3`.**

---

## Chunk 4: Delete the machinery the rescope orphans (M4)

The razor is the issue's own: **machinery that exists only to defend multi-owner or multi-host cases.** Applied honestly it deletes less than the Problem section's framing suggests, because several of the named candidates defend a *single-host* failure instead. Each survivor is listed with its reason — an un-deleted candidate with no stated reason is the ambiguity that caused the overrun in the first place.

### Deleted

| # | Subsystem | Files | Why it goes |
|---|---|---|---|
| D1 | Admission + fleet policy | `admission.go`, `admission_test.go`, `admission_reconcile_test.go`, `policyresolver.go`, **`policyresolver_exec.go`**, **`policyresolver_fake.go`**, `policyresolver_test.go`, `policy_shadow_test.go`, `guard_live_test.go`, `couchcmd/errors.go` | Capacity/incumbency across a *fleet* is the multi-owner case exactly. Also removes couch's dependency on ariadne's `sdlc fleet policy --json` provider, its stateful fake and the `make test-couch-policy-live` conformance target (plus `.github/workflows/couch-policy-conformance.yml`). **Blast radius beyond the file list:** `couchcore.New` takes a `PolicyResolver` and refuses nil (`couch.go:83,90-92`), so its signature changes at both production call sites — `couchcmd/run.go:92-95` and `cmd/probes/couchstartrecovery/main.go:123-126` (the probe passes `NewFakePolicyResolver()`, so it **does** break and must be repaired). `cmd/couch/main_test.go:38,121` stubs an `sdlc` binary and asserts the `sdlc fleet policy --path … --json` call; that stub and its assertions go too. |
| D2 | Start grants | `startgrant.go`, `startgrant_test.go` | A 256-bit one-shot capability table with TTL, capacity 16 and collision retries defends a prepared start against *another owner*. In-process and single-owner, the token is redundant with the start form's own armed-submit identity, and `SpawnPrepared` already re-resolves and compares the fingerprint. |
| D3 | Legacy migration + cutover | `migration.go`, `migration_test.go`, `CutoverLegacyActors`, `legacyThreadAddress`, manifest `LegacyCutover`/`LegacyMigrationVersion`, `ThreadIncarnation.LegacyActorID` | Upgrade compatibility for a pre-ThreadStore store, run on every `New` and a no-op since the first run. The operator's live store is already cut over — verified against `~/.local/share/pair/couch/threadstore/`: manifest carries `legacy_cutover: true` and `legacy_migration_version: 1`, and no record under `records/` contains a `legacy_actor_id`. |
| D4 | The actor loop | `actor.go`, `actor_test.go` only | Built, unit-tested, never instantiated — groundwork for `pair#147`, which is punted. `atlas/couch.md` already names it a `pair#170` deletion candidate. **`mailbox.go` stays untouched**: it holds `Message` and the pure bounded/collapsing `Enqueue`, which `couchtty/notice.go` uses for the exit/bell feed. `actor.go` is the goroutine loop around it and is the only unreferenced half. |
| D5 | Registry-era dead surface | in `couch.go`: `List`, `Get`, `Entry`, `SetName`, `SetDescription`, `IsLive`, `Views`, `Summarize`, `TreeSummary`, `ActorView`, `treeFor`; `ThreadStore.ManifestGeneration`; `ReconcileAdmission` (non-prepared); in `couchcmd/run.go`: `renderTrees` and the `case []couchcore.TreeSummary` | No non-test caller; verified by deleting the cluster and building. `renderTrees` is unreachable because nothing produces a `TreeSummary`. Three consequences to handle rather than discover: the `"sort"` import at `couch.go:11` becomes unused; removing `ManifestGeneration` breaks `storejournal_test.go` and `admission_reconcile_test.go` as well as `threadstore_test.go`; and `List`'s only caller is `couchcmd/run_test.go`, which must be updated in the same commit. |

### Kept, and why

- **Supervisor lease** — it is what *enforces* the singleton couch-lite asserts ("couch remains a singleton"). Deleting it would delete a property the Spec relies on, not a defence of a case we dropped. The Out-of-scope section names admission, start grants and park transactions as candidates; it does not name the lease.
- **Park transaction** — a two-process handshake with a deadline is a *protocol* defence, not a multi-owner one, and it stays wrong in exactly the same ways with one operator. Its Retry/Recover/Abandon modes are the operator's only escape from a stuck park. Deleting it also deletes `VerifiedPark`, which *is* the parked row (`actionableinventory.go:110`) and resume's precondition.
- **Write-ahead journal** — single-host crash safety for multi-file mutations (record + manifest + path preference). Its non-journal helpers (`writeAtomicBytes`, `readOptionalFile`, `syncDirectory`, `strictThreadStoreJSON`) are used throughout `threadstore.go` and must survive regardless, so the deletable remainder is small and the risk is not.
- **Fail-closed actionable projection** — the switcher's only data source, and the thing that keeps reserved/mid-park/mid-start records out of the operator's list. M2 *extends* it.
- **Artifact collision** — four capabilities behind one interface. Only `Claim`/`Release` are "collision", and they serialize tag allocation against standalone `pair`, which still runs. `Registration`, `Quiesce`, `PairSession`, `TriggerQuit` and `ResolveEstablished` are load-bearing for start promotion, post-ack cleanup, park and resume.
- **Start transaction (`starttransaction.go`)** — crash recovery for an interrupted start on one host, not multi-owner defence. Deleting it means rewriting the start path, which is precisely the ontology churn this rescope exists to stop; and `advanceSuccessfulStart` commits the path launch preference in the same journal transaction, so a careless deletion silently loses per-path agent memory.
- **`Couch.Spawn`** — an earlier draft listed it as dead. It has no *production* caller (the CLI and UI go through `PrepareStart`/`SpawnPrepared`), but it is the **test seam over the start path**: a thin wrapper on `resolveStartResolution` + `spawnResolved` (`couch.go:148-155`) with 24 call sites in `couch_test.go` plus `guard_live_test.go:105`. Task 13 Step 4 *rewrites* `spawnResolved` via `CommitStartClaim`, so deleting `Spawn` in the same milestone would drop the coverage of the path being rewritten, exactly when it is least safe to. Kept, and its doc comment updated to say it is the seam rather than a production entry point.
- **`LookupTrees` / `knownTrees` / `Describe`** — an earlier draft of this plan listed these under D5 as dead. They are not: `LookupTrees` (`couch.go:700`) ← `ResolveRef` (`couch.go:740`) ← `operationdispatch.go:197`, the live `"stop"` arm of `CouchLiveOwnerExecutor`; `knownTrees` (`couch.go:675`) and `Describe` (`couch.go:718`) ride the same path. Deleting them breaks the build. Kept.

### The one migration D1 forces

`advanceSuccessfulStart` keys the path launch preference by `incarnation.Policy.RepoIdentity` (`threadstore.go:613-618`), and `RepoIdentity` came from the policy provider. Deleting the provider therefore threatens to orphan `path-preferences/` — the operator's per-path agent+argv memory.

It does not have to. The provider's `repo_identity` is the git common dir (verified against the live store: `"repo_identity": "/Users/xianxu/workspace/tools/.git"` for `physical_path: /Users/xianxu/workspace/tools`, and against ariadne's own fixtures, `fleet/json_test.go:61`). couch already has a `GitRunner` seam, so the identical value is derivable locally. Replace `ThreadIncarnation.Policy` with `ThreadIncarnation.RepoIdentity string` populated from the local git resolution, and **every existing preference file stays readable**. A test pins the exact digest for a known `{repo identity, path}` pair against the current key so the migration is proved, not assumed.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ThreadIncarnation.Policy` → `RepoIdentity` | `cmd/internal/couchcore/thread.go` | modified |
| `Admission`, `AdmissionDecision`, `CapacityExceededError`, `PolicyResult` | `cmd/internal/couchcore/admission.go`, `policyresolver.go` | deleted |
| `StartGrantStore`, `StartGrantToken` | `cmd/internal/couchcore/startgrant.go` | deleted |
| `MigrateLegacyRecord` | `cmd/internal/couchcore/migration.go` | deleted |
| `Actor` | `cmd/internal/couchcore/actor.go` | deleted |
| `TreeSummary`, `ActorView` | `cmd/internal/couchcore/couch.go` | deleted |

- **`ThreadIncarnation.RepoIdentity`** — replaces the persisted `PolicyResult` with the one field anything still reads from it: the git common dir that keys the path launch preference. A plain string, validated non-empty at the same site `advanceSuccessfulStart` validates today (`threadstore.go:613`).
  - **DRY rationale:** collapses a six-field provider record persisted per incarnation down to the single value with a consumer — `ARCH-DRY`, and it is what makes the preference files survive (see "The one migration D1 forces").

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadStore.CommitStartClaim` | `cmd/internal/couchcore/threadstore.go` | new | journaled record CAS |
| `ThreadStore.CommitThreadReplacements`, `DeletePristineThread` | `cmd/internal/couchcore/threadstore.go` | deleted | — |
| `PolicyResolver` seam | `cmd/internal/couchcore/couch.go` | deleted | `sdlc fleet policy --json` |

- **`ThreadStore.CommitStartClaim(address, event StartEvent) (ThreadRecord, error)`** — the single revision-checked transition that replaces the four things admission did besides deciding capacity: clear the pristine `Reservation`, append the first `IncarnationCreating` carrying `{RepoIdentity, StartedAt}`, apply the supplied `StartEvent` (`StartClaimed`) through the **existing** `AdvanceStartTransaction`, and roll the record back to pristine on failure.
  - **Injected into:** `Couch.spawnResolved` (`couch.go:266`) and `Couch.ResumeContext` (`resume.go:219`) — the only two admission call sites. It is a `ThreadStore` method rather than a free function because it is one journaled multi-field CAS, which is exactly what `UpdateExistingThread` already is; it reuses that write path rather than opening a second one (`ARCH-DRY`).
  - **Why not just `UpdateExistingThread`:** the reservation clear and the incarnation append must be one revision-checked commit, or a crash between them leaves a reserved record with a live incarnation — the state `ProjectActionableThreads` hides, i.e. an invisible thread. Naming the transition keeps that atomicity checkable.
  - **Future extensions:** none intended. If capacity ever returns it is a *decision* taken before this call, not a widening of it.
- **`PolicyResolver` seam deletion** — removes couch's only cross-repo runtime dependency (ariadne's `sdlc fleet policy --json`), its stateful fake, its strict decoder, and the `make test-couch-policy-live` conformance target plus the workflow that runs it weekly. `ARCH-MOCK` obligations shrink rather than move: nothing replaces the seam.

**Architecture and operating envelope (M4)**

- `ARCH-PURPOSE`: deletion is the deliverable, so it is proved by *absence*, not by narrative. A replacement is incomplete while its superseded surface still compiles (`workshop/lessons.md`), so each deletion task ends with a build over the whole tree and the removal of the subject's tests — not their skipping.
- `ARCH-DRY`: `CommitStartClaim` reuses `AdvanceStartTransaction` and the existing journal write path rather than introducing a second start-state authority.
- `ARCH-CONSTRAINTS`: no runtime path changes shape. Startup loses two no-op migration passes and one subprocess policy query per start; the switcher's envelope is untouched. The one behavioural risk is the preference key, which Task 13 Step 1 pins with a characterization test before anything moves.

### Task 13: Delete admission and the policy provider

**Files:**
- Delete: `admission.go`, `admission_test.go`, `admission_reconcile_test.go`, `policyresolver.go`, `policyresolver_test.go`, `policy_shadow_test.go`, `guard_live_test.go`, `couchcmd/errors.go`
- Modify: `couch.go` (`spawnResolved`, `resolveStartResolution`), `resume.go`, `startresolution.go`, `thread.go`, `threadstore.go`, `threadrecord/record.go`, `couchcmd/run.go` (`renderError`), `Makefile.local` (drop `test-couch-policy-live`), the macOS policy-live workflow

- [ ] **Step 1: Write the failing preference-key test first.** Strategy: pin `pathLaunchPreferencePath` for a known `{git common dir, physical path}` pair to the exact digest produced today, and drive a successful start through the production seams asserting it reads back a preference written before the change. Mechanical guard: the key is byte-identical across the change, so an accidental re-keying fails loudly rather than silently orphaning the operator's agent memory.
- [ ] **Step 2: Run it against the pre-change tree and confirm it passes** (it is a characterization test — it must pass before, and after).
- [ ] **Step 3: Replace `ThreadIncarnation.Policy` with `RepoIdentity string`,** sourced from the existing git resolution in `resolveStartResolution`. Update `threadrecord` validation.
- [ ] **Step 4: Replace the admission call sites.** `spawnResolved` (`couch.go:266`) and `ResumeContext` (`resume.go:219`) need the three things admission did besides capacity: clear `Reservation`, append the first `IncarnationCreating`, and (for resume) apply `StartClaimed` atomically — plus the pristine-reservation rollback on failure. A single `ThreadStore.CommitStartClaim` CAS covers all four; it replaces `CommitThreadReplacements` and `DeletePristineThread`, whose only callers were admission. **It must carry M2's widened resume precondition forward** — `ReconcileResumeAdmission` accepts *either* a verified park or a proved-detached record after M2, and a `CommitStartClaim` that reverts to verified-park-only would silently re-break detached reattachment two milestones after it was fixed. The M2 resume tests are the regression guard; run them at this step, not just at the end.
- [ ] **Step 5: Delete the files; drop `Reservation`, `ClaimGeneration` and the capacity branch of `renderError`.** Keep `Reservation` only if Step 4 still needs the allocate→commit handshake; if allocation commits directly, delete it and its four `threadstore.go` guards.
- [ ] **Step 6: Run `go test ./cmd/... -count=1` and the preference-key test; confirm pass.** Commit.

### Task 14: Delete start grants, migration, the actor loop, and the dead surface

**Files:** as listed in D2–D5, plus `ops.go`, `operationdispatch.go`, `couchcmd/run.go`, `couchtty/menu.go`, `console_menu.go`, `menu_async.go`, `README`

- [ ] **Step 1: Write the failing tests for the collapsed start path.** Strategy: drive the start form's preview → submit sequence through production dispatch with a stale preview, a duplicate submit, an edit between preview and submit, and a concurrent inventory refresh. Mechanical guard: `start` still refuses a resolution whose fingerprint changed (`ErrStartResolutionChanged`), and a double submit still starts exactly one thread — the properties the grant token was carrying, now carried by the fingerprint plus the form's armed-submit identity.
- [ ] **Step 2: Run them; confirm the fingerprint path fails where the token used to cover it.**
- [ ] **Step 3: Delete D2.** `prepare-start` keeps returning a `StartResolution` (so the preview row survives); `start` takes the fingerprint instead of a token. Collapse `PrepareStart`/`SpawnPrepared`, `ops.go:132-147`, `operationdispatch.go:184-195`, `couchcmd/run.go:263-286`, and `MenuFrame.PreviewToken`.
- [ ] **Step 4: Delete D3, D4, D5.** For D4, confirm by compilation that `Enqueue`/`Message` survive for `couchtty/notice.go`.
- [ ] **Step 5: Repair the two string-level couplings the deletions break.** `cmd/internal/artifactpath/manifest.go`'s `NonArtifactSources` allowlist (every production `.go` file must be listed; `coverage_test.go:194`), and `couchcore/plan_contract_test.go`'s digest-pinned file/declaration ledgers for the `#149 M5` and `#151 M3` boundaries. The ledger pins *historical* boundary contracts from Git objects — if a pinned contract cannot survive the deletion, the correct repair is a `## Revisions` note on the pinned milestone, not a loosened digest.
- [ ] **Step 6: Repair `cmd/probes/couchstartrecovery`.** It is built on the start transaction (kept) and `SupervisorOwner` (kept), so the probe itself survives — but `main.go:123-126` constructs `Couch` with `couchcore.NewFakePolicyResolver()`, which D1 deletes, so it *will* fail to build and must be updated to the new `New` signature. Repair, do not delete.
- [ ] **Step 7: Run `go build ./... && env -u PAIR_SESSION_ID -u PAIR_TAG make test`; confirm pass.** Record the line count removed (`git diff --stat main`). Commit.

### Task 15: Close M4 and the issue

- [ ] **Step 0: Disposition ariadne's now-unconsumed policy arm.** couch was the only *programmatic* consumer of `sdlc fleet policy --path P --json` (`ariadne/cmd/sdlc/internal/fleet/`), which carries its own helptext and e2e tests. The CLI arm remains operator-facing on its own merits, so this plan does **not** delete a peer repo's surface — but leaving the cross-repo consequence unstated is how a surface rots. Record it in the M4 close: name the arm, say couch no longer calls it, and let ariadne decide. File a peer-repo note rather than an edit.
- [ ] **Step 1: Update `atlas/couch.md`.** Delete the "Identity and admission" section's policy/admission paragraphs, the start-grant paragraph in "Spawning", the "Actor loop — built, unit-tested, never instantiated" section, and the legacy-migration sentences in "What exists today". Move `pair#170` from "Planned, not built" into the delivered surface. Keep `atlas/index.md` linking every file.
- [ ] **Step 2: Update `workshop/projects/couch.md`** — tick `pair#170`, record `**actual:**`, and append a scope-event line noting `leave` now detaches. (The rescope scope event and the `#147`/`#148`/`#153` dispositions already landed on 2026-09-02 and need no repeat.)
- [ ] **Step 3: Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test` and `git diff --check`.** Record the exact commands and results in the issue `## Log`.
- [ ] **Step 4: Operator smoke on the real stack** (Ghostty → couch → pair → zellij → claude), because the switch rule, both `ctrl+backspace` encodings, `alt+d` and reattach are terminal behaviours no test proves end to end. Check: ctrl-space opens on the paged actor; ctrl+backspace returns home after two notification hops; alt+d detaches and the row stays listed; `couch` in that tree reattaches it; alt+x on the panel leaves couch without killing the agents.
- [ ] **Step 5: `sdlc milestone-close --issue 170 --milestone M4`, then `sdlc close --issue 170 --verified '<evidence>'`.** Let `close` measure actuals; do not hand-type `--actual`.

---

## Open, and deliberately not in this plan

Carried from the issue `## Log` so they are not silently absorbed:

- **couch-lite does not solve the problem the project was opened for.** The original pain was *forgetting a thread exists*, with a dated cost (the rogii submission whose 2026-08-05 deadline passed unnoticed). A switcher does not catch that; a durable `{tree, what, when}` list plus a clock would, and needs none of the fleet, transport or advisor machinery. M2's detached state makes such a list cheaper to build later — a detached thread is exactly a durable "this exists and nobody is looking at it" — but this plan does not build it.
- **A durable append log of operation attempts** (`{op, args, outcome, error}`) generalising `#169`. D1/D2 remove two sources of transient failure, which shrinks the problem without solving it.

---

## Revisions

### 2026-09-02 — plan review round 1

Fresh-context review of all four chunks. Eight blocking findings, all confirmed
against the source before acting.

**The material one (Chunk 2).** The first draft claimed detach needed "no
`ThreadRecord` field and no transaction". Half right: no field, but a durable
transition is unavoidable. `FinalizePark` (`threadstore.go:391`) is the only
path that removes an incarnation, and `reconcileInterruptedStarts` only touches
records with an open start transaction — so killing the client would leave
`Incarnations: [{IncarnationLive, PID: <dead>}]` forever. That hides the row in
the projector, and worse, `DecideResume` refuses any occupied incarnation at
`resume.go:73-86` *before* the `VerifiedPark` check, so the thread could never
reattach. The first draft would have shipped a detach whose sessions were
neither listed nor resumable — both Done-when bullets, failed. Fixed by adding
`ThreadStore.RetireIncarnation` (`FinalizePark`'s removal half, minus the park
transaction) and requiring **zero** incarnations for `ThreadDetached`, which
keeps the projector fail-closed rather than teaching it to tolerate stale state.

**Detach must not reuse `handleCleanup`** (advisory, taken): that path SIGKILLs
unconditionally and says so in its own comment. Under Decision 3 it would run
against every thread on every exit, which contradicts the safety argument
Decision 3 rests on. Detach now sends SIGTERM only and fails safe.

**Corrected factual claims.** `LookupTrees`/`knownTrees`/`Describe` are live via
`ResolveRef` ← `operationdispatch.go:197` and were wrongly listed for deletion —
the reviewer compiled the deletion to prove it. `Console.switchTo` is not the
only landing site: `installObservedThreadActor` seeds the first actor, so
seeding the tracker only at `switchTo` would break the Spec's leading
consequence. "The switcher opens on the current thread's row" was false and had
no covering task. `reduceParkHotkey` does need a change for a zero-live-pane
leave. `onExit`'s `c.active = c.root` is a use, not maintenance. D1's file list
and blast radius were incomplete (`policyresolver_exec.go`,
`policyresolver_fake.go`, the `New` signature, the `couchstartrecovery` probe,
`cmd/couch/main_test.go`'s `sdlc` stub). New production files must be registered
in `NonArtifactSources` — planned for deletions, missed for the two new files.
Chunk 3 silently narrowed the Spec's "live" to "detached"; the mapping and its
two excluded neighbours are now stated.

**Scope unchanged.** No milestone moved and nothing was added to the deliverable
beyond `RetireIncarnation`, which is the mechanism the agreed behavior already
required. One pre-existing gap (stale incarnations after a couch *crash*) is
named, excluded, and assigned a follow-up issue at M2 close.

### 2026-09-02 — plan review round 2

Verified the round-1 fixes and found seven more, of which two would have shipped
broken and one destroys data. All confirmed against source before acting.

**Detached reattach was blocked by a second gate the round-1 fix did not reach.**
`ReconcileResumeAdmission` (`admission.go:183`) refuses any candidate with
`VerifiedPark == nil`, and `ResumeContext` calls it at `resume.go:219` *after*
`DecideResume` returns. Widening only `DecideResume` would have shipped a
`ThreadDetached` row whose Enter fails with "is not verified parked". Task 10
now widens both. Admission dies at M4, so this is a one-line cost on
soon-deleted code — cheaper than reordering the milestones to dodge it — and
Task 13 now carries the widened precondition forward into `CommitStartClaim`,
which is where it would otherwise silently regress.

**Detached resume could delete the thread record.** `DeleteStart`
(`threadstore.go:724-756`) branches on the verified park: with one it clears the
incarnation and keeps the record; with none it falls through to `deleteThreadIf`,
whose predicate guards only revision, reservation, metadata and incarnation
count. So any post-claim failure on a detached resume removes the record, its
label, its description and its `LatestLaunchProfile` while the zellij session
keeps running. `starttransaction.go:83-86` names the premise: the verified park
*is* the rollback authority, and a detached thread has none. Fixed as the class
rather than the instance (`ARCH-PURPOSE`): a record carrying a
`LatestLaunchProfile` is durable history and is never deleted. Safe for every
existing caller, since a thread that never started successfully has no profile.

**Detach needed a seam that does not exist.** `Couch.Detach` runs from the live
owner executor with no `Handle`, and `ProcOps.Signal` is single-PID
(`procops.go:110-119`). Under couch the sidecars share the actor's process group
by design (`launcher/osruntime.go:399-408` suppresses `Setsid`), so as written
detach would orphan the session-watcher and title-poller on every `alt+d`. Added
`ProcOps.SignalGroup`, implemented by lifting the existing
`signalOwnedProcessGroup` (`runner.go:180-190`) out of `execHandle` so both
callers share one implementation. Also named the bounded wait as a poll over
`observeExactProcess`, since `Couch` has no wait seam.

**`leave`'s confirmation is thread-bound and must stop being.** A zero address
fails at five sites, not the two named in round 1 — including one that fires
*asynchronously* on the next inventory refresh (`menu.go:1016-1033`), which a
keystroke-only test would never catch. Rather than five `leave`-shaped
exceptions, the confirmation becomes a **global frame**, the shape the menu
already uses for the start form.

**`DetachedSessions` could not be satisfied as scoped.** The session-name index
is per repo scope (`artifactpath/paths.go:353,500`), so a whole-set method needed
a `repos/*` enumeration the checker does not have. It now takes addresses,
mirroring `PairSession(address)`, and the caller passes only *candidate* records
— which also bounds the refresh cost. Related: the envelope claim of "one
subprocess per refresh" was wrong; `ZellijSource.Snapshot` spawns 2 + N
(`launcher/zellij.go:15-41`). Corrected, bounded to candidates, with a
measurement step before M2 closes.

**A test pins the invariant this milestone reverses.**
`ops_declarations_test.go:55` fails the moment `detach` is declared. It is
inverted rather than deleted, in the same step that declares the operation, and
the matching claims in `README.md:302` and `atlas/couch.md:333` move with it.

Smaller corrections folded in: detach refuses `IncarnationUnknown` (retiring one
would let an unproven thread present as cleanly detached);
`consumeExpectedParkExitLocked` gains `detach`/`leave` or every detach pushes a
spurious exit notice; `couch --list` stops printing "(no agent running)" for a
detached thread whose agent is running; the no-notification default routes
through `reconcileRootSelection` rather than assigning a possibly-stale
`ActiveAddress`; `onExit` drops the exited thread from the tracker instead of
recording it as a landing; detached rows get physical-path resolution so alias
paths compare symmetrically; `actorAlive` is named for deletion because Go will
not flag an unused method; and `panelkeys.go:98` is explicitly left alone with
the reason, since the Spec names it and an implementer would otherwise change it.

**Scope still unchanged.** No milestone moved. The additions —
`ProcOps.SignalGroup`, the `DeleteStart` predicate, the global confirmation
frame — are all mechanisms the already-agreed behavior requires.

### 2026-09-02 — plan-quality gate round 1 (`sdlc change-code`)

Three Important findings, four Minor. All addressed as classes.

**PQ-1 — arrival never acknowledged the target's pending notification.** The
Spec says "an actor does not notify while the operator is attached to it", and
the only site honouring it was `onHotkey`'s home-landing block
(`console.go:1085-1090`) — which Task 2 deletes. `switchTo` never acknowledged,
and `onPreviousHotkey` bypasses `runMenuOperation` entirely, so `ctrl+backspace`
home to A would leave A lit, `NewestActor()` would name the actor the operator
is sitting in, and the next `ctrl-space` would open the switcher on it instead of
on whoever paged — the milestone's headline behavior, inverted. The class is
"every landing owes the same two rules", so `switchTo` gained an `arrival`
argument and now applies both there: update the tracker, and acknowledge.
Failure still cannot acknowledge, because a failed switch never lands.

**PQ-2 — the detached branch had to precede the `ParkHistory` tombstone scan.**
That scan (`resume.go:89-92`) refuses on any tombstoned entry, has no `break`,
and `AbandonPark` appends tombstones permanently. A thread once abandoned
mid-park, later started and detached, would have been permanently unreattachable.
Branch order is now specified and the tombstoned × detached cross added to the
test matrix.

**PQ-3 — D5 would have deleted `Couch.Spawn`, the test seam over the start path
M4 rewrites.** No production caller, but 24 `couch_test.go` call sites plus
`guard_live_test.go:105` ride it, and Task 13 rewrites `spawnResolved` beneath
them. Kept as the seam, with its doc comment corrected. `List`'s only caller
(`couchcmd/run_test.go`) is now named too.

**Minors.** `Leave` skips an unknown-incarnation thread and reports it rather
than parking it — parking is destructive and Decision 3 exists to avoid taking
the destructive option on unprovable state. The `Alt+d` help row moves from M1 to
M2 so no milestone documents a key that does nothing, and `README.md:350-352`'s
now-false leave prose gets an owner. The refresh measurement becomes an actual
step instead of a sentence in the envelope paragraph. ariadne's policy arm gets a
recorded disposition rather than a silent orphaning. And the `DeleteStart`
data-loss note was over-claimed: `threadHasMetadata` already protects a *named*
record, so the real exposure is an **unnamed** detached thread's
`LatestLaunchProfile` — narrowed, since the fix is right but the stated reason
was not.
