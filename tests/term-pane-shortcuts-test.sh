#!/usr/bin/env bash
# Regression test for the three-pane workbench shortcut gates (#116).
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-term-shortcuts.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

export PAIR_HOME="$tmp/home"
export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG=t
export PAIR_AGENT=codex
mkdir -p "$PAIR_HOME/bin" "$PAIR_DATA_DIR"
cp "$ROOT/bin/pair" "$PAIR_HOME/bin/pair"

fakebin="$tmp/fakebin"
mkdir -p "$fakebin"
cat > "$fakebin/zellij" <<EOF
#!/bin/sh
if [ "\$1 \$2" = "action list-panes" ]; then
  cat "$tmp/panes.json"
  exit 0
fi
if [ "\$1" = "action" ]; then
  shift
  printf '%s\n' "\$*" >> "$tmp/actions.log"
  exit 0
fi
if [ "\$1" = "run" ]; then
  printf '%s\n' "\$*" >> "$tmp/actions.log"
  exit 0
fi
exit 0
EOF
chmod +x "$fakebin/zellij"
export PATH="$fakebin:$PAIR_HOME/bin:$PATH"

agent='{"id":1,"is_plugin":false,"is_focused":FOCUS_AGENT,"is_floating":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"}'
draft='{"id":2,"is_plugin":false,"is_focused":FOCUS_DRAFT,"is_floating":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}'
filler='{"id":3,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"}'
terminal='{"id":4,"is_plugin":false,"is_focused":FOCUS_TERM,"is_floating":true,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}'
review='{"id":4,"is_plugin":false,"is_focused":FOCUS_REVIEW,"is_floating":true,"title":"review","terminal_command":"nvim -u /pair/nvim/review.lua /tmp/review.md"}'

write_panes() {
  focus="$1"
  printf '[%s,%s,%s,%s,%s]\n' \
    "${agent/FOCUS_AGENT/$([ "$focus" = agent ] && echo true || echo false)}" \
    "${draft/FOCUS_DRAFT/$([ "$focus" = draft ] && echo true || echo false)}" \
    "$filler" \
    "${terminal/FOCUS_TERM/$([ "$focus" = terminal ] && echo true || echo false)}" \
    "${review/FOCUS_REVIEW/$([ "$focus" = review ] && echo true || echo false)}" \
    > "$tmp/panes.json"
}

run_shortcut() {
  rm -f "$tmp/actions.log"
  "$PAIR_HOME/bin/pair" term --test-shortcut "$1" >/dev/null
}

run_shortcut_with_stdin() {
  rm -f "$tmp/actions.log"
  printf '%s\n' "$2" | "$PAIR_HOME/bin/pair" term --test-shortcut "$1" >/dev/null
}

actions() {
  [ -f "$tmp/actions.log" ] && cat "$tmp/actions.log" || true
}

fail=0
pass() { printf 'PASS %s\n' "$1"; }
check_eq() {
  name="$1"; got="$2"; want="$3"
  if [ "$got" = "$want" ]; then
    pass "$name"
  else
    printf 'FAIL %s: got [%s], want [%s]\n' "$name" "$got" "$want"
    fail=1
  fi
}

write_panes terminal
run_shortcut "Alt+t"
check_eq "right Alt+t stays local to pair term" "$(actions)" ""

write_panes terminal
run_shortcut "Alt+w"
check_eq "right Alt+w stays local to pair term" "$(actions)" ""

write_panes terminal
run_shortcut "Alt+r"
check_eq "right Alt+r stays local to pair term" "$(actions)" ""

write_panes terminal
run_shortcut "Alt+x"
check_eq "right Alt+x focuses then routes quit to draft" "$(actions)" "focus-pane-id 2
write --pane-id 2 28
write --pane-id 2 14
write-chars --pane-id 2 :lua PairConfirmQuit()
write --pane-id 2 13"

if grep -Fq "readRawPrompt" "$ROOT/cmd/internal/termcmd/run.go"; then
  printf 'FAIL Alt+r still uses a content-area prompt\n'
  fail=1
else
  printf 'PASS Alt+r no longer uses a content-area prompt\n'
fi

write_panes terminal
run_shortcut "Alt+j"
check_eq "right Alt+j is no-op" "$(actions)" ""

write_panes terminal
run_shortcut "Alt+Shift+Enter"
check_eq "right Alt+Shift+Enter changes floating geometry once" "$(actions)" "change-floating-pane-coordinates --pane-id 4 --x 37 --y 0 --width 113 --height 51 --borderless false --pinned true"

write_panes terminal
rm -f "$PAIR_DATA_DIR/last-left-pane-t"
run_shortcut "Alt+k"
check_eq "right Alt+k falls back to draft" "$(actions)" "focus-pane-id 2"

printf '1\n' > "$PAIR_DATA_DIR/last-left-pane-t"
run_shortcut "Alt+k"
check_eq "right Alt+k returns to last left pane" "$(actions)" "focus-pane-id 1"

write_panes agent
rm -f "$tmp/actions.log" "$PAIR_DATA_DIR/last-left-pane-t"
ZELLIJ_PANE_ID=2 nvim --headless -u "$ROOT/nvim/init.lua" "$tmp/draft.md" \
  -c 'lua vim.g.pair_test_has_ui = true; PairFocusTerminal()' -c 'qa!' >/dev/null 2>&1
check_eq "draft Alt+k production path moves right" "$(actions)" "move-focus right"
check_eq "draft Alt+k production path records pane" "$(cat "$PAIR_DATA_DIR/last-left-pane-t")" "2"

write_panes review
run_shortcut "Alt+r"
check_eq "review Alt+r does not rename tab" "$(actions)" ""

grep -Fq 'bind "Alt Shift Enter" { WriteChars "\u{1b}[13;4u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+Shift+Enter forwards distinct KKP sequence" \
  || { printf 'FAIL Alt+Shift+Enter bind missing\n'; fail=1; }

grep -Fq 'bind "Alt x" { WriteChars "\u{1b}[120;3u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+x forwards distinct KKP sequence" \
  || { printf 'FAIL Alt+x bind missing\n'; fail=1; }

for binding in \
  'bind "Alt d" { WriteChars "\u{1b}[100;3u"; }' \
  'bind "Alt n" { WriteChars "\u{1b}[110;3u"; }' \
  'bind "Ctrl Alt n" { WriteChars "\u{1b}[110;7u"; }' \
  'bind "Alt N" { WriteChars "\u{1b}[78;4u"; }' \
  'bind "Alt Up" { WriteChars "\u{1b}[1;3A"; }' \
  'bind "Alt Down" { WriteChars "\u{1b}[1;3B"; }' \
  'bind "Alt c" { WriteChars "\u{1b}[99;3u"; }'
do
  grep -Fq "$binding" "$ROOT/zellij/config.kdl" \
    || { printf 'FAIL global forwarding bind missing: %s\n' "$binding"; fail=1; }
done
[ "$fail" -ne 0 ] || pass "all draft-routed globals forward one distinct sequence"

if grep -Fq 'WriteChars ":lua ' "$ROOT/zellij/config.kdl"; then
  printf 'FAIL zellij still injects draft Lua commands directly\n'
  fail=1
else
  pass "global KDL contains no draft Lua injection"
fi

grep -Fq 'pane name="terminal-filler" borderless=true {' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'pane name="terminal" x="50%" y="0%" width="50%" height="100%" pinned=true {' "$ROOT/zellij/layouts/main-3.kdl" \
  && pass "terminal uses permanent floating layer over filler" \
  || { printf 'FAIL layered terminal layout missing\n'; fail=1; }

test ! -e "$ROOT/zellij/layouts/main.kdl" \
  && ! grep -Fq 'pair term' "$ROOT/zellij/layouts/main-2.kdl" \
  && pass "layout2 stays agent and draft only" \
  || { printf 'FAIL layout2 contains terminal topology\n'; fail=1; }

shared2="$(grep 'args "-c"' "$ROOT/zellij/layouts/main-2.kdl" | sed 's/^[[:space:]]*//' | head -n 2)"
shared3="$(grep 'args "-c"' "$ROOT/zellij/layouts/main-3.kdl" | sed 's/^[[:space:]]*//' | head -n 2)"
test "$shared2" = "$shared3" \
  && pass "layout2 and layout3 share agent and draft launch commands" \
  || { printf 'FAIL shared layout commands drifted\n'; fail=1; }

grep -Fq 'bind "Alt N" { WriteChars "\u{1b}[78;4u"; }' "$ROOT/zellij/config.kdl" \
  && grep -Fq 'function _G.PairConfirmAgentRestart()' "$ROOT/nvim/init.lua" \
  && pass "Alt+Shift+N restarts only supervised agent" \
  || { printf 'FAIL agent-only restart binding missing\n'; fail=1; }

grep -Fq 'show_startup_tips false' "$ROOT/zellij/config.kdl" \
  && pass "Zellij startup tips are disabled" \
  || { printf 'FAIL Zellij startup tips are enabled\n'; fail=1; }

grep -Fq 'focus_follows_mouse false' "$ROOT/zellij/config.kdl" \
  && pass "Zellij focus does not follow the mouse across asymmetric layers" \
  || { printf 'FAIL Zellij focus-follows-mouse is enabled\n'; fail=1; }

[ "$fail" -eq 0 ] || { printf 'term-pane-shortcuts-test FAILED\n'; exit 1; }
printf 'term-pane-shortcuts-test ok\n'
