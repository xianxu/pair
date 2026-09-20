---
id: 000285
status: open
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# Remember a thread's right-pane tabs across cold starts

## Problem

The right pane's tabs (`pair term`) die with every cold start of a thread:
Alt+x / Alt+n / Shift+Alt+N restart, Couch park→resume and continuation, the
layout-conflict relaunch, a reboot or zellij server death. The operator then
rebuilds the same tabs by hand — the names, the directories, the server / test /
nvim command each one was running. Nothing about tabs is persisted today: the
only per-tag right-pane files are the pane-id/pid shortcut records
(`terminal-panes-<tag>`, `last-terminal-pane-<tag>`).

## Spec

Per thread (`{repo scope, tag}`), remember the **primary right pane's** tabs and
recreate them on cold start.

**Remembered, per tab and in order:** name, working directory, last command
line run. Plus the active tab.

**Browser tabs (pair#292) are a second tab kind** and need their own record
fields and restore rule:
- **Remembered:** the tab kind, its URL, its name, and whether the operator
  renamed it (`named`). Not the profile dir — it is throwaway and dies with
  the tab.
- **Restore:** relaunch Carbonyl on the recorded URL, with `--zoom` re-derived
  from the *current* pane width rather than the recorded one (pair#292's
  `Zoom(cols)`), since a cold start often has a different geometry. There is no
  shell, so the prefill rule below does not apply — a browser tab restores by
  navigating, which is what its Enter would have done anyway.
- **Degraded:** if the engine is missing at restore (no `carbonyl`, no
  `PAIR_CARBONYL`), keep the entry, skip the launch, and notice once. Dropping
  the entry would silently lose the operator's URL.
- **Bound:** N browser tabs mean N Chromium process groups at cold start. Worth
  a cap or a "restore on first focus" rule if a thread accumulates them —
  decide when the record is designed, not after a resume starts six browsers.

**Restore:** when `pair term` starts as the layout's primary right pane and a
record exists, it opens one tab per entry, in order, with the recorded name and
cwd, selects the recorded active tab, and places each tab's last command in its
shell's edit buffer **without executing it** — the operator presses Return to
run it. That's the deliberate compromise: convenient, but nothing re-runs
unattended. A recorded cwd that no longer exists falls back to the launch
directory, with a notice. Warm attach needs nothing — the tabs are alive in
zellij's session.

### Design decisions

1. **One pair-owned zsh integration is the single source** for cwd, last
   command, and the prefill. Today pair injects nothing (the tab env carries
   only TERM/TERMINFO), OSC 7 reaches pair only if the operator's rc happens to
   emit it, and nothing observes commands. Inject a zsh integration via a
   `ZDOTDIR` shim — the Ghostty / VS Code pattern: the shim sources the
   operator's real dotfiles, restores `ZDOTDIR`, then adds hooks:
   - `chpwd` + first `precmd` → OSC 7. The vt emulator already parses OSC 7 into
     `DirectoryEffect`; the presenter drops it today — route it to the tab.
   - `preexec` → a pair-private OSC carrying the command line, registered with
     `RegisterOscHandler` like the OSC 9 handler.
   - Other shells: cwd only if they emit OSC 7 themselves; no command capture,
     no prefill, no errors.
2. **Prefill through the shell, never as raw PTY bytes.** Each restored tab is
   spawned with the command in an env var; the integration's first `precmd`
   runs `print -z -- "$PAIR_PREFILL"` (then unsets it), which puts the text in
   ZLE's buffer. Writing bytes into the PTY is rejected: an embedded newline
   *executes*, tab/control characters fire completion and key bindings, and
   bytes written before the line editor enters raw mode are echoed or lost. A
   multi-line command lands as a multi-line buffer, still unexecuted.
3. **Leading-space commands are not recorded** — the zsh `HIST_IGNORE_SPACE`
   convention, so the operator's existing "don't remember this" gesture keeps
   working. Otherwise the record's exposure equals `HISTFILE`'s (plaintext
   under the pair storage root).
4. **Storage is a per-tag Pair sidecar**, `terminal-tabs-<tag>` (versioned
   JSON), built only through `cmd/internal/artifactpath` and registered
   everywhere a sidecar family must be — `RenameArtifacts`,
   `EnvironmentBindings`, manifest retention, gc — like `workbench-layout-<tag>`.
   Not Couch's ThreadStore: its records decode strictly, so a new field breaks
   older binaries.
5. **Write on every change** (tab open / close / rename, cwd change, command
   start), atomically. No exit-time save — reboot and `kill -9` give no exit hook.
6. **Teardown must not erase the record.** A session kill can make the shells
   exit one at a time, and each exit runs `removeTab` — a naive writer would
   shrink the record tab by tab. Rule: once `pair term` begins stopping
   (SIGHUP / closing, or its last tab exiting), it stops writing; the last tab
   exiting never persists an empty or one-shorter list. Touches #274 (pair term's
   SIGHUP handling, unresolved).
7. **Primary pane only.** Alt+Shift+d split halves run the same command line as
   the layout pane and can't tell themselves apart, and pane IDs don't survive
   sessions. The split launch gets a marker (flag or env) so it neither reads
   nor writes the record. Restoring splits is out of scope.

### Out of scope

- Resurrecting processes or scrollback — only the command *text* comes back.
- New tabs (Alt+t) inheriting the active tab's cwd — a natural follow-up once
  cwd is tracked.
- Tab reordering — there's no reorder action to persist.

**Couch continuation keeps the thread's tag** (operator intent, and already
true: continuation parks the source and starts fresh with "the Pair scope and
tag … unchanged", `atlas/couch.md` §Continuation ownership). Continuation is
therefore just another cold start: the record carries across with no extra
wiring, and the tabs come back.

### Open questions

- Do the shim and Ghostty's own shell integration (or an operator rc that
  already emits OSC 7) coexist cleanly? Double OSC 7 is harmless; double
  `ZDOTDIR` rewriting may not be.

## Done when

- A cold restart (Alt+n) of a thread with three tabs — renamed, different cwds,
  last commands such as `make test` and `nvim README.md` — comes back with the
  same tabs in order, the same names, cwds and active tab, and each shell's edit
  buffer holding its last command, unexecuted.
- A Couch continuation of that thread restores the same tabs.
- Pressing Return runs the prefilled command; not pressing runs nothing.
- Session kill (Alt+x) and a SIGHUP cascade leave the record intact — a test
  with fake children exiting one by one after stop begins.
- A leading-space command is not recorded.
- A missing recorded cwd falls back to the launch directory with a visible
  notice.
- A split half neither reads nor writes the record.
- Under a non-zsh shell, tabs restore names and order (cwd if OSC 7 is
  emitted), with no prefill and no errors.
- The operator's zsh config still loads fully under the shim (prompt, plugins,
  history) — an e2e check against a probe rc.
- The new sidecar passes the artifactpath coverage / gc tests;
  `atlas/terminal.md` describes the record and the shell integration.

## Plan

- [ ] Probe first: what does a zsh in `pair term` emit today (OSC 7 from the
      operator rc or Ghostty's integration?), and does a `ZDOTDIR` shim load the
      operator's config intact?
- [ ] Design via `superpowers-writing-plans` → `workshop/plans/` (>3 files:
      termcmd, terminal, artifactpath, runtimebundle, launcher).

## Log

### 2026-09-18

Filed from a brain advisor session (operator request: remember tab order, name,
cwd, last command; cold start pre-types the last command but waits for Return).
Operator: continuation should keep the same tag — confirmed it already does
(`atlas/couch.md`), so that open question is closed.
Survey pointers for the implementing session:

- Tabs: `terminalTab{id,name,child}` + `terminalMux` in
  `cmd/internal/termcmd/presentation.go`; `newTab` never sets
  `ptychild.Options.Dir`; default name `terminal N`; rename via Alt+r;
  `removeTab` stops `pair term` when the last tab goes.
- Shell spawn: `OSRuntime.ShellCommand` → `$SHELL -i` (`termcmd/run.go`); env
  from `runtimebundle/terminal.go` (TERM/TERMINFO only).
- OSC: `terminal/endpoint.go` — OSC 9 handler is the `RegisterOscHandler`
  template; `WorkingDirectory` callback emits `DirectoryEffect`, which the
  presenter drops.
- Sidecar precedent: `workbench-layout-<tag>` (`launcher/layoutflow.go`);
  registration points in `artifactpath/paths.go`, `manifest.go`, `gc.go`.
- Cold vs warm: `runCreate` (`--new-session-with-layout`) vs `runAttach`
  (`zellij attach`) in `launcher/createflow.go`.
- Test seams: `ptychild.NewFakeChild` (records writes), `admitTab`-based fakes
  in `termcmd/presentation_test.go`, `hostty.FakeHost`, `fakeRuntime`.

### 2026-09-19 — browser tabs join the tab model (pair#292)

pair#292 adds a browser tab kind to `pair term`. Its Spec gained the record
fields and restore rule above. Two are load-bearing: `--zoom` is re-derived
from the pane width at restore rather than replayed, and a resume must not
silently start a browser per remembered tab without a bound.
