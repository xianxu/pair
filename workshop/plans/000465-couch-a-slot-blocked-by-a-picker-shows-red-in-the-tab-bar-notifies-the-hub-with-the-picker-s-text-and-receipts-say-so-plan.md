# pair#465 plan: a slot behind a picker is red, notifies the hub, and receipts say so

**Goal:** the operator sees a slot behind an input-holding picker at a glance (red in Couch's tab bar), the hub gets one specific notice after about a minute, and a message held by the picker says so instead of expiring silently.

**Architecture:** the wrapper owns one fact it already computes, *a picker is up* (`pickerActive`, corrected by pair#464). It reports that fact to Couch on the session frames that already carry `Settled` (#421). Couch keeps reporting facts: the registry's liveness carries it, the Console paints it, and the wait watcher (pair#451) reads the picker's text off the tail once per episode. What a picker *says* (kind, title, command, options) is read by one parser over the tail grammar, `couchmessage.ReadPicker`. The watcher and the wrapper's receipt both use it. The wrapper renders and never classifies the agent (tail.go); the parser classifies the rendered tail, so both sides read the same text the same way (ARCH-DRY).

**Base:** pair#464, which fixes `pickerActive` so that the composer coming back closes it. Build only after #464 is on main and merged in. Until then this branch holds only the design.

## Design

### 1. Wrapper → Couch: `Picker` on the session frame
- `couchmessage.SessionFrame` gains `Picker *bool` (omitempty). Like `Settled`, it rides activity/submit frames on hello-v2 sessions only. `nil` means unknown (an older wrapper), and Couch then shows nothing.
- `SessionClient.Picker(open bool)` mirrors `Settle`: it records the value, marks the frame dirty when it changes, and sends at most one frame per `peerActivityInterval` (1s).
- Wrapper (`peer_settle.go`): the settle probe returns `{settled, picker}`. `settleFired` publishes each value that changed. A picker that opens is reported when output goes quiet (3s, `SettleInterval`). One that closes is reported 3s after the confirming keystroke, through the same generation-guarded timer. No new timer and no new source of truth: `picker` is `pickerActive.Load()`.
- Registry: `SessionActivity`/`SessionSubmit` store `picker` the same way they store `settled`. `SlotLiveness.Picker *bool`.

### 2. Tab bar and switcher: red
- `messageService` derives the set of thread addresses whose liveness has `Picker=true` after each registry step. When the set changes, it hands the set to the Console through a setter installed like `SetBroadcastStatus`. The setter only stores the set and asks for a repaint, so the loop goroutine never waits on a Console lock (no lock-order coupling).
- `StatusActor.Blocked`, filled in `statusModelLocked`. `RenderStatusRow` precedence: placeholder > **blocked (red, `\x1b[31m`)** > bell > fade. Red shows even on the active chip: the operator asked to see it at a glance. `blockedSGR` is its own constant, not `broadcast.LiveSGR` (whose indicator check reads that byte string). The switcher's live-row branch states the same precedence (pair#247 keeps the two in step).

### 3. Hub notice: early and specific
- `waitFacts` gains `Picker bool`. `classifyWait` treats a picker as its own reason, `waitPicker = "behind a picker"`, with its own threshold `pickerNoticeAfter = time.Minute`. Quiet time is measured from the wrapper's last output or keystroke, the same quantity #451 uses, so the episode key (`LastActivity`) and the one-notice-per-episode rule are unchanged. The generic `on a prompt` path never fires for a slot that reports a picker, because the picker path owns that episode.
- The watcher's tick goes from 1m to 15s, so the notice lands at 60–75s. A tick reads memory only; tail IO still happens once per episode.
- Body, head first (pair#211): `slot pair:3 behind a picker 1 min (permission prompt): Bash command: git push origin HEAD — Do you want to proceed? [1. Yes | 2. No, and tell Claude what to do (esc)] (couch --peek pair:3; mute: …)`. Title and command are each bounded at 200 bytes; at most 6 options of 80 bytes each. The same mute file as #451.

### 4. `couchmessage.ReadPicker(lines []string) Picker`
- `Picker{Kind, Title, Detail string; Options []string}`. It is found from the bottom-most selected option (`❯ N.`), as `dialogTitle` does today, so numbered prose is never taken for a dialog. Options are the contiguous option lines around it. Title is the nearest non-option line above them. Detail is the box's lines between its top border and the title, joined with `: `, e.g. `Bash command: git push origin HEAD`.
- Kind is decided from the screen only. A title that starts with `Do you want to` or `Would you like to` is a **permission prompt**; anything else is a **question**. An empty Kind means the screen showed no recognizable picker, and the notice says `(picker text not recognized)`.
- `couchcmd`'s `dialogTitle` and `stripTailMarkup` move into couchmessage, next to the tail grammar they parse (tail.go). Existing callers use the moved functions.

### 5. Receipts: `blocked by a picker: <title>`
- In `dispatchPeer`, a picker that is up while the message is not yet pasted counts as a **hold**, the same mechanism as the operator's draft (pair#435). The status goes to `held`, the paste-by clock stops, and the deadline extends to `HoldLimit` (deadline + 30m `MaxHold`). The detail reads `blocked by a picker: <ReadPicker(agentTail(60)).Title>`, or `blocked by a picker` when the title is not recognized.
- The message is delivered when the picker closes: the next dispatch judges the composer as usual. If the picker stays up past the hold limit, the message expires with `held past 30m, blocked by a picker: <title> (…)` from `waitSummaryLocked`.
- A picker that opens during delivery (`PeerOverlayObserved`) is unchanged.

## ARCH notes
- **ARCH-FUNERAL:** this creates nothing durable. The registry's `picker` lives and dies with its session entry. The Console's blocked set is replaced on every change and dies with the process. Watcher episodes are pruned to the listing each tick, as in #451. Receipts already have their retention.
- **ARCH-CONSTRAINTS:** a 15s tick reads memory only (`Actors` answers from polled state). Tail IO is one 60-line read per picker episode per slot, with a 2s timeout. The parser is bounded by the 60 tail lines it reads.
- **Layer direction:** pair's wrapper reports a fact over the existing session protocol. Couch never calls into pair, and pair never calls sdlc.

## Tests (each fails if its code is reverted)
- `couchmessage`: `ReadPicker` on the #451 fixtures. The permission tail gives kind, title, detail and options. The selection tail is a question. Numbered prose is no picker. A codex done screen is no picker.
- `couchmessage`: a session-transport round trip. `Picker(true)` reaches `SlotLiveness.Picker`, and a hello-v1 session leaves it nil.
- `wrapcmd`: settle-probe tests. A picker that is up publishes `Picker(true)` once output is quiet, and closing it publishes `Picker(false)`. A real picker screen from `testdata/tty/claude` drives `pickerActive`.
- `wrapcmd`: a delivery to a slot behind a picker goes `held` with detail `blocked by a picker: Do you want to proceed?`, survives past the original deadline, and submits when the picker closes. A picker that stays past `HoldLimit` expires with that reason.
- `couchtty`: `RenderStatusRow` draws a blocked chip red and gives red precedence over bell and fade. The switcher row is red too. The precedence test extends `TestRenderStatusRowFadeYieldsToSelectionNotificationAndPlaceholder`.
- `couchcmd`: a picker slot gets one notice after 1m (none at 50s) with kind, title and options, and none from the generic path at 10m. Muted slots stay silent. The service hands the Console the blocked set when it changes.

## Steps
- [ ] Merge main once #464 lands, then `make build` and run `go test ./cmd/...` green as the baseline.
- [ ] `ReadPicker` in couchmessage (move `dialogTitle`/`stripTailMarkup`), with tests.
- [ ] `Picker` on the frame, client, registry and liveness, with tests.
- [ ] Wrapper settle probe reports the picker, with tests.
- [ ] Receipt hold `blocked by a picker: <title>`, with tests.
- [ ] Console blocked set, red chip and switcher row, with tests.
- [ ] Watcher: picker reason, 1m threshold, 15s tick, specific body, with tests.
- [ ] Atlas: couch messaging / tab-bar pages name the picker report and the red state.
- [ ] Full `make -k test`, then `go test ./...`. Live check on a scratch slot: a permission prompt turns its chip red within seconds and clears when answered, `--message-status` shows `held … blocked by a picker: …`, and the hub notice arrives at about 1m.
- [ ] Ask ops:0 to arrange the operator's smoke test, then close.
