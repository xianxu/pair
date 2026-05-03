---
type: pensive
date: 2026-05-02
topic: nvim as input field for TUI coding agents
mode: ideas
description: Pattern — run a TUI coding agent (Claude Code, Codex, Gemini) in one zellij pane and drive it from nvim in another. Independent scroll, rich editing, scriptable bidirectional flow. Universal across agents.
references: []
---

# Pensive: Nvim as Input Field for TUI Coding Agents

The TUI input box on every coding agent — Claude Code, Codex, Gemini CLI — is cramped, lacks editing power, and conflates the input affordance with the output affordance in the same scrollable region. The fix is structural, not cosmetic: split the surface into two specialized panes inside a terminal multiplexer. The agent runs in one pane and owns the *output* affordance (live streaming, tool calls, diffs, image rendering). Nvim runs in the other and owns the *input* affordance (undo, search/replace, syntax highlighting, snippets, telescope, persistent draft history). The two panes scroll independently. You compose in nvim with full editing power, then send the buffer (or a region) into the agent's input on a keystroke.

The mechanism is zellij's CLI: `zellij action write-chars "..."` writes a literal string to the focused pane, and `zellij action write 13` sends Enter. From nvim, a function gathers the buffer, focuses the agent's pane via `move-focus up`, types the content, sends Enter, and focuses back. Layouts are KDL files — pre-configure "agent on top 65%, nvim on bottom 35%" once and launch with `zellij --session work --layout claude-driver`. Idempotent attach. Toggle nvim to fullscreen for serious composition (`toggle-pane-fullscreen` preserves the underlying split), then back. Two flavors: single zellij with internal split (focus-relative, simplest), or two zellij sessions in separate ghostty panes (target by name with `--session`, no focus juggling). Single-zellij wins on ergonomics — fullscreen toggle, layouts, one process to manage.

Two refinements that matter beyond the basic plumbing. First, **paragraph reflow on copy from agent**: TUIs wrap output at terminal width, so selecting across wrapped lines lands hard `\n` mid-sentence in the clipboard. Pipe the pasted region through `par -w99999` (paragraph-aware reflow that respects list markers and blank-line breaks) on a `<leader>cp` keybind. Second, **reverse direction — hotkey from agent pane to nvim cursor**: when focus is in the agent pane and you've selected something into the clipboard, a hotkey should push that selection into nvim at the current cursor. Cleanest path is a zellij keybind running a small helper script (`zellij action move-focus down && zellij action write-chars "$(pbpaste)" && zellij action move-focus up`); next-cleanest is a Hammerspoon hotkey scoped to the terminal app. Wrap the pasted content as a `> ` quote block on the way in, so selections from the agent land as quoted context ready to react to.

The asymmetric-pane factoring composes well: any improvement to the *composer* (snippets for common openings, region-send between `---` markers, append-to-log on every send, prompt history grep) compounds because the agent side doesn't need to grow. Image paste is unchanged — Ctrl+V in the agent pane reads the OS clipboard directly, so as long as you put image bytes on the clipboard before triggering paste (via `osascript ... «class PNGf»` on macOS), it works. For non-clipboard file attachments, `@/abs/path.png` typed into the input is the cleaner path than going through the clipboard at all. Start with Claude Code; the same pattern transposes to Codex and Gemini CLI without modification — none of the mechanics depend on which agent is running, only on the agent presenting a TUI input that accepts typed text. Image-paste support varies per agent; clipboard-injection still works as long as the agent reads the clipboard on Ctrl+V.

## Open questions

- Does the bracketed-paste detection in each agent (Claude Code, Codex, Gemini) turn `write-chars` of a large block into a "Pasted text #N" chip, or inline it? Probably fine either way for this workflow, but worth checking — the chip behavior may actually be preferable.
- For the reverse hotkey (selection from agent → nvim), is zellij's `Run` action's transient pane flicker tolerable, or is Hammerspoon worth pulling in? Probably tolerable; only graduate if it grates.
- Persistent draft file (`~/scratch/claude-draft.md`) vs. per-session scratch buffer — the former gives grep-able prompt history forever, the latter is lighter. Lean toward persistent file with a section delimiter (`---`) so multiple drafts coexist and `SendSectionToClaude` extracts the one under the cursor.
- Does it make sense to log every send and every reply (extracted from agent pane) into a dated markdown file, turning the workflow into a self-recording conversation? That's the "data into central location, shell-ed agent runs free" pattern from earlier conversations applied to one's own conversations.
- How well does the pattern generalize to non-coding TUI agents (e.g., a chat-only TUI)? Probably the same, since none of the mechanics are coding-specific.
