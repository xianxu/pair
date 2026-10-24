#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
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
# --orchestrate`, which admits and executes the full auto-quote in process.
#
# #125 narrowed WHICH panes hand off: the AGENT pane still does (quoting what
# the agent said back into the draft is the point), but the RIGHT TERMINAL does
# not — selecting a path there is usually just a copy, and the flash + focus
# steal + `> ` insert was distracting. Case (c) pins that gate.
#
# The hand-off is observable as the staged quote file $PAIR_DATA_DIR/quote-<tag>.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
export PAIR_TEST_ZELLIJ_REPLIES="$REPO/tests/lib/zellij_fixture_replies.py"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-copyonselect.XXXXXX") || exit 1
trap 'status=$?
  if [ "$status" -ne 0 ]; then
    cat "$tmp/cache/pair/clipboard-debug.log" "$tmp/orchestrator.stderr" >&2 2>/dev/null || :
  fi
  rm -rf "${tmp:?}"' EXIT

# The hook self-execs its `pair` sibling. Alias the shared image beside the
# completion wrapper; copying it would trigger a new AMFI assessment (#749).
export ZELLIJ_SOCKET_DIR="$tmp/native"
export PAIR_HOME="$tmp/home"
export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG=t
export PAIR_AGENT=claude
mkdir -p "$PAIR_HOME/bin" "$PAIR_DATA_DIR"
bash "$REPO/tests/lib/pair-test-binary.sh" "$PAIR_HOME/bin/pair-real"
if [ -n "${PAIR_TEST_BINARY:-}" ]; then
  python3 - "$PAIR_TEST_BINARY" "$PAIR_HOME/bin/pair-real" <<'PY'
import os,sys
assert os.path.samefile(*sys.argv[1:]), 'copy fixture must execute the shared image, never a per-run copy (#749)'
PY
fi
export PAIR_TEST_COPY_BINARY="$PAIR_HOME/bin/pair-real"
export PAIR_TEST_COPY_DONE="$tmp/done"
export PAIR_TEST_COPY_LOG="$tmp/orchestrator.stderr"
mkfifo "$PAIR_TEST_COPY_DONE"
cat > "$PAIR_HOME/bin/pair" <<'SH'
#!/bin/sh
"$PAIR_TEST_COPY_BINARY" "$@" 2>>"$PAIR_TEST_COPY_LOG"
result=$?
printf '%s\n' "$result" > "$PAIR_TEST_COPY_DONE"
exit "$result"
SH
chmod +x "$PAIR_HOME/bin/pair"

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
  *list-clients*) python3 -c 'import json,sys; from pathlib import Path; sys.path.insert(0,str(Path("$REPO/tests/lib/zellij_fixture_replies.py").parent)); from zellij_fixture_replies import emit; rows=json.load(open("$tmp/panes.json")); p=next(p for p in rows if p.get("is_focused") and not p.get("is_plugin")); emit("clients",[{"id":1,"pane_id":"terminal_"+str(p["id"]),"running_command":"shell"}])' ;;
  *) : ;;
esac
EOF
chmod +x "$fakebin/pbcopy" "$fakebin/pbpaste" "$fakebin/zellij"
export PATH="$fakebin:$PATH"
export XDG_CACHE_HOME="$tmp/cache"

# Agent pane: a deliberately HOSTILE title containing "nvim" (via a legacy
# pre-#133 cwd suffix — production no longer emits one), while terminal_command is
# the pair-wrap launch (no nvim/draft). Keep the misleading title: the point is
# that detection keys on terminal_command, not the title. Do not "modernize" this
# to a bare "claude" — that makes the case pass trivially and tests nothing.
# Positive pane states, including the hostile title and launch templates above,
# are named catalog cases rather than hand-written native output.

quote="$PAIR_DATA_DIR/quote-t"
run() {
  rm -f "$quote" "$tmp/clipboard"
  python3 - "$PAIR_TEST_COPY_BINARY" "$PAIR_TEST_COPY_DONE" "${1:-0}" <<'PYTHON'
import os, select, subprocess, sys
fd = os.open(sys.argv[2], os.O_RDONLY | os.O_NONBLOCK)
try:
    subprocess.run([sys.argv[1], 'clip', 'copy-on-select'], input=b'selected text', check=True, timeout=3)
    if not select.select([fd], [], [], 3)[0]:
        raise SystemExit('detached copy orchestrator did not finish')
    result = os.read(fd, 32).strip()
    if result != sys.argv[3].encode():
        raise SystemExit(f'detached copy orchestrator returned {result!r}, wanted {sys.argv[3]}')
finally:
    os.close(fd)
PYTHON
}
# Completion comes from the owned orchestrator wrapper, including copy-only paths.

fail=0

# (a) Selection in the AGENT pane while cwd is parley.nvim → must hand off
#     (in_nvim=false). This is the regression: the title contains "nvim" but
#     in_nvim keys on terminal_command, so the paste must still happen.
python3 "$PAIR_TEST_ZELLIJ_REPLIES" --case copy-on-select.agent > "$tmp/panes.json"
run
[ -f "$quote" ] || { echo "FAIL (a) parley.nvim agent-pane selection did not hand off (quote not staged)"; fail=1; }

# (b) Selection in the DRAFT (nvim) pane → must NOT hand off (in_nvim=true),
#     else it would insert the selection beneath itself. The detached
#     orchestrator completion is joined before checking absence.
python3 "$PAIR_TEST_ZELLIJ_REPLIES" --case copy-on-select.draft > "$tmp/panes.json"
run
[ -f "$quote" ] && { echo "FAIL (b) draft-pane selection handed off (would self-insert)"; fail=1; }

# (c) #731: this fixture supplies a Pair-looking terminal template but starts
#     no registered wrapper owner. Refuse auto-quote without staging anything;
#     the clipboard mirror still succeeds. Registered-terminal copy-only
#     behavior is covered by TestCopyOnSelectOrchestrateSkipsRightTerminal.
python3 "$PAIR_TEST_ZELLIJ_REPLIES" --case copy-on-select.terminal > "$tmp/panes.json"
run 1
[ -f "$quote" ] && { echo "FAIL (c) right-terminal selection auto-pasted into the draft (#125)"; fail=1; }
[ "$(pbpaste 2>/dev/null)" = "selected text" ] || { echo "FAIL (c) right-terminal selection did not reach the clipboard"; fail=1; }

if [ "$fail" -eq 0 ]; then
  echo "PASS copy-on-select in_nvim detection + right-terminal gate (#125)"
fi
exit "$fail"
