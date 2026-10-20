# couch --keys special keys and a named interrupt: Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the hub drive the special keys an agent UI needs (Tab, Shift+Tab, Left/Right, Backspace, Delete, Home/End, Page Up/Down, F1–F12, Ctrl and Alt chords) through `couch --keys`, and interrupt a working turn only through a separate, deliberate `couch --interrupt`.

**Architecture:** Each caller key is one row of `agentdrive.Steps`, the table #716 introduced. The row carries the key's bytes, so the parser, `InputBytes`, `couch --keys --help` and the skill check all read it (ARCH-DRY). Family rows (`ctrl-LETTER`, `alt-LETTER`, `f1`–`f12`) are generated into the same table, minus the excluded chords below. The interrupt is a third request shape, beside keys and model. It passes through the one admission rule (`Request.Admits`), with its own proof: a turn is active. It writes exactly one Ctrl+C.

**Tech Stack:** Go; `cmd/internal/agentdrive` (pure), `cmd/internal/wrapcmd` (wrapper writer), `cmd/internal/couchcmd` (CLI, service), `cmd/internal/couchmessage` (wire).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `StepDoc` (adds `Bytes`) and `Steps` (adds named keys and generated families) | `cmd/internal/agentdrive/keys.go` | modified |
| `KeyStep.InputBytes` (reads the table) | `cmd/internal/agentdrive/keys.go` | modified |
| `Request.Interrupt` and `Request.InterruptOnly` | `cmd/internal/agentdrive/recipe.go` | new |
| `Admission.Interruptible` and `Request.Admits` (adds the interrupt clause) | `cmd/internal/agentdrive/recipe.go` | modified |
| Reducer: interrupt start, its turn exemption, its outcome | `cmd/internal/agentdrive/model.go` | modified |

- **Steps**: one row per caller key, as syntax, meaning and bytes. Families expand to concrete rows when the package initializes: `ctrl-a`…`ctrl-z` except the excluded chords below; `alt-a`…`alt-z`; `f1`…`f12`. Help prints a family once (its `Group` syntax, such as `ctrl-LETTER`). The parser accepts exactly the rows, so an unknown name refuses before anything is sent, listing `StepSyntax()`.
  - **Bytes:** classic xterm encodings, the family Pair's drive already uses live (`\r`, `\x1b\r`, `\x1b[A`, `\x1b`). Tab `\t`, Shift+Tab `\x1b[Z`, Left `\x1b[D`, Right `\x1b[C`, Backspace `\x7f`, Delete `\x1b[3~`, Home `\x1b[H`, End `\x1b[F`, Page Up `\x1b[5~`, Page Down `\x1b[6~`, F1–F4 `\x1bOP`…`\x1bOS`, F5–F12 `\x1b[15~ 17~ 18~ 19~ 20~ 21~ 23~ 24~`, `ctrl-x` is the letter AND 0x1f, `alt-x` is ESC then the letter. Claude turns on the kitty keyboard protocol, but it accepts these encodings, which the existing drive keys show. The live check covers Shift+Tab and one function key.
  - **Refused chords** (approved by the hub): `ctrl-c` (interrupt: only `couch --interrupt`), `ctrl-d` (EOF; a second one exits the agent), and `ctrl-z` (suspends the agent process under the wrapper). Each refuses with its own reason before anything is sent.
  - **Excluded aliases and flow control** (plan review PQ): `ctrl-i` is `tab`, `ctrl-m` is `enter`, and `ctrl-u` is `clear-line`. As a plain chord, `ctrl-u` would send clear-line's bytes without its recovery-only rule, and two names for one byte would split the admission rules. `ctrl-s` and `ctrl-q` are XON/XOFF flow control, which can freeze the agent's output on a PTY. Each refuses with its reason and points at the named key where there is one. One table row, one name, one rule.
  - **Recipe directions:** `KeyLeft` and `KeyRight` become the caller names `left` and `right`. The internal-only rule existed because the caller vocabulary had no arrows, and that reason is gone. `TestRecipeDirectionKeysAreInternal` becomes a bytes test.
- **Request.Interrupt**: a request carries exactly one of keys, model or interrupt. It is never a key step, so no sequence can carry an interrupt by accident. Wire: the new field is additive (ARCH-EVOLVE). An old wrapper decodes a request with none of the three, and its `Validate` refuses.
- **Admission.Interruptible**: the wrapper's proof that a turn is active, the frame is complete, no picker is open, and the wrapper owns input. In `Admits` the interrupt is checked first and exclusively: `if r.InterruptOnly() { return a.Interruptible }`. Otherwise the existing clauses apply, with `Idle` first. So an interrupt is never admitted on an idle composer (PQ-1), and keys never by `Interruptible`. See "One proof per request shape" below for the start gate, the turn gate and the pre-write recheck.

### One proof per request shape (plan review PQ-1, PQ-4)

Each request shape defines its admitting proof once, in `agentdrive`, as `Request.shapeProof(Admission)`. Every gate derives from it, and the pre-write recheck covers only the writes that the proof was taken for.

| Shape | Start admission (`Admits`) | Proof stands during a turn | Pre-write recheck (`WriteAdmits`) |
|---|---|---|---|
| Generic keys | `Idle` | no: stops on a turn | none after start, as today |
| Recovery only (esc, clear-line) | `Idle` or `Recoverable` | no | every write: `Recoverable` (as today) |
| Lone esc on a panel (#691) | `Idle`, `Recoverable` or `Dismissable` | yes | its write: `Recoverable` or `Dismissable` (as today) |
| Model | `Idle` | no | each effect's picker proof (`MatchesScreen`, as today) |
| Interrupt | `Interruptible` only, checked first and exclusively | yes | its one Ctrl+C: `Interruptible` |

- `Admits(a)`: `if r.InterruptOnly() { return r.shapeProof(a) }`, otherwise `a.Idle || r.shapeProof(a)`.
- `WriteAdmits(a)`: `shapeProof(a)` for recovery and interrupt shapes, otherwise true (generic keys and model keep today's guards). The wrapper's recheck calls `WriteAdmits`, replacing today's `RecoveryOnly`-only condition. Generic key drives are unchanged.
- The turn gate: the reducer's top-level stop and the wrapper's turn-active branch exempt the shapes whose proof stands during a turn. `DriveState.TurnTolerant()` (the dismissal, or an interrupt) replaces today's `Dismissal()` check.
- What `Interruptible` keeps from the wrapper's start gate: the same `common` conditions (not exited, no image, the observed sequence, no buffered operator input, no reply writes in flight, no reservation, the current message terminal; a hub-sent message is Submitted, which is terminal). For the input lease: Idle, or a residual wait (AwaitingEmpty, Alerted), as a recovery drive may take. An active peer or drive owner refuses, and so does NeedsOperator. It drops `settled` and the empty composer, because a working turn has neither. The screen part: `turnActive && Complete && !picker`.
- The Ctrl+C is an internal key kind (`interrupt`, bytes `\x03`) that only the interrupt shape emits. It is not a `Steps` row, so `Validate` refuses it inside any key sequence.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `agentDriveScreen` / `agentDriveObservation` (add `Interruptible`) | `cmd/internal/wrapcmd/agent_drive.go` | modified | the agent PTY and screen |
| `DriveObservation.Interruptible` | `cmd/internal/couchmessage/drive_endpoint.go` | modified | wrapper RPC |
| `couch --interrupt repo:N --confirm [--json]` | `cmd/internal/couchcmd/agent_drive_cli.go`, `run.go`, `cost.go` | new | CLI |
| Service admission (an interrupt skips the settled check, as recovery does) | `cmd/internal/couchcmd/agent_drive_service.go` | modified | supervisor |

The fakes are the existing drive service fake and harness session fake, with no new double. The live check is the conformance step.

## State design

Actions: `couch --interrupt SLOT --confirm`, `couch --peek`, the operator's own keys.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Turn active, wrapper owns input | interrupt admitted, one Ctrl+C written | none | the turn's unfinished work (the purpose) |
| No turn active | refused: "no turn to interrupt" | none | none |
| Turn flag set without work (a bare submission, or an open Claude panel whose progress report keeps the flag) | refused: no reported work, or no composer on screen (close review BR-4) | `--keys esc` for a panel | none |
| Turn ends between admission and the write | the pre-write recheck cancels; nothing written | none | none |
| Ctrl+C written, turn ends | done | none | as above |
| Ctrl+C written, turn still active at the deadline | unconfirmed with tail; never retried | peek, then decide on a new interrupt | none beyond the turn |
| A second interrupt after a successful one | refused (no turn), so two Ctrl+C presses cannot exit the agent | none | none |
| Picker or permission prompt open | refused | `--keys esc` or the operator | none |
| `ctrl-c`, `ctrl-d` or `ctrl-z` as a key step | refused before sending, with the reason | `couch --interrupt` for an interrupt | none |

Non-goals: special characters inside `text:` steps (quotes, `$`, backticks, newlines) and a file form for `--keys`. `text:` already takes any printable UTF-8. Shell quoting is the caller's concern, and the skill already says to write a file for a message with shell syntax. A `--keys` file form would be its own issue if the hub wants one.

Not covered: a cooldown across an operator's own Ctrl+C. The wrapper does not see that press as a drive, and each drive writes at most one Ctrl+C.

## Milestones

- [x] **M1 — named keys in the one table**
  - [x] Write the failing tests: every row's bytes (the table above); every generated family member parses; unknown names refuse with the list and send nothing; `ctrl-c`, `ctrl-d` and `ctrl-z` refuse with their reasons; help lists every row or its group; the skill sentence matches the group syntaxes (#716 test, extended).
  - [x] `StepDoc.Bytes`, `Group`; the generated families; `InputBytes` reads the table; recipe directions become caller names.
  - [x] `keysUsage` groups families; the skill steps sentence and quick table gain the new names; README and atlas updated.
  - [x] Wrapper test: a sequence of new keys written to the harness PTY emits the table bytes, each step separate.
  - [ ] `sdlc milestone-close --milestone M1`.
- [x] **M2 — `couch --interrupt`**
  - [x] Write the failing tests:
    - Pure: `Request.Validate` (exactly one shape), `InterruptOnly`, the `Admits` matrix: an interrupt only on `Interruptible`, never on `Idle`, `Recoverable` or `Dismissable`, even with all three set; keys never on `Interruptible`.
    - Reducer: start on an active turn; the turn exemption; done when the turn ends; unconfirmed at the deadline; refusal with no turn.
    - Wrapper: one `\x03` on an active turn; nothing when the turn ended before the write (the widened pre-write recheck); refusal with a picker; refusal on an idle composer.
    - Service: interrupt skips the settled check; refusal names the reason.
    - CLI: `--confirm` required; `--keys ctrl-c` points at `--interrupt`.
  - [x] Implement the request shape, `Interruptible` in the wrapper and observation, the reducer branch, the service and the CLI verb with its cost and help.
  - [x] Docs: help, skill quick-table row, README, atlas.
  - [ ] Full suite; `sdlc close`.
- [ ] Live check (hub, after landing, throwaway slot only for the interrupt): `--keys SLOT shift-tab` cycles Claude's mode; a function key reaches Codex. For the interrupt, an idle slot runs Bash `sleep 60`, then `couch --interrupt SLOT --confirm`; expect done and the sleep interrupted, and a second `--interrupt` refused.

ARCH notes: ARCH-DRY (one table for bytes, parser, help, skill); ARCH-PURE (bytes, admission and reducer stay pure); ARCH-SECURE (the interrupt is its own request shape, admitted only by a positive turn proof); ARCH-ORDER (the pre-write recheck closes the race where the turn ends first); ARCH-EVOLVE (additive wire field, fails closed on old wrappers); ARCH-CONSTRAINTS (still 64 steps; the generated table is about 80 rows, scanned linearly per step, fine at 64 steps).
