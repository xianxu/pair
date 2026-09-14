# Couch notification keyboard mode implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute this coupled output/framing change in the session that investigated it; the SDLC boundary owns the fresh-context code review.

**Goal:** Make physical Ctrl+Return reach Couch's notification jump even after the host loses a child's earlier keyboard setup.

**Architecture:** Couch owns a minimum Kitty keyboard-disambiguation requirement while hosting Zellij. Add bit 1 without overwriting other flags, at existing host-output framing boundaries. Keep notification selection and ordinary Return unchanged; model the terminal's effective keyboard state in tests.

**Tech Stack:** Go, existing Host/Child doubles, ANSI framing, Kitty keyboard protocol, Zellij.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `EnableKeyboardDisambiguation` | `cmd/internal/hostty/control.go` | new |
| `knownSequences` | `cmd/internal/couchtty/keys.go` | modified |

`EnableKeyboardDisambiguation` names `\x1b[=1;2u`, an additive, idempotent flag request. A single constant prevents copied protocol literals. It creates no stack entry. `knownSequences` adds explicit press/repeat forms `\x1b[13;5:1u` and `\x1b[13;5:2u` alongside the existing implicit press; release `:3` must never jump. This handles the relevant event forms when a child already requested event reporting, without creating a general keyboard parser.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console` host keyboard requirement | `cmd/internal/couchtty/console.go`, `cmd/internal/couchtty/keyboard.go` | modified | Existing `hostty.Host` output and `ptychild.Screen` framing |
| `keyboardHost` | `cmd/internal/couchtty/keyboard_host_test.go` | new | Stateful terminal double behind the existing Host seam |

The Console owns the policy; `ptychild.Screen.MidSequence` owns framing. Do not add keyboard state to Screen or teach the shared repaint composer a Couch-specific policy. The test host embeds FakeHost, consumes all actual writes using the existing ANSI framing utilities, tracks bounded primary/alternate stacks and set/add/remove operations, and converts a physical Ctrl+Return to CR or CSI-u based on effective flags. Its independent state model must not infer success by searching for our new constant. x/vt's existing host cannot encode Kitty keys, so extend the fake at the Host seam rather than use its unsupported key encoder.

## Decisions and alternatives

1. **Chosen: additive Couch-owned flag.** It directly maintains the condition Couch's own shortcut requires, with constant work at output boundaries. Child-requested event/alternate flags remain set.
2. **Rejected: restore every child's full keyboard stack.** This requires virtualizing queries, push/pop, buffers and input translation; it is substantially wider than the notification shortcut and still does not state Couch's minimum requirement.
3. **Rejected: recognize bare CR as Ctrl+Return or change the shortcut.** CR is ordinary Return in the agent. Consuming it would submit/jump unpredictably and would not deliver the requested key.

## Output/lifetime contract

| Event | Effect |
|-------|--------|
| Successful `Run` raw ownership, before stdin pump | Emit additive enable once. |
| Complete active-child output, including flags reset/pop, RIS and buffer switches | Append additive enable to the same host write after the raw bytes. |
| Partial/oversized skipped control string | Forward bytes; defer enable until `MidSequence` is false. No timer or unbounded new buffer. |
| Takeover to actor or panel | Compose and scan the takeover as today, then append enable if framing is complete. Same policy helper as live output. |
| Inactive ordinary child output | No host write and no mode change. The next takeover enforces the requirement. |
| Child holds cursor save but stream is complete | Enable is allowed: it touches no cursor; do not defer on `SafeToPaint`. |
| Shutdown/failed startup | Existing shell-safe reset remains authoritative; emit no enable after cleanup starts. |

The Run goroutine remains the sole host writer. A helper in `keyboard.go` accepts already-owned output bytes and a complete-frame boolean, returning the bytes with the named control appended only at a complete boundary (or an equally small writer helper with that contract). Read framing under the existing mutex; write outside it. Takeover must scan before choosing its suffix; feeding the keyboard suffix back is unnecessary because it has no effects on existing Screen state.

This does not repair modes changed by an unrelated process writing the same TTY without Couch's knowledge. Couch's next output/takeover reasserts the requirement. Unsupported terminal implementations may ignore the request and retain the existing two-gesture fallback.

## Constraints and architecture checks

- **ARCH-DRY / ARCH-PURE:** One named control and one policy helper shared by startup/live/takeover. Reuse the incremental ANSI framing scanner; no second production escape parser.
- **ARCH-PURPOSE:** Test from physical-key encoding through actual Console dispatch and notification acknowledgement, not pre-encoded key injection alone. Sweep startup, both takeover targets, active output and teardown.
- **ARCH-MOCK:** Stateful Host double models the used protocol operations. Compare the request/encoding and reset behavior on a real supporting terminal in the live smoke below.
- **ARCH-CONSTRAINTS:** Interactive output path: O(n) copy of the current output chunk plus an eight-byte suffix; at most one suffix per complete write, no extra goroutines, timers, subprocesses or network calls. No new retained production buffer. Verify repeated switches do not increase keyboard stack depth.
- **ARCH-SECURE:** Arbitrary PTY bytes must remain framed across every split and oversized strings. No capture of user keystrokes by default; tests use fake children and temporary stores only.
- **ARCH-ORDER:** Run serializes outputs and input. Tests explicitly sequence reset→keypress, partial→keypress→completion, background reset→switch, alternate-buffer changes and shutdown. Assertions cannot precede the reset they repair or follow teardown.
- **ARCH-FUNERAL:** No durable runtime artifacts or background workers added. Test doubles die with the test; optional live capture lives in a named temporary directory and records only deliberate probe keys.

## Chunk 1: Keyboard requirement and regression

### Task 1: Reproduce from terminal state

**Files:** Create `cmd/internal/couchtty/keyboard_host_test.go` and `cmd/internal/couchtty/console_keyboard_test.go`; reuse `console_newest_page_test.go` fixture patterns.

- [x] Build a Host double with independent main/alternate flag stacks, bounded to 16 entries, and set/add/remove/push/pop/RIS interpretation using the existing ANSI tokenizer. Write unit cases from the official protocol, including fragmented commands and overflow/underflow.
- [x] Add a running Console fixture with two real fake children and a pending notification. Feed a live `\x1b[=0u`, press physical Ctrl+Return via the terminal double, and assert the jump and acknowledgement. Observe the failure on main: CR reaches the original child.
- [x] Add startup and takeover cases: raw replay ending in pop; incoming child's enable aged out of the bounded ring; repeated actor/panel switching; primary/alternate mode independence; inactive reset isolation. Check the fake's emitted key, not only the final selected actor.
- [x] Run `go test ./cmd/internal/couchtty -run 'Test.*Keyboard' -count=1 -v`. Record the expected red result in the issue Log.

### Task 2: Enforce the requirement

**Files:** Modify `cmd/internal/hostty/control.go`, `cmd/internal/couchtty/console.go`, `cmd/internal/couchtty/keys.go`; create `cmd/internal/couchtty/keyboard.go`; extend `cmd/internal/couchtty/keys_test.go`.

- [x] Add `EnableKeyboardDisambiguation = "\x1b[=1;2u"` with a comment explaining additive flags and no stack allocation.
- [x] Add the small shared suffix/writer helper. Call at startup after successful MakeRaw, from live `writeChild` after feeding its bytes, and from takeover after feeding the composed repaint. Use `!hostScan.MidSequence()`, never the cursor-save paint gate.
- [x] Add the two exact Ctrl+Return event encodings to the existing sequence table. Test every read split, explicit press and repeat, plain Return passthrough, release not jumping, and bracketed-paste passthrough before adding the rows.
- [x] Add split CSI/OSC/DCS/APC, overlong control-string, cursor-save, non-disambiguation flags preservation and teardown cases. Ensure the suffix is not emitted after shell reset and repeated enforcement leaves stack depth unchanged.
- [x] Run `go test ./cmd/internal/couchtty ./cmd/internal/hostty ./cmd/internal/ptychild -count=1` and `go test -race ./cmd/internal/couchtty -run 'Test.*(Keyboard|NewestPage|Interceptor)' -count=1`.
- [x] Review actual changed files and keep tests targeted; retain existing host mode/notification tests. Commit with #251 and a Co-Authored-By trailer.

### Task 3: Verify in a real terminal and publish

**Files:** Update `atlas/couch.md`, issue Log and this plan's checkboxes.

- [x] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/hostty ./cmd/internal/ptychild -count=1` and `git diff --check`; broaden only for a new failure or affected consumer.
- [x] Build the Couch executable using the repository's existing build target. Determine a controlled reload path before replacing the running instance: this session is hosted and #250's stale Pair record remains unresolved. Do not kill or archive that thread just to test #251.
- [x] On the operator's supporting terminal, verify normal Return, physical Ctrl+Return to a yellow thread, repeated jumps, recent-thread return, switcher return and thread switches. To compare the fake with the real terminal, use a disposable terminal/probe to set/add/pop flags and query them; enter only deliberate test keys and restore modes. If live terminal access requires the operator, present exact smoke steps and leave live verification unchecked until confirmed.
- [x] Document the Couch-owned flag, explicit press/repeat handling, unsupported-host fallback and shell-safe teardown in the existing atlas keyboard section.
- [ ] Close with `sdlc close --issue 251 --verified '<actual evidence>'`, address the binary's fresh-context review findings, then `sdlc pr` and `sdlc merge`. The single-pass task has no separate milestone boundary.

## References

- Issue: `workshop/issues/000251-couch-notification-keyboard-mode.md`.
- Protocol: https://sw.kovidgoyal.net/kitty/keyboard-protocol/ (progressive enhancement, flags, stack lifetime).
- Current regression: `cmd/internal/couchtty/console_newest_page_test.go`.
- Existing framing authority: `cmd/internal/ptychild/screen.go` and `cmd/internal/ansi/`.

## Revisions

### 2026-09-14 — Clarify bounded enforcement and stack assertions

The control is seven bytes (the constraints paragraph's eight-byte count is an
off-by-one). Test that repeated Couch additive assertions do not change stack
depth; replaying a child's own push can still change it. Do not claim full stack
virtualization or require unchanged total depth across arbitrary child replays.
A complete replay ending in pop must still leave disambiguation enabled.

While an active child holds an incomplete control string, deferral preserves
framing; test the lack of injected bytes and successful re-enforcement on its
completion. Do not promise that a key arriving during an already-disabled,
incomplete string can be retroactively disambiguated. This is the safe framing
boundary of the fix, not a reason to consume plain CR.

### 2026-09-14 — Fresh review: serialize wire state and finish both-buffer cleanup

**Reason:** Review found the earlier sole-Run-writer statement false:
`operationQueue` also reaches takeover. Existing teardown resets the alternate
buffer before returning to a main buffer whose keyboard mode can remain enabled.
This revision supersedes that ordering/lifetime portion of the initial plan.

**Delta:** Add a narrow Console terminal-output mutex and released state,
encapsulated with the keyboard policy in `keyboard.go`. This is a correctness
prerequisite for this issue, not completion of #224's typed-door architecture.
The mutex serializes the entire scanner-decision-plus-host-write transaction,
not only the eventual syscall. Its lock order is terminal mutex then `c.mu`;
no caller may acquire the terminal mutex while already holding `c.mu`.
Enumerate and audit all seven current direct output sites:

- `writeChild`: terminal lock → read/update Screen under `c.mu` → append suffix
  if complete → write while still holding terminal lock → unlock.
- `takeOverScreen`: terminal lock → reset/feed composed takeover under `c.mu`
  → append suffix → write → unlock, then ask the child to repaint. Never hold
  the terminal lock while requesting repaint or invoking a child callback.
- `writeOwn`: same terminal lock across paint eligibility and actual write.
- `release`: terminal lock → mark released → write cleanup → unlock. Later
  output paths observe released and perform no writes; cleanup owns the end.
- Three `console_menu.go` hide/move/show writes: route through a small raw
  control method that acquires the same terminal lock and respects released.
  These complete cursor-only controls do not establish keyboard policy.

Add `cmd/internal/couchtty/console_menu.go` to Task 2's modified files. Startup
uses that serialized control method for the initial keyboard request.
All methods must recheck released only under this output mutex. Repeated
release is idempotent. There is no new goroutine or unbounded queue; a terminal
write retains existing blocking IO behavior. Keep lock scopes short and avoid
holding `c.mu` across IO. Audit every caller's lock order explicitly.

Cleanup resets the current buffer as before, leaves alternate screen, then
resets interactive modes again on the now-active main buffer. This clears the
keyboard flag regardless of which buffer startup or the child enabled. Check
normal stop, input EOF, signal, and startup failure paths with the terminal
double. Do not re-enable after cleanup has acquired output ownership.

**New regression steps before implementation:** Use a blocking Host double to
hold a live output write while a takeover or release is requested. Explicitly
release the write and observe ordering; assert scanner transitions match wire
order and no enable can follow final cleanup. Exercise reverse order too
(release wins first → later takeover/child/menu writes are dropped). Use
channels/barriers, not timing-only sleeps. Add startup on main → child enters
alternate → stop and assert physical Ctrl+Return encodes as legacy CR on the
restored main buffer.

**Validation delta:** Before publishing run the repository-required
`env -u PAIR_SESSION_ID -u PAIR_TAG make test`, since new production files can
also affect cross-package inventory contracts. Retain focused red/green and
race checks for the keyboard/output ordering tests. Log failures honestly and
fix newly affected consumers rather than declaring success from focused tests.


### 2026-09-14 — Implementation and verification progress

The physical-key and both-buffer regressions were observed failing on baseline,
then pass with the production fix. Core policy and output ownership are
implemented. Affected package tests pass; full/race/live checks are pending.
The shared helper is `keyboardDisambiguated`; the test model methods are
`keyboardModel.feed`/`command` behind `keyboardHost.Write`/`ctrlReturn`.

Source correction: pumpStdin returns on EOF without ending Console.Run.
Preserve that behavior; the EOF scenario verifies shell restoration when Console
subsequently stops, rather than adding an unrelated exit policy change.


### 2026-09-14 — Full-suite guard distinguishes keyboard control from painting

Full `make test` correctly found the two new MidSequence reads in the static
paint-gate guard. Keyboard controls have no cursor effects and must not wait for
cursor-save release, as the approved plan requires. Add an explicit
`keyboard-control-boundary: no cursor effects` annotation to these reads and
teach `tests/paint-gate-consumers-test.sh` that narrow non-paint exception;
unannotated partial paint-gate reads remain rejected. This is a newly affected
consumer of the documented framing contract, not a weakening to all MidSequence
calls. The stateful cursor-save regression verifies the distinct behavior.


### 2026-09-14 — Live acceptance scope at authorized shipment

Operator confirmed rebuilt Ctrl+Return and authorized shipment. Task 3's live
checkbox records that acceptance; the expanded live protocol-query exercise
was not performed. Remaining transitions are validated by the stateful terminal
tests and passing full suite. The close/publish row is pending gate execution,
not unfinished product behavior. Couch owns the terminal requirements for its
controls and re-establishes them across output boundaries (ARCH-ORDER).
