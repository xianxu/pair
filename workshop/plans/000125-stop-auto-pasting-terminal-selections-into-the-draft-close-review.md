# Boundary Review — pair#125 (whole-issue close)

| field | value |
|-------|-------|
| issue | 125 — Stop auto-pasting terminal selections into the draft |
| repo | pair |
| issue file | workshop/issues/000125-stop-auto-pasting-terminal-selections-into-the-draft.md |
| boundary | whole-issue close |
| milestone | — |
| window | e293d51acecd088c82ee5d4c077a608354319a6f..HEAD |
| command | sdlc close --issue 125 |
| reviewer | claude |
| timestamp | 2026-07-28T14:32:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The source-pane gate is correct and I verified it independently at the process level, not just from the diff: driving the real `bin/pair` against a fake zellij, a right-terminal selection logs `in_right_terminal: true` and stages no quote file; a registry-identified split half (`terminal_command: null`, title `[terminal 1]`) is gated only when the registry entry is present, and reverts to handing off when I remove it — so the gate, not some incidental path, is doing the work, and `pair term`'s `RegisterTerminalPane` (`termcmd/run.go:215`) does populate that registry in production. Reusing `workbenchshortcut.RoleForPaneWith` rather than inventing a local classifier is the right structural call. Nothing here blocks shipping. What holds it back from SHIP is a cluster of cheap consistency defects: the diff adds a **third** verbatim copy of `TerminalPaneIDs()` (ARCH-DRY), it swallows the registry error with no log line in a detached process whose only diagnostic channel is that log, the new shell assertion for the clipboard mirror is vacuous against a constant fake (ARCH-MOCK), and — most consequential for the tracker — issue #125's Plan checkboxes and the whole of open issue #126 still assert the *superseded* first approach, so both artifacts now claim things the code does not do.

## 1. Strengths

- **Gate placed in the orchestrator, not the hook** — keeps #100's reap-safety shape intact (`run.go:89` still detaches) while making the policy decision in the one process that can afford the zellij round-trips. The Revisions entry explains why the first shape was wrong; the correction is the right one.
- **Registry overlay reused, not reinvented** (`run.go:115`). The split-half case is exactly the class of bug a role check based on command/title would silently miss, and #123's registry was already the established answer.
- **Both gate paths are unit-pinned with non-vacuous assertions** (`run_test.go:352`, `:373`): setting `f.executable["/h/bin/pair"] = true` makes the flash reachable, so `len(f.subprocess) == 0` actually proves the skip rather than passing because `Executable` returned false.
- **Docs corrected rather than deleted** — `tests/term-pane-shortcuts-test.sh:230` and `zellij/config.kdl:8-10` were narrowed to "clipboard mirror" instead of dropped; `mouse_mode false` would still kill the surviving half, so the assertion stays true. The long-wrong "Alt+n flow" credit in `config.kdl` was fixed in passing.
- **README needs no change** and correctly wasn't touched: every user-facing description is already agent-pane framed (`README.md:56`, `:134`, `:154`). Docs gate satisfied on that axis; atlas was updated in three places.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/internal/clipcmd/runtime.go:29` — third verbatim copy of `TerminalPaneIDs()` (ARCH-DRY).**
Identical bodies now live at `layoutcmd/layoutcmd.go:183`, `termcmd/run.go:1102`, and this new one — same `TerminalPaneRegistry{DataDirFromEnv(), os.Getenv("PAIR_TAG")}` construction, same `LiveIDs` call, differing only cosmetically (`strconv.Itoa(pid)` here vs `fmt.Sprintf("%d", pid)` in the other two). Two copies was borderline; three is the point where the next consumer copies it again.
*Fix:* add `func LiveTerminalPaneIDsFromEnv(alive func(pid int) bool) ([]string, error)` (or take the liveness fn as a package var) to `workbenchshortcut`, and reduce all three `OSRuntime` methods to a one-line delegation.

**(b) `cmd/internal/clipcmd/run.go:112-114` — registry error swallowed with no log line.**
```go
terminalIDs, err := rt.TerminalPaneIDs()
if err != nil {
    terminalIDs = nil // degrade to command/title classification
}
```
This runs in the setsid'd orchestrator, whose stderr is `/dev/null` — the same function says so 35 lines down (`run.go:149-151`, a #100 close-review fix). The failure is invisible *and* behavior-changing: a permission/IO error on `terminal-panes-<tag>` silently restores auto-paste for split halves, which is the exact symptom the issue exists to remove, with nothing in `clipboard-debug.log` to explain it.
*Fix:* `rt.Log("terminal registry read failed: " + err.Error())` inside the branch. (`LiveIDs` already returns `(nil, nil)` for a missing file, so this only fires on real errors — no log noise.)

**(c) `tests/copy-on-select-test.sh:100` — the clipboard-mirror assertion cannot fail (ARCH-MOCK).**
The fake is a constant: `printf '#!/bin/sh\nprintf %%s "selected text"\n' > "$fakebin/pbpaste"` (`:42`), and `pbcopy` is `cat >/dev/null` (`:41`). So `[ "$(pbpaste)" = "selected text" ]` is a tautology — it would pass with `ClipboardCopy` deleted entirely. The Spec's second bullet ("selecting text STILL copies to the OS clipboard, **by the same mechanism as today**") is therefore unpinned at the process level; only the unit fake (`run_test.go:135`) covers it.
*Fix (2 lines, keeps (a)/(b) working since the hook copies before it spawns):*
```sh
printf '#!/bin/sh\ncat > "%s/clipboard"\n' "$tmp" > "$fakebin/pbcopy"
printf '#!/bin/sh\ncat "%s/clipboard" 2>/dev/null\n' "$tmp" > "$fakebin/pbpaste"
```
and `rm -f "$tmp/clipboard"` in `run()` so each case proves its own mirror.

**(d) `workshop/issues/000125-…md:118-126` and `:139-151` — checked Plan items describe the superseded approach.**
Item 1 is `[x]` but reads "delete the `SpawnDetached(...)` at `clipcmd/run.go:81`" and "rename `TestCopyOnSelectHookMirrorsThenDetaches` → `TestCopyOnSelectHookMirrorsOnly` and invert its `spawned` assertion". The shipped code keeps the spawn (`run.go:89`) and keeps the original test name (`run_test.go:129`). Item 3 is `[x]` but commits every annotated site to the string "unwired since #125 — no production caller" — that string appears nowhere in the tree (correctly, since nothing is unwired). The Spec bullets *were* marked `[SUPERSEDED]` in commit `27947c8`; the Plan section was not. As it stands the Plan claims delivered work that the code contradicts.

**(e) `workshop/issues/000126-…md:15-28` and `:47-48` — an open, `deps: [125]` issue built on the superseded premise.**
It states "#125 removes the AUTOMATIC quote-paste (selecting text anywhere outside the draft…)", "After #125 there is no way to invoke that at all", "the machinery #125 deliberately left unwired", and "the 'unwired since #125' atlas notes can be retired". All four are false against the shipped code: agent-pane selections still drive the full chain, nothing is unwired, and no such atlas notes exist. #125's own Revisions section already acknowledges the reframing ("#126 remains valuable as a DELIBERATE trigger, not as the only one") but the #126 file was never updated — it hasn't been touched since commit `936f9a6`. Whoever picks up #126 will start from a false problem statement.
*Fix:* rewrite #126's Problem to "the formatting is only reachable via an agent-pane mouse selection; give it a deliberate keyboard trigger", and drop the third Done-when bullet.

## 4. Minor findings

- `cmd/internal/clipcmd/run.go:95-96` and `cmd/internal/clipcmd/clipcmd.go:9-10` — both doc comments still say the chain hands off "unless the selection was made in the nvim draft pane", describing one of the two gates. The Plan's own generated grep (`copy-on-select` over `cmd/`) would have hit both; the sweep missed them.
- `atlas/architecture.md:417` states an **allowlist** ("hands off only for selections made in the AGENT pane") while the code implements a **denylist** (skip nvim-ish commands and right terminals; hand off otherwise). Equivalent today, but a future non-nvim pane type would hand off and contradict the atlas. Phrase it as the code's rule.
- `run.go:111` — `TerminalPaneIDs()` (file read + `kill -0` per entry) runs *before* the `inNvim` early return at `:123`, so every draft-pane selection pays for a lookup it discards. Move it below the `inNvim` check.
- `run_test.go:96` / `:106` — fixtures use `"sh -c exec pair-wrap claude"`, but the real layout is `exec pair wrap …` (`zellij/layouts/main-2.kdl:45`). `RoleForPane` matches `"pair wrap"`, so the fixture agent pane classifies as `PaneRoleOther` where production gives `PaneRoleLeftAgent`. Same outcome today; the fixture just no longer models the real pane.
- `runtime.go:30-32` reads `DataDirFromEnv()`/`PAIR_TAG` inside `OSRuntime`, whereas `CopyOnSelectOptions` (`run.go:57-61`) documents the package convention as "resolved at the CLI boundary so run.go stays pure". Consistent with `termcmd`/`layoutcmd`, so acceptable — but it's a second env-resolution site in a package that had one.

## 5. Test coverage notes

- The kind of bug this diff could ship — "the gate keys on something zellij doesn't report for split halves" — is covered at both layers: unit (`run_test.go:373`) and, per my own process-level run, in production shape. Good.
- **Gap:** `fakeRuntime.TerminalPaneIDs` (`run_test.go:59`) hardcodes a `nil` error, so the degrade branch at `run.go:112-114` is entirely uncovered. Add a `terminalPaneIDsErr` field plus one case asserting that a registry failure leaves the *command/title*-identified right terminal still gated (i.e. degradation loses only split halves).
- **Gap:** finding (c) — the mirror half of the Spec is asserted only against a constant fake at the process level.
- The shell test covers the right terminal by command/title only; the registry path is unit-only. Acceptable (the registry file is trivial to write, and I confirmed it works end-to-end), but a fourth shell case writing `$PAIR_DATA_DIR/terminal-panes-t` would close the process-level loop cheaply.

## 6. Architectural notes

- **ARCH-DRY — flag.** Finding (a): third copy of `TerminalPaneIDs()`. Secondary observation (no change requested): `RunCopyOnSelectOrchestrate` now runs two overlapping classifiers, `isNvimCommand` (`(?i)nvim|draft`) and `RoleForPaneWith`. They are *not* safely mergeable — `RoleForPane`'s `PaneRoleLeftDraft` requires `/nvim/init.lua`, so collapsing to `switch role { case LeftDraft, RightTerminal: return 0 }` would narrow the skip and start auto-pasting from the review pane and any other plain-nvim pane. Worth a one-line comment recording that `isNvimCommand` is deliberately the broader of the two, or the next reader will "simplify" it into a regression.
- **ARCH-PURE — pass.** The decision is pure (`RoleForPaneWith` over parsed pane data), the new IO sits behind the `Runtime` seam, and the orchestrator stays a thin caller. Tests run the decision with no IO.
- **ARCH-PURPOSE — pass.** Shadow-sweep of the consumers of "which panes hand off": enforcement lives in exactly one place (`run.go:132`); `atlas/architecture.md` (×3), `zellij/config.kdl`, `nvim/init.lua` (×3), `Makefile.local`, and both shell tests are documentation *of* that single point, all updated. No hand-maintained restatement re-implements the rule. The corrected scope delivers the issue's actual purpose (agent-pane retained, terminal excluded) rather than the easy global kill — that was the first pass's failure and it was properly caught and fixed.
- **ARCH-MOCK — flag.** The stateful-fake property holds for zellij (fake emits real captured JSON) and for the registry (a real file on a real path). It does **not** hold for the clipboard: `pbcopy` discards and `pbpaste` is a constant, so production flow and test flow don't share the boundary for the one behavior the Spec promises to preserve. Finding (c).

## 7. Plan revision recommendations

Add to `workshop/issues/000125-…md` `## Revisions`:

- **2026-07-28 (plan reconciliation, post-scope-correction):** Plan items 1 and 3 still describe the superseded first approach and must not stay checked as written. Item 1 → "Keep `RunCopyOnSelect` detaching the orchestrator (#100's shape); add the source-pane gate in `RunCopyOnSelectOrchestrate`, keyed on `workbenchshortcut.RoleForPaneWith` so registry-identified split halves are covered. `TestCopyOnSelectHookMirrorsThenDetaches` keeps its name and its spawn assertion." Item 3 → "Correct every doc site from 'auto-paste on any non-draft selection' to 'agent-pane only'; **no** site is annotated 'unwired since #125' — nothing was unwired."
- **2026-07-28 (dependency reframing):** #126 is re-scoped from "the only surviving trigger for `PairPasteQuote`" to "a deliberate keyboard trigger alongside the retained agent-pane selection path"; its Problem statement and third Done-when bullet must be rewritten before it is claimed. *(Also apply the edit in `000126-…md` itself — the note in #125's Revisions is not enough, since #126 is read on its own.)*
