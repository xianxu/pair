---
id: 000125
status: done
deps: []
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 0.45
started: 2026-07-28T12:27:14-07:00
actual_hours: 1.43
---

# Stop auto-pasting terminal selections into the draft

## Problem

Selecting text anywhere outside the draft pane currently flashes the source
pane, steals focus to the draft, and inserts the selection as a `> `-prefixed
reflowed quote. The user reports this as distracting: a selection made only to
copy a path or a line hijacks the draft.

The pipeline is zellij's `copy_command "pair clip copy-on-select"` →
`RunCopyOnSelect` (mirror to OS clipboard, spawn detached orchestrator) →
`RunCopyOnSelectOrchestrate` (flash + hand off) → `RunClipboardToPane` (stage
at `quote-<tag>`, focus draft, send Ctrl-_) → nvim `PairPasteQuote`.

## Spec

- A selection in the RIGHT TERMINAL no longer flashes, moves focus, or inserts
  into the draft. A selection in the AGENT pane still does — that is the
  feature's purpose and is explicitly retained (see ## Revisions).
- Selecting text STILL copies to the OS clipboard, **by the same mechanism as
  today** (user decision 2026-07-28: disable the auto-paste half only, not
  `copy_on_select` itself).
- **[SUPERSEDED by the 2026-07-28 scope correction — the hook detaches again;
  the gate moved into the orchestrator. The `copy_command` reasoning below still
  holds and is why it was never dropped.]**
  **Approach: keep `copy_command`, make the hook mirror-only.** `RunCopyOnSelect`
  keeps its `ClipboardCopy` and drops the `SpawnDetached(... --orchestrate)`.
  The tempting one-liner — deleting `copy_command` from the config — would hand
  the clipboard write to zellij's native path, which uses OSC 52 to the host
  terminal: a DIFFERENT mechanism with different failure modes (the terminal
  must honor OSC 52; nested tmux/ssh often does not), and one that sits outside
  any seam our fakes can drive, making it manual-verification-only. Mirror-only
  keeps the write on our tested `clipcmd.Runtime` seam (pbcopy/wl-copy/xclip),
  changes exactly one behavior, and stays unit-testable.
- **[SUPERSEDED — nothing is unwired: the agent-pane path still drives the whole
  chain. #126 remains valuable as a DELIBERATE trigger, not as the only one.]**
  The rest of the `pair clip` chain and `PairPasteQuote` stay in place,
  unwired: `RunCopyOnSelectOrchestrate`, `RunClipboardToPane`, `RunFlashPane`,
  the `quote-<tag>` staging file, and the insert-mode `<C-_>` map. They are the
  machinery **#126** (deliberate quote-paste keybind) will bind; deleting them
  would make that a rewrite rather than a binding (`Simplicity First` — unwire,
  don't demolish). Every such site is annotated "unwired since #125 — no
  production caller; see #126" so a reader can tell dormant-by-design from
  orphaned.
- **[SUPERSEDED — the `> `-reflow capability is retained for agent-pane
  selections; only the right terminal lost it.]**
  Recorded consequence: there is currently NO manual trigger for
  `PairPasteQuote`. Its only caller is the copy-on-select hand-off, and the
  insert-mode `<C-_>` keymap exists solely as that hand-off's delivery gate
  (Alt+n is `PairConfirmRestart`, not quote-paste). So this removes the
  `> `-reflow capability until #126 lands — accepted, since the automatic
  version is exactly what is being removed.

## Done when

- A live selection in the terminal populates the clipboard (`pbpaste` returns
  the selected text) and does nothing else — no pane flash, no focus change,
  no draft insert.
- **[SUPERSEDED]** ~~`RunCopyOnSelect` no longer spawns the orchestrator.~~
  Current: the hook still detaches; `RunCopyOnSelectOrchestrate` skips the
  hand-off for right-terminal sources (including registry-identified split
  halves), pinned by `TestCopyOnSelectOrchestrateSkipsRightTerminal` and
  `...SkipsSplitTerminalHalfViaRegistry`.
- A selection in the AGENT pane still hands off — verified live by the user
  after restart, and pinned by `TestCopyOnSelectOrchestrateHandsOff` plus shell
  case (a).
- `tests/copy-on-select-test.sh` asserts the new contract ("selection does not
  auto-paste into the draft") instead of driving the old chain end-to-end.

## Revisions

- 2026-07-28 (scope correction): the first implementation disabled the
  auto-paste for EVERY source pane. The request was "disable the copy on select
  and paste **from right pane** to draft nvim" — the user confirmed after
  restart that agent-pane auto-paste had stopped working, which they still
  want. Selecting something the agent said and having it land in the draft as a
  quote is the point of the feature; only the terminal case is noise. Corrected
  to a SOURCE-PANE GATE: the hook detaches the orchestrator again (restoring
  #100's shape) and the orchestrator skips the hand-off when the selection was
  made in the right terminal, in addition to the existing draft/nvim skip.
  Split halves are covered via #123's terminal-pane registry, since zellij
  reports them with no `terminal_command` and a `[terminal N]` title — a plain
  role check would miss them and they would keep auto-pasting. The docs written
  during the first pass ("unwired since #125", "REMOVED in #125") were
  corrected to "narrowed": nothing is unwired, and #126 remains useful only as
  a *deliberate* trigger, not as the sole surviving path.

## Estimate

Method A against `estimate-logic-v3.1` (source stale, same caveat as #123/#124):
a one-call removal plus test/doc reconciliation; no new logic.

Revised after the plan-quality gate: 0.2h covered the code change but not ~6
doc edits, two stale comment blocks, the reworked assertions, a full suite run,
and live in-session verification.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.05 impl=0.20
item: atlas-docs design=0.05 impl=0.15
total: 0.45
```

## Plan

- [x] **Make `RunCopyOnSelect` mirror-only.** Keep `ClipboardCopy`; delete the
      `SpawnDetached(SelfExe, "clip", "copy-on-select", "--orchestrate")` at
      `clipcmd/run.go:81`, with the rationale in the doc comment. RED first:
      rename `TestCopyOnSelectHookMirrorsThenDetaches`
      (`clipcmd/run_test.go:109`) → `TestCopyOnSelectHookMirrorsOnly` and
      invert its `spawned` assertion (`len(f.spawned) != 0` → fail) so the
      test name doesn't lie to `go test -run`. In
      `TestCopyOnSelectEmptySelection` (`:129`) keep `f.copied == ""` as the
      real assertion and drop the now-vacuous spawn check (nothing can spawn).
- [x] **Keep the process-level in_nvim regression — do not degrade it.**
      `tests/copy-on-select-test.sh` exists to prove, against the fake
      `zellij`/`pbcopy`/`pbpaste`, that in_nvim keys on `terminal_command`
      rather than the cwd-polluted `title` (the #copy-on-select-test fix). The
      `--orchestrate` leaf stays a live CLI entry (`clipcmd/runcli.go:20`,
      pinned by `dispatcher/dispatcher_test.go:55`), so keep those cases by
      invoking it **directly** as `pair clip copy-on-select --orchestrate`,
      and ADD a hook case asserting mirror-only (clipboard populated; no quote
      staged; no focus change). This preserves the only test that drives the
      real binary against a stateful process-level fake (`ARCH-MOCK`) — the
      alternative (dropping to unit-only) would leave the whole `zellij action`
      seam covered by `fakeRuntime` alone.
- [x] **Annotate every unwired site**, per the Spec's "every such site"
      commitment. Generate the set rather than hand-listing line numbers
      (they drift as edits land above them):
      `grep -rn "copy-on-select\|copy_command\|clipboard-to-pane\|PairPasteQuote\|quote-" atlas/ nvim/ zellij/ cmd/ Makefile.local`
      Known members: `atlas/architecture.md` (~371 copy_command, ~403-424 the
      two wrapper sections, ~552 "PairPasteQuote … triggered by
      clipboard-to-pane", ~618 auto-insert, ~889 "quote-<tag> … overwritten on
      every selection" — the last two become FALSE, not merely stale);
      `nvim/init.lua:1399-1404` and `:3800-3806`; `clipcmd/run.go:36-39`
      (`SpawnDetached`'s interface doc — after this change the seam method has
      no production caller; say it stays, per unwire-don't-demolish);
      `Makefile.local:212-219`. Each gets "unwired since #125 — no production
      caller; see #126".
      OUT of scope, deliberately: `atlas/go-migration-inventory.md` (historical
      migration record — editing it falsifies history) and `README.md:243`
      (names `pair clip copy-on-select` only as a dispatcher example, still
      true). Fix pre-existing `cmd/copy-on-select` / `cmd/clipboard-to-pane`
      naming staleness (#104 folded these into `pair clip …`) ONLY on lines
      already being touched — not a general atlas refresh.
- [x] **Correct, do not delete, the two mouse-mode mentions.**
      `tests/term-pane-shortcuts-test.sh:230` is NOT stale: `copy_command` and
      the clipboard mirror both survive, so `mouse_mode false` would still kill
      copy-on-select's remaining half. Narrow the phrase to "copy-on-select
      clipboard mirror" rather than removing the mention. Same for
      `zellij/config.kdl:8-11`. The block that genuinely needs rewriting is
      `zellij/config.kdl:45-50`, whose "runs the Alt+n flow" is already wrong
      today (Alt+n is `PairConfirmRestart`).
- [x] **Verify.** `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (session env
      leaks cause false failures); `zellij --config-dir zellij setup --check`
      (KDL syntax ONLY — it cannot tell you the copy path still works). Then
      live: `make build` AND install/`pair restart` first — the live check runs
      against the INSTALLED `pair`, so without that a running session still
      auto-pastes and the result is meaningless. Then select text in the right
      terminal → `pbpaste` returns it, the draft is untouched, focus has not
      moved, and no `quote-<tag>` file appears.
- [x] Do NOT hand-edit `cmd/internal/runtimebundle/assets/runtime/files/…` —
      generated and untracked; the config/nvim edits propagate on next build.

## Log

### 2026-07-28
- 2026-07-28: closed — Source-pane gate: agent-pane selections still auto-paste into the draft (quote reflow); right-terminal selections only reach the OS clipboard. Mutation-proven at both layers — removing the gate fails TestCopyOnSelectOrchestrateSkipsRightTerminal, TestCopyOnSelectOrchestrateSkipsSplitTerminalHalfViaRegistry, and shell case (c). Split halves are covered via #123 terminal-pane registry (zellij reports them with no terminal_command and a [terminal N] title, so a plain role check would miss them). The process-level in_nvim regression is preserved. Full env -u make test green through its pre-existing scrollback-open Alt+x abort (identical on main), post-abort targets run separately and green; zellij setup --check; git diff --check. USER LIVE-VERIFIED after restart: agent-pane paste works, terminal stays quiet.; review verdict: FIX-THEN-SHIP
- Implemented mirror-only: RED first (renamed `TestCopyOnSelectHookMirrorsOnly`,
  inverted its spawn assertion) then deleted the one `SpawnDetached` call.
  Mutation-proven at BOTH layers: restoring the spawn fails the unit test and
  fails the new shell case (c).
- Kept the process-level in_nvim regression by driving the `--orchestrate` leaf
  directly (it is still a live CLI entry), so the only test that runs the real
  binary against a process-level zellij/pbcopy fake survives (ARCH-MOCK), and
  added case (c) for the mirror-only hook.
- Annotated the unwired machinery from a generated grep rather than hand-picked
  lines: atlas ~371/413/552/618/889, `zellij/config.kdl`, `nvim/init.lua`
  (PairPasteQuote + its `<C-_>` gate + the cursor-styling rationale),
  `clipcmd/run.go` SpawnDetached seam doc, `Makefile.local`. Corrected rather
  than deleted the two `mouse_mode` mentions — `copy_command` and the clipboard
  mirror both survive, so "mouse_mode false kills it" is still true, narrowed to
  "clipboard mirror". Also fixed config.kdl's long-wrong "Alt+n flow" credit
  (Alt+n is PairConfirmRestart).
- Verified: `go test ./cmd/internal/clipcmd`; full `env -u ... make test` green
  through its pre-existing scrollback-open Alt+x abort (same on main), with the
  post-abort targets run separately; `zellij setup --check`; `git diff --check`.
- Live-verified by the user after a restart: selecting text in the right
  terminal copies to the clipboard and the draft is untouched — no flash, no
  focus change, no auto-paste. Closing on that.
- Scope correction implemented and live-verified by the user: agent-pane
  selections paste into the draft as before; right-terminal selections only
  reach the clipboard. Mutation-proven — removing the gate fails both new unit
  cases and shell case (c). Full `env -u ... make test` green through the
  pre-existing scrollback-open Alt+x abort (identical on main), post-abort
  targets green, `zellij setup --check`, `git diff --check`.
