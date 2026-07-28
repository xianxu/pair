#!/usr/bin/env bash
# Regression test for copy-on-select's in_nvim detection.
#
# Bug: the agent (e.g. claude) overwrites its zellij pane title with
# "claude [<cwd>]". The old detector matched the focused pane's TITLE against
# /nvim|draft/, so in a repo whose path contains "nvim" (e.g. parley.nvim) the
# agent pane was misclassified as the nvim draft pane → copy-on-select treated
# the selection as "made in nvim" and exited WITHOUT pasting. The fix keys the
# check on terminal_command (the fixed launch string), which never embeds cwd.
#
# Since #104 copy-on-select is `pair clip copy-on-select`: the hook mirrors the
# selection to the OS clipboard and detaches `pair clip copy-on-select
# --orchestrate`, which hands off in-chain to `pair clip clipboard-to-pane`.
#
# #125 narrowed WHICH panes hand off: the AGENT pane still does (quoting what
# the agent said back into the draft is the point), but the RIGHT TERMINAL does
# not — selecting a path there is usually just a copy, and the flash + focus
# steal + `> ` insert was distracting. Case (c) pins that gate.
#
# The hand-off is observable as the staged quote file $PAIR_DATA_DIR/quote-<tag>.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-copyonselect.XXXXXX"); trap 'rm -rf "$tmp"' EXIT

# The single `pair` self-execs the `pair` sibling of its own binary for the
# orchestrator + hand-off (dir(os.Executable())/pair), so a copy of it under
# $PAIR_HOME/bin drives the whole chain.
export PAIR_HOME="$tmp/home"
export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG=t
export PAIR_AGENT=claude
mkdir -p "$PAIR_HOME/bin" "$PAIR_DATA_DIR"
cp "$REPO/bin/pair" "$PAIR_HOME/bin/pair"

# Fakes on PATH: clipboard tools + a zellij that prints $tmp/panes.json for
# `list-panes` and no-ops other actions (jq is the real one). pbpaste returns the
# selection so the hand-off stages a non-empty quote file. PATH must NOT include
# $PAIR_HOME/bin so the `command -v` clipboard resolutions find these fakes.
fakebin="$tmp/fakebin"; mkdir -p "$fakebin"
printf '#!/bin/sh
cat > "%s/clipboard"
' "$tmp" > "$fakebin/pbcopy"
printf '#!/bin/sh
cat "%s/clipboard" 2>/dev/null
' "$tmp" > "$fakebin/pbpaste"
cat > "$fakebin/zellij" <<EOF
#!/bin/sh
case "\$*" in
  *list-panes*) cat "$tmp/panes.json" ;;
  *) : ;;
esac
EOF
chmod +x "$fakebin/pbcopy" "$fakebin/pbpaste" "$fakebin/zellij"
export PATH="$fakebin:$PATH"
export XDG_CACHE_HOME="$tmp/cache"

# Agent pane: title carries the cwd (parley.nvim → contains "nvim"), but
# terminal_command is the pair-wrap launch (no nvim/draft).
agent_pane='{"id":0,"is_plugin":false,"is_focused":FOCUS_AGENT,"is_floating":false,
  "title":"claude [~/workspace/parley.nvim]",
  "terminal_command":"sh -c zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair wrap --scrollback-log \"/data/scrollback-t-claude.raw\" claude"}'
# Draft pane: title is plain "draft", terminal_command launches nvim.
draft_pane='{"id":1,"is_plugin":false,"is_focused":FOCUS_DRAFT,"is_floating":false,
  "title":"draft",
  "terminal_command":"sh -c export PAIR_NVIM_PID_FILE=\"/data/nvim-pid-t-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"/data/draft-t.md\""}'

# Right terminal: `pair term` in the command is what identifies it (#125 gate).
term_pane='{"id":4,"is_plugin":false,"is_focused":true,"is_floating":false,
  "title":"terminal","terminal_command":"sh -c exec pair term"}'

quote="$PAIR_DATA_DIR/quote-t"
run() { rm -f "$quote" "$tmp/clipboard"; printf '%s' 'selected text' | "$PAIR_HOME/bin/pair" clip copy-on-select; }
# The hand-off exec-replaces into clipboard-to-pane, so poll for the staged file.
wait_staged() { for _ in $(seq 1 60); do [ -f "$quote" ] && return 0; sleep 0.05; done; return 1; }

fail=0

# (a) Selection in the AGENT pane while cwd is parley.nvim → must hand off
#     (in_nvim=false). This is the regression: the title contains "nvim" but
#     in_nvim keys on terminal_command, so the paste must still happen.
printf '[%s,%s]\n' \
  "${agent_pane/FOCUS_AGENT/true}" "${draft_pane/FOCUS_DRAFT/false}" > "$tmp/panes.json"
run
wait_staged || { echo "FAIL (a) parley.nvim agent-pane selection did not hand off (quote not staged)"; fail=1; }

# (b) Selection in the DRAFT (nvim) pane → must NOT hand off (in_nvim=true),
#     else it would insert the selection beneath itself. The detached
#     orchestrator decides quickly against the fake zellij; a 1s grace is ample,
#     after which the quote file's absence proves the gate held.
printf '[%s,%s]\n' \
  "${agent_pane/FOCUS_AGENT/false}" "${draft_pane/FOCUS_DRAFT/true}" > "$tmp/panes.json"
run
sleep 1
[ -f "$quote" ] && { echo "FAIL (b) draft-pane selection handed off (would self-insert)"; fail=1; }

# (c) #125: selection in the RIGHT TERMINAL → must NOT hand off, while the
#     clipboard mirror still happens. This is the distraction that was removed.
printf '[%s,%s,%s]\n' \
  "${agent_pane/FOCUS_AGENT/false}" "${draft_pane/FOCUS_DRAFT/false}" "$term_pane" > "$tmp/panes.json"
run
sleep 1
[ -f "$quote" ] && { echo "FAIL (c) right-terminal selection auto-pasted into the draft (#125)"; fail=1; }
[ "$(pbpaste 2>/dev/null)" = "selected text" ] || { echo "FAIL (c) right-terminal selection did not reach the clipboard"; fail=1; }

if [ "$fail" -eq 0 ]; then
  echo "PASS copy-on-select in_nvim detection + right-terminal gate (#125)"
fi
exit "$fail"
