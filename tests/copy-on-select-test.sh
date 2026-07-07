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
# We drive the real binary with a fake `zellij` that emits a captured panes
# JSON, stub the downstream handoff (`pair clip clipboard-to-pane`), and
# assert the handoff is reached when (and only when) the selection was NOT in
# the draft pane.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-copyonselect.XXXXXX"); trap 'rm -rf "$tmp"' EXIT

# Sandbox PAIR_HOME. Since #104 M2 copy-on-select self-execs the `pair` SIBLING
# of its own binary as `pair clip <leaf>` (dir(os.Executable())/pair), so we place
# a fake `$PAIR_HOME/bin/pair` next to the real copy-on-select: it routes the
# detached `clip copy-on-select --orchestrate` back to the real copy-on-select and
# stubs the flash/clipboard hand-off leaves, so the stub drives the real chain.
export PAIR_HOME="$tmp/home"
mkdir -p "$PAIR_HOME/bin"
cp "$REPO/bin/copy-on-select" "$PAIR_HOME/bin/"

# Fake pair sibling: route `clip copy-on-select [--orchestrate]` to the real
# copy-on-select; the flash leaf is a no-op; the clipboard-to-pane leaf records
# that the hand-off was reached.
cat > "$PAIR_HOME/bin/pair" <<EOF
#!/bin/sh
if [ "\$1" = clip ] && [ "\$2" = copy-on-select ]; then
  shift 2
  exec "$PAIR_HOME/bin/copy-on-select" "\$@"
elif [ "\$1" = clip ] && [ "\$2" = clipboard-to-pane ]; then
  echo reached > "$tmp/handoff"
elif [ "\$1" = clip ] && [ "\$2" = flash-pane ]; then
  exit 0
fi
EOF
chmod +x "$PAIR_HOME/bin/pair"

# Fakes on PATH: clipboard sink + a zellij that prints $tmp/panes.json for
# `list-panes` (jq is the real one). PATH must NOT include $PAIR_HOME/bin so
# the `command -v` resolutions in the script find these, not the stubs.
fakebin="$tmp/fakebin"; mkdir -p "$fakebin"
printf '#!/bin/sh\ncat >/dev/null\n' > "$fakebin/pbcopy"
cat > "$fakebin/zellij" <<EOF
#!/bin/sh
case "\$*" in
  *list-panes*) cat "$tmp/panes.json" ;;
  *) : ;;
esac
EOF
chmod +x "$fakebin/pbcopy" "$fakebin/zellij"
export PATH="$fakebin:$PATH"
export XDG_CACHE_HOME="$tmp/cache"

# Agent pane: title carries the cwd (parley.nvim → contains "nvim"), but
# terminal_command is the pair-wrap launch (no nvim/draft).
agent_pane='{"id":0,"is_plugin":false,"is_focused":FOCUS_AGENT,"is_floating":false,
  "title":"claude [~/workspace/parley.nvim]",
  "terminal_command":"sh -c zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" \"${PAIR_PANE_TITLE:-agent}\" 2>/dev/null; exec pair-wrap --scrollback-log \"/data/scrollback-t-claude.raw\" claude"}'
# Draft pane: title is plain "draft", terminal_command launches nvim.
draft_pane='{"id":1,"is_plugin":false,"is_focused":FOCUS_DRAFT,"is_floating":false,
  "title":"draft",
  "terminal_command":"sh -c export PAIR_NVIM_PID_FILE=\"/data/nvim-pid-t-draft\" && exec nvim -u \"$PAIR_HOME/nvim/init.lua\" \"/data/draft-t.md\""}'

run() { rm -f "$tmp/handoff"; printf '%s' 'selected text' | "$PAIR_HOME/bin/copy-on-select"; }
# Since #100 the hook returns immediately and the paste runs in a DETACHED
# `copy-on-select --orchestrate` (so zellij can't reap it mid-chain). The hand-off
# is therefore asynchronous: poll for the stub's marker instead of reading it once.
wait_reached() { for _ in $(seq 1 60); do [ -f "$tmp/handoff" ] && return 0; sleep 0.05; done; return 1; }

fail=0

# (a) Selection in the AGENT pane while cwd is parley.nvim → must hand off
#     (in_nvim=false). This is the regression: title contains "nvim" but the
#     paste must still happen.
printf '[%s,%s]\n' \
  "${agent_pane/FOCUS_AGENT/true}" "${draft_pane/FOCUS_DRAFT/false}" > "$tmp/panes.json"
run
wait_reached || { echo "FAIL (a) parley.nvim agent-pane selection did not hand off (paste skipped)"; fail=1; }

# (b) Selection in the DRAFT (nvim) pane → must NOT hand off (in_nvim=true), else
#     copy-on-select would insert your own selection beneath itself. The detached
#     orchestrator decides quickly against the fake zellij; a 1s grace is ample for
#     it to run and skip, after which the marker's absence proves the gate held.
printf '[%s,%s]\n' \
  "${agent_pane/FOCUS_AGENT/false}" "${draft_pane/FOCUS_DRAFT/true}" > "$tmp/panes.json"
run
sleep 1
[ -f "$tmp/handoff" ] && { echo "FAIL (b) draft-pane selection handed off (would self-insert)"; fail=1; }

if [ "$fail" -eq 0 ]; then
  echo "PASS copy-on-select in_nvim detection (terminal_command, not cwd-polluted title)"
fi
exit "$fail"
