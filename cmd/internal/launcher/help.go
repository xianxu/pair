package launcher

// UsageText is `pair --help` / `pair help` — a CONCISE synopsis of the launch
// verbs (#99 M5c, replacing the retired bin/pair-shell's help block). The full
// in-session keybindings / behavior notes live on Alt+h (bin/pair-help), not here,
// so this stays a short usage rather than a transcription that would drift.
func UsageText() string {
	return `pair — Neovim-backed input field for any TUI coding agent.

USAGE
  pair                          claude in the default session
  pair <agent>                  e.g. pair codex / pair agy
  pair [<agent>] --layout2      original agent/draft workbench
  pair [<agent>] --layout3      layered workbench with user terminal
  pair resume <tag>             attach this repo's tag if live, else create it
                                (agent inferred from saved state)
  pair continue [slug] [agent]  resume from a continuation doc; bare lists them
  pair [<agent>] -- <args>      forward args to the agent on create
  pair list | ls                list this repo's Pair sessions and attach state
  pair rename <old> <new>       rename every tag-scoped file from <old> to <new>
  pair -h | --help              this message

Use ` + "`--`" + ` to separate pair's args from the agent's. When creating a
session you're prompted for a name; ` + "`resume <tag>`" + ` skips the prompt.
Layout flags are Pair arguments before ` + "`--`" + ` and may appear before or
after the agent. A new tag defaults to layout2; an omitted flag reuses the tag's
recorded layout. Changing a live tag explicitly asks before recreating the whole
workbench. In-session keybindings are on Alt+h.
`
}
