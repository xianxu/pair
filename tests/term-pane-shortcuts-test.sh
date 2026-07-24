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
exit 0
EOF
chmod +x "$fakebin/zellij"
export PATH="$fakebin:$PAIR_HOME/bin:$PATH"

agent='{"id":1,"is_plugin":false,"is_focused":FOCUS_AGENT,"is_floating":false,"title":"codex","terminal_command":"pair wrap codex"}'
draft='{"id":2,"is_plugin":false,"is_focused":FOCUS_DRAFT,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}'
terminal='{"id":3,"is_plugin":false,"is_focused":FOCUS_TERM,"is_floating":false,"title":"terminal","terminal_command":"pair term"}'
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
check_eq "right Alt+t creates terminal tab" "$(actions)" "new-tab --name terminal --layout-string layout {
    pane command=\"pair\" name=\"terminal\" {
        args \"term\"
    }
}"

write_panes terminal
run_shortcut "Alt+w"
check_eq "right Alt+w closes tab" "$(actions)" "close-tab"

write_panes terminal
run_shortcut_with_stdin "Alt+r" "work"
check_eq "right Alt+r renames tab" "$(actions)" "rename-tab work"

write_panes terminal
run_shortcut "Alt+j"
check_eq "right Alt+j is no-op" "$(actions)" ""

write_panes terminal
rm -f "$PAIR_DATA_DIR/last-left-pane-t"
run_shortcut "Alt+k"
check_eq "right Alt+k falls back to draft" "$(actions)" "focus-pane-id 2"

printf '1\n' > "$PAIR_DATA_DIR/last-left-pane-t"
run_shortcut "Alt+k"
check_eq "right Alt+k returns to last left pane" "$(actions)" "focus-pane-id 1"

write_panes agent
run_shortcut "Alt+k"
check_eq "left Alt+k focuses terminal" "$(actions)" "focus-pane-id 3"
check_eq "left Alt+k records last left pane" "$(cat "$PAIR_DATA_DIR/last-left-pane-t")" "1"

write_panes review
run_shortcut "Alt+r"
check_eq "review Alt+r does not rename tab" "$(actions)" ""

[ "$fail" -eq 0 ] || { printf 'term-pane-shortcuts-test FAILED\n'; exit 1; }
printf 'term-pane-shortcuts-test ok\n'
