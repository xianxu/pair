#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
# Regression test for the three-pane workbench shortcut gates (#116).
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
export PAIR_TEST_ZELLIJ_REPLIES="$ROOT/tests/lib/zellij_fixture_replies.py"
layout_terminal_shell=$(grep 'exec pair term' "$ROOT/zellij/layouts/main-3.kdl" | sed 's/^[[:space:]]*args "-c" "//; s/"$//; s/\\"/"/g')
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pair-term-shortcuts.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

export PAIR_HOME="$tmp/home"
export PAIR_DATA_DIR="$tmp/data"
export PAIR_TAG=t
export PAIR_AGENT=codex
export ZELLIJ_SESSION_NAME=fixture-term
export PAIR_TERM_FIXTURE_ROOT="$tmp"
mkdir -p "$PAIR_HOME/bin" "$PAIR_DATA_DIR"
export PAIR_DRAFT_PATH="$tmp/draft.md"
export PAIR_DRAFT_PANE_PATH="$PAIR_DATA_DIR/draft-pane-t.json"
export PAIR_LAST_LEFT_PANE_PATH="$PAIR_DATA_DIR/last-left-pane-t"
export PAIR_LAYOUT_MODE_PATH="$PAIR_DATA_DIR/layout-mode-t"
cp "$ROOT/bin/pair" "$PAIR_HOME/bin/pair"

fakebin="$tmp/fakebin"
mkdir -p "$fakebin"
cat > "$fakebin/zellij" <<'PYTHON'
#!/usr/bin/env python3
import json, os, pathlib, sys
sys.path.insert(0, str(pathlib.Path(os.environ['PAIR_TEST_ZELLIJ_REPLIES']).parent))
from zellij_fixture_replies import emit, case_payload
root = pathlib.Path(os.environ['PAIR_TERM_FIXTURE_ROOT'])
args = sys.argv[1:]
if args[:1] == ['action']:
    command, params = args[1], args[2:]
    panes = json.loads((root / 'panes.json').read_text())
    if command in {'list-panes', 'current-tab-info'}:
        with (root / 'queries.log').open('a') as log:
            log.write(' '.join(args[1:]) + '\n')
        if command == 'list-panes':
            assert {'--json', '--command', '--state', '--geometry'} <= set(params)
            emit('panes', panes)
        else:
            assert params == ['--json']
            tab = case_payload('term-shortcuts.tab-normal')
            tab.update(is_fullscreen_active=any(p['is_fullscreen'] for p in panes),
                       are_floating_panes_visible=any(p['is_floating'] for p in panes))
            emit('tab', tab)
        sys.exit(0)
    with (root / 'actions.log').open('a') as log:
        log.write(' '.join(args[1:]) + '\n')
    if command == 'focus-pane-id':
        target = int(params[0])
        assert any(p['id'] == target for p in panes)
        for pane in panes:
            pane['is_focused'] = pane['id'] == target
    elif command in {'toggle-fullscreen', 'toggle-pane-embed-or-floating', 'change-floating-pane-coordinates'}:
        target = int(params[params.index('--pane-id') + 1])
        selected = [p for p in panes if p['id'] == target]
        assert len(selected) == 1
        pane = selected[0]
        if command == 'toggle-fullscreen':
            pane['is_fullscreen'] = not pane['is_fullscreen']
        elif command == 'toggle-pane-embed-or-floating':
            pane['is_floating'] = not pane['is_floating']
        else:
            assert pane['is_floating']
            # The known 150x51 fixture implements the requested focus rectangle.
            assert params == ['--pane-id', str(target), '-x', '12%', '-y', '0', '--width', '75%', '--height', '100%', '--pinned', 'true']
            pane.update(pane_x=18, pane_y=0, pane_columns=112, pane_rows=51)
    elif command == 'show-floating-panes':
        assert params == ['--tab-id', '2']
    else:
        assert command in {'write', 'write-chars', 'new-pane'}, command
    (root / 'panes.json').write_text(json.dumps(panes))
    sys.exit(0)
if args[:1] == ['run']:
    with (root / 'actions.log').open('a') as log:
        log.write(' '.join(args) + '\n')
    sys.exit(0)
raise AssertionError('unexpected fixture command: ' + repr(args))
PYTHON
chmod +x "$fakebin/zellij"
export PATH="$fakebin:$PAIR_HOME/bin:$PATH"

write_panes() {
  focus="$1"
  case "$focus" in
    agent) export ZELLIJ_PANE_ID=1 ;;
    draft) export ZELLIJ_PANE_ID=2 ;;
    terminal) export ZELLIJ_PANE_ID=4 ;;
    review) export ZELLIJ_PANE_ID=5 ;;
    *) printf 'unknown fixture focus: %s\n' "$focus" >&2; exit 1 ;;
  esac
  case_name="$focus"
  if [ "${2:-false}" = true ]; then case_name=fullscreen-terminal; fi
  python3 "$PAIR_TEST_ZELLIJ_REPLIES" --case "term-shortcuts.$case_name" > "$tmp/panes.json"
}

run_shortcut() {
  rm -f "$tmp/actions.log" "$tmp/queries.log"
  "$PAIR_HOME/bin/pair" term --test-shortcut "$1" >/dev/null
}

run_shortcut_with_stdin() {
  rm -f "$tmp/actions.log" "$tmp/queries.log"
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
check_eq "right Alt+Shift+d splits terminal down as a native tiled split" "$(actions)" "new-pane --direction down --name terminal -- sh -c $layout_terminal_shell"

# #216/#243: the from-any-pane right-terminal chords, driven from the DRAFT's
# focus. They now deliver the GLOBAL chord bytes (Alt+Shift+Left/Right/t =
# ESC[1;4D / ESC[1;4C / ESC[84;4u), not the role-scoped Alt+Left/Right (;3):
# #227 passes a role-scoped chord THROUGH to a full-screen right pane, so the
# delivery must use a global pair term always handles (#243). handleTerminalChord
# short-circuits these inside the right pane, so `--test-shortcut` is the only
# reachable caller of runDecision's case.
write_panes draft
run_shortcut "Alt+Shift+Left"
check_eq "Alt+Shift+Left delivers previous-tab into the right terminal" "$(actions)" "write --pane-id 4 27 91 49 59 52 68"

write_panes draft
run_shortcut "Alt+Shift+Right"
check_eq "Alt+Shift+Right delivers next-tab into the right terminal" "$(actions)" "write --pane-id 4 27 91 49 59 52 67"

write_panes draft
run_shortcut "Alt+Shift+t"
check_eq "Alt+Shift+t delivers new-tab into the right terminal (#243)" "$(actions)" "write --pane-id 4 27 91 56 52 59 52 117"

# No focus action of any kind: the cursor must not move.
write_panes draft
run_shortcut "Alt+Shift+Left"
check_eq "Alt+Shift+Left moves no focus" "$(actions | grep -c 'focus' || true)" "0"

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

# #297/#417: from normal the draft press enters focus (the tab info records the
# swap layout to re-tile onto); collapse from the fullscreen terminal restores
# the original caller. The fake pane report models each observed state.
write_panes draft
ZELLIJ_PANE_ID=2 run_shortcut "Alt+Shift+Enter"
check_eq "draft Alt+Shift+Enter enters focus on the right terminal" "$(actions)" "toggle-pane-embed-or-floating --pane-id 4
change-floating-pane-coordinates --pane-id 4 -x 12% -y 0 --width 75% --height 100% --pinned true
show-floating-panes --tab-id 2"
check_eq "focus entry confirms current tab independently of effects" "$(grep -c '^current-tab-info --json$' "$tmp/queries.log" || true)" "2"
check_eq "focus entry confirms pane context before effects" "$(grep -c '^list-panes ' "$tmp/queries.log" || true)" "2"

write_panes terminal true
ZELLIJ_PANE_ID=4 run_shortcut "Alt+Shift+Enter"
check_eq "right Alt+Shift+Enter exits fullscreen and restores draft focus" "$(actions)" "toggle-fullscreen --pane-id 4
focus-pane-id 2"

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
#
# The third grep is a REGEX, not a fixed string, and that is the point: it used
# to be `pane name="terminal" {`, which pinned the pane's exact SPELLING as a
# proxy for its POSITION in the tree. #199 M4 added `borderless=true` to that
# declaration -- a change to chrome, not to tiling -- and the assertion failed
# for a property it does not test. What "tiled" actually means here is the two
# negations above plus a terminal pane declared with a command block; attributes
# between the name and the brace are none of this check's business.
! grep -Fq 'floating_panes' "$ROOT/zellij/layouts/main-3.kdl" \
  && ! grep -Fq 'terminal-filler' "$ROOT/zellij/layouts/main-3.kdl" \
  && grep -Eq 'pane name="terminal"[^{]*\{' "$ROOT/zellij/layouts/main-3.kdl" \
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
  && { printf 'FAIL Zellij mouse support is disabled (kills click-to-focus rescue + the copy-on-select clipboard mirror, #123 lockout)\n'; fail=1; } \
  || pass "Zellij mouse support stays enabled"

grep -Fq "'pair', 'layout', 'focus-terminal'" "$ROOT/nvim/init.lua" \
  && pass "draft Alt+k focuses terminal by pane id" \
  || { printf 'FAIL draft Alt+k does not use pair layout focus-terminal\n'; fail=1; }

grep -Fq '"move-focus", "right"' "$ROOT/cmd/internal/wrapcmd/wrap.go" \
  && { printf 'FAIL agent Alt+k still uses relative move-focus right (must target a split half by id)\n'; fail=1; } \
  || pass "agent Alt+k no longer uses relative move-focus right"

# The split must not pass --borderless AT THE CALL SITE. The reason changed with
# pair#199 M4 and the assertion outlived it, which is why it is restated here
# rather than left to read as the old rationale:
#
#   BEFORE M4 — frames were wanted, so opting out was the bug.
#   AFTER  M4 — the terminal IS borderless, but the LAYOUT says so, at all nine
#               `name="terminal"` rungs, where TestEveryTerminalPaneRungIsBorderless
#               enumerates it. A `--borderless` flag on `new-pane` would be a
#               SECOND place declaring the same thing, and the two would drift.
#
# So the check is unchanged and its meaning is inverted: one declaration, in the
# layout, not two.
! grep -Fq '"--borderless"' "$ROOT/cmd/internal/termcmd/run.go" \
  && pass "right terminal split leaves borderless to the layout, not the call site" \
  || { printf 'FAIL right terminal split passes --borderless; the layout already declares it at all nine rungs\n'; fail=1; }

grep -Fq "$layout_terminal_shell" "$ROOT/cmd/internal/workbenchshortcut/terminal_command.go" \
  && grep -Fq 'const rightTerminalPaneShell = workbenchshortcut.RightTerminalPaneShell' "$ROOT/cmd/internal/termcmd/run.go" \
  && grep -Fq 'workbenchshortcut.RightTerminalPaneShell' "$ROOT/cmd/internal/layoutcmd/layoutcmd.go" \
  && pass "split and recovery share the layout3 terminal command" \
  || { printf 'FAIL shared right terminal command or its consumers drifted from layout3\n'; fail=1; }

grep -Fq 'support_kitty_keyboard_protocol true' "$ROOT/zellij/config.kdl" \
  && pass "Zellij explicitly enables Kitty keyboard protocol" \
  || { printf 'FAIL Zellij Kitty keyboard protocol is not enabled\n'; fail=1; }

! grep -Fq "'session-inventory', '--activity', '--agent'" "$ROOT/nvim/init.lua" \
  && pass "Neovim confirmations do not block on inventory activity" \
  || { printf 'FAIL Neovim confirmation still blocks on inventory activity\n'; fail=1; }

if grep -Eq '\.codex/sessions|stat -f.*session|find .*sessions' "$ROOT/nvim/init.lua"; then
  printf 'FAIL Neovim retains native session path/stat discovery\n'; fail=1
else
  pass "Neovim has no native session path/stat fallback"
fi

[ "$fail" -eq 0 ] || { printf 'term-pane-shortcuts-test FAILED\n'; exit 1; }
printf 'term-pane-shortcuts-test ok\n'
