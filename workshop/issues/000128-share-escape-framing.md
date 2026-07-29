---
id: 000128
status: open
deps: [pair#127]
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours:
---

# share escape-sequence framing between termcmd and wrapcmd

## Problem

Two packages in one binary now carry their own escape-sequence framing:

- `wrapcmd`: `otherEscRe` (wrap.go:189) encodes CSI plus OSC terminated by
  BEL-or-ST, and `stripCodexOutputMarkers` (wrap.go:766) is a byte-level marker
  stripper with tail carry (`p.stdoutPending`, wrap.go:310).
- `termcmd`: `queries.go` (#127) frames CSI and OSC to strip capability queries
  out of a tab redraw.

The *policy* tables must stay separate — they are in one case opposed, since
`wrapcmd` strips `\x1b[>7u` so codex stops pushing Kitty flags while `termcmd`
requires `\x1b[>1u` to survive a replay. What should not be duplicated is the
**framing**: "where does this CSI/OSC end".

#127 deliberately scoped the extraction out: the repo is a flat
`cmd/internal/<pkg>` layout with no shared home, so extracting would have created
a package as a side effect of a two-defect bugfix. `otherEscRe` is also consumed
three ways — `ReplaceAll` in `stripTerminalControls` (wrap.go:812) and the
capture-early path (wrap.go:1151), and `FindIndex` **at an offset** in the
colored-run walker (wrap.go:1018) — so a byte-scanner does not drop in; sharing
means rewriting three call sites that feed scrollback capture and agent-output
detection.

## Spec

- One framing helper answering "length of the CSI/OSC/escape at buf[i]", used by
  both packages. Policy tables stay where they are (`ARCH-DRY` applies to the
  framing, not the policy).
- Decide its home: a new `cmd/internal/ansi` package, or hosting it in one of the
  two existing packages if the dependency direction stays acyclic.
- These wrapcmd tests must stay green across the change:
  `stdout_filter_test.go`, `extract_fg_test.go`, `update_agent_output_test.go`.

## Done when

- One framing implementation; `otherEscRe`'s three call sites still behave
  identically, pinned by the tests above.
- `termcmd/queries.go` and `wrapcmd`'s stripper both call it.
- The opposed-policy note in `atlas/architecture.md` still reads true.

## Log

### 2026-07-28

- Filed from pair#127's close. #127 added `termcmd/queries.go`, which frames CSI
  and OSC for a second time in this binary; the deferral and its reasoning are
  recorded in #127's Plan and in `atlas/architecture.md`.
- Note for whoever picks this up: the deny-list's two failure directions are NOT
  symmetric. A missed query degrades benignly, but an **over-strip** silently
  removes mouse mode, Kitty encoding, or the cursor shape. #127 added
  `FuzzStripTerminalQueries` as the cheap structural guard; the shared framing
  helper is the natural place to attach a real conformance check against
  emitters if that ever becomes worth building.
