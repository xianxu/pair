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
terminal='{"id":4,"is_plugin":false,"is_focused":FOCUS_TERM,"is_floating":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}'
review='{"id":4,"is_plugin":false,"is_focused":FOCUS_REVIEW,"is_floating":true,"title":"review","terminal_command":"nvim -u /pair/nvim/review.lua /tmp/review.md"}'

write_panes() {
  focus="$1"
  printf '[%s,%s,%s,%s]\n' \
    "${agent/FOCUS_AGENT/$([ "$focus" = agent ] && echo true || echo false)}" \
    "${draft/FOCUS_DRAFT/$([ "$focus" = draft ] && echo true || echo false)}" \
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
run_shortcut "Alt+Shift+d"
check_eq "right Alt+Shift+d splits terminal down as a native tiled split" "$(actions)" 'new-pane --direction down --name terminal -- sh -c zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term'

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

# The stateless fake zellij reports unchanged geometry after the first resize
# step, so the toggle's no-progress guard stops after one op.
write_panes terminal
run_shortcut "Alt+Shift+Enter"
check_eq "right Alt+Shift+Enter steps tiled resize toward two thirds" "$(actions)" "resize increase left"

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
check_eq "draft Alt+k production path focuses terminal by id" "$(actions)" "focus-pane-id 4"
check_eq "draft Alt+k production path records pane" "$(cat "$PAIR_DATA_DIR/last-left-pane-t")" "2"

write_panes review
run_shortcut "Alt+r"
check_eq "review Alt+r does not rename tab" "$(actions)" ""

write_panes review
run_shortcut "Alt+Shift+d"
check_eq "review Alt+Shift+d is not hijacked by terminal split" "$(actions)" ""

grep -Fq 'bind "Alt Shift Enter" { WriteChars "\u{1b}[13;4u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+Shift+Enter forwards distinct KKP sequence" \
  || { printf 'FAIL Alt+Shift+Enter bind missing\n'; fail=1; }

grep -Fq 'bind "Alt D" { WriteChars "\u{1b}[68;4u"; }' "$ROOT/zellij/config.kdl" \
  && pass "Alt+Shift+d forwards distinct KKP sequence" \
  || { printf 'FAIL Alt+Shift+d bind missing\n'; fail=1; }

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

# #123 tiled pivot: the right terminal lives in the tiled tree. Floating
# panes are frame-draggable with no zellij 0.44.3 config gate; tiled panes
# have no mouse-move operation at all, so drag-immunity is architectural.
! grep -Fq 'floating_panes' "$ROOT/zellij/layouts/main-3.kdl" \
  && ! grep -Fq 'terminal-filler' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'pane name="terminal" {' "$ROOT/zellij/layouts/main-3.kdl" \
  && pass "right terminal is a tiled pane (no floating layer, no filler)" \
  || { printf 'FAIL right terminal is not tiled (floating layer or filler remains)\n'; fail=1; }

grep -Fq 'swap_tiled_layout name="minimized"' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'swap_tiled_layout name="minimized-split"' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'swap_tiled_layout name="third"' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'swap_tiled_layout name="third-split"' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Fq 'swap_tiled_layout name="small-split"' "$ROOT/zellij/layouts/main-3.kdl" \
  && [ "$(grep -c 'tab exact_panes=3' "$ROOT/zellij/layouts/main-3.kdl")" = 2 ] \
  && [ "$(grep -c 'tab exact_panes=4' "$ROOT/zellij/layouts/main-3.kdl")" = 3 ] \
  && pass "draft rungs have 3-pane and 4-pane (split) swap variants incl. small" \
  || { printf 'FAIL swap layouts missing split variants (rung ladder breaks after Alt+Shift+d)\n'; fail=1; }

# small-split must be the LAST swap layout: the 4-pane cycle order
# [minimized-split, third-split, small-split] is what keeps nvim's
# next/prev rung semantics identical to the 3-pane cycle.
[ "$(grep 'swap_tiled_layout name=' "$ROOT/zellij/layouts/main-3.kdl" | tail -1)" = "$(grep 'swap_tiled_layout name="small-split"' "$ROOT/zellij/layouts/main-3.kdl")" ] \
  && pass "small-split is the last swap layout (cycle order preserved)" \
  || { printf 'FAIL small-split is not last in the swap cycle\n'; fail=1; }

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

grep -Eq '^[[:space:]]*mouse_mode[[:space:]]+false' "$ROOT/zellij/config.kdl" \
  && { printf 'FAIL Zellij mouse support is disabled (kills click-to-focus rescue + copy-on-select, #123 lockout)\n'; fail=1; } \
  || pass "Zellij mouse support stays enabled"

grep -Fq "'pair', 'layout', 'focus-terminal'" "$ROOT/nvim/init.lua" \
  && pass "draft Alt+k focuses terminal by pane id" \
  || { printf 'FAIL draft Alt+k does not use pair layout focus-terminal\n'; fail=1; }

grep -Fq '"move-focus", "right"' "$ROOT/cmd/internal/wrapcmd/wrap.go" \
  && { printf 'FAIL agent Alt+k still uses relative move-focus right (must target a split half by id)\n'; fail=1; } \
  || pass "agent Alt+k no longer uses relative move-focus right"

# Tiled panes are framed by zellij default (pane_frames true); the split must
# not opt out — the frame is the divider and carries the #118 tab title.
! grep -Fq '"--borderless"' "$ROOT/cmd/internal/termcmd/run.go" \
  && pass "right terminal split panes keep zellij default frames" \
  || { printf 'FAIL right terminal split passes --borderless (frames are the divider)\n'; fail=1; }

layout_terminal_shell=$(grep 'exec pair term' "$ROOT/zellij/layouts/main-3.kdl" | sed 's/^[[:space:]]*args "-c" "//; s/"$//; s/\\"/"/g')
grep -Fq "$layout_terminal_shell" "$ROOT/cmd/internal/termcmd/run.go" \
  && pass "right terminal split command matches layout3 terminal command" \
  || { printf 'FAIL right terminal split command drifted from layout3 terminal command\n'; fail=1; }

grep -Fq 'support_kitty_keyboard_protocol true' "$ROOT/zellij/config.kdl" \
  && pass "Zellij explicitly enables Kitty keyboard protocol" \
  || { printf 'FAIL Zellij Kitty keyboard protocol is not enabled\n'; fail=1; }

[ "$fail" -eq 0 ] || { printf 'term-pane-shortcuts-test FAILED\n'; exit 1; }
printf 'term-pane-shortcuts-test ok\n'
